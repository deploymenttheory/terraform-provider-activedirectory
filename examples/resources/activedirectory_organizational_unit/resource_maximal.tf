resource "activedirectory_organizational_unit" "full" {
  name                               = "Terraform Full Example"
  parent_dn                          = "OU=Managed,DC=example,DC=com"
  description                        = "Managed organizational unit"
  display_name                       = "Terraform Full Example"
  city                               = "Cardiff"
  country                            = "GB"
  state                              = "Wales"
  street_address                     = "1 Example Street"
  postal_code                        = "CF10 1AA"
  managed_by                         = "CN=Directory Manager,OU=People,DC=example,DC=com"
  protected_from_accidental_deletion = true
}
