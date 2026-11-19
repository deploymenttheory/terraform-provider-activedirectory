variable "standalone_initial_password" {
  type      = string
  sensitive = true
  ephemeral = true
}

resource "activedirectory_service_account" "standalone" {
  name                = "Standalone Build"
  sam_account_name    = "tfstandalone$"
  parent_dn           = "OU=Service Accounts,DC=example,DC=test"
  kind                = "standalone"
  enabled             = false
  password_wo         = var.standalone_initial_password
  password_wo_version = 1
}
