#!/usr/bin/env sh

set -ex

instances=$(gcloud compute instances list --format "table(name)" | tail -n +1)
if [ -n "$instances" ]; then
  echo "deleting $(echo "$instances" | wc -l) instances"
  xargs "$instances" gcloud compute instances delete --delete-disks=all --
else
  echo nothing to delete
fi

