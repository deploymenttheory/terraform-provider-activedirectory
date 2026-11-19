#Requires -Version 5.1
[CmdletBinding()]
param(
    [Parameter(Mandatory)][switch]$AuthorizeLab,
    [Parameter(Mandatory)][string]$Target,
    [Parameter(Mandatory)][ValidateSet('2016','2019','2022','2025')][string]$ServerVersion,
    [Parameter(Mandatory)][ValidatePattern('\A[a-z][a-z0-9-]{5,39}\z')][string]$RunId,
    [Parameter(Mandatory)][string]$Partition,
    [string]$Package='./internal/provider/core',
    [string]$Test='^TestAccProviderLabConnection$',
    [string]$Go='go',
    [string]$TestBinary,
    [string]$ExpectedBinarySHA256
)
$ErrorActionPreference='Stop'
if(-not$AuthorizeLab-or$env:TF_ACC-cne'1'){throw 'Live acceptance requires -AuthorizeLab and explicit TF_ACC=1'}
if(-not$Target-or$env:AD_HOST-ine$Target-or$Target-match'[/\\:@\s]'){throw 'AD_HOST must match the explicitly selected lab target'}
if($env:GITHUB_EVENT_NAME-in@('pull_request','pull_request_target')){throw 'Live acceptance is forbidden from pull-request jobs'}
if($TestBinary){
    $TestBinary=(Resolve-Path -LiteralPath $TestBinary).ProviderPath
    if($ExpectedBinarySHA256-cnotmatch'\A[a-f0-9]{64}\z'-or(Get-FileHash -LiteralPath $TestBinary -Algorithm SHA256).Hash.ToLowerInvariant()-cne$ExpectedBinarySHA256){throw 'Acceptance binary checksum differs'}
}
$values=@{
    TF_ACC_AD_LAB='1';AD_ACCEPTANCE_AUTHORIZED='1';AD_ACCEPTANCE_HOSTS=$Target
    AD_ACCEPTANCE_RUN_ID=$RunId;AD_ACCEPTANCE_PARTITION=$Partition;AD_ACCEPTANCE_SERVER_VERSION=$ServerVersion
}
$saved=@{}
foreach($name in $values.Keys){$saved[$name]=[Environment]::GetEnvironmentVariable($name,'Process');[Environment]::SetEnvironmentVariable($name,$values[$name],'Process')}
Push-Location (Join-Path $PSScriptRoot '..')
try{
    if($TestBinary){
        $testArguments=@(('-test.run='+$Test),'-test.count=1','-test.timeout=15m')
        & $TestBinary @testArguments
    }
    else{& $Go test -mod=readonly $Package -run $Test -count=1 -timeout=15m}
    if($LASTEXITCODE-ne0){throw 'Provider acceptance failed; preserve failure and cleanup records before another write'}
}finally{
    Pop-Location
    foreach($name in $saved.Keys){[Environment]::SetEnvironmentVariable($name,$saved[$name],'Process')}
}
