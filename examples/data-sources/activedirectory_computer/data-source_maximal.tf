data "activedirectory_computer" "by_dn" {
  dn = "CN=Terraform Automation,OU=Automation,DC=example,DC=test"
}
