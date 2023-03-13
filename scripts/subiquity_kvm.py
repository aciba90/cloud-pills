#!/usr/bin/env python3

import logging
import tempfile
import shutil
import os
from pathlib import Path
import requests
import subprocess
import sys
from typing import Optional, Tuple

LOG = logging.getLogger(__name__)


def cloud_localds(
    tmpdir: Path,
    user_data: str,
    meta_data: str,
    vendor_data: Optional[str] = None,
    network_config: Optional[str] = None,
) -> Path:
    """Create a CIDATA disk image containing NoCloud meta-data/user-data

    This image can be mounted as a disk in qemu-kvm to provide #cloud-config
    """

    img_path = tmpdir.joinpath("my-seed.img")
    ud_path = tmpdir.joinpath("user-data")
    md_path = tmpdir.joinpath("meta-data")
    cmd = ["cloud-localds", img_path, ud_path, md_path]
    ud_path.write_text(user_data)
    md_path.write_text(meta_data)
    if vendor_data:
        tmpdir.joinpath("vendor-data").write_text(vendor_data)
        cmd += ["-v", tmpdir.joinpath("vendor-data")]
    if network_config:
        tmpdir.joinpath("network-config").write_text(network_config)
        cmd += ["-N", tmpdir.joinpath("network-config")]
    subprocess.run(cmd)
    return img_path


def create_qemu_disk(tmpdir: Path, size: str):
    img_path = tmpdir.joinpath("image.iso")
    if img_path.exists():
        LOG.debug("Reusing %s", path)
    else:
        subprocess.run(["truncate", "-s", size, img_path])
    return img_path


from enum import Enum


class InstallFlavor(Enum):
    DESKTOP = "desktop"
    LIVE_SERVER = "live-server"


def get_release_iso(
    distro: str,
    release: str,
    flavor: InstallFlavor = InstallFlavor.LIVE_SERVER,
    arch: Optional[str] = "amd64",
) -> Path:
    base_name = f"{release}-{flavor.value}-{arch}"
    flavor_subdir = "ubuntu-server/" if flavor == InstallFlavor.LIVE_SERVER else ""
    iso_base_url = f"https://cdimage.ubuntu.com/{flavor_subdir}daily-live/current/"
    manifest_url = f"{iso_base_url}{base_name}.manifest"
    iso_url = f"{iso_base_url}{base_name}.iso"
    iso_path = Path(f"/srv/iso/{release}/{base_name}.iso")
    manifest_path = Path(f"/srv/iso/{release}/{base_name}.manifest")
    manifest_sum = b"ABSENT"
    if manifest_path.exists():
        p = subprocess.run(["md5sum", manifest_path], capture_output=True)
        manifest_sum = p.stdout
    r = requests.get(manifest_url, allow_redirects=True)
    manifest_path.write_bytes(r.content)
    if iso_path.exists():
        LOG.info(f"Checking md5sum of {manifest_path} for stale local ISO.")
        p = subprocess.run(["md5sum", manifest_path], capture_output=True)
        if p.stdout == manifest_sum:
            LOG.info(f"Using cached {iso_path} no manifest changes.")
            return iso_path
    LOG.info(f"Downloading {iso_url}...")
    with requests.get(iso_url, stream=True) as r:
        total_len = int(r.headers.get("Content-Length"))
        r.raise_for_status()
        total_chunks = total_len / 8192
        print_step = total_chunks // 25
        with open(iso_path, "wb") as stream:
            for idx, chunk in enumerate(r.iter_content(chunk_size=8192)):
                if idx % print_step == 0:
                    print(
                        f"{idx/print_step * 4}%"
                        f" of {total_len/1024/1024/1000:.2f} downloaded"
                    )
                stream.write(chunk)
    return iso_path


def extract_kernel_initrd_from_iso(tmpdir: Path, iso_path: Path) -> Tuple[Path, Path]:
    """Mount iso_path in tmpdir and extract vmlinuz and initrd to tmpdir."""
    mnt_path = tmpdir.joinpath("mnt")
    mnt_path.mkdir()
    cmd = f"mount {iso_path} -o loop {mnt_path}"
    LOG.info(f"cmd: {cmd}")
    kernel_path = tmpdir.joinpath("vmlinuz")
    initrd_path = tmpdir.joinpath("initrd")
    subprocess.run(cmd.split(), capture_output=True)
    shutil.copy(mnt_path.joinpath("casper/vmlinuz"), kernel_path)
    shutil.copy(mnt_path.joinpath("casper/initrd"), initrd_path)
    umount_cmd = f"umount {mnt_path}"
    LOG.info(f"cmd: {umount_cmd}")
    subprocess.run(umount_cmd.split(), capture_output=True)
    return (kernel_path, initrd_path)


def launch_kvm(
    tmpdir: Path,
    ram_size: str,
    disk_img_path: Path,
    iso_path: Optional[Path] = None,
    seed_path: Optional[Path] = None,
    kernel_cmdline: Optional[str] = "",
    cmdline: Optional[list] = None,
):
    """use qemu-kvm to setup and launch a test VM with optional kernel params"""
    cmd = [
        "kvm",
        "-no-reboot",
        "-m",
        ram_size,
        "-drive",
        f"file={disk_img_path},format=raw,cache=none,if=virtio",
        "-net",
        "nic",
        "-net",
        "user,hostfwd=tcp::2222-:22",
    ]
    if iso_path:
        cmd += ["-cdrom", iso_path]
    if seed_path:
        cmd += ["-drive", f"file={seed_path},format=raw,if=ide"]
    if kernel_cmdline:
        # Mount and extract kernel and initrd from iso
        (kernel_path, initrd_path) = extract_kernel_initrd_from_iso(tmpdir, iso_path)
        cmd += [
            "-kernel",
            kernel_path,
            "-initrd",
            initrd_path,
            "-append",
            kernel_cmdline,
        ]
    if cmdline:
        cmd += cmdline
    LOG.info(f"cmd: {cmd}")
    subprocess.check_output(cmd)


USER_DATA = """
#cloud-config
ssh_import_id: [chad.smith]
users:
- default
- name: ephemeral
  ssh_import_id: [chad.smith]
  sudo: ALL=(ALL) NOPASSWD:ALL
  shell: /bin/bash
"""

USER_DATA_AUTOINSTALL = (
    USER_DATA
    + """
autoinstall:
  version: 1
  user-data:
    users:
    - default
    chpasswd: { expire: False }
    ssh_pwauth: True
    hostname: ubuntu-server3
    ssh_import_id: [chad.smith]
    password: passw0rd
"""
)


def main():
    logging.basicConfig(stream=sys.stdout, level=logging.DEBUG)
    with tempfile.TemporaryDirectory() as tmpdir:
        tdir = Path(tmpdir)
        seed_path = cloud_localds(tdir, USER_DATA, meta_data="")
        disk_img_path = create_qemu_disk(tdir, "20G")
        iso_path = get_release_iso("ubuntu", "lunar", InstallFlavor.LIVE_SERVER, "amd64")
        LOG.info("Ephemeral boot: ssh ephemeral@localhost -p 2222")
        launch_kvm(
            tmpdir=tdir,
            ram_size="4G",
            iso_path=iso_path,
            # seed_path=seed_path,
            disk_img_path=disk_img_path,
            kernel_cmdline="autoinstall ds=nocloud-net;s=http://blackboxsw.com:800/",
        )
        LOG.info("First boot: ssh ubuntu@localhost -p 2222")
        launch_kvm(
            tmpdir=tdir,
            ram_size="4G",
            disk_img_path=disk_img_path,
            cmdline=["-daemonize"],
        )
        LOG.info(tmpdir)


main()
