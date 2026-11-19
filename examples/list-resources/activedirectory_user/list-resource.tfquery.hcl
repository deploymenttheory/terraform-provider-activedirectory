list "activedirectory_user" "automation" {
  provider = activedirectory
  config {
    base_dn = "OU=Automation,DC=example,DC=test"
  }
}
