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

resource "activedirectory_service_account" "test" {
  name             = __NAME__
  sam_account_name = __SAM__
  parent_dn        = __PARENT__
  kind             = "group_managed"
  dns_host_name    = __DNS__
  description      = __MARKER__
  enabled          = false
  __ATTRIBUTES__
}

data "activedirectory_service_account" "by_guid" {
  object_guid = activedirectory_service_account.test.id
}

data "activedirectory_service_account" "by_dn" {
  dn = activedirectory_service_account.test.distinguished_name
}

data "activedirectory_service_accounts" "scoped" {
  base_dn      = __ROOT_DN__
  ldap_filter  = __FILTER__
  page_size    = 1
  result_limit = 10
  depends_on   = [activedirectory_service_account.test]
}
