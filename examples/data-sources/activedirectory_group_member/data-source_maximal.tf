data "activedirectory_group_member" "existing_link" {
  group_guid  = "11111111-1111-4111-8111-111111111111"
  member_guid = "22222222-2222-4222-8222-222222222222"
}

output "direct_link" {
  value = {
    id        = data.activedirectory_group_member.existing_link.id
    group_dn  = data.activedirectory_group_member.existing_link.group_dn
    member_dn = data.activedirectory_group_member.existing_link.member_dn
  }
}
