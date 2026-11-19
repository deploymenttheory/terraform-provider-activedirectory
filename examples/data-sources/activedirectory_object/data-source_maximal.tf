data "activedirectory_object" "contact" {
  dn         = "CN=Example Contact,OU=Contacts,DC=example,DC=test"
  properties = ["description", "jpegPhoto", "telephoneNumber"]
}
