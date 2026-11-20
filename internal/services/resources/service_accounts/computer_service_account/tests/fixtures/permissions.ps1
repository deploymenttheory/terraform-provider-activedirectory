#Requires -Version 5.1
# Dot-source alongside the verified SDK TLS and directory fixture functions on the owned DC.
function Get-ProviderComputerServiceAccountAclShape([string]$Sddl) {
    # AD lexicographically orders ACEs within canonical groups (MS-ADTS 6.1.3.1).
    # Compare the exact ACE multiset, owner/group and control flags after requiring canonical ordering.
    $security=[DirectoryServices.ActiveDirectorySecurity]::new()
    $security.SetSecurityDescriptorSddlForm($Sddl)
    if(-not$security.AreAccessRulesCanonical){throw 'Permission fixture ACL is not canonical'}
    $raw=[Security.AccessControl.RawSecurityDescriptor]::new($Sddl)
    if(-not$raw.DiscretionaryAcl){throw 'Permission fixture ACL is absent'}
    $aces=@(foreach($ace in $raw.DiscretionaryAcl){
        $bytes=[byte[]]::new($ace.BinaryLength)
        $ace.GetBinaryForm($bytes,0)
        [Convert]::ToBase64String($bytes)
    })
    [Array]::Sort($aces,[StringComparer]::Ordinal)
    return ($raw.Owner.Value+'|'+$raw.Group.Value+'|'+[int]$raw.ControlFlags+'|'+($aces-join'|'))
}

# New-ProviderComputerServiceAccountPermissionFixture owns a fresh computer/account pair and one exact association-property read denial to an ordinary actor.
function New-ProviderComputerServiceAccountPermissionFixture(
    [string]$RunId, [string]$Server, [string]$ExpectedRootGuid, [string]$Platform,
    [string]$ActorRun, [string]$ActorGuid, [PSCredential]$Credential, [PSCredential]$ActorCredential
) {
    $ErrorActionPreference = 'Stop'
    $null = Assert-PortableTlsFixtureIdentity $RunId $Server
    if ($Platform -cnotin @('windows', 'linux') -or $ActorRun -cnotmatch '\A[a-z0-9][a-z0-9-]{5,39}\z') { throw 'Invalid association permission fixture identity' }
    $root = Get-ADOrganizationalUnit -Server $Server -Identity ([guid]$ExpectedRootGuid) -Properties description -ErrorAction Stop
    $rootDn = Get-PortableDirectoryFixtureDn $RunId
    if ($root.DistinguishedName -cne $rootDn -or $root.Description -cne ('sdk-portable-parent:' + $RunId)) { throw 'Owned association permission root differs' }
    $actor = Get-ADUser -Server $Server -Identity ([guid]$ActorGuid) -Properties description, memberOf -ErrorAction Stop
    if ($actor.DistinguishedName -cne ('CN=client,OU=sdk-kerberos-' + $ActorRun + ',DC=ad,DC=sdk,DC=test') -or
        $actor.Description -cne ('sdk-kerberos-client:' + $ActorRun) -or @($actor.memberOf).Count -ne 0 -or $ActorCredential.UserName -cne $actor.UserPrincipalName) {
        throw 'Association permission actor is not the ordinary independently owned account'
    }
    $schema = Get-ADRootDSE -Server $Server -ErrorAction Stop
    $attributes = @(Get-ADObject -Server $Server -SearchBase $schema.schemaNamingContext -LDAPFilter '(&(objectClass=attributeSchema)(lDAPDisplayName=msDS-HostServiceAccount))' -Properties schemaIDGUID -ErrorAction Stop)
    if ($attributes.Count -ne 1 -or $attributes[0].schemaIDGUID.Length -ne 16) { throw 'Member schema identity is ambiguous' }
    $attributeGuid = [guid]::new([byte[]]$attributes[0].schemaIDGUID)
    $marker = 'tfad-owned:' + $RunId + ':' + $Platform
    $state = [ordered]@{ run = $RunId; server = $Server; platform = $Platform; phase = 'creating'; computer_guid = ''; service_account_guid = ''; baseline_sddl = ''; expected_sddl = ''; observed_sddl = '' }
    $journal = 'C:\adsdk-portable-tls\' + $RunId + '\association-permissions-' + $Platform + '.json'
    foreach ($kind in @('computer', 'service_account')) {
        $name = 'tfad-' + $RunId + '-hidden-' + $kind + '-' + $Platform
        $hasher = [Security.Cryptography.SHA256]::Create()
        try { $digest = $hasher.ComputeHash([Text.Encoding]::UTF8.GetBytes($name)) } finally { $hasher.Dispose() }
        $sam = 'tfh' + (($digest[0..5] | ForEach-Object { $_.ToString('x2') }) -join '')
        if ($kind -ceq 'computer') {
            $created = New-ADComputer -Server $Server -Name $name -SamAccountName $sam -Path $rootDn -Enabled $false -Description $marker -PassThru -ErrorAction Stop
        } else {
            $created = New-ADServiceAccount -Server $Server -Name $name -SamAccountName $sam -Path $rootDn -RestrictToSingleComputer -Enabled $false -Description $marker -PassThru -ErrorAction Stop
        }
        $state[$kind + '_guid'] = $created.ObjectGUID.ToString()
        $state[$kind + '_dn'] = $created.DistinguishedName
        [IO.File]::WriteAllText($journal, ($state | ConvertTo-Json -Depth 5), [Text.UTF8Encoding]::new($false))
    }
    # Add-ADComputerServiceAccount binds its account operand through -ServiceAccount.
    Add-ADComputerServiceAccount -Server $Server -Identity ([guid]$state.computer_guid) -ServiceAccount ([guid]$state.service_account_guid) -ErrorAction Stop
    $known = Get-ADComputer -Server $Server -Identity ([guid]$state.computer_guid) -Properties msDS-HostServiceAccount, description -ErrorAction Stop
    if ($known.Description -cne $marker -or @($known.'msDS-HostServiceAccount').Count -ne 1 -or $known.'msDS-HostServiceAccount'[0] -cne $state.service_account_dn) { throw 'Independent native direct link did not verify' }
    $entry = Open-KerberosWriteDirectoryEntry $Server ([guid]$state.computer_guid) $Credential
    $sections = [Security.AccessControl.AccessControlSections]::Owner -bor [Security.AccessControl.AccessControlSections]::Group -bor [Security.AccessControl.AccessControlSections]::Access
    $rights = [DirectoryServices.ActiveDirectoryRights]::ReadProperty
    try {
        $state.baseline_sddl = Get-KerberosWriteSDDL $entry
        $security = [DirectoryServices.ActiveDirectorySecurity]::new()
        $security.SetSecurityDescriptorSddlForm($state.baseline_sddl, $sections)
        $existing = @($security.GetAccessRules($true, $false, [Security.Principal.SecurityIdentifier]) | Where-Object { $_.IdentityReference.Value -ceq $actor.SID.Value })
        if ($existing.Count -ne 0) { throw 'Fresh association fixture already has an actor ACE' }
        $rule = [DirectoryServices.ActiveDirectoryAccessRule]::new($actor.SID, $rights, [Security.AccessControl.AccessControlType]::Deny, $attributeGuid, [DirectoryServices.ActiveDirectorySecurityInheritance]::None)
        $security.AddAccessRule($rule)
        $state.expected_sddl = $security.GetSecurityDescriptorSddlForm($sections)
        $state.phase = 'acl-intent'
        [IO.File]::WriteAllText($journal, ($state | ConvertTo-Json -Depth 5), [Text.UTF8Encoding]::new($false))
        $entry.psbase.RefreshCache(@('nTSecurityDescriptor'))
        if ((Get-KerberosWriteSDDL $entry) -cne $state.baseline_sddl) { throw 'Association fixture ACL changed before submission' }
        $entry.psbase.ObjectSecurity = $security
        $entry.psbase.CommitChanges()
    } finally { $entry.psbase.Dispose() }
    $entry = Open-KerberosWriteDirectoryEntry $Server ([guid]$state.computer_guid) $Credential
    try {
        $state.observed_sddl = Get-KerberosWriteSDDL $entry
        [IO.File]::WriteAllText($journal, ($state | ConvertTo-Json -Depth 5), [Text.UTF8Encoding]::new($false))
        if ((Get-ProviderComputerServiceAccountAclShape $state.observed_sddl) -cne (Get-ProviderComputerServiceAccountAclShape $state.expected_sddl)) { throw 'Association fixture ACL did not independently verify' }
        $rules = @($entry.psbase.ObjectSecurity.GetAccessRules($true, $false, [Security.Principal.SecurityIdentifier]) | Where-Object { $_.IdentityReference.Value -ceq $actor.SID.Value })
        if ($rules.Count -ne 1 -or $rules[0].ActiveDirectoryRights -ne $rights -or $rules[0].AccessControlType -ne [Security.AccessControl.AccessControlType]::Deny -or
            $rules[0].ObjectType -ne $attributeGuid -or $rules[0].IsInherited -or $rules[0].InheritanceType -ne [DirectoryServices.ActiveDirectorySecurityInheritance]::None) {
            throw 'Association fixture ACE exceeds its exact property boundary'
        }
    } finally { $entry.psbase.Dispose() }
    $reader = Get-ADComputer -Server $Server -Credential $ActorCredential -Identity ([guid]$state.computer_guid) -Properties msDS-HostServiceAccount -ErrorAction Stop
    if (@($reader.'msDS-HostServiceAccount').Count -ne 0) { throw 'Member attribute was not hidden from the ordinary actor' }
    $native = Get-ADComputer -Server $Server -Identity ([guid]$state.computer_guid) -Properties msDS-HostServiceAccount -ErrorAction Stop
    if (@($native.'msDS-HostServiceAccount').Count -ne 1 -or $native.'msDS-HostServiceAccount'[0] -cne $state.service_account_dn) { throw 'Hidden association changed during native qualification' }
    $guidFilter = (([guid]$state.computer_guid).ToByteArray() | ForEach-Object { '\' + $_.ToString('x2') }) -join ''
    $state.reader_association_count = @($reader.'msDS-HostServiceAccount').Count
    $state.admin_association_count = @($native.'msDS-HostServiceAccount').Count
    $state.native_positive_count = @(Get-ADComputer -Server $Server -Credential $ActorCredential -SearchBase $state.computer_dn -SearchScope Base -LDAPFilter ('(&(objectGUID=' + $guidFilter + ')(msDS-HostServiceAccount=' + $state.service_account_dn + '))') -ErrorAction Stop).Count
    $state.native_negative_count = @(Get-ADComputer -Server $Server -Credential $ActorCredential -SearchBase $state.computer_dn -SearchScope Base -LDAPFilter ('(&(objectGUID=' + $guidFilter + ')(!(msDS-HostServiceAccount=' + $state.service_account_dn + ')))') -ErrorAction Stop).Count
    $state.phase = 'verified'
    $state.actor_guid = $ActorGuid
    $state.actor_sid = $actor.SID.Value
    $state.attribute_guid = $attributeGuid.ToString()
    [IO.File]::WriteAllText($journal, ($state | ConvertTo-Json -Depth 5), [Text.UTF8Encoding]::new($false))
    [pscustomobject]$state
}
