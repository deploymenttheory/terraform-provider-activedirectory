terraform {
  required_version = ">= 1.14.0"
  required_providers {
    activedirectory = { source = "deploymenttheory/activedirectory" }
  }
}

variable "service_account_password" {
  type      = string
  sensitive = true
  ephemeral = true
}

provider "activedirectory" {
  auth_method = "simple"
  transport   = "ldaps"
}

resource "activedirectory_service_account" "automatic" {
  name             = __AUTO_NAME__
  sam_account_name = __AUTO_SAM__
  parent_dn        = __ROOT_DN__
  kind             = "standalone"
  description      = __MARKER__
  enabled          = false
}

resource "activedirectory_service_account" "explicit" {
  name                = __EXPLICIT_NAME__
  sam_account_name    = __EXPLICIT_SAM__
  parent_dn           = __ROOT_DN__
  kind                = "standalone"
  description         = __MARKER__
  enabled             = false
  password_wo         = var.service_account_password
  password_wo_version = __PASSWORD_VERSION__
}

resource "activedirectory_service_account" "outbound" {
  name                         = __OUTBOUND_NAME__
  sam_account_name             = __OUTBOUND_SAM__
  parent_dn                    = __ROOT_DN__
  kind                         = "group_managed"
  description                  = __MARKER__
  outbound_authentication_only = true
  enabled                      = false
}

data "activedirectory_service_accounts" "scoped" {
  base_dn      = __ROOT_DN__
  ldap_filter  = __FILTER__
  page_size    = 1
  result_limit = 10
  depends_on   = [activedirectory_service_account.automatic, activedirectory_service_account.explicit, activedirectory_service_account.outbound]
}

data "activedirectory_service_accounts" "bounded" {
  base_dn      = __ROOT_DN__
  ldap_filter  = __FILTER__
  page_size    = 1
  result_limit = 1
  depends_on   = [activedirectory_service_account.automatic, activedirectory_service_account.explicit, activedirectory_service_account.outbound]
}
