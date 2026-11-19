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

resource "activedirectory_group_member" "denied" {
  provider = activedirectory.reader
  # __ATTEMPT_BLOCK__
  group_guid  = "__GROUP_GUID__"
  member_guid = "__MEMBER_GUID__"
}

data "activedirectory_group_member" "visible" {
  provider    = activedirectory.reader
  group_guid  = "__GROUP_GUID__"
  member_guid = "__UNMANAGED_GUID__"
}
