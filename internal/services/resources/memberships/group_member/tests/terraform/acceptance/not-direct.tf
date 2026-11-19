data "activedirectory_group_member" "not_direct" {
  group_guid  = "__GROUP_GUID__"
  member_guid = "__MEMBER_GUID__"
}
