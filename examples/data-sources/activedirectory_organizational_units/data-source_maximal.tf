data "activedirectory_organizational_units" "filtered" {
  base_dn      = "OU=Managed,DC=example,DC=com"
  ldap_filter  = "(name=Terraform*)"
  scope        = "one_level"
  page_size    = 256
  result_limit = 1000
}
