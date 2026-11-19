# Import this existing contact before applying clear/remove intent.
resource "activedirectory_object" "clear_existing" {
  name             = "Existing Contact"
  parent_dn        = "OU=Contacts,DC=example,DC=test"
  object_class     = "contact"
  remove_values    = { otherTelephone = [base64encode("+44 20 7946 0123")] }
  clear_attributes = ["displayName"]
}
