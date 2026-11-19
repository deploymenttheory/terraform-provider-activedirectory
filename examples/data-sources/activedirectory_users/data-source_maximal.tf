data "activedirectory_users" "engineering" {
  base_dn      = "DC=example,DC=test"
  ldap_filter  = "(department=Engineering)"
  scope        = "subtree"
  page_size    = 256
  result_limit = 1000
}
