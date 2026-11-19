terraform {
  required_providers {
    activedirectory = { source = "deploymenttheory/activedirectory" }
  }
}
provider "activedirectory" {
  auth_method = "simple"
  transport   = "ldaps"
}
variable "object_guid" { type = string }
data "activedirectory_object" "test" {
  object_guid = var.object_guid
  properties  = ["description", "jpegPhoto", "telephoneNumber"]
}
