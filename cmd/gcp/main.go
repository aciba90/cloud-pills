package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	compute "cloud.google.com/go/compute/apiv1"
	"google.golang.org/api/iterator"
	computepb "google.golang.org/genproto/googleapis/cloud/compute/v1"
	"google.golang.org/protobuf/proto"
)

// ListAllInstances prints all instances present in a project, grouped by their zone.
func ListAllInstances(projectID string) error {
	// projectID := "your_project_id"
	ctx := context.Background()
	instancesClient, err := compute.NewInstancesRESTClient(ctx)
	if err != nil {
		return fmt.Errorf("NewInstancesRESTClient: %v", err)
	}
	defer instancesClient.Close()

	// Use the `MaxResults` parameter to limit the number of results that the API returns per response page.
	req := &computepb.AggregatedListInstancesRequest{
		Project:    projectID,
		MaxResults: proto.Uint32(3),
	}

	it := instancesClient.AggregatedList(ctx, req)

	log.Printf("Instances found:\n")
	// Despite using the `MaxResults` parameter, you don't need to handle the pagination
	// yourself. The returned iterator object handles pagination
	// automatically, returning separated pages as you iterate over the results.
	for {
		pair, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return err
		}
		instances := pair.Value.Instances
		if len(instances) > 0 {
			log.Printf("%s\n", pair.Key)
			for _, instance := range instances {
				log.Printf("- %s %s\n", instance.GetName(), instance.GetMachineType())
			}
		}
	}
	return nil
}

// DeleteInstance sends a delete request to the Compute Engine API and waits for it to complete.
func DeleteInstance(projectID, zone, instanceName string) error {
	// projectID := "your_project_id"
	// zone := "europe-central2-b"
	// instanceName := "your_instance_name"
	ctx := context.Background()
	instancesClient, err := compute.NewInstancesRESTClient(ctx)
	if err != nil {
		return fmt.Errorf("NewInstancesRESTClient: %v", err)
	}
	defer instancesClient.Close()

	req := &computepb.DeleteInstanceRequest{
		Project:  projectID,
		Zone:     zone,
		Instance: instanceName,
	}

	op, err := instancesClient.Delete(ctx, req)
	if err != nil {
		return fmt.Errorf("unable to delete instance: %v", err)
	}

	if err = op.Wait(ctx); err != nil {
		return fmt.Errorf("unable to wait for the operation: %v", err)
	}

	log.Printf("Instance deleted\n")

	return nil
}

func DeleteAllInstances(projectID string) error {
	ctx := context.Background()
	instancesClient, err := compute.NewInstancesRESTClient(ctx)
	if err != nil {
		return fmt.Errorf("NewInstancesRESTClient: %v", err)
	}
	defer instancesClient.Close()

	// Use the `MaxResults` parameter to limit the number of results that the API returns per response page.
	req := &computepb.AggregatedListInstancesRequest{
		Project:    projectID,
		MaxResults: proto.Uint32(3),
	}

	it := instancesClient.AggregatedList(ctx, req)

	for {
		pair, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return err
		}
		instances := pair.Value.Instances
		tmp := strings.Split(pair.Key, "/")
		zone := tmp[len(tmp) -1]
		if len(instances) > 0 {
			log.Printf("%s\n", pair.Key)
			for _, instance := range instances {
				log.Printf("- Deleting %s %s\n", instance.GetName(), instance.GetMachineType())

				err := DeleteInstance(projectID, zone, instance.GetName())
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

type Config struct {
	ProjectId string `json:"project_id"`
}

func main() {
	f, err := os.ReadFile(os.Getenv("HOME") + "/.config/clean_cloud.json")
	if err != nil {
		log.Println(err)
	}
	cfg := Config{}
	json.Unmarshal([]byte(f), &cfg)
	log.Println(cfg.ProjectId)
	err = DeleteAllInstances(cfg.ProjectId)
	if err != nil {
		log.Println(err)
	}

	println("Hello GCP")
}
