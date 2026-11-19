# After importing the existing group, explicitly clear only the selected property.
resource "activedirectory_group" "adopted" {
  name             = "Application Readers"
  sam_account_name = "app-readers"
  parent_dn        = "OU=Groups,DC=example,DC=test"
  category         = "security"
  scope            = "global"
  clear_properties = ["description"]
}
