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

data "activedirectory_group_members" "direct" {
  group_guid = "__GROUP_GUID__"
}
data "activedirectory_group_members" "recursive" {
  group_guid = "__GROUP_GUID__"
  recursive  = true
}
data "activedirectory_group_members" "depth" {
  group_guid = "__GROUP_GUID__"
  recursive  = true
  max_depth  = 1
}
data "activedirectory_group_members" "objects" {
  group_guid  = "__GROUP_GUID__"
  recursive   = true
  max_objects = 8
}
data "activedirectory_group_members" "empty" {
  group_guid = "__EMPTY_GUID__"
}

data "activedirectory_group_members" "groups" {
  group_guid = "__GROUP_GUID__"
  recursive  = true
  max_groups = 1
}

data "activedirectory_group_members" "ranges" {
  group_guid = "__GROUP_GUID__"
  recursive  = true
  max_ranges = 1
}

data "activedirectory_group_members" "bytes" {
  group_guid = "__GROUP_GUID__"
  recursive  = true
  max_bytes  = 1024
}
