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
data "activedirectory_object" "test" {
  properties  = ["description", "jpegPhoto", "telephoneNumber", "member"]
  object_guid = "dc93656c-f745-4926-9bbb-31e7bbecb68b"
}
