variable "jpeg_photo_path" {
  type        = string
  description = "Path to the contact JPEG photo on the Terraform client."
}
resource "activedirectory_object" "application" {
  name         = "Application Contact"
  parent_dn    = "OU=Contacts,DC=example,DC=test"
  object_class = "contact"
  attributes = {
    description = [base64encode("Application contact")]
    jpegPhoto   = [filebase64(var.jpeg_photo_path)]
  }
  add_values = {
    otherTelephone = [base64encode("+44 20 7946 0123")]
  }
  properties = ["displayName", "telephoneNumber"]
}
