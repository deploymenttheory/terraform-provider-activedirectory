# Separate read-only lookup by DN. Account settings are never changed by a data source.
data "activedirectory_account_settings" "by_dn" {
  dn = "CN=Existing Service User,OU=Service Accounts,DC=example,DC=com"
}

output "account_status" {
  value = {
    guid             = data.activedirectory_account_settings.by_dn.id
    enabled          = data.activedirectory_account_settings.by_dn.enabled
    never_expires    = data.activedirectory_account_settings.by_dn.never_expires
    expires_at       = data.activedirectory_account_settings.by_dn.expires_at
    locked_out       = data.activedirectory_account_settings.by_dn.locked_out
    password_expired = data.activedirectory_account_settings.by_dn.password_expired
  }
}
