list "activedirectory_user" "engineering" {
  provider         = activedirectory
  include_resource = true
  limit            = 100
  config {
    base_dn      = "DC=example,DC=test"
    ldap_filter  = "(department=Engineering)"
    scope        = "subtree"
    page_size    = 100
    result_limit = 100
  }
}
