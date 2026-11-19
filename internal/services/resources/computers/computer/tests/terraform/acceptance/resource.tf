terraform {
  required_version = ">= 1.14.0"
  required_providers {
    activedirectory = { source = "deploymenttheory/activedirectory" }
  }
}

variable "computer_password" {
  type      = string
  sensitive = true
  ephemeral = true
}

provider "activedirectory" {
  auth_method = "simple"
  transport = "ldaps"
}

resource "activedirectory_computer" "test" {
  provider            = __PROVIDER__
  name                = __NAME__
  sam_account_name    = __SAM__
  user_principal_name = __UPN__
  dns_host_name = __DNS__
  parent_dn           = __PARENT__
  display_name        = __MARKER__
  __DESCRIPTION_BLOCK__
  enabled             = __ENABLED__
  __PASSWORD_BLOCK__
  account_expires     = __EXPIRATION__
}

data "activedirectory_computer" "lookup" {
  object_guid = activedirectory_computer.test.id
}

data "activedirectory_computers" "scoped" {
  base_dn      = __ROOT_DN__
  ldap_filter  = __FILTER__
  page_size    = 2
  result_limit = 10
  depends_on   = [activedirectory_computer.test]
}
