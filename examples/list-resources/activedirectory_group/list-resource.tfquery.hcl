list "activedirectory_group" "example" {
  provider = activedirectory
  limit    = 100
  config {
    base_dn = "OU=Groups,DC=example,DC=test"
  }
}
