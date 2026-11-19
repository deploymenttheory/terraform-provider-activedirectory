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

resource "activedirectory_organizational_unit" "test" {
  provider                           = __PROVIDER__
  name                               = __NAME__
  parent_dn                          = __ROOT_DN__
  display_name                       = __MARKER__
  description                        = __DESCRIPTION__
  protected_from_accidental_deletion = false
}

__IMPORT_BLOCK__

__LOOKUP_BLOCK__
