# Account settings acceptance

The resource adopts an existing account by GUID and manages only configured
account-control/expiration fields. Delete relinquishes management without deleting
or resetting the account. The singular data source observes settings only.

Feature-local Terraform unit tests exercise adoption, no-change planning, finite
expiration/explicit clearing, drift repair, both lookup forms, string/identity import
with generated configuration, target replacement and relinquishment with the
accounts retained. A deferred Terraform dependency resolves to invalid expiration
at apply time and prevents the earlier enable write. An uncertain submitted effect
must reconcile positively; unresolved private intent blocks tainted replacement.

SDK-boundary tests cover accepted effects followed by failed readback, hidden
configured fields, missing durable storage, malformed recovery state, contradictory
returned GUIDs and no later-phase replay. Complete identity lookup proof is separate
from attribute visibility. Data-source tests preserve missing values as null,
reject invalid/unknown lookup forms before SDK access and redact directory peer text.

## Real lab cases

Both embedded cases pass from Windows/Linux amd64 against Server 2016, 2019,
2022 and 2025: sixteen distinct cases. The independent matrix audit also verifies
source/binary/module provenance, process retirement, directory/reader/TLS cleanup
and restoration of the original lab baseline. Use the
[guarded acceptance runner](testing.md) for later qualification. macOS and arm64
have build/unit coverage only; they are not claimed as live clients.

- `TestAccAccountSettingsLifecycle`: independently provision strong-password,
  disabled user and unjoined computer fixtures plus an alternate user. Terraform
  adopts their settings, changes flags/finite expiration, imports both account
  classes and generates import configuration. GUID/DN lookup includes different DN
  case. Independent owned drift and external rename prove GUID stability and that
  an unconfigured flag is preserved. Explicit clearing, target replacement and
  omission/no-change plans leave previous and current accounts intact. The
  independent observer verifies actual effects before separate fixture retirement.
- `TestAccAccountSettingsDenied`: use an independently created ordinary reader
  with an ephemeral provider password. Read-only adoption succeeds; an attempted
  enable write is refused and the independent observer proves the disabled account
  and its expiration remain unchanged. Resolved relinquishment performs no reset
  or deletion. No privileged group membership or policy change is used.

The observer opens its own portable, verified LDAPS session before fixture setup
and validates the external root GUID/DN/marker. Each subject keeps a permanent
marker, bounded name, expected class and root location. Both creation and bind
passwords are checked against complete Terraform plans and state. No password is
retrieved and no login probe or host/domain join occurs.

After Terraform destroy, every independently provisioned account must still exist.
Then the observer retires only exact, positively owned leaf fixtures and independently
confirms absence. The outer PowerShell fixture also refuses unknown, moved or
nonempty descendants. Process retirement, directory/TLS cleanup, immutable source/
binary/module provenance, actual client/server builds and original VM/Linux
baselines must all verify before a live case is qualified.

These user/computer fixtures require no KDS root key. Managed service-account
qualification has a separate [KDS prerequisite](service-account-kds-fixtures.md)
and must not delay independent account-settings implementation or fabricate passes.

## Qualified revision

The sixteen-case matrix qualified provider runtime
`a9ccfa4423d6f1e8e5af32d8e92a75f8e4f52f14` with SDK
`v0.2.1-0.20261119222120-bfcadb3f012d`, Go 1.27.4 and Terraform 1.17.2.
The initial lower-case-DN comparison exposed an SDK projection defect; SDK PR
[218](https://github.com/deploymenttheory/go-sdk-activedirectory/pull/218) fixes
the returned account names without a provider workaround. The corrected matrix
includes that exact failing comparison.

Local run identifiers are `as-25-20261119t222811104z` (Windows),
`as-25-20261119t223020211z` (Linux), `as-16-20261119t223349174z`,
`as-19-20261119t224628723z` and `as-22-20261119t225124132z` (both clients).
Independent audit: `account-settings-matrix-audit-20261119t225728869z`.
Original failed receipts are retained separately and do not count as passes.
