data "activedirectory_group_member_relationships" "bounded" {
  group_guid   = "11111111-1111-4111-8111-111111111111"
  result_limit = 10000
}

output "direct_members" {
  value = data.activedirectory_group_member_relationships.bounded.members
}

output "discovery_complete" {
  value = data.activedirectory_group_member_relationships.bounded.complete
}
