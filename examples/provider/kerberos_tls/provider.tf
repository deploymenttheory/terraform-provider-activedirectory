terraform {
  required_version = ">= 1.14.0"
  required_providers {
    activedirectory = { source = "deploymenttheory/activedirectory" }
  }
}

variable "directory_password" {
  type      = string
  sensitive = true
  ephemeral = true
}

provider "activedirectory" {
  backend                    = "portable"
  host                       = "dc.example.test"
  partition                  = "DC=example,DC=test"
  auth_method                = "kerberos"
  transport                  = "ldaps"
  username                   = "terraform-service"
  password                   = var.directory_password
  kerberos_realm             = "EXAMPLE.TEST"
  kerberos_spn               = "ldap/dc.example.test"
  kerberos_credential_source = "password"
  timeout_seconds            = 30
}
