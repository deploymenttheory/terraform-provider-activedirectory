terraform {
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
data "activedirectory_groups" "test" {
  base_dn      = "DC=unit,DC=invalid"
  scope        = "one_level"
  page_size    = 2
  result_limit = 10
}
