terraform {
  required_version = ">= 1.14.0"

  required_providers {
    activedirectory = {
      source = "deploymenttheory/activedirectory"
    }
  }
}

# The foundation has no directory services or authentication configuration yet.
provider "activedirectory" {}
