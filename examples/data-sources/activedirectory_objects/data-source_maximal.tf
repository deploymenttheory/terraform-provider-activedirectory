data "activedirectory_objects" "contacts" {
  base_dn      = "OU=Contacts,DC=example,DC=test"
  ldap_filter  = "(objectClass=contact)"
  scope        = "one_level"
  page_size    = 100
  result_limit = 500
  properties   = ["description", "thumbnailPhoto", "telephoneNumber"]
}
