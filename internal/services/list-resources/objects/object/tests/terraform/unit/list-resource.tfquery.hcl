provider "activedirectory" {
  host        = "unit.invalid"
  auth_method = "simple"
  username    = "unit-fixture"
  password    = "not-a-real-credential"
}
list "activedirectory_object" "test" {
  provider         = activedirectory
  include_resource = true
  limit            = 10
  config {
    properties   = ["description", "jpegPhoto"]
    base_dn      = "DC=unit,DC=invalid"
    scope        = "one_level"
    page_size    = 2
    result_limit = 10
  }
}
