# Provider and feature testing

The test layout follows the Microsoft 365 provider's feature-local pattern, replacing Graph HTTP
responders with mocks at the public Active Directory SDK boundary. Tests import the shared core
and their feature packages; they do not import the production registration package.

`internal/mocks.Session` requires explicit read/protection callbacks and fails unexpected calls.
Its factory creates a fresh session on every Configure invocation. `internal/acceptance.ProviderFactories`
registers only the supplied services and closes configured clients during test cleanup. The shared
protocol V6 shutdown adapter lives in the core, keeping the production service registry out of tests.

Run all unit checks without directory credentials:

```powershell
./scripts/run-unit.ps1
```

```shell
bash ./scripts/run-unit.sh
```

These entry points remove AD configuration from the test environment and leave live acceptance
disabled. Terraform-running unit tests set `IsUnitTest: true`, inject SDK mocks and use only
synthetic configuration. The Terraform executable comes from `TF_ACC_TERRAFORM_PATH`, or the
pinned 1.17.2 version through HashiCorp's test harness. Tool installation is distinct from AD access.

For each feature, keep `provider_factories_test.go`, unit tests, acceptance tests, response fixtures
and `tests/terraform/{unit,acceptance}/` beside its implementation. Supply only explicit fixture
dependencies in the factory registry. A list feature also registers its corresponding managed
resource, as required by Framework identity validation.

Live tests call `acceptance.PreCheck(t)` before opening clients or creating fixtures. The guard
requires all of:

- `TF_ACC=1`, `TF_ACC_AD_LAB=1` and explicit `AD_ACCEPTANCE_AUTHORIZED=1`.
- `AD_HOST` matching the explicit comma-separated `AD_ACCEPTANCE_HOSTS` allowlist.
- `AD_ACCEPTANCE_PARTITION`, an externally unique `AD_ACCEPTANCE_RUN_ID`, and the selected
  `AD_ACCEPTANCE_SERVER_VERSION` (2016, 2019, 2022 or 2025).
- A Linux/Windows amd64 client. macOS and arm64 receive build/unit checks; no live qualification
  is inferred from those checks.

Pull-request jobs are refused even if those variables are present. Live jobs belong in an
explicit trusted lab invocation, never an untrusted fork PR. The PowerShell acceptance entry point
requires `-AuthorizeLab`; both platform entry points require the operator to set `TF_ACC=1`.

See [client acceptance](client-acceptance.md) for commands and authentication cases. The scripts
can execute checksum-pinned test binaries on lab clients without installing Go: PowerShell uses
`-TestBinary` and `-ExpectedBinarySHA256`; Bash uses `AD_ACCEPTANCE_TEST_BINARY` and
`AD_ACCEPTANCE_BINARY_SHA256`. Credentials remain in protected files or process environment,
never command arguments or test artifacts.

Use `acceptance.FixtureName(runID, t.Name())` for deterministic names inside a unique run boundary.
Register each feature's owned cleanup and independent absence check with `RegisterCleanup`.
An error, partial result, canceled check or unverified absence fails the test, including after an
earlier test failure. Raw SDK/peer error text is withheld. The callback owns feature-specific
GUID/marker checks; the shared helper does not infer object ownership or replay failed writes.

Successful teardown calls alone are not proof of cleanup. Retain the verification failure and
reconcile accepted or uncertain effects before another mutation. Historical failure records
remain unchanged. Summarize validation in implementation PRs; keep bulk lab logs local.
