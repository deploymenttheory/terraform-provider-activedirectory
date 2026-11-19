terraform {
  required_version = ">= 1.14.0"
  required_providers {
    activedirectory = { source = "deploymenttheory/activedirectory" }
  }
}

# Credentials come from AD_USERNAME and AD_PASSWORD. For a private CA, set AD_CA_CERT_PEM.
provider "activedirectory" {
  host        = "dc.example.test"
  auth_method = "simple"
  transport   = "ldaps"
}
