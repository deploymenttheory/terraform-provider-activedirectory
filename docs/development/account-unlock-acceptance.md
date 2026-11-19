# Account unlock acceptance

`TestAccAccountUnlock` executes actual Terraform action triggers over production SDK clients.
The test first refuses a different DC, then verifies an ordinary account's denied write, and
finally performs an explicitly selected administrator unlock. A separate authenticated SDK
observer checks the target GUID, exact `lockoutTime` and unchanged stored account-control flags.
The account stays disabled. A repeated no-change plan does not trigger another action; destroying
the Terraform trigger resource leaves the directory account in place until fixture cleanup.

Follow the authorization, allowlist and protected authentication requirements in
[testing](testing.md). Prepare the SDK's marker-owned TLS and empty directory fixtures for the
selected run. The parent DN must be `OU=sdk-portable-<run-id>,<partition>` with description
`sdk-portable-parent:<run-id>` and an independently observed GUID. Prepare an ordinary SDK account
with `kerberos-account.ps1 -Action Prepare`; no privileged group or policy change is required.

Dot-source the feature-local
[directory fixture](../../internal/services/actions/accounts/account_unlock/tests/fixtures/directory.ps1)
on the owned DC alongside the SDK's `portable-tls-fixture.ps1`, `portable-directory-fixture.ps1`
and `kerberos-write-fixture.ps1`. Call `New-ProviderAccountUnlockFixture` with the run ID, DC,
verified parent GUID, `windows` or `linux` platform, ordinary actor run ID/GUID and existing
protected setup credential. It creates a fresh random-password account and a marker-owned password
settings object applied directly to that account alone. It verifies that scope and effective
three-attempt threshold, performs at most three invalid-password binds over verified LDAPS, then
independently observes lockout and disables the account before Terraform runs. It denies only
that attribute's write permission to the ordinary actor. Existing accounts, policies and group
memberships are unchanged. Intent is journaled and exact ACL content is verified while allowing
AD's canonical ACE ordering.

The setup observes real lockout before disabling its fresh subject; the action checks resetting
the stored timestamp without enabling it. A nonzero `lockoutTime` alone does not prove current
account lockout or login eligibility. The default domain password policy is never changed.
See Microsoft's
[Lockout-Time attribute documentation](https://learn.microsoft.com/en-us/windows/win32/adschema/a-lockouttime).

Supply these process environment variables in addition to the common lab inputs:

| Input | Value |
| --- | --- |
| `AD_ACCEPTANCE_PARENT_DN` | Verified owned outer OU DN |
| `AD_ACCEPTANCE_ACCOUNT_GUID` | Returned subject account GUID |
| `AD_ACCEPTANCE_ACCOUNT_DN` | Returned subject account DN |
| `TF_VAR_reader_upn` | Ordinary actor UPN |
| `TF_VAR_reader_password` | Ordinary actor password from protected storage |

The alias password variable is sensitive and ephemeral. Credentials must remain in protected
storage or process environment, never command arguments, configuration files or captured logs.

```shell
go test ./internal/services/actions/accounts/account_unlock \
  -run '^TestAccAccountUnlock$' -count=1 -timeout=15m
```

The shared PowerShell/Bash runners also accept checksum-pinned test binaries. Run the same case
from Windows and Linux amd64 against Server 2016, 2019, 2022 and 2025. Record source revision,
binary/tool hashes, selected DC/transport, actual result and independent cleanup evidence.
Unit tests separately cover cancellation, capability refusal, accepted/rejected/uncertain writes,
unavailable/conflicting readback, redaction and actual Terraform refusal/recovery sequencing.

The test removes only its exact GUID after validating DN and ownership marker, then proves
deletion independently. After its process retires, `Remove-ProviderAccountChildren` reconciles
any retained owned subject and checks every journaled GUID, refusing moved or unowned objects.
It then verifies that each owned password settings object has no remaining subjects before
deleting it by its verified GUID and marker and independently confirming absence.
Remove the empty outer directory fixture, retire the ordinary actor with its SDK cleanup helper,
then retire TLS. Restore original VM/client profile state. Missing or uncertain retirement and
cleanup remains a failure; never rerun an unknown write to manufacture a passing result.
