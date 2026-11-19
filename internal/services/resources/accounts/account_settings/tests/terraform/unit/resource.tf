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

resource "activedirectory_account_settings" "test" {
  account_guid = "dc93656c-f745-4926-9bbb-31e7bbecb68b"
  # __SETTINGS__
}

data "activedirectory_account_settings" "by_guid" {
  object_guid = activedirectory_account_settings.test.id
}

data "activedirectory_account_settings" "by_dn" {
  dn = activedirectory_account_settings.test.distinguished_name
}
