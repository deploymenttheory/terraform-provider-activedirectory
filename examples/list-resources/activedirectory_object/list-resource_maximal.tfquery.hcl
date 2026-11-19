list "activedirectory_object" "contacts" {
  provider         = activedirectory
  include_resource = true
  limit            = 100
  config {
    base_dn      = "OU=Contacts,DC=example,DC=test"
    ldap_filter  = "(objectClass=contact)"
    scope        = "one_level"
    page_size    = 50
    result_limit = 500
    properties   = ["description", "thumbnailPhoto"]
  }
}
