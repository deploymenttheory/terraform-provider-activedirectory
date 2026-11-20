terraform {
  required_version = ">= 1.14.0"
  required_providers { activedirectory = { source = "deploymenttheory/activedirectory" } }
}
provider "activedirectory" {
  auth_method = "simple"
  transport   = "ldaps"
}
resource "activedirectory_computer_service_account" "adopted" {
  computer_guid        = "__COMPUTER_GUID__"
  service_account_guid = "__ACCOUNT_GUID__"
}
resource "activedirectory_computer_service_account" "added" {
  computer_guid        = "__COMPUTER_GUID__"
  service_account_guid = "__SECOND_GUID__"
}
resource "activedirectory_computer_service_account" "solo" {
  computer_guid        = "__SOLO_COMPUTER_GUID__"
  service_account_guid = "__SOLO_ACCOUNT_GUID__"
}
data "activedirectory_computer_service_account" "owned" {
  computer_guid        = activedirectory_computer_service_account.adopted.computer_guid
  service_account_guid = activedirectory_computer_service_account.adopted.service_account_guid
}
data "activedirectory_computer_service_accounts" "owned" {
  computer_guid = "__COMPUTER_GUID__"
  result_limit  = 10
  depends_on    = [activedirectory_computer_service_account.adopted, activedirectory_computer_service_account.added]
}
data "activedirectory_computer_service_accounts" "bounded" {
  computer_guid = "__COMPUTER_GUID__"
  result_limit  = 1
  depends_on    = [activedirectory_computer_service_account.adopted, activedirectory_computer_service_account.added]
}
data "activedirectory_computer_service_accounts" "empty" {
  computer_guid = "__EMPTY_COMPUTER_GUID__"
  result_limit  = 10
}
