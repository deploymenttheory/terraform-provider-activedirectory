resource "activedirectory_computer" "computer" {
  name             = "terraform-computer"
  sam_account_name = "terraform-computer$"
  parent_dn        = "OU=Automation,DC=example,DC=test"
  enabled          = false
}
