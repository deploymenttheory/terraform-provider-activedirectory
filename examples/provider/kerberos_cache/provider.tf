terraform {
  required_version = ">= 1.14.0"
  required_providers {
    activedirectory = { source = "deploymenttheory/activedirectory" }
  }
}

# Supply a valid caller-owned cache. The provider never looks up an ambient cache.
provider "activedirectory" {
  backend                        = "portable"
  host                           = "dc.example.test"
  auth_method                    = "kerberos"
  transport                      = "starttls"
  username                       = "terraform-service"
  kerberos_realm                 = "EXAMPLE.TEST"
  kerberos_spn                   = "ldap/dc.example.test"
  kerberos_credential_source     = "credential_cache"
  kerberos_credential_cache_file = "/run/secrets/terraform.ccache"
}
