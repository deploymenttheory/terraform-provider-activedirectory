variable "user_password" {
  type        = string
  sensitive   = true
  ephemeral   = true
  description = "Initial or explicitly rotated password from your secret manager. Never persist it in configuration files."
}

resource "activedirectory_user" "full" {
  name                   = "Terraform Automation"
  sam_account_name       = "tf-automation"
  parent_dn              = "OU=Automation,DC=example,DC=test"
  user_principal_name    = "tf-automation@example.test"
  given_name             = "Terraform"
  surname                = "Automation"
  display_name           = "Terraform Automation"
  description            = "Explicitly managed automation account"
  email_address          = "tf-automation@example.test"
  department             = "Engineering"
  company                = "Example"
  title                  = "Automation"
  manager_dn             = "CN=Automation Owner,OU=People,DC=example,DC=test"
  account_expires        = 0
  password_not_required  = false
  password_never_expires = false
  password_wo            = var.user_password
  password_wo_version    = 1
  enabled                = true
}
