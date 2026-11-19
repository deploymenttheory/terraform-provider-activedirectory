resource "activedirectory_service_account" "example" {
  name             = "Terraform Build"
  sam_account_name = "tfbuild$"
  parent_dn        = "OU=Service Accounts,DC=example,DC=test"
  kind             = "group_managed"
  dns_host_name    = "tfbuild.example.test"
  enabled          = false
}
