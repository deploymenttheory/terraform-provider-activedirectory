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

data "activedirectory_group_member" "test" {
  group_guid  = "11111111-1111-4111-8111-111111111111"
  member_guid = "22222222-2222-4222-8222-222222222222"
}
