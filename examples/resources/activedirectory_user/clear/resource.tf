resource "activedirectory_user" "user" {
  name             = "terraform-user"
  sam_account_name = "terraform-user"
  parent_dn        = "OU=Automation,DC=example,DC=test"
  clear_properties = ["description", "telephoneNumber"]
}
