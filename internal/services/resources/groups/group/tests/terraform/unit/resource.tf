terraform {
  required_providers {
    activedirectory = { source = "deploymenttheory/activedirectory" }
  }
}

provider "activedirectory" {
  host        = "dc.unit.invalid"
  username    = "fixture@unit.invalid"
  password    = "synthetic-bind-password"
  auth_method = "simple"
  transport   = "ldaps"
}

resource "activedirectory_group" "test" {
  name             = "initial"
  sam_account_name = "fixture-group"
  parent_dn        = "OU=first,DC=unit,DC=invalid"
  category         = "security"
  scope            = "global"
  description      = "first description"
}
