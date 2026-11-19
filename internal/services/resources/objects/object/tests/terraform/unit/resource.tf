terraform {
  required_providers {
    activedirectory = { source = "deploymenttheory/activedirectory" }
  }
}
provider "activedirectory" {
  host        = "unit.invalid"
  auth_method = "simple"
  transport   = "ldaps"
  username    = "unit-fixture"
  password    = "not-a-real-credential"
}
resource "activedirectory_object" "test" {
  name         = "fixture"
  parent_dn    = "OU=first,DC=unit,DC=invalid"
  object_class = "contact"
  attributes   = { description = ["Zmlyc3Q="], thumbnailPhoto = ["/wCA"] }
  add_values   = { otherTelephone = ["b2xk", "a2VlcA=="] }
  properties   = ["telephoneNumber"]
}
