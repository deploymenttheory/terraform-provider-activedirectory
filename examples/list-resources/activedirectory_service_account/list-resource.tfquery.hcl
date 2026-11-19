list "activedirectory_service_account" "automation" {
  provider = activedirectory
  config {
    base_dn = "OU=Service Accounts,DC=example,DC=test"
  }
}
