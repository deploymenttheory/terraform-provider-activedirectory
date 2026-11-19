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
    $upn=$sam+'@ad.sdk.test'
    $random=[byte[]]::new(48);$rng=[Security.Cryptography.RandomNumberGenerator]::Create()
    try{$rng.GetBytes($random)}finally{$rng.Dispose()}
    $password=ConvertTo-SecureString -String ('Tfau!'+[Convert]::ToBase64String($random)+'a9') -AsPlainText -Force
    [Array]::Clear($random,0,$random.Length)
    try{
        $subject=New-ADUser -Server $Server -Name $name -SamAccountName $sam -UserPrincipalName $upn -Path $rootDn -AccountPassword $password -Enabled $true -Description $marker -OtherAttributes @{displayName=('tfad-owned:'+$RunId+':'+$Platform)} -PassThru -ErrorAction Stop
    }finally{$password=$null}
    $journal='C:\adsdk-portable-tls\'+$RunId+'\account-'+$Platform+'.json'
    $policyDn='CN='+$name+',CN=Password Settings Container,CN=System,DC=ad,DC=sdk,DC=test'
    $state=[ordered]@{run=$RunId;server=$Server;guid=$subject.ObjectGUID.ToString();dn=$subject.DistinguishedName;sam=$sam;phase='created';policy_dn=$policyDn;policy_guid=$null;initial_lockout=$null;initial_uac=$null;baseline_sddl=$null;expected_sddl=$null;observed_sddl=$null}
    [IO.File]::WriteAllText($journal,($state|ConvertTo-Json -Depth 5),[Text.UTF8Encoding]::new($false))
    $existingPolicy=$null
    try{$existingPolicy=Get-ADObject -Server $Server -Identity $policyDn -ErrorAction Stop}catch [Microsoft.ActiveDirectory.Management.ADIdentityNotFoundException]{}
    if($existingPolicy){throw 'Account lockout policy must be fresh'}
    $state.phase='policy-intent'
    [IO.File]::WriteAllText($journal,($state|ConvertTo-Json -Depth 5),[Text.UTF8Encoding]::new($false))
    $policyOptions=@{
        Server=$Server;Name=$name;Description=('tfad-unlock-policy:'+$RunId+':'+$Platform);Precedence=1000
        ComplexityEnabled=$true;ReversibleEncryptionEnabled=$false;MinPasswordLength=16;PasswordHistoryCount=1
        MinPasswordAge=[TimeSpan]::Zero;MaxPasswordAge=[TimeSpan]::FromDays(30)
        LockoutDuration=[TimeSpan]::FromMinutes(30);LockoutObservationWindow=[TimeSpan]::FromMinutes(30);LockoutThreshold=3
        PassThru=$true;ErrorAction='Stop'
    }
    $policy=New-ADFineGrainedPasswordPolicy @policyOptions
    if($policy.DistinguishedName-cne$policyDn){throw 'Owned account policy location differs'}
    $state.policy_guid=$policy.ObjectGUID.ToString();$state.phase='policy-created'
    [IO.File]::WriteAllText($journal,($state|ConvertTo-Json -Depth 5),[Text.UTF8Encoding]::new($false))
    Add-ADFineGrainedPasswordPolicySubject -Server $Server -Identity $policy.ObjectGUID -Subjects $subject.ObjectGUID -ErrorAction Stop
    $linked=Get-ADObject -Server $Server -Identity $policy.ObjectGUID -Properties 'msDS-PSOAppliesTo' -ErrorAction Stop
    $subjects=@($linked.'msDS-PSOAppliesTo')
    if($subjects.Count-ne1-or[string]$subjects[0]-cne$subject.DistinguishedName){throw 'Owned password policy scope exceeds its one fixture account'}
    $effective=Get-ADUserResultantPasswordPolicy -Server $Server -Identity $subject.ObjectGUID -ErrorAction Stop
    if($effective.ObjectGUID-ne$policy.ObjectGUID-or$effective.LockoutThreshold-ne3){throw 'Owned account effective policy did not verify'}
    # Only this new random-password subject receives at most three invalid attempts over verified LDAPS.
    # No existing account, default policy or group membership is changed.
    Add-Type -AssemblyName System.DirectoryServices.Protocols
    for($attempt=0;$attempt-lt3;$attempt++){
        $observed=Get-ADUser -Server $Server -Identity $subject.ObjectGUID -Properties LockedOut -ErrorAction Stop
        if($observed.LockedOut){break}
        $identifier=[DirectoryServices.Protocols.LdapDirectoryIdentifier]::new($Server,636,$true,$false)
        $connection=[DirectoryServices.Protocols.LdapConnection]::new($identifier)
        try{
            $connection.AuthType=[DirectoryServices.Protocols.AuthType]::Basic
            $connection.SessionOptions.ProtocolVersion=3;$connection.SessionOptions.SecureSocketLayer=$true
            $connection.Timeout=[TimeSpan]::FromSeconds(10)
            $connection.Credential=[Net.NetworkCredential]::new($upn,'intentionally-incorrect-owned-fixture-password')
            $accepted=$false
            try{$connection.Bind();$accepted=$true}catch{
                $failure=$_.Exception
                while($failure-and$failure-isnot[DirectoryServices.Protocols.LdapException]){$failure=$failure.InnerException}
                if(-not$failure-or$failure.ErrorCode-ne49){throw 'Owned invalid-password probe failed before a confirmed LDAP credential rejection'}
            }
            if($accepted){throw 'Unexpected authentication success for an intentionally incorrect owned-fixture password'}
        }finally{$connection.Dispose()}
    }
    $locked=Get-ADUser -Server $Server -Identity $subject.ObjectGUID -Properties LockedOut,lockoutTime -ErrorAction Stop
    if(-not$locked.LockedOut-or[long]$locked.lockoutTime-eq0){throw 'Owned account did not reach independently observed lockout'}
    Disable-ADAccount -Server $Server -Identity $subject.ObjectGUID -ErrorAction Stop
    $initial=Get-ADUser -Server $Server -Identity $subject.ObjectGUID -Properties description,lockoutTime,userAccountControl -ErrorAction Stop
    if($initial.Enabled-or$initial.Description-cne$marker-or[long]$initial.lockoutTime-ne[long]$locked.lockoutTime){throw 'Disabled account lockout fixture did not independently verify'}
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
    $retiredPolicies=@()
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
    foreach($platform in @('windows','linux')){
        $journal='C:\adsdk-portable-tls\'+$RunId+'\account-'+$platform+'.json'
        if(-not(Test-Path -LiteralPath $journal)){continue}
        $state=Get-Content -LiteralPath $journal -Raw|ConvertFrom-Json
        $expectedPolicyDn='CN=tfad-'+$RunId+'-unlock-'+$platform+',CN=Password Settings Container,CN=System,DC=ad,DC=sdk,DC=test'
        if($state.policy_dn-cne$expectedPolicyDn){throw 'Owned account policy journal location differs'}
        $policy=$null
        $policyIdentity=if($state.policy_guid){[guid]$state.policy_guid}else{$expectedPolicyDn}
        try{$policy=Get-ADObject -Server $Server -Identity $policyIdentity -Properties description,'msDS-PSOAppliesTo' -ErrorAction Stop}catch [Microsoft.ActiveDirectory.Management.ADIdentityNotFoundException]{}
        if($policy){
            if($policy.DistinguishedName-cne$expectedPolicyDn-or$policy.ObjectClass-cne'msDS-PasswordSettings'-or$policy.description-cne('tfad-unlock-policy:'+$RunId+':'+$platform)-or
                ($state.policy_guid-and$policy.ObjectGUID-ne[guid]$state.policy_guid)-or@($policy.'msDS-PSOAppliesTo').Count-ne0){throw 'Unowned or still-linked account policy prevents cleanup'}
            Remove-ADFineGrainedPasswordPolicy -Server $Server -Identity $policy.ObjectGUID -Confirm:$false -ErrorAction Stop
            $hex=($policy.ObjectGUID.ToByteArray()|ForEach-Object {'\'+$_.ToString('x2')})-join''
            if(@(Get-ADObject -Server $Server -LDAPFilter ('(objectGUID='+$hex+')') -ErrorAction Stop).Count-ne0){throw 'Owned account policy remains live'}
            $retiredPolicies+=@{guid=$policy.ObjectGUID.ToString();dn=$expectedPolicyDn;verified_absent=$true}
        }
    }
    [pscustomobject]@{verified=$true;reconciled_children=$removed;known_account_guids=@($known|ForEach-Object {$_.ToString()});retired_policies=$retiredPolicies;remaining=0}
}
