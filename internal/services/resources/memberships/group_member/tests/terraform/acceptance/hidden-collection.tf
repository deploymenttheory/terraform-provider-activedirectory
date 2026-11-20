data "activedirectory_group_member_relationships" "hidden" {
  provider     = activedirectory.reader
  group_guid   = "__GROUP_GUID__"
  result_limit = 10
}
