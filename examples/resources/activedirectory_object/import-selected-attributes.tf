# Use this form to adopt specific existing raw values. Configure matching ownership
# on the target resource; import performs reads only and never changes the directory.
import {
  to = activedirectory_object.application
  id = jsonencode({
    id         = "dc93656c-f745-4926-9bbb-31e7bbecb68b"
    attributes = ["description", "thumbnailPhoto"]
    properties = ["displayName", "telephoneNumber"]
  })
}
