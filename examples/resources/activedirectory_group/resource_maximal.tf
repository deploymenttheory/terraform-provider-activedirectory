resource "activedirectory_group" "maximal" {
  name             = "Application Operators"
  sam_account_name = "app-operators"
  parent_dn        = "OU=Groups,DC=example,DC=test"
  category         = "security"
  scope            = "domain_local"
  description      = "Operators for application access"
  display_name     = "Application Operators"
  home_page        = "https://applications.example.test/"
  managed_by_dn    = "CN=Application Owner,OU=People,DC=example,DC=test"
}
