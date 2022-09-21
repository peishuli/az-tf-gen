/************************************************************************************************************************************
* References:
  https://github.com/Azure-Samples/azure-sdk-for-go-samples/blob/main/sdk/resourcemanager/resource/resourcegroups/main.go
  https://pkg.go.dev/github.com/hashicorp/hcl/v2/hclwrite#pkg-overview
*************************************************************************************************************************************/

package main

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	// "github.com/hashicorp/hcl2/hclwrite" //DO NOT use this package, generated blocks from it don't have nice line breaks/indentations
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"

	"log"
	"os"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
)

var (
	subscriptionID    string
	location          = "South Central US"
	resourceGroupName = "pli-demo-rg"
)

func main() {
	subscriptionID = os.Getenv("AZURE_SUBSCRIPTION_ID")
	if len(subscriptionID) == 0 {
		log.Fatal("AZURE_SUBSCRIPTION_ID is not set.")
	}

	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		log.Fatal(err)
	}
	ctx := context.Background()

	resourceGroup, err := getResourceGroup(ctx, cred)
	if err != nil {
		log.Fatal(err)
	}
	writeTFProviders()
	writeTFConfig(resourceGroup)
}

func writeTFProviders() {
	tfFile, err := os.Create("providers.tf")
	if err != nil {
		log.Println(err)
		return
	}

	f := hclwrite.NewEmptyFile()
	rootBody := f.Body()
	
	tfBlock := rootBody.AppendNewBlock("terraform", nil)
	tfBody := tfBlock.Body()
	requiredProviderBlock := tfBody.AppendNewBlock("required_providers", nil)
	requiredProviderBody := requiredProviderBlock.Body()
	requiredProviderBody.SetAttributeValue("azurerm", cty.ObjectVal(map[string]cty.Value{
		"source":  cty.StringVal("hashicorp/azurerm"),
		"version": cty.StringVal("3.23.0"),
	}))

	rootBody.AppendNewline()

	providerBlock := rootBody.AppendNewBlock("provider", []string{"azurerm"})
	providerBody := providerBlock.Body()
	providerBody.AppendNewBlock("features", nil)

	tfFile.Write(f.Bytes())

}

func writeTFConfig(rg *armresources.ResourceGroup) {
	tfFile, err := os.Create("main.tf")
	if err != nil {
		log.Println(err)
		return
	}
	f := hclwrite.NewEmptyFile()
	rootBody := f.Body()

	rgBlock := rootBody.AppendNewBlock("resource", []string{"azurerm_resource_group", "this"})
	rgBody := rgBlock.Body()
	rgBody.SetAttributeValue("name", cty.StringVal(*rg.Name))
	// rgBody.SetAttributeValue("name", cty.StringVal(fmt.Sprintf("new-%s", *rg.Name)))
	rgBody.SetAttributeValue("location", cty.StringVal(*rg.Location))

	tags := make(map[string]cty.Value)
	for k, v := range rg.Tags {
		tags[k] = cty.StringVal(*v)
	}

	rgBody.SetAttributeValue("tags", cty.ObjectVal(tags))

	// log.Printf("%s", f.Bytes())
	tfFile.Write(f.Bytes())
}

func getResourceGroup(ctx context.Context, cred azcore.TokenCredential) (*armresources.ResourceGroup, error) {
	resourceGroupClient, err := armresources.NewResourceGroupsClient(subscriptionID, cred, nil)
	if err != nil {
		return nil, err
	}

	resourceGroupResp, err := resourceGroupClient.Get(ctx, resourceGroupName, nil)
	if err != nil {
		return nil, err
	}
	return &resourceGroupResp.ResourceGroup, nil
}
