param([Parameter(Mandatory=$true)][string]$PersonalDir)
$ErrorActionPreference = "Stop"
$root = (Resolve-Path -LiteralPath $PersonalDir).Path
$path = Join-Path $root "desktop/main.cjs"
$source = [System.IO.File]::ReadAllText($path)
$patches = @(
  @('join(app.getPath("appData"), "SIAAS");', 'join(app.getPath("appData"), "SIAAS-RC4-Preview");'),
  @('app.setName("SIAAS Personal");', 'app.setName("SIAAS Personal RC4 Preview");'),
  @('app.setAppUserModelId("it.siaas.desktop");', 'app.setAppUserModelId("it.siaas.desktop.rc4.preview");')
)
foreach ($pair in $patches) {
  if (-not $source.Contains($pair[0])) { throw "Expected RC4 identity anchor missing: $($pair[0])" }
  $source = $source.Replace($pair[0], $pair[1])
}
[System.IO.File]::WriteAllText($path, $source, [System.Text.UTF8Encoding]::new($false))
$overlay = @'
const original = require("./electron-builder.cjs");
module.exports = {
  ...original,
  appId: "it.siaas.desktop.rc4.preview",
  productName: "SIAAS Personal RC4 Preview",
  artifactName: "SIAAS-Personal-RC4-Preview-__D__{version}-__D__{os}-__D__{arch}.__D__{ext}",
};
'@
$overlay = $overlay.Replace("__D__", [string][char]36)
[System.IO.File]::WriteAllText((Join-Path $root "electron-builder.preview.cjs"), $overlay, [System.Text.UTF8Encoding]::new($false))
if (-not ([System.IO.File]::ReadAllText($path)).Contains('SIAAS-RC4-Preview')) { throw "Preview profile isolation missing" }
Write-Host "Preview-only RC4 identity, appId and user-data directory isolated from Stable."
