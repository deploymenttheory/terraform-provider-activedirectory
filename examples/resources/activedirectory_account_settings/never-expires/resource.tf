# Separate recipe: explicitly clear expiration on an existing account.
# Both native never-expire representations read as never_expires = true.
resource "activedirectory_account_settings" "never" {
  account_guid  = "6d682f3f-6a4d-4beb-ab06-6d6115b256e1"
  never_expires = true
}
