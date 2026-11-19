terraform {
  required_version = ">= 1.14.0"

  required_providers {
    activedirectory = {
      source = "deploymenttheory/activedirectory"
    }
  }
}

# Supply AD_USERNAME and AD_PASSWORD through your secret manager or process environment.
# TLS uses system trust. For a private CA, set ca_cert_pem = file("/path/to/ca.pem").
provider "activedirectory" {
  host        = "dc.example.test"
  auth_method = "simple"
  transport   = "ldaps"
}
