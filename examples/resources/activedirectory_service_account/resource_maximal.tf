variable "password_reader_sids" {
  type        = set(string)
  description = "Canonical SIDs of principals allowed explicit gMSA Read Property grants."
}

resource "activedirectory_service_account" "extended" {
  name                                 = "Terraform Extended Build"
  sam_account_name                     = "tfextended$"
  parent_dn                            = "OU=Service Accounts,DC=example,DC=test"
  kind                                 = "group_managed"
  dns_host_name                        = "tfextended.example.test"
  description                          = "Terraform-owned build identity"
  display_name                         = "Extended build service"
  home_page                            = "https://example.test/build"
  enabled                              = false
  managed_password_interval_days       = 30
  service_principal_names              = ["HTTP/tfextended.example.test"]
  password_reader_sids                 = var.password_reader_sids
  kerberos_encryption_types            = ["aes128", "aes256"]
  compound_identity_supported          = true
  account_expires                      = 0
  account_not_delegated                = true
  allow_reversible_password_encryption = false
  does_not_require_pre_auth            = false
  home_directory_required              = false
  mns_logon_account                    = false
  smartcard_logon_required             = false
  trusted_for_delegation               = false
  trusted_to_auth_for_delegation       = false
  use_des_key_only                     = false
}
