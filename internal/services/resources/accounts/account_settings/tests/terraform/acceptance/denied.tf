terraform {
  required_version = ">= 1.14.0"
  required_providers {
    activedirectory = { source = "deploymenttheory/activedirectory" }
  }
}

variable "reader_upn" {
  type = string
}

variable "reader_password" {
  type      = string
  sensitive = true
  ephemeral = true
}

provider "activedirectory" {
  auth_method = "simple"
  transport   = "ldaps"
}

provider "activedirectory" {
  alias       = "reader"
  auth_method = "simple"
  transport   = "ldaps"
  username    = var.reader_upn
  password    = var.reader_password
}

resource "activedirectory_account_settings" "user" {
  provider     = activedirectory.reader
  account_guid = __USER_GUID__
  enabled      = __ENABLED__
}

data "activedirectory_account_settings" "by_guid" {
  provider    = activedirectory.reader
  object_guid = activedirectory_account_settings.user.id
}
