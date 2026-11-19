provider "activedirectory" {}

list "activedirectory_organizational_unit" "owned" {
  provider         = activedirectory
  include_resource = true
  limit            = 10
  config {
    base_dn      = __ROOT_DN__
    ldap_filter  = __FILTER__
    scope        = "subtree"
    page_size    = 2
    result_limit = 10
  }
}
