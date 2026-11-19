#Requires -Version 5.1
# Copyright (c) 2026 Deployment Theory.
# SPDX-License-Identifier: MIT

# Get-ProviderServiceAccountPrerequisites reads selected-DC class/KDS metadata only; it never creates keys or requests secret attributes.
function Get-ProviderServiceAccountPrerequisites([string]$Server, [string]$ExpectedPartition) {
    $ErrorActionPreference = 'Stop'
    $env:ADPS_LoadDefaultDrive = '0'
    Import-Module ActiveDirectory
    $root = Get-ADRootDSE -Server $Server -Properties isSynchronized -ErrorAction Stop
    $capabilities = @($root.supportedCapabilities | ForEach-Object { [string]$_.Value })
    if ($root.dnsHostName -ine $Server -or $root.defaultNamingContext -cne $ExpectedPartition -or
        $capabilities -cnotcontains '1.2.840.113556.1.4.800' -or
        $capabilities -ccontains '1.2.840.113556.1.4.1920' -or
        [string]$root.isSynchronized -ine 'TRUE') { throw 'Selected writable AD DS target did not verify' }
    if ($root.currentTime -isnot [DateTime]) { throw 'Selected DC time metadata is unavailable' }
    $now = $root.currentTime.ToUniversalTime()
    [long]$ticks = $now.ToFileTimeUtc()
    [long]$cycle = 360000000000
    [long]$intervalStart = $ticks - ($ticks % $cycle)
    $classes = @()
    foreach ($name in @('msDS-ManagedServiceAccount', 'msDS-GroupManagedServiceAccount', 'msDS-DelegatedManagedServiceAccount')) {
        $classRows = @(Get-ADObject -Server $Server -SearchBase $root.schemaNamingContext -SearchScope OneLevel `
            -LDAPFilter ('(&(objectClass=classSchema)(ldapDisplayName=' + $name + '))') `
            -Properties ldapDisplayName, isDefunct, objectClassCategory -ResultSetSize 2 -ErrorAction Stop)
        if ($classRows.Count -gt 1) { throw 'Service account class observation is ambiguous' }
        $available = $classRows.Count -eq 1 -and -not [bool]$classRows[0].isDefunct -and [int]$classRows[0].objectClassCategory -eq 1
        $classes += [pscustomobject]@{ name = $name; available = $available }
    }
    $base = 'CN=Master Root Keys,CN=Group Key Distribution Service,CN=Services,' + $root.configurationNamingContext
    $keys = @(Get-ADObject -Server $Server -SearchBase $base -SearchScope OneLevel -LDAPFilter '(objectClass=msKds-ProvRootKey)' `
        -Properties 'msKds-CreateTime', 'msKds-UseStartTime' -ResultSetSize 128 -ErrorAction Stop)
    if ($keys.Count -ge 128) { throw 'Bounded KDS metadata query cannot establish completion' }
    $metadata = @()
    foreach ($key in $keys) {
        [long]$created = $key.'msKds-CreateTime'
        [long]$effective = $key.'msKds-UseStartTime'
        if ($created -le 0 -or $effective -le 0) { throw 'KDS time metadata is unavailable' }
        $createdAt = [DateTime]::FromFileTimeUtc($created)
        $metadata += [pscustomobject]@{
            guid = $key.ObjectGUID.ToString()
            created_utc = $createdAt.ToString('o')
            effective_utc = [DateTime]::FromFileTimeUtc($effective).ToString('o')
            eligible_interval = $effective -le $intervalStart
            aged_ten_hours = $createdAt.AddHours(10) -le $now
        }
    }
    [pscustomobject]@{
        server = $Server
        partition = [string]$root.defaultNamingContext
        dc_functionality = [int]$root.domainControllerFunctionality
        domain_functionality = [int]$root.domainFunctionality
        forest_functionality = [int]$root.forestFunctionality
        observed_utc = $now.ToString('o')
        schema = $classes
        kds_metadata = $metadata
        secret_attributes_requested = $false
        directory_mutations = 0
    }
}
