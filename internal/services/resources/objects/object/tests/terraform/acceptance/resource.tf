terraform {
  required_version = ">= 1.14.0"
  required_providers {
    activedirectory = { source = "deploymenttheory/activedirectory" }
  }
}
provider "activedirectory" {
  auth_method = "simple"
  transport   = "ldaps"
}
resource "activedirectory_object" "test" {
  name             = __NAME__
  parent_dn        = __PARENT__
  object_class     = __CLASS__
  attributes       = __ATTRIBUTES__
  add_values       = __ADD_VALUES__
  remove_values    = __REMOVE_VALUES__
  clear_attributes = __CLEAR_ATTRIBUTES__
  properties       = ["description", "displayName", "thumbnailPhoto", "manager", "otherTelephone", "telephoneNumber"]
}
data "activedirectory_object" "lookup" {
  object_guid = activedirectory_object.test.id
  properties  = ["description", "displayName", "thumbnailPhoto", "manager", "otherTelephone"]
}
data "activedirectory_objects" "scoped" {
  base_dn      = __ROOT_DN__
  ldap_filter  = __FILTER__
  page_size    = 2
  result_limit = 10
  properties   = ["description", "displayName", "thumbnailPhoto", "manager", "otherTelephone"]
  depends_on   = [activedirectory_object.test]
}
data "activedirectory_object" "by_dn" {
  dn         = activedirectory_object.test.distinguished_name
  properties = ["description", "thumbnailPhoto"]
}
data "activedirectory_objects" "paged" {
  base_dn      = __ROOT_DN__
  ldap_filter  = __ALL_FILTER__
  page_size    = 1
  result_limit = 10
  properties   = ["description"]
  depends_on   = [activedirectory_object.test]
}
data "activedirectory_objects" "limited" {
  base_dn      = __ROOT_DN__
  ldap_filter  = __ALL_FILTER__
  page_size    = 1
  result_limit = 1
  properties   = ["description"]
  depends_on   = [activedirectory_object.test]
}
