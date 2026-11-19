list "activedirectory_group_member" "owned" {
  provider         = activedirectory
  include_resource = true
  limit            = 10
  config {
    group_guid   = "__GROUP_GUID__"
    result_limit = 10
  }
}
