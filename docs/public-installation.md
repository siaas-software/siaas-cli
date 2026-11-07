# Public CLI bootstrap installation

Status on 2026-11-07: repository public; **v0.1.0 is not published**. No working
release download or remote install is claimed. The following is the reviewed
installation procedure **for use after separately authorized publication**.

1. Open <https://github.com/siaas-software/siaas-cli/releases> anonymously. Choose
   the approved stable CLI release and its assets. For v0.1.0, download `install.sh`
   (Unix) or `install.ps1` (Windows), plus `SHA256SUMS`, using the browser's normal
   HTTPS/certificate checks. Do not execute an HTML error page as a script.
2. Review the script and compare its SHA-256 to the corresponding `SHA256SUMS`
   entry. Unix: `sha256sum install.sh` (Linux) or `shasum -a 256 install.sh`
   (macOS). Windows: `Get-FileHash -LiteralPath .\install.ps1 -Algorithm SHA256`.
   This establishes integrity relative to the publisher, not independent trust
   in the publisher or the downloaded checksum list. No signatures are provided.
3. Run the reviewed script from its download directory:

   ```sh
   bash ./install.sh
   "$HOME/.local/bin/siaas" --version
   ```

   ```powershell
   .\install.ps1
   & (Join-Path $env:LOCALAPPDATA 'SIAAS CLI\bin\siaas.exe') --version
   ```

   Use your existing script execution policy. If policy rejects the unsigned
   script, obtain an organization-approved execution method; do not bypass TLS,
   PowerShell policy, antivirus or Gatekeeper. Windows/macOS native offline
   acceptance now passes on the [recorded hosted runners](native-acceptance.md);
   downloaded-file OS policy and remote installation remain unverified.

No application version argument, administrator rights or separately installed
Node.js/npm/Electron/Docker/Ollama/Go is required. Bash/curl and standard Unix
utilities, or Windows PowerShell 5.1 and Windows APIs, provide the bootstrap.
The script always selects the **latest stable CLI** manifest from the fixed
repository, then fetches the artifact under the manifest's versioned release path.
Running an older bootstrap does not pin the CLI version. M1 provides no downgrade
prevention or authenticated freshness guarantee beyond trusted HTTPS/repository
control; review the selected publisher release.

For installation with a specific reviewed local release directory, download its
`cli-release-v1.txt` and the raw executable matching the OS/architecture (or build
them locally), then use `--from-dir '/absolute/release directory'` on Unix or
`-FromDir 'C:\release directory'` on Windows. This explicitly trusts local
metadata. Both paths validate the exact binary size and SHA-256 before replacement.

An optional `--install-dir` / `-InstallDir` selects an absolute private directory;
spaces and Unicode are permitted. Defaults are `~/.local/bin` and
`%LOCALAPPDATA%\SIAAS CLI\bin`. The scripts do not edit PATH or shell profiles.
Invoking `siaas install personal` returns **3: not implemented**; no product is
installed or changed by M1.

On failure, retain the current executable and `siaas.backup.*` /
`siaas.exe.backup.*`. Close CLI processes before retrying a Windows replacement.
If a crashed installer left a lock, confirm every installer targeting that
directory has exited, inspect its `.siaas-cli-install.lock` and
`.siaas-cli-stage.*` entries, then remove only abandoned installer state. Never
delete the CLI directory recursively or touch application data. Backups are not
pruned automatically. To restore a CLI backup, first verify its origin/bytes,
close running CLI/installers, preserve the current file, and use your OS's safe
file replacement procedure; no automated rollback command exists in M1.

Publisher routing (prospective, not tested downloads):

```text
https://github.com/siaas-software/siaas-cli/releases/latest/download/cli-release-v1.txt
https://github.com/siaas-software/siaas-cli/releases/download/v0.1.0/<asset-name>
```

The public repository permits anonymous Release access once a release is
published. A draft or prerelease is not a valid `latest` source. Before publication,
upload the complete asset inventory and check metadata. After separately approved
publication, an operator must verify the manifest and each asset without login,
then run native remote bootstraps with no tokens and normal certificate validation.
Those network checks cannot be passed in advance while assets do not exist.
