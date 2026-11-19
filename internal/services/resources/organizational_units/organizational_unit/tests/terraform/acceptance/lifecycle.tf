terraform {
  required_version = ">= 1.14.0"
  required_providers {
    activedirectory = { source = "deploymenttheory/activedirectory" }
  }
}

provider "activedirectory" {}

locals {
  prefix = __PREFIX__
  marker = __MARKER__
  root   = __ROOT_DN__
}

resource "activedirectory_organizational_unit" "parent_a" {
  name                               = "${local.prefix}-a"
  parent_dn                          = local.root
  display_name                       = local.marker
  protected_from_accidental_deletion = false
}

resource "activedirectory_organizational_unit" "parent_b" {
  name                               = "${local.prefix}-b"
  parent_dn                          = local.root
  display_name                       = local.marker
  protected_from_accidental_deletion = false
}

resource "activedirectory_organizational_unit" "test" {
  name         = "${local.prefix}-__NAME__"
  parent_dn    = __TARGET_PARENT__
  display_name = local.marker
  __DESCRIPTION_BLOCK__
  protected_from_accidental_deletion = __PROTECTED__
}

data "activedirectory_organizational_unit" "lookup" {
  object_guid = activedirectory_organizational_unit.test.id
}

data "activedirectory_organizational_units" "scoped" {
  base_dn      = local.root
  ldap_filter  = "(name=${local.prefix}*)"
  scope        = "subtree"
  page_size    = 2
  result_limit = 10
  depends_on = [activedirectory_organizational_unit.parent_a, activedirectory_organizational_unit.parent_b, activedirectory_organizational_unit.test]
}
