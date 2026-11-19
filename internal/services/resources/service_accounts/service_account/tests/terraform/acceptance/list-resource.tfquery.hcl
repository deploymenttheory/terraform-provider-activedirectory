list "activedirectory_service_account" "owned" {
  provider         = activedirectory
  include_resource = true
  limit            = __LIMIT__
  config {
    base_dn      = __ROOT_DN__
    ldap_filter  = __FILTER__
    page_size    = 1
    result_limit = 10
  }
}
