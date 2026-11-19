variable "computer_password" {
  type        = string
  sensitive   = true
  ephemeral   = true
  description = "Initial or explicitly rotated password from your secret manager. Never persist it in configuration files."
}

resource "activedirectory_computer" "full" {
  name                                 = "tf-machine"
  sam_account_name                     = "tf-machine$"
  parent_dn                            = "OU=Automation,DC=example,DC=test"
  dns_host_name                        = "tf-machine.example.test"
  user_principal_name                  = "tf-machine$@example.test"
  display_name                         = "Terraform managed machine account"
  description                          = "Directory account metadata only; no host join or configuration"
  location                             = "Lab rack 1"
  managed_by_dn                        = "CN=Automation Owner,OU=People,DC=example,DC=test"
  home_page                            = "https://inventory.example.test/machines/tf-machine"
  operating_system                     = "Windows Server 2025"
  operating_system_version             = "10.0"
  operating_system_service_pack        = "Inventory metadata"
  operating_system_hotfix              = "Inventory metadata"
  account_expires                      = 0
  kerberos_encryption_types            = ["aes128", "aes256"]
  compound_identity_supported          = true
  account_not_delegated                = true
  allow_reversible_password_encryption = false
  does_not_require_pre_auth            = false
  home_directory_required              = false
  mns_logon_account                    = false
  password_never_expires               = false
  password_not_required                = false
  smartcard_logon_required             = false
  trusted_for_delegation               = false
  trusted_to_auth_for_delegation       = false
  use_des_key_only                     = false
  change_password_at_logon             = false
  password_wo                          = var.computer_password
  password_wo_version                  = 1
  enabled                              = true
}
