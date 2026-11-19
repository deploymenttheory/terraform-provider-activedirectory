terraform {
  required_version = ">= 1.14.0"
  required_providers {
    activedirectory = { source = "deploymenttheory/activedirectory" }
  }
}

provider "activedirectory" {}

resource "activedirectory_organizational_unit" "test" {
  name                               = __NAME__
  parent_dn                          = __ROOT_DN__
  display_name                       = __MARKER__
  description                        = "owned lab OU"
  protected_from_accidental_deletion = __PROTECTED__
}
