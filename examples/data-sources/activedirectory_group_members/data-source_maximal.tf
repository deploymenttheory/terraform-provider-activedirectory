data "activedirectory_group_members" "recursive" {
  group_guid  = "11111111-1111-4111-8111-111111111111"
  recursive   = true
  max_objects = 5000
  max_groups  = 100
  max_ranges  = 100
  max_depth   = 8
  max_bytes   = 8388608

  lifecycle {
    postcondition {
      condition     = self.complete
      error_message = "Membership discovery is incomplete; verify visibility, unresolved references and traversal bounds before using these results as a complete collection."
    }
  }
}
