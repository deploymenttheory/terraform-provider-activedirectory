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

data "activedirectory_account_settings" "by_guid" {
  object_guid = "dc93656c-f745-4926-9bbb-31e7bbecb68b"
}

data "activedirectory_account_settings" "by_dn" {
  dn = "CN=Account,OU=owned,DC=unit,DC=invalid"
}
