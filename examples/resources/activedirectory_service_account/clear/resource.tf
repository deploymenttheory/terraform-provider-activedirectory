# Import an existing account by GUID before applying this explicit clear recipe.
resource "activedirectory_service_account" "example" {
  name                    = "Terraform Build"
  sam_account_name        = "tfbuild$"
  parent_dn               = "OU=Service Accounts,DC=example,DC=test"
  kind                    = "group_managed"
  clear_properties        = ["displayName", "description"]
  service_principal_names = []
  password_reader_sids    = []
}
