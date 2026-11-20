data "activedirectory_computer_service_account" "hidden" {
  provider             = activedirectory.reader
  computer_guid        = "__COMPUTER_GUID__"
  service_account_guid = "__ACCOUNT_GUID__"
}
