terraform {
  required_version = ">= 1.14.0"
  required_providers {
    activedirectory = { source = "deploymenttheory/activedirectory" }
  }
}

provider "activedirectory" {
  host        = "unit.invalid"
  auth_method = "simple"
  username    = "unit-fixture"
  password    = "not-a-real-credential"
}

resource "activedirectory_organizational_unit" "test" {
  name                               = "initial"
  parent_dn                          = "DC=unit,DC=invalid"
  description                        = "first description"
  protected_from_accidental_deletion = false
}
