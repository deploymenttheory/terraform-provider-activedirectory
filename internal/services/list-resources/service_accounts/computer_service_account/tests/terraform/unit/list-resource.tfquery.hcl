list "activedirectory_computer_service_account" "test" {
  provider         = activedirectory
  include_resource = true
  limit            = 10
  config {
    computer_guid   = "11111111-1111-4111-8111-111111111111"
    result_limit = 10
  }
}
