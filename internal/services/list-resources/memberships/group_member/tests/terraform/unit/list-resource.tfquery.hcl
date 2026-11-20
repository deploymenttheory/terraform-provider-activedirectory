list "activedirectory_group_member" "test" {
  provider         = activedirectory
  include_resource = true
  limit            = 10
  config {
    group_guid   = "11111111-1111-4111-8111-111111111111"
    result_limit = 10
  }
}
