# Separate recipe: manage this existing account only through this settings resource
# for the fields below. Omitted fields remain unmanaged.
resource "activedirectory_account_settings" "maximal" {
  account_guid                         = "6d682f3f-6a4d-4beb-ab06-6d6115b256e1"
  enabled                              = true
  account_not_delegated                = true
  allow_reversible_password_encryption = false
  does_not_require_preauth             = false
  home_directory_required              = false
  mns_logon_account                    = false
  password_never_expires               = false
  password_not_required                = false
  smartcard_logon_required             = false
  trusted_for_delegation               = false
  trusted_to_auth_for_delegation       = false
  use_des_key_only                     = false
  change_password_at_logon             = true
  expires_at                           = "2031-04-05T11:34:56Z"
}
