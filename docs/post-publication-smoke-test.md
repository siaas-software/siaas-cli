# Anonymous public release smoke test and incident response

This is a **future procedure, not executed evidence**. v0.1.0 has not been
published. Run only after separate authorization to publish, on disposable native
Linux, Windows and macOS accounts with their normal security settings. Record
OS/CPU, account privilege, filesystem, shell, execution policy, timestamps, HTTP
status/redirect hosts, checksums, exit codes and security prompts. Never record
credentials or signed redirect query strings. Browser tests must use a signed-out
session; downloads must send no Authorization header, cookies or GitHub token.

## Publication checks

1. Confirm the release is public, stable, non-draft and designated latest. Download
   `https://github.com/siaas-software/siaas-cli/releases/latest/download/cli-release-v1.txt`
   anonymously. It must begin `siaas-cli-release-v1 0.1.0` and contain exactly the
   six canonical target rows. Stop if latest points at a different version; do
   not silently test a different release.
2. Fetch all twelve inventory assets through
   `https://github.com/siaas-software/siaas-cli/releases/download/v0.1.0/<asset-name>`.
   Keep TLS validation enabled. Review every redirect before following it:
   HTTPS/443 only, at most five hops, exact hosts `github.com`,
   `release-assets.githubusercontent.com`, `objects.githubusercontent.com`.
   Do not approve a new host merely because a redirect requests it.
3. Compare the six binary sizes and SHA-256 values with both the manifest and
   `SHA256SUMS`. Verify the other five checksum entries, including both scripts
   and the manifest. Compare against the reviewed candidate inventory obtained
   through a separately trusted review channel. Checksums are not signatures.
4. Review the downloaded script. Detect HTML/login/error responses before
   execution. A browser download should retain origin metadata; record Windows
   `Zone.Identifier` or macOS `com.apple.quarantine` where present. Do not remove
   those attributes to make acceptance succeed.

## Native bootstrap and CLI checks

On Linux and macOS, use **Bash**, not `sh` or a piped download:

```sh
bash ./install.sh
"$HOME/.local/bin/siaas" --version --json
"$HOME/.local/bin/siaas" --help
```

On Windows, use **Windows PowerShell 5.1**, without elevation:

```powershell
$PSVersionTable
Get-ExecutionPolicy -List
Get-FileHash -LiteralPath .\install.ps1 -Algorithm SHA256
.\install.ps1
& (Join-Path $env:LOCALAPPDATA 'SIAAS CLI\bin\siaas.exe') --version --json
```

If policy blocks the unsigned script or binary, record the refusal and **stop**.
Do not use ExecutionPolicy Bypass/Unrestricted, Unblock-File, xattr deletion,
Gatekeeper disabling, antivirus exclusions or “Run anyway” to claim acceptance.
An organization-approved distribution method or an explicitly scoped unsigned
support decision must be resolved by the release owner. M1 does not provide
publisher signing or notarization.

Where normal policy allows execution, verify version 0.1.0, help and JSON (exit 0),
an unknown command (exit 2), and `siaas install personal` (exit 3; no product side
effects). Run installed CLI with no external language runtime on PATH. Repeat the
bootstrap and verify identical bytes and retained origin metadata. Use a second
private destination containing spaces and Unicode. Check Unix executable mode
0755 and default newly created bin directory 0700; inspect Windows inherited
ACLs to confirm the destination is private to the account and trusted system
administrators. Do not test against a real product data directory.

Using disposable offline copies of the downloaded release, test corrupted and
truncated binaries (exit 6, old executable unchanged), missing binary (exit 7),
concurrent lock (exit 4), replacement of a known fixture, and backup retention.
On Windows test an exclusive handle and a running disposable executable, record
whether replacement safely succeeds or refuses, then close it and retry. Do not
weaken ACLs or terminate unrelated processes. Compare outcomes with native CI.

Browser/Explorer SmartScreen and browser/Finder Gatekeeper behavior require a
separate interactive observation on clean supported systems. CI's synthetic
origin markers and command-line assessment do not establish download reputation,
user-interface behavior, antivirus clearance or notarization acceptance.

## Defective release / incident response

M1 has no product rollback operation and no automated CLI rollback command.

1. Stop installation attempts and preserve the release inventory, logs, current
   executable and backups. Report suspected publisher compromise through the
   owner's established security channel; do not include tokens or personal paths.
2. The release owner must explicitly authorize any withdrawal/latest-release
   changes or account/security actions. This procedure grants no authorization
   to modify GitHub settings, releases or protected tags. Never move an immutable
   release tag or silently replace reviewed assets under the same version.
3. If a safe older CLI is available, verify its provenance and bytes against a
   trusted inventory. A file named `backup` is not automatically trustworthy.
   Stop only the affected CLI and installer. Preserve the defective current file
   under a new private name. Restore the reviewed older binary using a staged
   same-filesystem atomic rename (Unix), or `File.Replace` with a unique backup
   (Windows); if the filesystem refuses, leave files intact and seek assistance.
   Retain origin attributes and normal permissions. Never delete application
   data, models, profiles, the installation directory or all backups.
4. Inspect a stale lock only after confirming no installer is running. Remove
   only abandoned `.siaas-cli-install.lock` / `.siaas-cli-stage.*` entries owned
   by that interrupted installation, then retry with the reviewed local release.
   Online bootstrap always selects latest; do not retry online while latest is
   defective. Local fixture mode explicitly trusts the selected local metadata.
5. Prepare a new fixed version, rerun native and public distribution checks, and
   obtain separate publication approval. For suspected publisher compromise,
   checksums served from the same compromised account cannot establish trust;
   owner-led credential/security recovery and an independent trusted inventory
   are required before resuming distribution.
