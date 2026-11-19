# Update an existing/imported OU; omit description while explicitly requesting its clear.
resource "activedirectory_organizational_unit" "example" {
  name             = "Terraform Example"
  parent_dn        = "OU=Managed,DC=example,DC=com"
  clear_properties = ["description"]
}
