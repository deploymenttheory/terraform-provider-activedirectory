# Examples and generated documentation

Run `make generate-docs` from the repository root. On Windows without Make, run
`go run ./internal/tools/docscheck` followed by `go generate ./...`.
The generator pins tfplugindocs 0.25.0 and Terraform 1.17.2. It reads schemas without
configuring the provider or connecting to Active Directory.

Each registered resource, data source, list resource and action needs its own template
under `templates/<category>/<short_type>.md.tmpl`. Example directories include the full
type name, for example `examples/resources/activedirectory_organizational_unit/`.
Templates must embed `.SchemaMarkdown` and the examples through `tffile`:

| Category | Minimal example | Maximal example |
| --- | --- | --- |
| resources | resource.tf | resource_maximal.tf |
| data-sources | data-source.tf | data-source_maximal.tf |
| list-resources | list-resource.tfquery.hcl | list-resource_maximal.tfquery.hcl |
| actions | action.tf | action_maximal.tf |

Resource templates must also embed `import-by-string-id.tf` using the resource's immutable
GUID. List examples use Terraform's `.tfquery.hcl` format. Maximal examples demonstrate the
relevant optional configuration; they must not invent supported fields or unsafe defaults.
Keep common provider authentication in the secure provider examples. Deliver credentials
through environment variables or ephemeral sensitive variables, never embedded values.

`docscheck` compares these authored inputs with the actual protocol V6 registration schemas
for all four surfaces. A generator-created fallback page cannot satisfy this check. Run
`make check-docs` to regenerate and reject changed or untracked documentation inputs/outputs.
CI also exercises the actual pinned generator with representative synthetic schemas for all
four surfaces, verifying that schema descriptions and both examples reach the rendered pages.
That integration check is offline with respect to AD; it does not qualify directory operations.
