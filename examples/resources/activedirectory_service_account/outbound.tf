resource "activedirectory_service_account" "outbound" {
  name                         = "Outbound Build"
  sam_account_name             = "tfoutbound$"
  parent_dn                    = "OU=Service Accounts,DC=example,DC=test"
  kind                         = "group_managed"
  outbound_authentication_only = true
  enabled                      = false
}
