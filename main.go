/************************************************************************************************************************************
References:
  https://github.com/Azure-Samples/azure-sdk-for-go-samples/blob/main/sdk/resourcemanager/resource/resourcegroups/main.go
  https://pkg.go.dev/github.com/hashicorp/hcl/v2/hclwrite#pkg-overview
  https://magodo.github.io/editing-hcl/
  https://discuss.hashicorp.com/t/hclwrite-v2-a-object-with-a-reference-value/26524
  https://stackoverflow.com/questions/67945463/how-to-use-hcl-write-to-set-expressions-with
  https://github.com/hashicorp/hcl/issues/373
  https://github.com/hashicorp/hcl/issues/442
*************************************************************************************************************************************/
package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
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
	resourceGroupName = "pli-rg-demogroup"
	
)

const module = "./generated/module"
const terragrunt = "./generated/terragrunt"

var var_infos = []VariableInfo{}

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

	var_rg := VariableInfo {
		prop_name: "name",
		var_name: "rg_name",
		var_type: "string",
		default_val: *resourceGroup.Name,
	}
	var_infos = append(var_infos, var_rg)
	
	var_location := VariableInfo{
		prop_name: "location",
		var_name: "location",
		var_type: "string",
		default_val: *resourceGroup.Location,
	}
	var_infos = append(var_infos, var_location)

	writeTFProviders()
	writeTFConfig(resourceGroup)
}

func writeTFProviders() {
	filename := fmt.Sprintf("%s/providers.tf", module)
	tfFile, err := os.Create(filename)
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
	filename := fmt.Sprintf("%s/main.tf", module)
	tfFile, err := os.Create(filename)
	if err != nil {
		log.Println(err)
		return
	}
	f := hclwrite.NewEmptyFile()
	rootBody := f.Body()

	rgBlock := rootBody.AppendNewBlock("resource", []string{"azurerm_resource_group", "this"})
	rgBody := rgBlock.Body()
	
	for _, var_info := range var_infos {
		var_name := var_info.var_name
		rgBody.SetAttributeTraversal(var_info.prop_name, hcl.Traversal{
			hcl.TraverseRoot{Name: "var"},
			hcl.TraverseAttr{Name: var_name},
		})
	}
	
	
	tags := make(map[string]cty.Value)		
	for k, v := range rg.Tags {		
		tag_var := fmt.Sprintf("${var.%s}", strings.ToLower(k))
		tags[k] = cty.StringVal(tag_var)
		var_tag := VariableInfo {
			var_name: strings.ToLower(k),
			var_type: "string",
			default_val: *v,
		}
		var_infos = append(var_infos, var_tag)
	}	
	rgBody.SetAttributeValue("tags", cty.ObjectVal(tags))
		
	WriteVariableFile()
	GenerateDefaultTFVars()
	WriteTerragruntFile()

	//TODO: temporary workaround to get rid of extras in "{$$name=var.name}"
	bodyString := string(f.Bytes())
	bodyString = strings.Replace(bodyString, "\"$${", "", -1)
	bodyString = strings.Replace(bodyString, "}\"", "", -1)
	tfFile.Write([]byte(bodyString))
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

func WriteVariableFile() {
	filename := fmt.Sprintf("%s/variables.tf", module)
	tfFile, err := os.Create(filename)
	if err != nil {
		log.Println(err)
		return
	}

	f := hclwrite.NewEmptyFile()
	rootBody := f.Body()
	rootBody.AppendNewline()
	for _,  var_info := range var_infos {		
		varBlock := *rootBody.AppendNewBlock("variable",[]string{var_info.var_name})
		// varBody := varBlock.Body()
		typeTokens := hclwrite.Tokens {
			{
				Type: hclsyntax.TokenIdent,
				Bytes: []byte(var_info.var_type),
			},
		}
		varBlock.Body().SetAttributeRaw("type", typeTokens)
		// varBody.SetAttributeValue("default", cty.StringVal(var_info.default_val)) //TODO: make it type dependent
	}
	tfFile.Write(f.Bytes())

}

func GenerateDefaultTFVars() {
	filename := fmt.Sprintf("%s/defualt.tfvars", module)
	tfFile, err := os.Create(filename)
	if err != nil {
		log.Println(err)
		return
	}
	f := hclwrite.NewEmptyFile()
	rootBody := f.Body()
	for _,  var_info := range var_infos {
		rootBody.SetAttributeValue(var_info.var_name, cty.StringVal(var_info.default_val))
	}
	tfFile.Write(f.Bytes())
}

func WriteTerragruntFile() {
	filename := fmt.Sprintf("%s/terragrunt.hcl", terragrunt)
	tfFile, err := os.Create(filename)
	if err != nil {
		log.Println(err)
		return
	}

	f := hclwrite.NewEmptyFile()
	rootBody := f.Body()
	
	// Write the terraform block
	tfBlock := rootBody.AppendNewBlock("terraform", nil)
	tfBody := tfBlock.Body()
	tfBody.SetAttributeValue("source", cty.StringVal("../module"))
	
	// Write a line break
	rootBody.AppendNewline()
	
	// Write the inputs block
	inputs := make(map[string]cty.Value)
	for _,  var_info := range var_infos {		
		inputs[var_info.var_name] = cty.StringVal(var_info.default_val)
	}
	rootBody.SetAttributeValue("inputs", cty.ObjectVal(inputs))
	tfFile.Write(f.Bytes())

}

type VariableInfo struct {
	prop_name string
	var_name string
	var_type string
	default_val string
}