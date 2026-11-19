terraform {
  required_version = ">= 1.14.0"
  required_providers {
    activedirectory = { source = "deploymenttheory/activedirectory" }
  }
}

provider "activedirectory" {}

resource "activedirectory_group" "test" {
  name             = __NAME__
  sam_account_name = __SAM__
  parent_dn        = __PARENT__
  category         = __CATEGORY__
  scope            = __SCOPE__
  display_name     = __MARKER__
  __DESCRIPTION_BLOCK__
}

data "activedirectory_group" "lookup" {
  object_guid = activedirectory_group.test.id
}

data "activedirectory_groups" "scoped" {
  base_dn      = __ROOT_DN__
  ldap_filter  = __FILTER__
  page_size    = 2
  result_limit = 10
  depends_on   = [activedirectory_group.test]
}
