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

resource "activedirectory_account_settings" "user" {
  account_guid = __USER_GUID__
  # __USER_SETTINGS__
}

resource "activedirectory_account_settings" "computer" {
  account_guid = __COMPUTER_GUID__
  # __COMPUTER_SETTINGS__
}

data "activedirectory_account_settings" "by_guid" {
  object_guid = activedirectory_account_settings.user.id
}

data "activedirectory_account_settings" "by_dn" {
  dn = lower(activedirectory_account_settings.user.distinguished_name)
}
