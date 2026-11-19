terraform {
  required_providers {
    activedirectory = { source = "deploymenttheory/activedirectory" }
  }
}

provider "activedirectory" {
  host        = "dc.unit.invalid"
  username    = "fixture@unit.invalid"
  password    = "synthetic-password"
  auth_method = "simple"
  transport   = "ldaps"
}

action "activedirectory_account_unlock" "test" {
  config {
    object_guid = "dc93656c-f745-4926-9bbb-31e7bbecb68b"
    server      = "dc.unit.invalid"
  }
}

resource "terraform_data" "invoke" {
  input = "explicit unlock"
  lifecycle {
    action_trigger {
      events  = [before_create]
      actions = [action.activedirectory_account_unlock.test]
    }
  }
}
