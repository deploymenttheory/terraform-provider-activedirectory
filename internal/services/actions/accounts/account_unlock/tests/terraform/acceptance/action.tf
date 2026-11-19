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

provider "activedirectory" {}

provider "activedirectory" {
  alias    = "reader"
  username = var.reader_upn
  password = var.reader_password
}

action "activedirectory_account_unlock" "test" {
  provider = __PROVIDER__
  config {
    object_guid = __GUID__
    server      = __SERVER__
  }
}

resource "terraform_data" "request" {
  input = "explicit owned fixture unlock"
  lifecycle {
    action_trigger {
      events  = [before_create]
      actions = [action.activedirectory_account_unlock.test]
    }
  }
}
