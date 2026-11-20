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

resource "activedirectory_computer_service_account" "test" {
  computer_guid        = "11111111-1111-4111-8111-111111111111"
  service_account_guid = "22222222-2222-4222-8222-222222222222"
}
