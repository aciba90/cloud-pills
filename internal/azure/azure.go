package azure

import (
	"context"
	"log"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
)

func ListResourceGroup(ctx context.Context, cred azcore.TokenCredential, subscriptionID string) ([]*armresources.ResourceGroup, error) {
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

func DeleteResourceGroup(ctx context.Context, cred azcore.TokenCredential, subscriptionID string, resourceGroupName string) error {
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

func DeleteAllResourceGroups(ctx context.Context, cred azcore.TokenCredential, subscriptionID string) error {
	resourceGroups, err := ListResourceGroup(ctx, cred, subscriptionID)
	if err != nil {
		log.Fatal(err)
		return err
	}
	for _, resource := range resourceGroups {
		log.Printf("Deleting Resource Group with Name: %s, and ID: %s", *resource.Name, *resource.ID)
		error := DeleteResourceGroup(ctx, cred, subscriptionID, *resource.Name)
		if error != nil {
			log.Fatal(err)
		}
	}
	return nil
}
