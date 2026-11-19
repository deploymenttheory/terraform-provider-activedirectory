data "activedirectory_group" "by_dn" {
  dn = "CN=Application Readers,OU=Groups,DC=example,DC=test"
}
