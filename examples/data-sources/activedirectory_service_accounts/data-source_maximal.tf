data "activedirectory_service_accounts" "group_managed" {
  base_dn      = "OU=Service Accounts,DC=example,DC=test"
  ldap_filter  = "(objectClass=msDS-GroupManagedServiceAccount)"
  scope        = "one_level"
  page_size    = 100
  result_limit = 500
}
