data "activedirectory_computer_service_account" "existing_link" {
  computer_guid        = "11111111-1111-4111-8111-111111111111"
  service_account_guid = "22222222-2222-4222-8222-222222222222"
}

output "direct_link" {
  value = {
    id                 = data.activedirectory_computer_service_account.existing_link.id
    computer_dn        = data.activedirectory_computer_service_account.existing_link.computer_dn
    service_account_dn = data.activedirectory_computer_service_account.existing_link.service_account_dn
  }
}
