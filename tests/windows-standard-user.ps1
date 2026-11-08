# CI-only: create a disposable non-administrator account on the hosted VM.
# No execution-policy, UAC, Defender, SmartScreen or repository setting changes.
param([switch]$Child, [string]$ConfigPath)
$ErrorActionPreference = 'Stop'
if ($Child) {
    $config = Get-Content -LiteralPath $ConfigPath -Raw | ConvertFrom-Json
    $identity = [Security.Principal.WindowsIdentity]::GetCurrent()
    $principal = New-Object Security.Principal.WindowsPrincipal($identity)
    if ($identity.User.Value -ne $config.Sid -or $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator) -or $identity.Groups.Value -contains 'S-1-5-32-544') { throw 'Not a genuine standard-user token.' }
    Remove-Item Env:PSExecutionPolicyPreference -ErrorAction SilentlyContinue
    $env:USERPROFILE = [Environment]::GetFolderPath('UserProfile')
    $env:LOCALAPPDATA = [Environment]::GetFolderPath('LocalApplicationData')
    if (-not $env:USERPROFILE -or -not $env:LOCALAPPDATA -or $env:USERPROFILE -notlike "*\$($config.UserName)") { throw 'Standard-user profile was not loaded.' }
    $env:HOME = $env:USERPROFILE
    $env:TEMP = Join-Path $env:LOCALAPPDATA 'Temp'
    $env:TMP = $env:TEMP
    [void][IO.Directory]::CreateDirectory($env:TEMP)
    $env:GOCACHE = Join-Path $env:LOCALAPPDATA 'go-build'
    $env:GOPATH = Join-Path $env:USERPROFILE 'go'
    $env:GOROOT = $config.GoRoot
    $env:GOTOOLCHAIN = 'local'; $env:GOPROXY = 'off'; $env:GOSUMDB = 'off'; $env:GOENV = 'off'
    $env:SIAAS_STANDARD_USER_ACCEPTANCE = '1'
    $env:SIAAS_EXPECTED_OS = 'windows'; $env:SIAAS_EXPECTED_ARCH = $config.Arch
    Write-Output "STANDARD_USER identity=$($identity.Name) SID=$($identity.User.Value) AdministratorsMember=False"
    Write-Output "STANDARD_USER USERPROFILE=$env:USERPROFILE LOCALAPPDATA=$env:LOCALAPPDATA"
    Get-ExecutionPolicy -List | Format-Table
    Set-Location -LiteralPath (Join-Path $config.Repository 'tests')
    & $config.TestBinary '-test.v' '-test.timeout=10m'
    if ($LASTEXITCODE -ne 0) { throw "Standard-user acceptance failed (exit $LASTEXITCODE)." }
    Write-Output 'STANDARD_USER_ACCEPTANCE=PASS'
    exit 0
}

$userName = 'siaas' + [Guid]::NewGuid().ToString('N').Substring(0,10)
$stage = Join-Path $env:PUBLIC ('Siaas Acceptance ' + [Guid]::NewGuid().ToString('N'))
$created = $false
$childProcess = $null
$savedPolicyOverride = $env:PSExecutionPolicyPreference
try {
    $password = ConvertTo-SecureString ('Siaas!9x' + [Guid]::NewGuid().ToString('N')) -AsPlainText -Force
    $user = New-LocalUser -Name $userName -Password $password -Description 'Disposable SIAAS CLI hosted-runner acceptance'
    $created = $true
    $usersGroup = Get-LocalGroup -SID 'S-1-5-32-545'
    if (-not (Get-LocalGroupMember -Group $usersGroup | Where-Object { $_.SID -eq $user.SID })) { Add-LocalGroupMember -Group $usersGroup -Member $user }
    [void][IO.Directory]::CreateDirectory($stage)
    # Apply ACLs only to this new test staging directory; source is read-only to child.
    $acl = Get-Acl -LiteralPath $stage
    $acl.SetAccessRuleProtection($true, $false)
    foreach ($sid in @('S-1-5-18','S-1-5-32-544')) {
        $rule = New-Object Security.AccessControl.FileSystemAccessRule(([Security.Principal.SecurityIdentifier]$sid),'FullControl','ContainerInherit,ObjectInherit','None','Allow')
        $acl.AddAccessRule($rule)
    }
    $rule = New-Object Security.AccessControl.FileSystemAccessRule($user.SID,'ReadAndExecute','ContainerInherit,ObjectInherit','None','Allow')
    $acl.AddAccessRule($rule)
    Set-Acl -LiteralPath $stage -AclObject $acl
    $repository = Join-Path $stage 'repository'
    [void][IO.Directory]::CreateDirectory($repository)
    foreach ($path in @('.go-version','go.mod','install.sh','install.ps1','cmd','internal','tools','tests')) {
        Copy-Item -LiteralPath (Join-Path $PWD $path) -Destination $repository -Recurse
    }
    $testBinary = Join-Path $stage 'acceptance.test.exe'
    go test -c -o $testBinary ./tests
    if ($LASTEXITCODE -ne 0) { throw 'Cannot build native acceptance executable.' }
    $configFile = Join-Path $stage 'context.json'
    @{ Sid=$user.SID.Value; UserName=$userName; Repository=$repository; TestBinary=$testBinary; GoRoot=(go env GOROOT); Arch=$env:SIAAS_EXPECTED_ARCH } | ConvertTo-Json | Set-Content -LiteralPath $configFile -Encoding UTF8
    $logs = Join-Path $stage 'logs'
    [void][IO.Directory]::CreateDirectory($logs)
    $logACL = Get-Acl -LiteralPath $logs
    $logACL.AddAccessRule((New-Object Security.AccessControl.FileSystemAccessRule($user.SID,'Modify','ContainerInherit,ObjectInherit','None','Allow')))
    Set-Acl -LiteralPath $logs -AclObject $logACL
    $credential = New-Object PSCredential("$env:COMPUTERNAME\$userName",$password)
    # Do not inherit a runner's process-level policy override into the new account.
    Remove-Item Env:PSExecutionPolicyPreference -ErrorAction SilentlyContinue
    $childScript = Join-Path $repository 'tests\windows-standard-user.ps1'
    $arguments = '-NoProfile -NonInteractive -File "' + $childScript + '" -Child -ConfigPath "' + $configFile + '"'
    $childProcess = Start-Process -FilePath "$env:SystemRoot\System32\WindowsPowerShell\v1.0\powershell.exe" -ArgumentList $arguments -Credential $credential -LoadUserProfile -WorkingDirectory $repository -PassThru -RedirectStandardOutput (Join-Path $logs 'stdout.txt') -RedirectStandardError (Join-Path $logs 'stderr.txt')
    # Retain the native process handle before exit so .NET can retrieve its code.
    $null = $childProcess.Handle
    if (-not $childProcess.WaitForExit(720000)) { $childProcess.Kill(); throw 'Standard-user process timed out.' }
    $childProcess.WaitForExit()
    Get-Content -LiteralPath (Join-Path $logs 'stdout.txt')
    Get-Content -LiteralPath (Join-Path $logs 'stderr.txt')
    if ($childProcess.ExitCode -ne 0) { throw "Standard-user process failed (exit $($childProcess.ExitCode)); no policy bypass attempted." }
} finally {
    if ($childProcess -and -not $childProcess.HasExited) { $childProcess.Kill(); $childProcess.WaitForExit() }
    if ($savedPolicyOverride) { $env:PSExecutionPolicyPreference = $savedPolicyOverride }
    if ($created) { Remove-LocalUser -Name $userName }
    # The ephemeral VM discards the loaded profile; remove only our staging directory.
    if (Test-Path -LiteralPath $stage) { Remove-Item -LiteralPath $stage -Recurse -Force }
}
