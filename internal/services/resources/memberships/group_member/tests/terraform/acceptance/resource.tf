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

resource "activedirectory_group_member" "user" {
  group_guid  = "__GROUP_GUID__"
  member_guid = "__USER_GUID__"
}

resource "activedirectory_group_member" "group" {
  group_guid  = "__GROUP_GUID__"
  member_guid = "__MEMBER_GROUP_GUID__"
}

resource "activedirectory_group_member" "computer" {
  group_guid  = "__GROUP_GUID__"
  member_guid = "__COMPUTER_GUID__"
}

data "activedirectory_group_member" "owned" {
  group_guid  = activedirectory_group_member.user.group_guid
  member_guid = activedirectory_group_member.user.member_guid
  depends_on  = [activedirectory_group_member.user]
}

data "activedirectory_group_member_relationships" "owned" {
  group_guid   = "__GROUP_GUID__"
  result_limit = 10
  depends_on   = [activedirectory_group_member.user, activedirectory_group_member.group, activedirectory_group_member.computer]
}

data "activedirectory_group_member_relationships" "bounded" {
  group_guid   = "__GROUP_GUID__"
  result_limit = 2
  depends_on   = [activedirectory_group_member.user, activedirectory_group_member.group, activedirectory_group_member.computer]
}

data "activedirectory_group_member_relationships" "empty" {
  group_guid   = "__EMPTY_GROUP_GUID__"
  result_limit = 10
}

resource "activedirectory_group_member" "solo" {
  group_guid  = "__SOLO_GROUP_GUID__"
  member_guid = "__USER_GUID__"
}
