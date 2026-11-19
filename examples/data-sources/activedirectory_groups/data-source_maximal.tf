data "activedirectory_groups" "application" {
  base_dn      = "OU=Groups,DC=example,DC=test"
  ldap_filter  = "(name=Application*)"
  scope        = "one_level"
  page_size    = 100
  result_limit = 500
}
