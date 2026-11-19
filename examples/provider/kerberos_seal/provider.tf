terraform {
  required_version = ">= 1.14.0"
  required_providers {
    activedirectory = { source = "deploymenttheory/activedirectory" }
  }
}

# The keytab is caller-owned and must be protected on the Terraform execution host.
# Use sasl_protection = "sign" when integrity without confidentiality is explicitly intended.
provider "activedirectory" {
  backend                    = "portable"
  host                       = "dc.example.test"
  auth_method                = "kerberos"
  transport                  = "ldap"
  sasl_protection            = "seal"
  username                   = "terraform-service"
  kerberos_realm             = "EXAMPLE.TEST"
  kerberos_spn               = "ldap/dc.example.test"
  kerberos_credential_source = "keytab"
  kerberos_keytab_file       = "/run/secrets/terraform.keytab"
}
