terraform {
  required_providers {
    activedirectory = { source = "deploymenttheory/activedirectory" }
  }
}

provider "activedirectory" {
  host        = "dc.unit.invalid"
  username    = "fixture@unit.invalid"
  password    = "synthetic-bind-password"
  auth_method = "simple"
  transport   = "ldaps"
}

variable "computer_password" {
  type      = string
  sensitive = true
  ephemeral = true
}

resource "activedirectory_computer" "test" {
  name                = "initial"
  sam_account_name    = "fixture$"
  parent_dn           = "OU=first,DC=unit,DC=invalid"
  description         = "first description"
  enabled             = true
  password_wo         = var.computer_password
  password_wo_version = 1
  account_expires     = 0
}

data "activedirectory_computer" "lookup" {
  object_guid = activedirectory_computer.test.id
}

data "activedirectory_computers" "scoped" {
  base_dn      = "DC=unit,DC=invalid"
  page_size    = 2
  result_limit = 10
  depends_on   = [activedirectory_computer.test]
}
