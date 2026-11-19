list "activedirectory_user" "test" {
  provider         = activedirectory
  include_resource = true
  limit            = 10
  config {
    base_dn      = "DC=unit,DC=invalid"
    page_size    = 2
    result_limit = 10
  }
}
