# Persistent KDS lab fixtures

The approved LAB-03 work includes KDS/gMSA prerequisites. An ordinary gMSA needs a
KDS root key in its forest; the service-account provider must not create this
infrastructure during Terraform resource creation. The lab helper is separate
from provider execution and uses authenticated PowerShell Direct.

Run from PowerShell 7 on the owned Hyper-V host, using the existing private
two-DC profile and its protected credential file:

```powershell
# Inspect the existing key metadata. No key is created without -Provision.
./scripts/lab/service-account-kds.ps1 -ServerVersion 2016

# Bootstrap one missing key as persistent infrastructure for this isolated lab.
./scripts/lab/service-account-kds.ps1 -ServerVersion 2016 -Provision
```

The helper validates the profile, private switch, VM identities, observed AD DS
topology, selected server versions and synchronized DCs. It starts only owned DCs
that were originally off and restores their original power states. Member and
Linux VMs remain outside this operation. Receipts live in the ignored
`tmp/lab-kds/` directory.

Creation is permitted only when both DCs report zero keys. The helper journals
uncertainty before one `Add-KdsRootKey -EffectiveImmediately` submission, records
the returned GUID immediately, and checks replication through read-only GUID/time
observations on both DCs. A later invocation first observes actual state and reuses
an existing single key. Disagreement or multiple keys prevents another creation.
It never serializes a `Get-KdsRootKey` object, key data, passwords or credentials.

Keys remain as persistent lab prerequisites after subject/root/TLS fixture cleanup.
They are not temporary test objects. The helper does not delete or rotate them,
backdate timestamps, change clocks, restart KDC services or relax any policy.
Do not run the old SDK lab-wide configuration script to address this prerequisite:
that script also changes unrelated settings and uses a different timing approach.

Microsoft documents a ten-hour convergence interval for a multi-DC environment.
Its expedited backdating recipe is explicitly for single-DC test environments;
these profiles have two DCs. The receipt reports `awaiting-key-maturity` until both
DC observations pass the ten-hour age and effective key interval checks.
Replication alone is not readiness. See the
[Microsoft KDS root-key guidance](https://learn.microsoft.com/en-us/windows-server/identity/ad-ds/manage/group-managed-service-accounts/group-managed-service-accounts/create-the-key-distribution-services-kds-root-key).

While keys mature, continue unrelated provider implementation and tests. Rerun
inspection before actual Windows/Linux gMSA qualification. Missing or immature
fixtures remain pending; they are not passing or unsupported results.

The initial Server 2016 diagnostic observed zero keys on both DCs. Both a minimal
native `New-ADServiceAccount` request and the lifecycle-shaped request failed with
native error code `-2146893811`, leaving no gMSA. The equivalent provider case also
failed and its owned subjects/root/TLS fixtures were independently retired.
This evidence identifies an infrastructure prerequisite, not an SDK behavior fix.
The diagnostic receipts are `service-account-preflight-2016-20261119t205752624z`
and `sn-16-20261119t205752775z`; the failed provider receipt is
`sa-16-20261119t205108524z`. Historical receipts remain unchanged locally.

Server 2016 provisioning created one key on 19 November 2026. The independent
read-only rerun `kds-2016-20261119t210533492z` confirmed the same key on both DCs,
created no second key and restored both original power states. Its reported
maturity threshold is `2026-11-20T07:03:51.091Z`; inspect both DCs again before use.
The native KDS key ID and AD object GUID are separate identifiers. The first
provisioning run incorrectly compared them after successful creation/replication;
that historical failure receipt and its matching helper source hash are retained.
The corrected helper records both identities and matches the native result to
the key ID, not to `objectGUID`.

This helper covers only the KDS portion of LAB-03. RODC, trust, policy, dMSA and
disposable high-impact fixtures remain separate work; this PR must not close the
whole infrastructure ticket.
