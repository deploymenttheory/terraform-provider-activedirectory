data "activedirectory_service_account" "by_dn" {
  dn = "CN=Terraform Build,OU=Service Accounts,DC=example,DC=test"
}
