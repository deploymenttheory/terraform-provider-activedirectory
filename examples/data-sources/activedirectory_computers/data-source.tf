data "activedirectory_computers" "automation" {
  base_dn = "OU=Automation,DC=example,DC=test"
}
