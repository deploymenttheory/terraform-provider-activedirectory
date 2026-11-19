list "activedirectory_organizational_unit" "filtered" {
  provider         = activedirectory
  include_resource = true
  limit            = 100
  config {
    base_dn      = "OU=Managed,DC=example,DC=com"
    ldap_filter  = "(name=Terraform*)"
    scope        = "one_level"
    page_size    = 100
    result_limit = 100
  }
}
