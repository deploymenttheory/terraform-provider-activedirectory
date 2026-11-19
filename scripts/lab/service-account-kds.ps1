#Requires -Version 7.0
<#
.SYNOPSIS
Inspect or provision the single KDS root key in an existing owned two-DC lab profile.
.DESCRIPTION
Uses authenticated PowerShell Direct against the private Hyper-V lab. Provisioning
is explicit, happens only when both DCs report no keys, and uses the documented
EffectiveImmediately operation. Keys are persistent lab infrastructure. This helper
never backdates, deletes, exports, or replaces a key. Only GUID/time metadata enters
the receipt. Original DC power states are restored even when an operation fails.
#>
[CmdletBinding()]
param(
    [Parameter(Mandatory)][ValidateSet('2016', '2019', '2022', '2025')][string]$ServerVersion,
    [string]$LabRoot = 'H:\ad-sdk-lab',
    [switch]$Provision
)
$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest

$profileRoot = Join-Path ([IO.Path]::GetFullPath($LabRoot)) ('server' + $ServerVersion)
$profile = Get-Content -LiteralPath (Join-Path $profileRoot 'state.json') -Raw | ConvertFrom-Json
if ([IO.Path]::GetFullPath($profile.root) -cne $profileRoot -or
    $profile.domain -cne 'ad.sdk.test' -or $profile.server_version -cne $ServerVersion) { throw 'Owned lab profile differs' }
$controllers = @($profile.machines | Where-Object role -In @('dc1', 'dc2'))
if ($controllers.Count -ne 2 -or @($controllers.role | Sort-Object -Unique).Count -ne 2) { throw 'Exactly two owned writable DCs are required' }
Import-Module Hyper-V
$switch = Get-VMSwitch -Id ([guid]$profile.switch_id)
if ($switch.SwitchType -ne 'Private' -or $switch.Notes -cne ('ad-sdk-lab:' + $profile.id)) { throw 'Owned private lab switch differs' }
$original = @{}
foreach ($entry in $controllers) {
    $vm = Get-VM -Id ([guid]$entry.id)
    $nics = @(Get-VMNetworkAdapter -VM $vm)
    if ($vm.Name -cne $entry.name -or $vm.Notes -cne ('ad-sdk-lab:' + $profile.id) -or
        $vm.State -notin @('Running', 'Off') -or $nics.Count -ne 1 -or $nics[0].SwitchId.ToString() -cne $profile.switch_id) {
        throw 'Owned DC identity, state or network differs'
    }
    $original[$entry.id] = $vm.State.ToString()
}
$repoRoot = Split-Path (Split-Path $PSScriptRoot -Parent) -Parent
$outputRoot = Join-Path $repoRoot 'tmp/lab-kds'
for ($ancestor = $outputRoot; $ancestor; $ancestor = [IO.Path]::GetDirectoryName($ancestor)) {
    if ((Test-Path -LiteralPath $ancestor) -and ((Get-Item -LiteralPath $ancestor).Attributes -band [IO.FileAttributes]::ReparsePoint)) {
        throw 'Receipt path crosses a reparse point'
    }
}
$null = New-Item -ItemType Directory -Path $outputRoot -Force
$run = 'kds-' + $ServerVersion + '-' + [DateTime]::UtcNow.ToString('yyyyMMddTHHmmssfffZ').ToLowerInvariant()
$receiptPath = Join-Path $outputRoot ($run + '.json')
$record = [ordered]@{
    run = $run; profile_id = $profile.id; version = $ServerVersion; phase = 'starting'; provision_requested = [bool]$Provision
    key_created = $false; created_key_id = $null; before = @(); after = @(); original_vm_states = $original
    vm_baseline_restored = $false; persistent_lab_infrastructure = $true; secret_attributes_requested = $false
    source_sha256 = (Get-FileHash -LiteralPath $PSCommandPath -Algorithm SHA256).Hash
}
function Save-Receipt { $record | ConvertTo-Json -Depth 10 | Set-Content -LiteralPath $receiptPath -Encoding utf8NoBOM }

# No Get-KdsRootKey object is serialized: it contains more information than this preflight needs.
$observe = {
    param($ExpectedName)
    $ErrorActionPreference = 'Stop'
    if ($env:COMPUTERNAME -cne $ExpectedName) { throw 'Guest identity differs' }
    $env:ADPS_LoadDefaultDrive = '0'
    Import-Module ActiveDirectory
    $server = $ExpectedName.ToLowerInvariant() + '.ad.sdk.test'
    $root = Get-ADRootDSE -Server $server -Properties isSynchronized -ErrorAction Stop
    $capabilities = @($root.supportedCapabilities | ForEach-Object { [string]$_.Value })
    if ($root.dnsHostName -ine $server -or $root.defaultNamingContext -cne 'DC=ad,DC=sdk,DC=test' -or
        [string]$root.isSynchronized -ine 'TRUE' -or $capabilities -cnotcontains '1.2.840.113556.1.4.800' -or
        $capabilities -ccontains '1.2.840.113556.1.4.1920') { throw 'Synchronized writable AD DS target did not verify' }
    $domainControllers = @(Get-ADDomainController -Server $server -Filter * -ErrorAction Stop)
    if ($domainControllers.Count -ne 2 -or @($domainControllers | Where-Object IsReadOnly).Count -ne 0) { throw 'Expected two writable lab DCs' }
    $base = 'CN=Master Root Keys,CN=Group Key Distribution Service,CN=Services,' + $root.configurationNamingContext
    $keys = @(Get-ADObject -Server $server -SearchBase $base -SearchScope OneLevel -LDAPFilter '(objectClass=msKds-ProvRootKey)' `
        -Properties 'msKds-CreateTime', 'msKds-UseStartTime' -ResultSetSize 3 -ErrorAction Stop)
    if ($keys.Count -gt 1) { throw 'Multiple KDS keys require independent reconciliation; none will be created or removed' }
    $metadata = @()
    foreach ($key in $keys) {
        [long]$created = $key.'msKds-CreateTime'
        [long]$effective = $key.'msKds-UseStartTime'
        if ($created -le 0 -or $effective -le 0) { throw 'Key time metadata is incomplete' }
        [long]$cycle = 360000000000
        [long]$effectiveBoundary = $effective - ($effective % $cycle)
        if ($effectiveBoundary -lt $effective) { $effectiveBoundary += $cycle }
        $createdAt = [DateTime]::FromFileTimeUtc($created)
        $eligibleAt = $createdAt.AddHours(10)
        $boundary = [DateTime]::FromFileTimeUtc($effectiveBoundary)
        if ($boundary -gt $eligibleAt) { $eligibleAt = $boundary }
        $metadata += [pscustomobject]@{
            guid = $key.ObjectGUID.ToString(); key_id = ([guid]$key.Name).ToString(); created_utc = $createdAt.ToString('o')
            effective_utc = [DateTime]::FromFileTimeUtc($effective).ToString('o')
            eligible_after_utc = $eligibleAt.ToString('o'); mature = $root.currentTime.ToUniversalTime() -ge $eligibleAt
        }
    }
    $os = Get-CimInstance Win32_OperatingSystem -ErrorAction Stop
    [pscustomobject]@{
        server = $server; caption = $os.Caption; build = $os.BuildNumber; observed_utc = $root.currentTime.ToUniversalTime().ToString('o')
        configuration_partition = [string]$root.configurationNamingContext
        controllers = @($domainControllers.HostName | Sort-Object); keys = $metadata
    }
}
$mutex = [Threading.Mutex]::new($false, ('Local\TerraformActiveDirectoryKds-' + $profile.id))
$held = $false
$sessions = @{}
$started = [Collections.Generic.List[guid]]::new()
$credential = $null
try {
    try { $held = $mutex.WaitOne(0) } catch [Threading.AbandonedMutexException] { $held = $true }
    if (-not $held) { throw 'Another KDS profile operation is active' }
    Save-Receipt
    $saved = Import-Clixml -LiteralPath (Join-Path $profileRoot 'credential.clixml')
    $credential = [PSCredential]::new('ADSDK\Administrator', $saved.Password)
    $saved = $null
    foreach ($entry in $controllers) {
        if ($original[$entry.id] -ceq 'Off') {
            $started.Add([guid]$entry.id)
            Start-VM -VM (Get-VM -Id ([guid]$entry.id)) | Out-Null
        }
    }
    foreach ($entry in $controllers) {
        $deadline = [DateTime]::UtcNow.AddMinutes(5)
        do {
            $session = $null
            try {
                $session = New-PSSession -VMId ([guid]$entry.id) -Credential $credential
                $observed = Invoke-Command -Session $session -ArgumentList $entry.name -ScriptBlock $observe
                $sessions[$entry.id] = $session
                $record.before += $observed
                break
            } catch {
                if ($session) { Remove-PSSession $session }
                if ([DateTime]::UtcNow -ge $deadline) { throw 'Owned DC readiness or key observation did not verify; no create is authorized' }
                Start-Sleep -Seconds 5
            }
        } while ($true)
    }
    $expectedControllers = @($controllers | ForEach-Object { $_.name.ToLowerInvariant() + '.ad.sdk.test' } | Sort-Object)
    foreach ($observation in $record.before) {
        if (($observation.controllers -join ',') -ine ($expectedControllers -join ',') -or
            $observation.configuration_partition -cne $record.before[0].configuration_partition -or
            $observation.caption.IndexOf($ServerVersion, [StringComparison]::OrdinalIgnoreCase) -lt 0) { throw 'Observed forest topology or server version differs' }
    }
    Save-Receipt
    $firstKeys = @($record.before[0].keys)
    $secondKeys = @($record.before[1].keys)
    if ($firstKeys.Count -eq 0 -and $secondKeys.Count -eq 0 -and $Provision) {
        # Durable uncertainty precedes the one native submission. Reruns reobserve keys; they never replay a lost create response.
        $record.phase = 'key-create-submission-uncertain'
        Save-Receipt
        $dc1 = $controllers | Where-Object role -EQ dc1
        $created = Invoke-Command -Session $sessions[$dc1.id] -ArgumentList $dc1.name -ScriptBlock {
            param($ExpectedName)
            $ErrorActionPreference = 'Stop'
            if ($env:COMPUTERNAME -cne $ExpectedName) { throw 'Selected key-creation DC differs' }
            $env:ADPS_LoadDefaultDrive = '0'
            Import-Module ActiveDirectory
            $root = Get-ADRootDSE -Server ($ExpectedName + '.ad.sdk.test') -ErrorAction Stop
            $base = 'CN=Master Root Keys,CN=Group Key Distribution Service,CN=Services,' + $root.configurationNamingContext
            if (@(Get-ADObject -Server ($ExpectedName + '.ad.sdk.test') -SearchBase $base -SearchScope OneLevel -LDAPFilter '(objectClass=msKds-ProvRootKey)' -ResultSetSize 2 -ErrorAction Stop).Count -ne 0) {
                throw 'A key appeared before submission; no duplicate key was created'
            }
            Import-Module Kds
            ([guid](Add-KdsRootKey -EffectiveImmediately -ErrorAction Stop)).ToString()
        }
        $record.key_created = $true
        $record.created_key_id = ([guid]$created).ToString()
        $record.phase = 'key-created-awaiting-replication'
        Save-Receipt
    } elseif ($firstKeys.Count -ne $secondKeys.Count -or
        ($firstKeys.Count -eq 1 -and $firstKeys[0].guid -cne $secondKeys[0].guid)) {
        throw 'Existing key observations disagree; reconcile replication before provisioning'
    }
    $deadline = [DateTime]::UtcNow.AddMinutes(3)
    do {
        $record.after = @()
        foreach ($entry in $controllers) { $record.after += Invoke-Command -Session $sessions[$entry.id] -ArgumentList $entry.name -ScriptBlock $observe }
        Save-Receipt
        $firstKeys = @($record.after[0].keys)
        $secondKeys = @($record.after[1].keys)
        if ($firstKeys.Count -eq 1 -and $secondKeys.Count -eq 1 -and $firstKeys[0].guid -ceq $secondKeys[0].guid) { break }
        if (-not $record.key_created -or [DateTime]::UtcNow -ge $deadline) { break }
        Start-Sleep -Seconds 5
    } while ($true)
    $record.replication_verified = $firstKeys.Count -eq 1 -and $secondKeys.Count -eq 1 -and $firstKeys[0].guid -ceq $secondKeys[0].guid
    if ($record.key_created -and (-not $record.replication_verified -or $record.created_key_id -cne $firstKeys[0].key_id)) {
        throw 'Created key replication did not verify; preserve the receipt and reobserve without creating another key'
    }
    if ($record.replication_verified) { $record.directory_key_guid = $firstKeys[0].guid; $record.key_id = $firstKeys[0].key_id }
    $record.ready = $record.replication_verified -and [bool]$firstKeys[0].mature -and [bool]$secondKeys[0].mature
    $record.phase = if ($record.ready) { 'ready' } elseif ($record.replication_verified) { 'awaiting-key-maturity' } else { 'missing-key' }
    Save-Receipt
} finally {
    try {
        foreach ($session in $sessions.Values) { Remove-PSSession $session }
        foreach ($id in $started) {
            $vm = Get-VM -Id $id
            if ($vm.Notes -cne ('ad-sdk-lab:' + $profile.id)) { throw 'Started DC ownership changed before shutdown' }
            if ($vm.State -eq 'Running') { Stop-VM -VM $vm -Confirm:$false }
        }
        if ($held) {
            foreach ($entry in $controllers) {
                if ((Get-VM -Id ([guid]$entry.id)).State.ToString() -cne $original[$entry.id]) { throw 'Original DC power state did not restore' }
            }
            $record.vm_baseline_restored = $true
            $record.finished_utc = [DateTime]::UtcNow.ToString('o')
            Save-Receipt
        }
    } finally {
        $credential = $null
        if ($held) { $mutex.ReleaseMutex() }
        $mutex.Dispose()
    }
    Write-Output ('KDS fixture receipt: ' + $receiptPath)
}
Write-Output ([pscustomobject]@{ version = $ServerVersion; phase = $record.phase; created = $record.key_created; ready = $record.ready; vm_baseline_restored = $record.vm_baseline_restored } | ConvertTo-Json -Compress)
