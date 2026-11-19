#Requires -Version 7.0
[CmdletBinding()]
param([string]$Go='go')
$ErrorActionPreference='Stop'
$saved=@{}
foreach($entry in Get-ChildItem Env:){
    if($entry.Name.StartsWith('AD_')-or$entry.Name-in@('TF_ACC','TF_ACC_AD_LAB')){
        $saved[$entry.Name]=$entry.Value
        [Environment]::SetEnvironmentVariable($entry.Name,$null,'Process')
    }
}
$previousVersion=$env:TF_ACC_TERRAFORM_VERSION
if(-not$env:TF_ACC_TERRAFORM_PATH){$env:TF_ACC_TERRAFORM_VERSION='1.17.2'}
Push-Location (Join-Path $PSScriptRoot '..')
try{
    & $Go test -mod=readonly -count=1 ./...
    if($LASTEXITCODE-ne0){throw 'Provider unit tests failed'}
}finally{
    Pop-Location
    foreach($name in $saved.Keys){[Environment]::SetEnvironmentVariable($name,$saved[$name],'Process')}
    [Environment]::SetEnvironmentVariable('TF_ACC_TERRAFORM_VERSION',$previousVersion,'Process')
}
