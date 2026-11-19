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

resource "activedirectory_service_account" "test" {
  name                = __NAME__
  sam_account_name    = __SAM__
  parent_dn           = __ROOT_DN__
  kind                = "standalone"
  description         = __MARKER__
  enabled             = true
  password_wo         = var.service_account_password
  password_wo_version = 1
}
