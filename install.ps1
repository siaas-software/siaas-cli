# SIAAS CLI bootstrap. PowerShell 5.1+; no administrator privileges or product install.
$InstallDir = ''
$FromDir = ''
$Help = $false
$ErrorActionPreference = 'Stop'
$script:ResultCode = 1
$stage = $null
$lockStream = $null
$lockPath = $null
$oldTLS = [Net.ServicePointManager]::SecurityProtocol
function Fail([int]$Code, [string]$Message) {
    $script:ResultCode = $Code
    throw [InvalidOperationException]::new($Message)
}
function Get-Artifact([string]$Name, [string]$Destination, [long]$Limit, [string]$URL) {
    if ($FromDir) {
        $source = [IO.Path]::Combine($FromDir, $Name)
        if (-not [IO.File]::Exists($source)) { Fail 7 'Required local release artifact is missing. Build CLI artifacts first.' }
        if (([IO.File]::GetAttributes($source) -band [IO.FileAttributes]::ReparsePoint) -ne 0) { Fail 6 'Symlink release artifacts are refused.' }
        if (([IO.FileInfo]$source).Length -gt $Limit) { Fail 6 'Release artifact exceeds its permitted size.' }
        [IO.File]::Copy($source, $Destination, $false)
        return
    }
    $uri = [Uri]$URL
    for ($redirect = 0; $redirect -le 5; $redirect++) {
        if ($uri.Scheme -ne 'https' -or $uri.UserInfo -or $uri.Host -notin @('github.com', 'release-assets.githubusercontent.com', 'objects.githubusercontent.com')) {
            Fail 6 'Untrusted release redirect refused.'
        }
        $request = [Net.HttpWebRequest]::Create($uri)
        $request.AllowAutoRedirect = $false
        $request.Timeout = 30000
        $request.ReadWriteTimeout = 30000
        $request.UserAgent = 'SIAAS-CLI-Bootstrap/1'
        try { $response = $request.GetResponse() }
        catch { Fail 7 'CLI release unavailable or download failed. Check release access/network or use local fixtures; remote installation is not guaranteed before publication.' }
        try {
            if ([int]$response.StatusCode -in @(301,302,303,307,308)) {
                $uri = [Uri]::new($uri, $response.Headers['Location'])
                continue
            }
            if ([int]$response.StatusCode -ne 200) { Fail 7 'Release server returned an unexpected response.' }
            if ($response.ContentLength -gt $Limit) { Fail 6 'Release artifact exceeds its permitted size.' }
            $inputStream = $response.GetResponseStream()
            $outputStream = [IO.File]::Open($Destination, [IO.FileMode]::CreateNew)
            try {
                $buffer = New-Object byte[] 65536
                $total = 0L
                $timer = [Diagnostics.Stopwatch]::StartNew()
                while (($count = $inputStream.Read($buffer, 0, $buffer.Length)) -gt 0) {
                    $total += $count
                    if ($total -gt $Limit) { Fail 6 'Release artifact exceeds its permitted size.' }
                    if ($timer.Elapsed.TotalSeconds -gt 180) { Fail 7 'Download deadline exceeded. Existing CLI preserved; retry later.' }
                    $outputStream.Write($buffer, 0, $count)
                }
            } finally { $outputStream.Dispose(); $inputStream.Dispose() }
            return
        } finally { $response.Dispose() }
    }
    Fail 7 'Too many release redirects. Existing CLI preserved.'
}
try {
    # Parse manually so PowerShell's parameter binder cannot echo unknown secret arguments.
    for ($index = 0; $index -lt $args.Count; $index++) {
        switch ($args[$index]) {
            '-Help' { if ($args.Count -ne 1) { Fail 2 'Use -Help alone.' }; $Help = $true }
            { $_ -in @('-InstallDir','-FromDir') } {
                $option = $args[$index]
                $index++
                if ($index -ge $args.Count -or -not $args[$index]) { Fail 2 'An option value is missing. Use -Help.' }
                if ($option -eq '-InstallDir') { $InstallDir = [string]$args[$index] } else { $FromDir = [string]$args[$index] }
            }
            default { Fail 2 'Unsupported arguments. Use -Help. No product argument is accepted.' }
        }
    }
    if ($Help) {
        Write-Output 'Usage: .\install.ps1 [-InstallDir ABSOLUTE_PATH] [-FromDir LOCAL_RELEASE_DIRECTORY]'
        Write-Output 'Installs SIAAS CLI itself; no product version is required. -FromDir explicitly trusts local metadata.'
        $script:ResultCode = 0
    } else {
        if ([Environment]::OSVersion.Platform -ne [PlatformID]::Win32NT) { Fail 2 'This bootstrap requires Windows. Use install.sh on Linux or macOS.' }
        $machine = if ($env:PROCESSOR_ARCHITEW6432) { $env:PROCESSOR_ARCHITEW6432 } else { $env:PROCESSOR_ARCHITECTURE }
        $arch = switch ($machine) { 'AMD64' { 'amd64' } 'ARM64' { 'arm64' } default { Fail 2 'Supported Windows architectures are x64 and arm64.' } }
        if (-not $InstallDir) { $InstallDir = $env:SIAAS_CLI_INSTALL_DIR }
        if (-not $InstallDir) {
            if (-not $env:LOCALAPPDATA) { Fail 2 'LOCALAPPDATA is unavailable. Choose an explicit user-owned install directory.' }
            $InstallDir = [IO.Path]::Combine($env:LOCALAPPDATA, 'SIAAS CLI', 'bin')
        }
        if ($InstallDir -notmatch '^[A-Za-z]:[\\/]' -or $InstallDir -match '[\x00-\x1f\x7f]' -or $InstallDir -match '(^|[\\/])\.{1,2}([\\/]|$)') {
            Fail 2 'Use an absolute local-volume directory without control characters or dot components.'
        }
        $InstallDir = [IO.Path]::GetFullPath($InstallDir).TrimEnd('\','/')
        if ($InstallDir.Length -le 3) { Fail 2 'Installing at a volume root is refused.' }
        $cursor = $InstallDir
        while ($cursor) {
            if ([IO.Directory]::Exists($cursor) -or [IO.File]::Exists($cursor)) {
                if (([IO.File]::GetAttributes($cursor) -band [IO.FileAttributes]::ReparsePoint) -ne 0) { Fail 6 'Reparse-point install paths are refused. Choose a private real directory.' }
            }
            $parent = [IO.Directory]::GetParent($cursor)
            $cursor = if ($parent) { $parent.FullName } else { $null }
        }
        [void][IO.Directory]::CreateDirectory($InstallDir)
        $lockPath = [IO.Path]::Combine($InstallDir, '.siaas-cli-install.lock')
        try { $lockStream = [IO.File]::Open($lockPath, [IO.FileMode]::CreateNew, [IO.FileAccess]::Write, [IO.FileShare]::None) }
        catch { Fail 4 'Installation locked or directory unwritable. Wait for the installer; after interruption confirm it has exited before removing only .siaas-cli-install.lock.' }
        $stage = [IO.Path]::Combine($InstallDir, '.siaas-cli-stage.' + [Guid]::NewGuid().ToString('N'))
        [void][IO.Directory]::CreateDirectory($stage)
        $target = [IO.Path]::Combine($InstallDir, 'siaas.exe')
        if ([IO.Directory]::Exists($target)) { Fail 6 'Existing siaas.exe is a directory. Inspect it without deleting application data.' }
        if ([IO.File]::Exists($target) -and (([IO.File]::GetAttributes($target) -band [IO.FileAttributes]::ReparsePoint) -ne 0)) { Fail 6 'Existing siaas.exe is a reparse point. Choose a regular file location.' }
        [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
        $base = 'https://github.com/siaas-software/siaas-cli/releases'
        $manifest = [IO.Path]::Combine($stage, 'manifest')
        Get-Artifact 'cli-release-v1.txt' $manifest 65536 "$base/latest/download/cli-release-v1.txt"
        $lines = [IO.File]::ReadAllLines($manifest)
        if ($lines.Length -lt 2 -or $lines[0] -cnotmatch '^siaas-cli-release-v1 ((?:0|[1-9][0-9]*)\.(?:0|[1-9][0-9]*)\.(?:0|[1-9][0-9]*))$') { Fail 6 'Invalid release metadata schema/version.' }
        $version = $Matches[1]
        $selected = @()
        foreach ($line in $lines[1..($lines.Length-1)]) {
            $fields = $line.Split(' ')
            if ($fields.Length -ne 5 -or ($fields[0]+'/'+$fields[1]) -notin @('linux/amd64','linux/arm64','darwin/amd64','darwin/arm64','windows/amd64','windows/arm64')) { Fail 6 'Invalid release platform entry.' }
            $suffix = if ($fields[0] -eq 'windows') { '.exe' } else { '' }
            if ($fields[2] -cne "siaas-cli_${version}_$($fields[0])_$($fields[1])$suffix" -or $fields[3] -cnotmatch '^[a-f0-9]{64}$' -or $fields[4] -notmatch '^[1-9][0-9]{0,8}$') { Fail 6 'Invalid release artifact metadata.' }
            if ($fields[0] -eq 'windows' -and $fields[1] -eq $arch) { $selected += ,$fields }
        }
        if ($selected.Count -ne 1) { Fail 7 'Release has no unique artifact for this platform. Existing CLI preserved.' }
        $artifact = $selected[0][2]; $expected = $selected[0][3]; $bytes = [long]$selected[0][4]
        if ($bytes -gt 100000000) { Fail 6 'CLI artifact is unexpectedly large.' }
        $candidate = [IO.Path]::Combine($stage, 'candidate.exe')
        Get-Artifact $artifact $candidate $bytes "$base/download/v$version/$artifact"
        if (([IO.FileInfo]$candidate).Length -ne $bytes -or (Get-FileHash -LiteralPath $candidate -Algorithm SHA256).Hash.ToLowerInvariant() -cne $expected) { Fail 6 'Artifact checksum/size verification failed. Existing CLI preserved; obtain a fresh verified release.' }
        if ([IO.File]::Exists($target) -and (Get-FileHash -LiteralPath $target -Algorithm SHA256).Hash.ToLowerInvariant() -ceq $expected) {
            Write-Output 'SIAAS CLI is already installed with identical verified bytes.'
        } elseif ([IO.File]::Exists($target)) {
            $backup = [IO.Path]::Combine($InstallDir, 'siaas.exe.backup.' + [Guid]::NewGuid().ToString('N'))
            # Never fall back to delete-then-move. Unsupported filesystems fail safely.
            [IO.File]::Replace($candidate, $target, $backup)
            Write-Output "Installed SIAAS CLI $version. Previous executable retained as siaas.exe.backup.*."
        } else {
            [IO.File]::Move($candidate, $target)
            Write-Output "Installed SIAAS CLI $version. No SIAAS product was installed."
        }
        Write-Output 'Add the selected directory to your user PATH manually, or invoke its siaas.exe directly.'
        $script:ResultCode = 0
    }
} catch {
    # Only our fixed messages are printable; native exceptions may include secrets/paths.
    $message = if ($_.Exception -is [InvalidOperationException] -and $script:ResultCode -ne 1) { $_.Exception.Message } else { 'Installation failed. Existing CLI/backup preserved. Close CLI processes, check permissions/free space and use a private user directory; retry after inspection.' }
    if ($script:ResultCode -eq 1) { $script:ResultCode = 6 }
    [Console]::Error.WriteLine('siaas installer: ' + $message)
} finally {
    [Net.ServicePointManager]::SecurityProtocol = $oldTLS
    if ($stage -and [IO.Directory]::Exists($stage)) { try { [IO.Directory]::Delete($stage, $true) } catch { [Console]::Error.WriteLine('Inspect the remaining .siaas-cli-stage directory after the installer exits.'); if ($script:ResultCode -eq 0) { $script:ResultCode = 6 } } }
    if ($lockStream) {
        $lockStream.Dispose()
        try { [IO.File]::Delete($lockPath) } catch { [Console]::Error.WriteLine('Installer lock remains; inspect it before retrying.'); if ($script:ResultCode -eq 0) { $script:ResultCode = 6 } }
    }
}
exit $script:ResultCode
