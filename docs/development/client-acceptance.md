# Provider client acceptance

The client foundation pins Active Directory SDK v0.2.0. Unit tests use narrow interfaces over
public SDK types. Live acceptance uses a test-only connection probe registered with the shared
core; it is never exposed as a production data source. Terraform configures the provider,
validates negotiated protection and AD DS capabilities, and reads authenticated RootDSE again
through the configured client. Assertions check the lab endpoint and naming context.

Enable live acceptance explicitly on an owned lab client:

```powershell
$env:TF_ACC = "1"
$env:TF_ACC_AD_LAB = "1"
$env:TF_ACC_TERRAFORM_PATH = (Get-Command terraform).Source
$env:AD_HOST = "adsdk25dc1.ad.sdk.test"
$env:AD_AUTH_METHOD = "simple"
$env:AD_TRANSPORT = "ldaps"
# Supply AD_USERNAME, AD_PASSWORD and AD_CA_CERT_PEM through the lab's secret delivery.
go test ./internal/provider/core -run '^TestAccProviderLabConnection$' -count=1 -timeout=10m
```

```shell
TF_ACC=1 TF_ACC_AD_LAB=1 TF_ACC_TERRAFORM_PATH="$(command -v terraform)" \
  go test ./internal/provider/core -run '^TestAccProviderLabConnection$' -count=1 -timeout=10m
```

Both commands execute Terraform CLI. The probe expects `DC=ad,DC=sdk,DC=test` and the exact
`AD_HOST` FQDN in the lab RootDSE. Run the same test separately for each applicable Server
2016/2019/2022/2025 endpoint from Linux and Windows. A successful SDK-only test or a cross-build
does not qualify Terraform acceptance or another server/client target.

Authentication scenarios use the provider's ordinary configuration environment variables:

| Scenario | AD_AUTH_METHOD | AD_TRANSPORT | Additional settings |
| --- | --- | --- | --- |
| Verified LDAPS simple bind | simple | ldaps | AD_USERNAME, AD_PASSWORD; private CA when required |
| Verified StartTLS simple bind | simple | starttls | AD_USERNAME, AD_PASSWORD; private CA when required |
| Kerberos with TLS channel binding | kerberos | ldaps or starttls | Explicit realm, SPN and credential source |
| Kerberos integrity | kerberos | ldap | AD_SASL_PROTECTION=sign; explicit Kerberos identity/source |
| Kerberos confidentiality | kerberos | ldap | AD_SASL_PROTECTION=seal; explicit Kerberos identity/source |

Kerberos requires `AD_KERBEROS_REALM`, `AD_KERBEROS_SPN` and
`AD_KERBEROS_CREDENTIAL_SOURCE=password|keytab|credential_cache`. Supply only that source's
password, keytab path or cache path. Do not leave a password variable set while testing a file
source. Caches/keytabs remain caller-owned and are never generated or renewed by the provider.

Provider aliases have independent clients. The provider publishes the same bundle to all four
Configure interfaces. SDK operations serialize access to a shared session. Configure completion
does not close it. The protocol V6 stop adapter and test cleanup release sessions explicitly;
abrupt process termination remains the operating system's responsibility. No mutation retry or
automatic authentication/transport downgrade is introduced.

Generate provider docs from the committed template and examples with `go generate ./...`.
The generator is pinned to tfplugindocs v0.25.0 in `main.go`. The four-surface documentation
coverage and drift gate are tracked separately in FND-05.
