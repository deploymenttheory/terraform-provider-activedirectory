list "activedirectory_computer_service_account" "owned" {
  provider         = activedirectory
  include_resource = true
  limit            = 10
  config {
    computer_guid = "__COMPUTER_GUID__"
    result_limit  = 10
  }
}
