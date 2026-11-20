terraform {
  required_version = ">= 1.14.0"
  required_providers { activedirectory = { source = "deploymenttheory/activedirectory" } }
}
provider "activedirectory" {
  auth_method = "simple"
  transport   = "ldaps"
}
list "activedirectory_computer_service_account" "owned" {
  provider = activedirectory
  config { computer_guid = "__COMPUTER_GUID__" }
}
