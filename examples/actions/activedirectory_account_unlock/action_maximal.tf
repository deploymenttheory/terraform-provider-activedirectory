variable "unlock_request" {
  type        = string
  description = "Explicit request identifier. Change this value only after verifying any earlier unlock outcome."
}

action "activedirectory_account_unlock" "requested" {
  config {
    object_guid = "8b48ae8e-aea5-4c45-acef-b319f8db9cf4"
    server      = "dc.example.test"
  }
}

resource "terraform_data" "request" {
  input = var.unlock_request

  lifecycle {
    action_trigger {
      events  = [before_create, before_update]
      actions = [action.activedirectory_account_unlock.requested]
    }
  }
}
