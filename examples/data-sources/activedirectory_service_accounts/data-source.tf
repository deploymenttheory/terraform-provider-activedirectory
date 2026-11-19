data "activedirectory_service_accounts" "automation" {
  base_dn = "OU=Service Accounts,DC=example,DC=test"
}
