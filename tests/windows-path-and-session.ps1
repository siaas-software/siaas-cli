# Hosted Windows acceptance: exact IEX invocation, persistence and opt-out.
$ErrorActionPreference = 'Stop'
if ($env:GITHUB_ACTIONS -ne 'true' -or $env:GITHUB_REPOSITORY -ne 'siaas-software/siaas-cli') { throw 'Run only in the disposable official Windows runner.' }
$repository = Split-Path $PSScriptRoot -Parent
$stage = Join-Path $env:TEMP ('Siaas PATH Test ' + [Guid]::NewGuid().ToString('N'))
$originalProcess = $env:Path
$originalLocal = $env:LOCALAPPDATA
$originalOverride = $env:SIAAS_CLI_INSTALL_DIR
$key = [Microsoft.Win32.Registry]::CurrentUser.CreateSubKey('Environment')
$hadPath = $key.GetValueNames() -contains 'Path'
$originalRaw = $key.GetValue('Path', '', [Microsoft.Win32.RegistryValueOptions]::DoNotExpandEnvironmentNames)
$originalKind = if ($hadPath) { $key.GetValueKind('Path') } else { [Microsoft.Win32.RegistryValueKind]::String }
try {
    $raw = '%SystemRoot%\System32;C:\Siaas fixture existing'
    $key.SetValue('Path',$raw,[Microsoft.Win32.RegistryValueKind]::ExpandString)
    [void][IO.Directory]::CreateDirectory($stage)
    # TEMP on hosted Windows can use an 8.3 alias (RUNNER~1). Match the installer's full-path normalization before testing exact strings.
    $stage = [IO.Path]::GetFullPath($stage)
    $env:LOCALAPPDATA = $stage
    Remove-Item Env:SIAAS_CLI_INSTALL_DIR -ErrorAction SilentlyContinue
    $installer = Get-Content -LiteralPath (Join-Path $repository 'install.ps1') -Raw
    $dest = Join-Path $stage 'SIAAS CLI\bin'
    # Use real public release downloads, but the reviewed candidate installer.
    Invoke-Expression $installer
    if ($LASTEXITCODE -ne 0) { throw 'IEX remote install failed.' }
    $expectedExe = [IO.Path]::GetFullPath((Join-Path $dest 'siaas.exe'))
    $resolved = Get-Command siaas -ErrorAction Stop
    if ($resolved.CommandType -ne 'Application' -or [IO.Path]::GetFullPath($resolved.Source) -ne $expectedExe) { throw "CLI not on session PATH. Expected=$expectedExe; resolved=$($resolved.Source); type=$($resolved.CommandType)" }
    if (-not [IO.File]::Exists($expectedExe)) { throw 'Expected installed binary missing.' }
    Write-Output "Resolved CLI: $($resolved.Source)"
    if ((siaas --version) -notmatch '^siaas 0\.2\.[01]$') { throw 'Unexpected remote CLI version.' }
    $dest = [IO.Path]::GetFullPath($dest)
    $expected = $raw + ';' + $dest
    if ($key.GetValue('Path','',[Microsoft.Win32.RegistryValueOptions]::DoNotExpandEnvironmentNames) -cne $expected -or $key.GetValueKind('Path') -ne [Microsoft.Win32.RegistryValueKind]::ExpandString) { throw 'User PATH or registry type not preserved.' }
    Invoke-Expression $installer
    if ($LASTEXITCODE -ne 0 -or $key.GetValue('Path','',[Microsoft.Win32.RegistryValueOptions]::DoNotExpandEnvironmentNames) -cne $expected) { throw 'Repeated install duplicated PATH.' }
    if (@($env:Path.Split(';') | Where-Object { $_ -eq $dest }).Count -ne 1) { throw 'Duplicate process PATH.' }
    $env:SIAAS_CLI_INSTALL_DIR = 'relative-invalid'
    Invoke-Expression $installer
    if ($LASTEXITCODE -ne 2) { throw 'IEX failure did not preserve exit code.' }
    if ($key.GetValue('Path','',[Microsoft.Win32.RegistryValueOptions]::DoNotExpandEnvironmentNames) -cne $expected) { throw 'Failed install changed PATH.' }
    $env:SIAAS_CLI_INSTALL_DIR = Join-Path $stage 'custom'
    Invoke-Expression $installer
    if ($LASTEXITCODE -ne 0 -or $key.GetValue('Path','',[Microsoft.Win32.RegistryValueOptions]::DoNotExpandEnvironmentNames) -cne $expected) { throw 'Custom install unexpectedly edited user PATH.' }
    & (Join-Path $repository 'install.ps1') -InstallDir $env:SIAAS_CLI_INSTALL_DIR -AddToPath
    if ($LASTEXITCODE -ne 0) { throw 'Explicit PATH opt-in failed.' }
    $expected += ';' + $env:SIAAS_CLI_INSTALL_DIR
    if ($key.GetValue('Path','',[Microsoft.Win32.RegistryValueOptions]::DoNotExpandEnvironmentNames) -cne $expected) { throw 'Explicit PATH opt-in not persisted.' }
    Remove-Item Env:SIAAS_CLI_INSTALL_DIR -ErrorAction SilentlyContinue
    $env:LOCALAPPDATA = Join-Path $stage 'opt-out'
    [void][IO.Directory]::CreateDirectory($env:LOCALAPPDATA)
    $beforeOptOut = $env:Path
    & (Join-Path $repository 'install.ps1') -NoPath
    if ($LASTEXITCODE -ne 0 -or $key.GetValue('Path','',[Microsoft.Win32.RegistryValueOptions]::DoNotExpandEnvironmentNames) -cne $expected -or $env:Path -cne $beforeOptOut -or -not [IO.File]::Exists((Join-Path $env:LOCALAPPDATA 'SIAAS CLI\bin\siaas.exe'))) { throw 'PATH opt-out failed.' }
    Write-Output ('SESSION_PATH_ACCEPTANCE=PASS PowerShell ' + $PSVersionTable.PSVersion)
} finally {
    if ($hadPath) { $key.SetValue('Path',$originalRaw,$originalKind) } else { $key.DeleteValue('Path',$false) }
    $key.Dispose()
    $env:Path = $originalProcess
    $env:LOCALAPPDATA = $originalLocal
    $env:SIAAS_CLI_INSTALL_DIR = $originalOverride
    if (Test-Path -LiteralPath $stage) { Remove-Item -LiteralPath $stage -Recurse -Force }
}
exit 0
