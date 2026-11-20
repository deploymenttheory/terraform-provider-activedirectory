# Each instance owns one link. Members omitted from this set remain unmanaged.
resource "activedirectory_group_member" "application_readers" {
  for_each = toset([
    "22222222-2222-4222-8222-222222222222",
    "33333333-3333-4333-8333-333333333333",
  ])
  group_guid  = "11111111-1111-4111-8111-111111111111"
  member_guid = each.value
}
