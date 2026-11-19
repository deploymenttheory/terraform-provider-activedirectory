list "activedirectory_group" "application" {
  provider         = activedirectory
  include_resource = true
  limit            = 500
  config {
    base_dn      = "OU=Groups,DC=example,DC=test"
    ldap_filter  = "(name=Application*)"
    scope        = "subtree"
    page_size    = 100
    result_limit = 500
  }
}
