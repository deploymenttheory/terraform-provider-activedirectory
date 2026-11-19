resource "activedirectory_object" "example" {
  name         = "Example Contact"
  parent_dn    = "OU=Contacts,DC=example,DC=test"
  object_class = "contact"
}
