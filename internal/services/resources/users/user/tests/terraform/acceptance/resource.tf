terraform {
  required_version = ">= 1.14.0"
  required_providers {
    activedirectory = { source = "deploymenttheory/activedirectory" }
  }
}

variable "user_password" {
  type      = string
  sensitive = true
  ephemeral = true
}

variable "reader_upn" {
  type    = string
  default = "unused"
}

variable "reader_password" {
  type      = string
  sensitive = true
  ephemeral = true
  default   = "unused"
}

provider "activedirectory" {}

provider "activedirectory" {
  alias    = "reader"
  username = var.reader_upn
  password = var.reader_password
}

resource "activedirectory_user" "test" {
  provider            = __PROVIDER__
  name                = __NAME__
  sam_account_name    = __SAM__
  user_principal_name = __UPN__
  parent_dn           = __PARENT__
  display_name        = __MARKER__
  __DESCRIPTION_BLOCK__
  enabled             = __ENABLED__
  __PASSWORD_BLOCK__
  account_expires     = __EXPIRATION__
}

data "activedirectory_user" "lookup" {
  object_guid = activedirectory_user.test.id
}

data "activedirectory_users" "scoped" {
  base_dn      = __ROOT_DN__
  ldap_filter  = __FILTER__
  page_size    = 2
  result_limit = 10
  depends_on   = [activedirectory_user.test]
}
