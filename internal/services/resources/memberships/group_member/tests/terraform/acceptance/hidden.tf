data "activedirectory_group_member" "hidden" {
  provider    = activedirectory.reader
  group_guid  = "__GROUP_GUID__"
  member_guid = "__MEMBER_GUID__"
}
