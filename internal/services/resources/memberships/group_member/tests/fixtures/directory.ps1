#Requires -Version 5.1
# Copyright (c) 2026 Deployment Theory.
# SPDX-License-Identifier: MIT

# Remove-ProviderGroupMemberChildren reconciles only marker-owned leaf groups, users, computers and OUs below the independently verified run root.
function Remove-ProviderGroupMemberChildren([string]$RunId, [string]$Server, [string]$ExpectedGuid) {
    $ErrorActionPreference = 'Stop'
    $null = Assert-PortableTlsFixtureIdentity $RunId $Server
    $dn = Get-PortableDirectoryFixtureDn $RunId
    $root = Get-ADObject -Server $Server -Identity ([guid]$ExpectedGuid) -Properties description -ErrorAction Stop
    if ($root.DistinguishedName -cne $dn -or $root.ObjectClass -cne 'organizationalUnit' -or
        $root.description -cne ('sdk-portable-parent:' + $RunId)) { throw 'Owned group member fixture root differs' }
    $children = @(Get-ADObject -Server $Server -SearchBase $dn -SearchScope Subtree -LDAPFilter '(objectClass=*)' `
        -Properties description -ErrorAction Stop | Where-Object ObjectGUID -NE $root.ObjectGUID)
    if ($children.Count -gt 20) { throw 'Unexpected fixture count; preserve for reconciliation' }
    $markers = @(('tfad-owned:' + $RunId + ':windows'), ('tfad-owned:' + $RunId + ':linux'))
    foreach ($child in $children) {
        if (([string]$child.ObjectClass).ToLowerInvariant() -cnotin @('organizationalunit', 'group', 'user', 'computer') -or [string]$child.description -cnotin $markers -or
            -not ([string]$child.Name).StartsWith(('tfad-' + $RunId + '-'), [StringComparison]::Ordinal) -or
            -not ([string]$child.DistinguishedName).EndsWith(',' + $dn, [StringComparison]::OrdinalIgnoreCase)) {
            throw 'Unowned or moved descendant prevents group member fixture cleanup'
        }
    }
    $removed = @()
    foreach ($child in @($children | Sort-Object { $_.DistinguishedName.Length } -Descending)) {
        $actual = Get-ADObject -Server $Server -Identity $child.ObjectGUID -Properties description -ErrorAction Stop
        if ($actual.DistinguishedName -cne $child.DistinguishedName -or $actual.ObjectClass -cne $child.ObjectClass -or
            $actual.description -cne $child.description) { throw 'Fixture identity changed before cleanup' }
        $descendants = @(Get-ADObject -Server $Server -SearchBase $actual.DistinguishedName -SearchScope OneLevel `
            -LDAPFilter '(objectClass=*)' -ErrorAction Stop)
        if ($descendants.Count -ne 0) { throw 'Leaf cleanup refuses a nonempty fixture object' }
        Remove-ADObject -Server $Server -Identity $actual.ObjectGUID -Confirm:$false -ErrorAction Stop
        $hex = ($actual.ObjectGUID.ToByteArray() | ForEach-Object { '\' + $_.ToString('x2') }) -join ''
        if (@(Get-ADObject -Server $Server -LDAPFilter ('(objectGUID=' + $hex + ')') -ErrorAction Stop).Count -ne 0) {
            throw 'Owned fixture child remains live'
        }
        $removed += @{ guid = $actual.ObjectGUID.ToString(); dn = $actual.DistinguishedName }
    }
    [pscustomobject]@{ verified = $true; reconciled_children = $removed; remaining = 0 }
}
