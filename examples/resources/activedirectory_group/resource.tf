resource "activedirectory_group" "example" {
  name             = "Application Readers"
  sam_account_name = "app-readers"
  parent_dn        = "OU=Groups,DC=example,DC=test"
  category         = "security"
  scope            = "global"
}
