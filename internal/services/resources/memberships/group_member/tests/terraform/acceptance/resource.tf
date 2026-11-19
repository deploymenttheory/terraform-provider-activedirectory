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
variable "group_guid" {
  type = string
}

variable "member_guid" {
  type = string
}

resource "activedirectory_group_member" "test" {
  group_guid  = var.group_guid
  member_guid = var.member_guid
}

data "activedirectory_group_member" "owned" {
  group_guid  = activedirectory_group_member.test.group_guid
  member_guid = activedirectory_group_member.test.member_guid
  depends_on  = [activedirectory_group_member.test]
}

data "activedirectory_group_member_relationships" "owned" {
  group_guid   = var.group_guid
  result_limit = 10
  depends_on   = [activedirectory_group_member.test]
}
