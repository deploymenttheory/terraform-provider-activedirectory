list "activedirectory_service_account" "group_managed" {
  provider         = activedirectory
  include_resource = true
  limit            = 100
  config {
    base_dn      = "OU=Service Accounts,DC=example,DC=test"
    ldap_filter  = "(objectClass=msDS-GroupManagedServiceAccount)"
    scope        = "one_level"
    page_size    = 100
    result_limit = 100
  }
}
