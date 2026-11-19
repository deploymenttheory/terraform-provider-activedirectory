data "activedirectory_users" "automation" {
  base_dn = "OU=Automation,DC=example,DC=test"
}
