action "activedirectory_account_unlock" "unlock" {
  config {
    object_guid = "8b48ae8e-aea5-4c45-acef-b319f8db9cf4"
    server      = "dc.example.test"
  }
}

# Explicit invocation: terraform apply -invoke=action.activedirectory_account_unlock.unlock
