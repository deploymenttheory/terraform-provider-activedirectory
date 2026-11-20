data "activedirectory_computer_service_accounts" "hidden" {
  provider      = activedirectory.reader
  computer_guid = "__COMPUTER_GUID__"
  result_limit  = 10
}
