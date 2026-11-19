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

data "activedirectory_group_member_relationships" "test" {
  group_guid   = "11111111-1111-4111-8111-111111111111"
  result_limit = 10
}
