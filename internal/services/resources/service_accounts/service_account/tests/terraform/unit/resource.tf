provider "activedirectory" {
  host        = "dc.unit.invalid"
  auth_method = "simple"
  transport   = "ldaps"
  username    = "fixture@unit.invalid"
  password    = "synthetic-bind-only"
}

resource "activedirectory_service_account" "test" {
  name                    = "initial"
  parent_dn               = "OU=first,DC=unit,DC=invalid"
  sam_account_name        = "initial$"
  kind                    = "group_managed"
  dns_host_name           = "initial.unit.invalid"
  description             = "first description"
  enabled                 = true
  service_principal_names = ["HTTP/initial.unit.invalid"]
  password_reader_sids    = ["S-1-5-21-1-2-3-1001"]
}
