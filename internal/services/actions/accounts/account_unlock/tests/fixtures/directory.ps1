#Requires -Version 5.1
# Dot-source on the owned DC alongside the SDK TLS/directory and kerberos-write fixture functions.

function Get-ProviderAccountAclShape([string]$Sddl) {
    $security=[DirectoryServices.ActiveDirectorySecurity]::new()
    $security.SetSecurityDescriptorSddlForm($Sddl)
    if(-not$security.AreAccessRulesCanonical){throw 'Account fixture ACL is not canonical'}
    $raw=[Security.AccessControl.RawSecurityDescriptor]::new($Sddl)
    if(-not$raw.DiscretionaryAcl){throw 'Account fixture ACL is absent'}
    $aces=@(foreach($ace in $raw.DiscretionaryAcl){
        $bytes=[byte[]]::new($ace.BinaryLength)
        $ace.GetBinaryForm($bytes,0)
        [Convert]::ToBase64String($bytes)
    })
    [Array]::Sort($aces,[StringComparer]::Ordinal)
    return ($raw.Owner.Value+'|'+$raw.Group.Value+'|'+[int]$raw.ControlFlags+'|'+($aces-join'|'))
}

function New-ProviderAccountUnlockFixture([string]$RunId,[string]$Server,[string]$RootGuid,[string]$Platform,[string]$ActorRun,[string]$ActorGuid,[PSCredential]$Credential) {
    $ErrorActionPreference='Stop'
    $null=Assert-PortableTlsFixtureIdentity $RunId $Server
    if($Platform-cnotin@('windows','linux')-or$ActorRun-cnotmatch'\A[a-z0-9][a-z0-9-]{5,39}\z'){throw 'Account fixture identity is invalid'}
    $rootDn=Get-PortableDirectoryFixtureDn $RunId
    $root=Get-ADOrganizationalUnit -Server $Server -Identity ([guid]$RootGuid) -Properties description -ErrorAction Stop
    if($root.DistinguishedName-cne$rootDn-or$root.Description-cne('sdk-portable-parent:'+$RunId)){throw 'Account fixture root differs'}
    $actor=Get-ADUser -Server $Server -Identity ([guid]$ActorGuid) -Properties description,memberOf -ErrorAction Stop
    if($actor.DistinguishedName-cne('CN=client,OU=sdk-kerberos-'+$ActorRun+',DC=ad,DC=sdk,DC=test')-or
        $actor.Description-cne('sdk-kerberos-client:'+$ActorRun)-or@($actor.memberOf).Count-ne0){throw 'Unlock actor must be the ordinary marker-owned account'}
    $name='tfad-'+$RunId+'-unlock-'+$Platform
    $sha=[Security.Cryptography.SHA256]::Create()
    try{$digest=$sha.ComputeHash([Text.Encoding]::UTF8.GetBytes($name))}finally{$sha.Dispose()}
    $sam='tfau'+([BitConverter]::ToString($digest).Replace('-','').Substring(0,12).ToLowerInvariant())
    $marker='tfad-unlock:'+$RunId
    $subject=New-ADUser -Server $Server -Name $name -SamAccountName $sam -Path $rootDn -Enabled $false -Description $marker -OtherAttributes @{displayName=('tfad-owned:'+$RunId+':'+$Platform)} -PassThru -ErrorAction Stop
    $journal='C:\adsdk-portable-tls\'+$RunId+'\account-'+$Platform+'.json'
    $state=[ordered]@{run=$RunId;server=$Server;guid=$subject.ObjectGUID.ToString();dn=$subject.DistinguishedName;sam=$sam;phase='created';initial_lockout=$null;initial_uac=$null;baseline_sddl=$null;expected_sddl=$null;observed_sddl=$null}
    [IO.File]::WriteAllText($journal,($state|ConvertTo-Json -Depth 5),[Text.UTF8Encoding]::new($false))
    $lockout=[DateTime]::UtcNow.ToFileTimeUtc()
    Set-ADUser -Server $Server -Identity $subject.ObjectGUID -Replace @{lockoutTime=[long]$lockout} -ErrorAction Stop
    $initial=Get-ADUser -Server $Server -Identity $subject.ObjectGUID -Properties description,lockoutTime,userAccountControl -ErrorAction Stop
    if($initial.Enabled-or$initial.Description-cne$marker-or[long]$initial.lockoutTime-ne$lockout){throw 'Disabled account lockout fixture did not independently verify'}
    $state.initial_lockout=[long]$initial.lockoutTime;$state.initial_uac=[long]$initial.userAccountControl;$state.phase='lockout-observed'
    [IO.File]::WriteAllText($journal,($state|ConvertTo-Json -Depth 5),[Text.UTF8Encoding]::new($false))
    $schema=Get-ADRootDSE -Server $Server -ErrorAction Stop
    $attributes=@(Get-ADObject -Server $Server -SearchBase $schema.schemaNamingContext -LDAPFilter '(&(objectClass=attributeSchema)(lDAPDisplayName=lockoutTime))' -Properties schemaIDGUID -ErrorAction Stop)
    if($attributes.Count-ne1-or$attributes[0].schemaIDGUID.Length-ne16){throw 'Lockout attribute schema identity is ambiguous'}
    $attributeGuid=[guid]::new([byte[]]$attributes[0].schemaIDGUID)
    $sections=[Security.AccessControl.AccessControlSections]::Owner-bor[Security.AccessControl.AccessControlSections]::Group-bor[Security.AccessControl.AccessControlSections]::Access
    $entry=Open-KerberosWriteDirectoryEntry $Server $subject.ObjectGUID $Credential
    try{
        $state.baseline_sddl=Get-KerberosWriteSDDL $entry
        $security=[DirectoryServices.ActiveDirectorySecurity]::new()
        $security.SetSecurityDescriptorSddlForm($state.baseline_sddl,$sections)
        if(@($security.GetAccessRules($true,$false,[Security.Principal.SecurityIdentifier])|Where-Object {$_.IdentityReference.Value-ceq$actor.SID.Value}).Count-ne0){throw 'Fresh account has an unexpected actor ACE'}
        $security.AddAccessRule([DirectoryServices.ActiveDirectoryAccessRule]::new($actor.SID,[DirectoryServices.ActiveDirectoryRights]::WriteProperty,[Security.AccessControl.AccessControlType]::Deny,$attributeGuid,[DirectoryServices.ActiveDirectorySecurityInheritance]::None))
        $state.expected_sddl=$security.GetSecurityDescriptorSddlForm($sections);$state.phase='acl-intent'
        [IO.File]::WriteAllText($journal,($state|ConvertTo-Json -Depth 5),[Text.UTF8Encoding]::new($false))
        $entry.psbase.RefreshCache(@('nTSecurityDescriptor'))
        if((Get-KerberosWriteSDDL $entry)-cne$state.baseline_sddl){throw 'Account fixture ACL changed before submission'}
        $entry.psbase.ObjectSecurity=$security
        $entry.psbase.CommitChanges()
    }finally{$entry.psbase.Dispose()}
    $entry=Open-KerberosWriteDirectoryEntry $Server $subject.ObjectGUID $Credential
    try{
        $state.observed_sddl=Get-KerberosWriteSDDL $entry
        [IO.File]::WriteAllText($journal,($state|ConvertTo-Json -Depth 5),[Text.UTF8Encoding]::new($false))
        if((Get-ProviderAccountAclShape $state.observed_sddl)-cne(Get-ProviderAccountAclShape $state.expected_sddl)){throw 'Account fixture ACL did not independently verify'}
        $rules=@($entry.psbase.ObjectSecurity.GetAccessRules($true,$false,[Security.Principal.SecurityIdentifier])|Where-Object {$_.IdentityReference.Value-ceq$actor.SID.Value})
        if($rules.Count-ne1-or$rules[0].ActiveDirectoryRights-ne[DirectoryServices.ActiveDirectoryRights]::WriteProperty-or
            $rules[0].AccessControlType-ne[Security.AccessControl.AccessControlType]::Deny-or$rules[0].ObjectType-ne$attributeGuid-or
            $rules[0].IsInherited-or$rules[0].InheritanceType-ne[DirectoryServices.ActiveDirectorySecurityInheritance]::None){throw 'Account fixture deny exceeds the exact lockout property'}
    }finally{$entry.psbase.Dispose()}
    $state.phase='verified'
    [IO.File]::WriteAllText($journal,($state|ConvertTo-Json -Depth 5),[Text.UTF8Encoding]::new($false))
    [pscustomobject]$state
}

function Remove-ProviderAccountChildren([string]$RunId,[string]$Server,[string]$RootGuid) {
    $ErrorActionPreference='Stop'
    $null=Assert-PortableTlsFixtureIdentity $RunId $Server
    $rootDn=Get-PortableDirectoryFixtureDn $RunId
    $root=Get-ADOrganizationalUnit -Server $Server -Identity ([guid]$RootGuid) -Properties description -ErrorAction Stop
    if($root.DistinguishedName-cne$rootDn-or$root.Description-cne('sdk-portable-parent:'+$RunId)){throw 'Account cleanup root differs'}
    $objects=@(Get-ADObject -Server $Server -SearchBase $rootDn -SearchScope Subtree -LDAPFilter '(objectClass=*)' -Properties description,displayName -ErrorAction Stop|Where-Object ObjectGUID -NE $root.ObjectGUID)
    if($objects.Count-gt2){throw 'Unexpected account fixture descendants'}
    $known=@()
    foreach($platform in @('windows','linux')){
        $journal='C:\adsdk-portable-tls\'+$RunId+'\account-'+$platform+'.json'
        if(Test-Path -LiteralPath $journal){
            $state=Get-Content -LiteralPath $journal -Raw|ConvertFrom-Json
            if($state.run-cne$RunId-or$state.server-cne$Server-or$state.dn-cne('CN=tfad-'+$RunId+'-unlock-'+$platform+','+$rootDn)-or([guid]$state.guid)-eq[guid]::Empty){throw 'Account fixture journal identity differs'}
            $known+=[guid]$state.guid
            $live=$null
            try{$live=Get-ADObject -Server $Server -Identity ([guid]$state.guid) -ErrorAction Stop}catch [Microsoft.ActiveDirectory.Management.ADIdentityNotFoundException]{}
            if($live-and$live.DistinguishedName-cne$state.dn){throw 'Known account moved outside its owned location; preserve it for reconciliation'}
        }
    }
    foreach($object in $objects){
        $platform=if($object.Name-ceq('tfad-'+$RunId+'-unlock-windows')){'windows'}elseif($object.Name-ceq('tfad-'+$RunId+'-unlock-linux')){'linux'}else{throw 'Unowned account name prevents cleanup'}
        if($object.ObjectClass-cne'user'-or$object.DistinguishedName-cne('CN='+$object.Name+','+$rootDn)-or$object.description-cne('tfad-unlock:'+$RunId)-or
            $object.displayName-cne('tfad-owned:'+$RunId+':'+$platform)){throw 'Unowned or moved account prevents cleanup'}
    }
    $removed=@()
    foreach($object in $objects){
        $actual=Get-ADUser -Server $Server -Identity $object.ObjectGUID -Properties description,displayName -ErrorAction Stop
        if($actual.DistinguishedName-cne$object.DistinguishedName-or$actual.Description-cne$object.description-or$actual.DisplayName-cne$object.displayName){throw 'Account identity changed before cleanup'}
        if(@(Get-ADObject -Server $Server -SearchBase $actual.DistinguishedName -SearchScope OneLevel -LDAPFilter '(objectClass=*)' -ErrorAction Stop).Count-ne0){throw 'Account cleanup refuses descendants'}
        Remove-ADUser -Server $Server -Identity $actual.ObjectGUID -Confirm:$false -ErrorAction Stop
        $hex=($actual.ObjectGUID.ToByteArray()|ForEach-Object {'\'+$_.ToString('x2')})-join''
        if(@(Get-ADObject -Server $Server -LDAPFilter ('(objectGUID='+$hex+')') -ErrorAction Stop).Count-ne0){throw 'Owned account remains live'}
        $removed+=@{guid=$actual.ObjectGUID.ToString();dn=$actual.DistinguishedName}
    }
    foreach($guid in $known){
        $hex=($guid.ToByteArray()|ForEach-Object {'\'+$_.ToString('x2')})-join''
        if(@(Get-ADObject -Server $Server -LDAPFilter ('(objectGUID='+$hex+')') -ErrorAction Stop).Count-ne0){throw 'A known account GUID remains live after cleanup'}
    }
    [pscustomobject]@{verified=$true;reconciled_children=$removed;remaining=0}
}
