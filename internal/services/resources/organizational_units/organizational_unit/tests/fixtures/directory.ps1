#Requires -Version 5.1
# Dot-source alongside the verified SDK TLS and directory fixture functions on the owned DC.
function Get-ProviderOuAclShape([string]$Sddl) {
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

function New-ProviderOuPermissionFixture([string]$RunId,[string]$Server,[string]$ExpectedRootGuid,[string]$Platform,[string]$ActorRun,[string]$ActorGuid,[PSCredential]$Credential) {
    $ErrorActionPreference='Stop'
    $null=Assert-PortableTlsFixtureIdentity $RunId $Server
    if($Platform-cnotin@('windows','linux')-or$ActorRun-cnotmatch'\A[a-z0-9][a-z0-9-]{5,39}\z'){throw 'Invalid permission fixture identity'}
    $root=Get-ADOrganizationalUnit -Server $Server -Identity ([guid]$ExpectedRootGuid) -Properties description -ErrorAction Stop
    $rootDn=Get-PortableDirectoryFixtureDn $RunId
    if($root.DistinguishedName-cne$rootDn-or$root.Description-cne('sdk-portable-parent:'+$RunId)){throw 'Owned permission root differs'}
    $actor=Get-ADUser -Server $Server -Identity ([guid]$ActorGuid) -Properties description,memberOf -ErrorAction Stop
    if($actor.DistinguishedName-cne('CN=client,OU=sdk-kerberos-'+$ActorRun+',DC=ad,DC=sdk,DC=test')-or
        $actor.Description-cne('sdk-kerberos-client:'+$ActorRun)-or@($actor.memberOf).Count-ne0){throw 'Permission actor must remain the ordinary marker-owned account'}
    $schema=Get-ADRootDSE -Server $Server -ErrorAction Stop
    $attributes=@(Get-ADObject -Server $Server -SearchBase $schema.schemaNamingContext -LDAPFilter '(&(objectClass=attributeSchema)(lDAPDisplayName=description))' -Properties schemaIDGUID -ErrorAction Stop)
    if($attributes.Count-ne1-or$attributes[0].schemaIDGUID.Length-ne16){throw 'Description schema identity is ambiguous'}
    $attributeGuid=[guid]::new([byte[]]$attributes[0].schemaIDGUID)
    $name='tfad-'+$RunId+'-hidden-'+$Platform
    $marker='tfad-owned:'+$RunId+':'+$Platform
    $unit=New-ADOrganizationalUnit -Server $Server -Name $name -Path $rootDn -Description ('tfad-hidden:'+$RunId) -ProtectedFromAccidentalDeletion $false -OtherAttributes @{displayName=$marker} -PassThru -ErrorAction Stop
    $entry=Open-KerberosWriteDirectoryEntry $Server $unit.ObjectGUID $Credential
    $sections=[Security.AccessControl.AccessControlSections]::Owner-bor[Security.AccessControl.AccessControlSections]::Group-bor[Security.AccessControl.AccessControlSections]::Access
    $journal='C:\adsdk-portable-tls\'+$RunId+'\permissions-'+$Platform+'.json'
    $state=[ordered]@{run=$RunId;server=$Server;guid=$unit.ObjectGUID.ToString();name=$name;dn=$unit.DistinguishedName;actor_guid=$ActorGuid;actor_sid=$actor.SID.Value;attribute_guid=$attributeGuid.ToString();phase='created';baseline_sddl=$null;expected_sddl=$null;observed_sddl=$null}
    try{
        $state.baseline_sddl=Get-KerberosWriteSDDL $entry
        $security=[DirectoryServices.ActiveDirectorySecurity]::new()
        $security.SetSecurityDescriptorSddlForm($state.baseline_sddl,$sections)
        $existing=@($security.GetAccessRules($true,$false,[Security.Principal.SecurityIdentifier])|Where-Object {$_.IdentityReference.Value-ceq$actor.SID.Value})
        if($existing.Count-ne0){throw 'Fresh permission fixture already has an actor ACE'}
        $rights=[DirectoryServices.ActiveDirectoryRights]::ReadProperty-bor[DirectoryServices.ActiveDirectoryRights]::WriteProperty
        $rule=[DirectoryServices.ActiveDirectoryAccessRule]::new($actor.SID,$rights,[Security.AccessControl.AccessControlType]::Deny,$attributeGuid,[DirectoryServices.ActiveDirectorySecurityInheritance]::None)
        $security.AddAccessRule($rule)
        $state.expected_sddl=$security.GetSecurityDescriptorSddlForm($sections);$state.phase='acl-intent'
        [IO.File]::WriteAllText($journal,($state|ConvertTo-Json -Depth 5),[Text.UTF8Encoding]::new($false))
        $entry.psbase.RefreshCache(@('nTSecurityDescriptor'))
        if((Get-KerberosWriteSDDL $entry)-cne$state.baseline_sddl){throw 'Permission fixture ACL changed before submission'}
        $entry.psbase.ObjectSecurity=$security
        $entry.psbase.CommitChanges()
    }finally{$entry.psbase.Dispose()}
    $entry=Open-KerberosWriteDirectoryEntry $Server $unit.ObjectGUID $Credential
    try{
        $state.observed_sddl=Get-KerberosWriteSDDL $entry
        [IO.File]::WriteAllText($journal,($state|ConvertTo-Json -Depth 5),[Text.UTF8Encoding]::new($false))
        if((Get-ProviderOuAclShape $state.observed_sddl)-cne(Get-ProviderOuAclShape $state.expected_sddl)){throw 'Permission fixture ACL did not independently verify'}
        $rules=@($entry.psbase.ObjectSecurity.GetAccessRules($true,$false,[Security.Principal.SecurityIdentifier])|Where-Object {$_.IdentityReference.Value-ceq$actor.SID.Value})
        if($rules.Count-ne1-or$rules[0].ActiveDirectoryRights-ne$rights-or$rules[0].AccessControlType-ne[Security.AccessControl.AccessControlType]::Deny-or
            $rules[0].ObjectType-ne$attributeGuid-or$rules[0].IsInherited-or$rules[0].InheritanceType-ne[DirectoryServices.ActiveDirectorySecurityInheritance]::None){throw 'Permission ACE exceeds its exact property boundary'}
    }finally{$entry.psbase.Dispose()}
    $state.phase='verified'
    [IO.File]::WriteAllText($journal,($state|ConvertTo-Json -Depth 5),[Text.UTF8Encoding]::new($false))
    [pscustomobject]$state
}

function Remove-ProviderOuChildren([string]$RunId,[string]$Server,[string]$ExpectedGuid) {
    $ErrorActionPreference='Stop'
    $null=Assert-PortableTlsFixtureIdentity $RunId $Server
    $dn=Get-PortableDirectoryFixtureDn $RunId
    $root=Get-ADObject -Server $Server -Identity ([guid]$ExpectedGuid) -Properties description -ErrorAction Stop
    if($root.DistinguishedName-cne$dn-or$root.ObjectClass-cne'organizationalUnit'-or$root.description-cne("sdk-portable-parent:"+$RunId)){throw 'Owned provider fixture root differs'}
    $children=@(Get-ADObject -Server $Server -SearchBase $dn -SearchScope Subtree -LDAPFilter '(objectClass=*)' -Properties displayName -ErrorAction Stop|Where-Object ObjectGUID -NE $root.ObjectGUID)
    if($children.Count-gt20){throw 'Unexpected fixture object count; preserve the directory for reconciliation'}
    $expectedMarkers=@(('tfad-owned:'+$RunId+':windows'),('tfad-owned:'+$RunId+':linux'))
    foreach($child in $children){
        if($child.ObjectClass-cnotin@('organizationalUnit','contact')-or[string]$child.displayName-cnotin$expectedMarkers-or
            -not([string]$child.Name).StartsWith('tfad-',[StringComparison]::Ordinal)-or
            -not([string]$child.DistinguishedName).EndsWith(','+$dn,[StringComparison]::OrdinalIgnoreCase)){
            throw 'Unowned or moved descendant prevents provider fixture cleanup'
        }
    }
    $removed=@()
    foreach($child in @($children|Sort-Object {$_.DistinguishedName.Length} -Descending)){
        $actual=Get-ADObject -Server $Server -Identity $child.ObjectGUID -Properties displayName -ErrorAction Stop
        if($actual.DistinguishedName-cne$child.DistinguishedName-or$actual.ObjectClass-cne$child.ObjectClass-or$actual.displayName-cne$child.displayName){throw 'Fixture identity changed before cleanup'}
        $descendants=@(Get-ADObject -Server $Server -SearchBase $actual.DistinguishedName -SearchScope OneLevel -LDAPFilter '(objectClass=*)' -ErrorAction Stop)
        if($descendants.Count-ne0){throw 'Leaf cleanup refuses a nonempty fixture object'}
        if($actual.ObjectClass-ceq'organizationalUnit'){
            $unit=Get-ADOrganizationalUnit -Server $Server -Identity $actual.ObjectGUID -Properties ProtectedFromAccidentalDeletion -ErrorAction Stop
            if($unit.ProtectedFromAccidentalDeletion){Set-ADOrganizationalUnit -Server $Server -Identity $actual.ObjectGUID -ProtectedFromAccidentalDeletion $false -ErrorAction Stop}
            Remove-ADOrganizationalUnit -Server $Server -Identity $actual.ObjectGUID -Confirm:$false -ErrorAction Stop
        }else{Remove-ADObject -Server $Server -Identity $actual.ObjectGUID -Confirm:$false -ErrorAction Stop}
        $hex=($actual.ObjectGUID.ToByteArray()|ForEach-Object {'\'+$_.ToString('x2')})-join''
        if(@(Get-ADObject -Server $Server -LDAPFilter ('(objectGUID='+$hex+')') -ErrorAction Stop).Count-ne0){throw 'Owned fixture child remains live'}
        $removed+=@{guid=$actual.ObjectGUID.ToString();dn=$actual.DistinguishedName}
    }
    [pscustomobject]@{verified=$true;reconciled_children=$removed;remaining=0}
}
