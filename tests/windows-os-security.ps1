# Offline OS-policy probes; never unblock files or disable security controls.
$ErrorActionPreference = 'Stop'
$repository = Split-Path $PSScriptRoot -Parent
$stage = Join-Path $env:TEMP ('Siaas OS Security ' + [Guid]::NewGuid().ToString('N'))
$savedPolicyOverride = $env:PSExecutionPolicyPreference
try {
    Remove-Item Env:PSExecutionPolicyPreference -ErrorAction SilentlyContinue
    go run ./tools/build -out $stage
    if ($LASTEXITCODE -ne 0) { throw 'Cannot build policy fixture.' }
    $binary = Get-ChildItem -LiteralPath $stage -Filter '*.exe' | Select-Object -First 1
    $before = (Get-FileHash -LiteralPath $binary.FullName).Hash
    $tokens = $null; $parseErrors = $null
    $ast = [Management.Automation.Language.Parser]::ParseFile((Join-Path $repository 'install.ps1'),[ref]$tokens,[ref]$parseErrors)
    if ($parseErrors.Count) { throw 'Installer parse error.' }
    foreach ($name in @('Fail','Set-InternetZone')) {
        $function = $ast.Find({ param($n) $n -is [Management.Automation.Language.FunctionDefinitionAst] -and $n.Name -eq $name },$false)
        . ([ScriptBlock]::Create($function.Extent.Text))
    }
    Set-InternetZone $binary.FullName
    if ((Get-Content -LiteralPath $binary.FullName -Stream Zone.Identifier -Raw) -notmatch 'ZoneId=3') { throw 'Internet-origin marker missing.' }
    if ((Get-FileHash -LiteralPath $binary.FullName).Hash -ne $before) { throw 'Origin metadata changed executable bytes.' }
    $signature = Get-AuthenticodeSignature -LiteralPath $binary.FullName
    Write-Output "OS_SECURITY Authenticode=$($signature.Status); ZoneId=3; SHA256 unchanged"
    if ($signature.Status -ne 'NotSigned') { throw 'Unexpected signing status; review distribution assumptions.' }
    $markedScript = Join-Path $stage 'install.ps1'
    Copy-Item -LiteralPath (Join-Path $repository 'install.ps1') -Destination $markedScript
    Set-InternetZone $markedScript
    Get-ExecutionPolicy -List | Format-Table
    $policy = Get-ExecutionPolicy
    # RemoteSigned is only used when it strengthens, or equals, current policy.
    $options = @('-NoProfile','-NonInteractive')
    if ($policy -in @('Unrestricted','Bypass','RemoteSigned')) { $options += @('-ExecutionPolicy','RemoteSigned') }
    $options += @('-File',$markedScript,'-Help')
    $savedErrorPreference = $ErrorActionPreference
    $ErrorActionPreference = 'Continue'
    $result = & powershell.exe @options 2>&1
    $code = $LASTEXITCODE
    $ErrorActionPreference = $savedErrorPreference
    Write-Output "OS_SECURITY Internet-marked unsigned script: policy=$policy; probe exit=$code"
    if ($code -eq 0) { throw 'Internet-marked unsigned script unexpectedly executed; policy acceptance is unverified.' }
    if (($result | Out-String) -notmatch 'PSSecurityException|UnauthorizedAccess|not digitally signed|running scripts is disabled') { throw 'The negative probe failed for an unexpected reason, not a recognized script-policy refusal.' }
    Write-Output 'OS_SECURITY unsigned Internet script blocked as expected; no unblock or policy downgrade performed.'
    Write-Output 'OS_SECURITY SmartScreen interactive/browser download verdict=UNVERIFIED; marked executable was not launched.'
} finally {
    if ($savedPolicyOverride) { $env:PSExecutionPolicyPreference = $savedPolicyOverride }
    if (Test-Path -LiteralPath $stage) { Remove-Item -LiteralPath $stage -Recurse -Force }
}
# The negative policy probe intentionally returned nonzero; do not leak its
# LASTEXITCODE through the Actions PowerShell wrapper after all assertions pass.
exit 0
