# Terraform Provider for Active Directory

Terraform Plugin Framework provider for Active Directory Domain Services, built on the
[Deployment Theory Active Directory SDK](https://github.com/deploymenttheory/go-sdk-activedirectory).

The provider is under development. The current foundation serves Terraform plugin protocol V6;
directory resources, data sources, list resources and actions are being added through the
[approved implementation backlog](docs/plans/activedirectory-provider/backlog/INDEX.md).
The foundation does not connect to Active Directory or manage directory objects.

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

provider "activedirectory" {}
```

The provider has not been published to the Terraform Registry. This configuration becomes usable
after installing a development build; the empty foundation has no authentication settings yet.
Secure SDK authentication configuration is the next client foundation step.

## Development

Run commands from the repository root:

```shell
go mod download
go test ./...
go vet ./...
go build -o bin/terraform-provider-activedirectory .
golangci-lint run --fix=false
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

Report defects through [GitHub issues](https://github.com/deploymenttheory/terraform-provider-activedirectory/issues).
See [CONTRIBUTING.md](CONTRIBUTING.md) for contribution guidance.
