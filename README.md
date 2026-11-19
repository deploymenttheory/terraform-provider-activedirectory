# Terraform Provider for Active Directory

Developer documentation: [testing](docs/development/client-acceptance.md) and
[examples and generated documentation](docs/development/documentation.md).

Terraform Plugin Framework provider for Active Directory Domain Services, built on the
[Deployment Theory Active Directory SDK](https://github.com/deploymenttheory/go-sdk-activedirectory).

The provider is under development. The current foundation serves Terraform plugin protocol V6;
directory resources, data sources, list resources and actions are being added through the
[approved implementation backlog](docs/plans/activedirectory-provider/backlog/INDEX.md).
Provider configuration opens and qualifies a protected AD DS session using SDK v0.2.0.
Directory service implementations are being added separately.

Requirements: Go 1.27.4 for development and Terraform 1.14 or newer for the provider's four service surfaces.

```hcl
terraform {
  required_version = ">= 1.14.0"

  required_providers {
    activedirectory = {
      source = "deploymenttheory/activedirectory"
    }
  }
}

# Credentials come from AD_USERNAME and AD_PASSWORD.
provider "activedirectory" {
  host        = "dc.example.test"
  auth_method = "simple"
  transport   = "ldaps"
}
```

The provider has not been published to the Terraform Registry. This configuration becomes usable
after installing a development build. See [provider configuration](docs/index.md) for verified TLS,
Kerberos channel binding, signing/sealing, explicit credential sources and private CA trust.

## Development

Run commands from the repository root:

```shell
go mod download
go test ./...
go vet ./...
go build -o bin/terraform-provider-activedirectory .
golangci-lint run --fix=false
go generate ./...
```

The Terraform CLI foundation test runs without a domain controller:

```powershell
$env:TF_ACC = "1"
$env:TF_ACC_TERRAFORM_PATH = (Get-Command terraform).Source
go test ./internal/provider -run '^TestAccProviderFoundation$' -count=1
```

```shell
TF_ACC=1 TF_ACC_TERRAFORM_PATH="$(command -v terraform)" \
  go test ./internal/provider -run '^TestAccProviderFoundation$' -count=1
```

This verifies Terraform CLI planning and state, alongside the unit tests for protocol V6 schema,
configuration validation and provider instance isolation. It does not claim directory acceptance.
Directory feature acceptance will run against the owned lab from Linux and Windows clients on
Windows Server 2016, 2019, 2022 and 2025 where applicable. macOS receives build and unit coverage.

## Repository layout

```text
main.go                         Provider process and build version
internal/provider/              Provider factory and service registrations
internal/provider/core/         Shared schema and configuration
internal/client/                Public SDK contracts and session qualification
examples/provider/              Provider configuration example
docs/plans/activedirectory-provider/  Approved architecture and implementation backlog
```

The architecture follows the
[Microsoft 365 provider](https://github.com/deploymenttheory/terraform-provider-microsoft365),
with service implementations added under `internal/services/` and SDK connections owned by
`internal/client/`. See the [architecture](docs/plans/activedirectory-provider/architecture.md)
and [implementation sequence](docs/plans/activedirectory-provider/implementation-plan.md).
See [provider registrations](docs/development/provider-registrations.md) for production constructor
lists and the isolated factory pattern used by feature tests.
The [client acceptance guide](docs/development/client-acceptance.md) describes the real Terraform
connection probe and its lab-only environment variables.

Report defects through [GitHub issues](https://github.com/deploymenttheory/terraform-provider-activedirectory/issues).
See [CONTRIBUTING.md](CONTRIBUTING.md) for contribution guidance.
