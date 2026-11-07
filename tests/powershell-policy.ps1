# Supplemental portable parser/URI tests, NOT native Windows bootstrap acceptance.
$ErrorActionPreference = 'Stop'
$tokens = $null
$errors = $null
$path = Join-Path (Split-Path $PSScriptRoot -Parent) 'install.ps1'
$ast = [Management.Automation.Language.Parser]::ParseFile($path, [ref]$tokens, [ref]$errors)
if ($errors.Count) { throw 'Installer syntax errors.' }
# Load only the reviewed pure predicate, never execute the installer entry point.
$function = $ast.Find({ param($node) $node -is [Management.Automation.Language.FunctionDefinitionAst] -and $node.Name -eq 'Test-TrustedReleaseUri' }, $false)
if (-not $function) { throw 'Missing URI policy function.' }
. ([ScriptBlock]::Create($function.Extent.Text))
$accepted = @(
    'https://github.com/siaas-software/siaas-cli/releases/latest/download/cli-release-v1.txt',
    'https://release-assets.githubusercontent.com/asset?signature=secret-sentinel',
    'https://objects.githubusercontent.com/asset',
    'https://github.com:443/asset'
)
$rejected = @(
    'http://github.com/asset', 'https://evil.example/asset',
    'https://github.com.evil.example/asset', 'https://github.com@evil.example/asset',
    'https://secret-sentinel@github.com/asset', 'https://github.com:444/asset',
    'https://github.com/asset#secret-sentinel', 'https://github.com/\secret-sentinel',
    "https://github.com/`nsecret-sentinel", 'file:///secret-sentinel'
)
foreach ($value in $accepted) { if (-not (Test-TrustedReleaseUri ([Uri]$value))) { throw 'Trusted URI rejected.' } }
foreach ($value in $rejected) { if (Test-TrustedReleaseUri ([Uri]$value)) { throw 'Untrusted URI accepted.' } }
$resolved = [Uri]::new([Uri]$accepted[0], '/siaas-software/siaas-cli/releases/download/v0.1.0/cli-release-v1.txt')
if (-not (Test-TrustedReleaseUri $resolved)) { throw 'Relative GitHub redirect rejected.' }
Write-Output 'PASS: installer syntax; 4 accepted, 10 rejected URIs; relative GitHub redirect. Native Windows remains unverified on non-Windows hosts.'
