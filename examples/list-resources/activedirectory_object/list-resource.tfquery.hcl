list "activedirectory_object" "example" {
  provider = activedirectory
  config {
    base_dn     = "OU=Contacts,DC=example,DC=test"
    ldap_filter = "(objectClass=contact)"
  }
}
