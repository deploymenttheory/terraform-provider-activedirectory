data "activedirectory_computers" "engineering" {
  base_dn      = "DC=example,DC=test"
  ldap_filter  = "(operatingSystem=Windows*)"
  scope        = "subtree"
  page_size    = 256
  result_limit = 1000
}
