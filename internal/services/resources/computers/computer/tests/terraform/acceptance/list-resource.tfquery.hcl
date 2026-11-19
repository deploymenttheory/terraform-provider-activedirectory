list "activedirectory_computer" "owned" {
  provider         = activedirectory
  include_resource = true
  limit            = 10
  config {
    base_dn      = __ROOT_DN__
    ldap_filter  = __FILTER__
    page_size    = 2
    result_limit = 10
  }
}
