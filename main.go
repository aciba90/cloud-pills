package main
import (
    "context"
    "github.com/Azure/azure-sdk-for-go/sdk/azcore"
    "github.com/Azure/azure-sdk-for-go/sdk/azidentity"
    "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
    "log"
    "os"
)

import "fmt"

var (
	subscriptionID    string
	location          = "westus"
	resourceGroupName = "sample-resource-group"
)

func main() {
    fmt.Println("hello world")
    subscriptionID = os.Getenv("AZURE_SUBSCRIPTION_ID")
    if len(subscriptionID) == 0 {
		log.Fatal("AZURE_SUBSCRIPTION_ID is not set.")
	}
    cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		log.Fatal(err)
	}
	ctx := context.Background()

    resourceGroups, err := listResourceGroup(ctx, cred)
	if err != nil {
		log.Fatal(err)
	}
	for _, resource := range resourceGroups {
		log.Printf("Deleting Resource Group Name: %s,ID: %s", *resource.Name, *resource.ID)
        error := deleteResourceGroup(ctx, cred, *resource.Name)
        if error != nil {
            log.Fatal(err)
        }
	}
}

func listResourceGroup(ctx context.Context, cred azcore.TokenCredential) ([]*armresources.ResourceGroup, error) {
	resourceGroupClient, err := armresources.NewResourceGroupsClient(subscriptionID, cred, nil)
	if err != nil {
		return nil, err
	}

	resultPager := resourceGroupClient.NewListPager(nil)

	resourceGroups := make([]*armresources.ResourceGroup, 0)
	for resultPager.More() {
		pageResp, err := resultPager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		resourceGroups = append(resourceGroups, pageResp.ResourceGroupListResult.Value...)
	}
	return resourceGroups, nil
}

func deleteResourceGroup(ctx context.Context, cred azcore.TokenCredential, resourceGroupName string) error {
	resourceGroupClient, err := armresources.NewResourceGroupsClient(subscriptionID, cred, nil)
	if err != nil {
		return err
	}

	pollerResp, err := resourceGroupClient.BeginDelete(ctx, resourceGroupName, nil)
	if err != nil {
		return err
	}

	_, err = pollerResp.PollUntilDone(ctx, nil)
	if err != nil {
		return err
	}
	return nil
}
