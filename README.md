# An Azure Terrafy Inspired Sample Application
## Overview
This little sample GoLang app pulls resource group data from Azure and generates Terraform code. It was inspired by [Azure Terrafy](https://github.com/Azure/aztfy) and intended to overcome a few Azure Terrafy limitations such as:
- Generating refactored .tf files that follows Terraform best practices instead a single monolithic main.tf 
- Promoting better naming/labelling in  (instead of resource_1, resource_2, etc. )
- Using variables.tf file instead of hard coding properties in resource .tf files.
- Generating Terragrunt .hcl files

The following screenshot shows an existing resource group in Azure:

![An AZ RG Screenshot](./rg.png)

To run the sample:
1. Set up environment variables (with data from a Service Principle):
```bash
export AZURE_SUBSCRIPTION_ID="......"
export AZURE_TENANT_ID="......"
export AZURE_CLIENT_ID="......"
export AZURE_CLIENT_SECRET="......"
```

2. Modify the localtion and resourceGroupName variables in main.go (line 26 & 27):
```go
var (
	subscriptionID    string
	location          = "South Central US"
	resourceGroupName = "pli-demo-rg"
)
```
3. Run `go run main.go` and it will generates two Terraform configuration files, providers.tf and main.tf, respectively.

## Generated Sample Terraform Code:
```hcl
#providers.tf
terraform {
  required_providers {
    azurerm = {
      source  = "hashicorp/azurerm"
      version = "3.23.0"
    }
  }
}

provider "azurerm" {
  features {
  }
}

# main.tf
resource "azurerm_resource_group" "this" {
  name     = "pli-demo-rg"
  location = "southcentralus"
  tags = {
    CostCenter  = "123456789"
    Environment = "Demo"
  }
}

# variables.tf
variable "rg_name" {
  type = string
}
variable "location" {
  type = string
}
variable "costcenter" {
  type = string
}

# default.tfvars
rg_name    = "pli-rg-demogroup"
location   = "eastus2"
costcenter = "12345678"

```

## Generated Sample Terragrunt Code:
```hcl
# terragrunt.hcl
terraform {
  source = "../module"
}

inputs = {
  costcenter = "12345678"
  location   = "eastus2"
  rg_name    = "pli-rg-demogroup"
}

```
