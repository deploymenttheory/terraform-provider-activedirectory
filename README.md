# Terraform Provider for Active Directory

Developer documentation: [testing](docs/development/client-acceptance.md) and
[examples and generated documentation](docs/development/documentation.md).

Terraform Plugin Framework provider for Active Directory Domain Services, built on the
[Deployment Theory Active Directory SDK](https://github.com/deploymenttheory/go-sdk-activedirectory).

The provider is under development. The current foundation serves Terraform plugin protocol V6;
directory resources, data sources, list resources and actions are being added through the
[approved implementation backlog](docs/plans/activedirectory-provider/backlog/INDEX.md).
Provider configuration opens and qualifies a protected AD DS session using the public SDK.
Organizational unit lifecycle and discovery passed real Terraform lab acceptance from Windows and
Linux amd64 against Windows Server 2016, 2019, 2022 and 2025. The 32 cases cover CRUD, rename/move,
GUID/identity import, generated configuration, data/list discovery, explicit clearing, protected
and nonempty deletion, denied writes and hidden attributes. Owned fixture cleanup was verified.

| Surface | Type | Documentation |
| --- | --- | --- |
| Resource | `activedirectory_organizational_unit` | [Lifecycle, protection and import](docs/resources/organizational_unit.md) |
| Data source | `activedirectory_organizational_unit` | [GUID/DN lookup](docs/data-sources/organizational_unit.md) |
| Data source | `activedirectory_organizational_units` | [Bounded discovery](docs/data-sources/organizational_units.md) |
| List resource | `activedirectory_organizational_unit` | [Query and import discovery](docs/list-resources/organizational_unit.md) |

OUs retain their GUID through same-domain rename/move. Deletion is nonrecursive and never
disables protection automatically. Omitted optional properties remain unmanaged; clearing an
attribute requires explicit `clear_properties` intent. Uncertain writes retain recovery state
and block further mutations until reconciliation.

The `activedirectory_account_unlock` action includes unit tests, examples and
[generated documentation](docs/actions/account_unlock.md). Actual Terraform acceptance passed from
Windows/Linux amd64 against Server 2016, 2019, 2022 and 2025, including target refusal, denied writes,
explicit recovery, retained disabled status and independently verified fixture/policy cleanup.
It selects an account by GUID on an explicitly acknowledged DC and never enables a disabled
account or retries a failed or uncertain unlock automatically.
This branch adds the `activedirectory_user` resource, singular/plural user data sources and
the `activedirectory_user` list resource. User management supports GUID import, rename/move,
typed properties, account flags, expiration and versioned write-only passwords.
See the [user resource](docs/resources/user.md) and [user acceptance guide](docs/development/user-acceptance.md).

Computer lifecycle, singular/plural data sources and list discovery are implemented with typed
account settings, versioned write-only passwords and stable GUID import. This manages directory
objects only; it never joins or configures a host. See the [computer resource](docs/resources/computer.md)
and [computer acceptance guide](docs/development/computer-acceptance.md). The lifecycle and password-observation
cases pass from Windows/Linux amd64 against Server 2016, 2019, 2022 and 2025 using verified LDAPS.

This branch adds the `activedirectory_group` resource, singular/plural group data sources and
the `activedirectory_group` list resource. Group management supports GUID import, same-domain
rename/move, typed properties and semantic category/scope changes. Membership belongs to
separate edge resources; deleting a group does not delete its members.
See the [group resource](docs/resources/group.md) for ownership and recovery guidance.
The [group acceptance guide](docs/development/group-acceptance.md) describes live lab cases
and their independently verified cleanup requirements.

Standalone and group-managed service accounts have a [managed lifecycle](docs/resources/service_account.md),
[single-account](docs/data-sources/service_account.md) and [bounded-query](docs/data-sources/service_accounts.md)
data sources, and [list discovery](docs/list-resources/service_account.md). They retain GUID identity
through rename/move, preserve unrelated reader-policy ACEs through the SDK, and never retrieve
managed passwords or install an account on a host. Explicit sMSA password input is write-only
and creation-only; changing its version replaces the account. See the
[acceptance guide](docs/development/service-account-acceptance.md) for the current lab qualification gate.

Generic object lifecycle, GUID/DN lookup, bounded data sources and list discovery are implemented
with explicit schema classes and binary-safe raw attribute ownership. See the
[object resource](docs/resources/object.md) for import and recovery semantics and the
[object acceptance guide](docs/development/object-acceptance.md) for guarded lab cases.
Both lifecycle and nonempty-leaf acceptance cases pass from Windows/Linux amd64 against
Server 2016, 2019, 2022 and 2025 using verified LDAPS. All 16 cases have independently
verified process, directory and TLS cleanup; the original lab baselines were restored.

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
pwsh ./scripts/run-unit.ps1
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
OU feature acceptance runs against the owned lab from Linux and Windows clients on Windows Server
2016, 2019, 2022 and 2025. See the [OU acceptance guide](docs/development/organizational-unit-acceptance.md)
for fixtures, cases and cleanup. Each subsequent feature needs its own applicable qualification.
macOS receives build and unit coverage.

## Repository layout

```text
main.go                         Provider process and build version
internal/provider/              Provider factory and service registrations
internal/provider/core/         Shared schema and configuration
internal/client/                Public SDK contracts and session qualification
internal/services/              Feature-local resources, data sources and list resources
examples/provider/              Provider configuration example
examples/                       Minimal and maximal examples for each service surface
templates/                      Source templates for generated service documentation
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
See [testing](docs/development/testing.md) for feature-local SDK mocks, isolated Terraform factories,
the Linux unit command and explicit live-lab authorization/cleanup requirements.

Report defects through [GitHub issues](https://github.com/deploymenttheory/terraform-provider-activedirectory/issues).
See [CONTRIBUTING.md](CONTRIBUTING.md) for contribution guidance.
