terraform {
  required_providers {
    activedirectory = { source = "deploymenttheory/activedirectory" }
  }
}
provider "activedirectory" {
  auth_method = "simple"
  transport   = "ldaps"
}
variable "parent_dn" { type = string }
data "activedirectory_objects" "test" {
  base_dn      = var.parent_dn
  ldap_filter  = "(objectClass=contact)"
  scope        = "subtree"
  page_size    = 2
  result_limit = 10
  properties   = ["description", "jpegPhoto", "telephoneNumber"]
}
