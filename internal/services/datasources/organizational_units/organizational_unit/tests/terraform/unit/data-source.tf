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
data "activedirectory_organizational_unit" "test" {
  object_guid = "a3e9767a-200e-43dd-bb1d-e5ab400e5d6f"
}
