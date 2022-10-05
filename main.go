package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github/aciba90/clean-cloud/internal/azure"
	"github/aciba90/clean-cloud/internal/gcp"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/alecthomas/kong"
)

type Context struct {
	Debug bool
}

type GcpListCmd struct{}

func (r *GcpListCmd) Run(ctx *Context) error {
	var projectID = os.Getenv("PROJECT_ID")
	if len(projectID) == 0 {
		log.Fatal("PROJECT_ID is not set.")
	}
	gcp.ListAllInstances(projectID)
	return nil
}

type GcpCmd struct {
	List  GcpListCmd `cmd:"" help:"List elements to clean"`
	Clean struct{}   `cmd:"" help:"Clean elements"`
}

func (r *GcpCmd) Run(ctx *Context) error {
	return nil
}

type AzureListCmd struct{}

func (r *AzureListCmd) Run(ctx *Context) error {
	subscriptionID := os.Getenv("AZURE_SUBSCRIPTION_ID")
	if len(subscriptionID) == 0 {
		log.Fatal("AZURE_SUBSCRIPTION_ID is not set.")
	}
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		log.Fatal(err)
	}
	ctx_2 := context.TODO()

	resourceGroups, err := azure.ListResourceGroup(ctx_2, cred, subscriptionID)
	if err != nil {
		log.Fatal(err)
	}

	if len(resourceGroups) > 0 {
		fmt.Println("Resource groups:")
		for _, rg := range resourceGroups {
			fmt.Println(rg)
		}
	} else {
		fmt.Println("No Resource groups")
	}

	return nil
}

type AzureCmd struct {
	List  AzureListCmd `cmd:"" help:"List elements to clean"`
	Clean struct{}     `cmd:"" help:"Clean elements"`
}

func (r *AzureCmd) Run(ctx *Context) error {
	return nil
}

var cli struct {
	Debug bool     `help:"Enable debug mode"`
	Gcp   GcpCmd   `cmd:"" help:"GCP commands"`
	Azure AzureCmd `cmd:"" help:"Azure commands"`
}

func main() {
	ctx := kong.Parse(&cli)
	err := ctx.Run(&Context{Debug: cli.Debug})
	ctx.FatalIfErrorf(err)
}
