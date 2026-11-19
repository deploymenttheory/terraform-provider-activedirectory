# Adopt an existing account. This resource does not create it.
# Do not also configure this account's enabled setting in a typed account resource.
resource "activedirectory_account_settings" "example" {
  account_guid = "b591fb30-6292-4744-bf23-5f346cb74559"
  enabled      = false
}
