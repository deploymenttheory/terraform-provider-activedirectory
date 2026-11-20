terraform {
  required_version = ">= 1.14.0"
  required_providers {
    activedirectory = { source = "deploymenttheory/activedirectory" }
  }
}

# Supply AD_HOST, AD_USERNAME, AD_PASSWORD and, for a private CA, AD_CA_CERT_PEM
# through your secret manager or process environment. Hostname/chain checks remain enabled.
provider "activedirectory" {
  auth_method = "simple"
  transport   = "ldaps"
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
  alias       = "reader"
  username    = var.reader_upn
  password    = var.reader_password
  auth_method = "simple"
  transport   = "ldaps"
}

resource "activedirectory_computer_service_account" "denied" {
  provider = activedirectory.reader
  # __ATTEMPT_BLOCK__
  computer_guid        = "__COMPUTER_GUID__"
  service_account_guid = "__ACCOUNT_GUID__"
}

data "activedirectory_computer_service_account" "visible" {
  provider             = activedirectory.reader
  computer_guid        = "__COMPUTER_GUID__"
  service_account_guid = "__UNMANAGED_GUID__"
}
