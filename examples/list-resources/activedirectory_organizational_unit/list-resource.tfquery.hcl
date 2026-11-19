list "activedirectory_organizational_unit" "example" {
  provider = activedirectory
  config {
    base_dn = "OU=Managed,DC=example,DC=com"
  }
}
