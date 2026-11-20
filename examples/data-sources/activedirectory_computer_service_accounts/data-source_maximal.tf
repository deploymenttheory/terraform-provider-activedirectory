data "activedirectory_computer_service_accounts" "bounded" {
  computer_guid = "11111111-1111-4111-8111-111111111111"
  result_limit  = 10000
}

output "direct_members" {
  value = data.activedirectory_computer_service_accounts.bounded.members
}

output "discovery_complete" {
  value = data.activedirectory_computer_service_accounts.bounded.complete
}
