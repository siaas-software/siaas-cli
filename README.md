> Linux backend native lifecycle evidence: [successful cycle and limits](docs/m2-native-success.md).
> Public CLI install/update acceptance still requires a compatible release.

> Earlier blocked attempt: [historical first-party M2 report](docs/m2-working-lifecycle.md).

# SIAAS CLI

A standalone, local command-line application that will manage SIAAS products.
This repository is independent of `siaas-personal`. **M1 implements only the
CLI foundation and installation of the CLI itself.** It does not install, start,
stop, update or inspect local Personal installations. This branch adds read-only
verification of published Personal source releases; it does not complete M2.

End users run one executable. **Node.js, npm, Electron, Docker, Ollama and Go are
not runtime prerequisites.** Developers use Go to build and test it.

## Commands

```text
siaas --help
siaas help
siaas --version
siaas --version --json
siaas --help --json
siaas install personal --check
siaas install personal --rc --check
siaas install personal --version 1.1.0-rc.3 --check --json
```

No arguments displays help. `--json` may precede or follow the command. Duplicate
`--json`, unknown options, ambiguous combinations and unsupported commands fail.
Arguments are never echoed in errors, avoiding accidental disclosure of tokens,
credential-bearing URLs and private paths. Release checks contact fixed public
GitHub HTTPS endpoints and verify source archives in bounded memory. They do not
extract or execute source, provision Node, inspect profiles or write installations.
No token, alternate publisher or endpoint override is accepted.

| Exit | Meaning |
| --- | --- |
| 0 | Help/version or read-only release verification completed; not installation success |
| 1 | Output/internal failure |
| 2 | Invalid usage / unknown command or options |
| 3 | Unsupported lifecycle capability or reserved command |
| 4 | Release, network or integrity verification failed |

These codes describe completed invocations. OS signals or a closed output pipe
can terminate a process before it writes a result.

Human successes go to stdout and errors to stderr. JSON mode writes one JSON
object to stdout, including errors, with no human prose on stderr. Its stable
envelope has `schemaVersion: 1`, `ok`, `command`, `version`, and either `message`
or `error: {code, message}`. Help also includes the embedded Go runtime copyright
notice; `--version` remains short.

```json
{"schemaVersion":1,"ok":true,"command":"version","version":"0.1.0","message":"siaas 0.1.0"}
```

Mutating Personal commands validate their options and then return exit **3**
(`unsupported_capability`); invalid options return **2**. `--yes` cannot bypass
the missing backend. `--purge` is explicitly rejected. Other lifecycle commands
remain reserved:

```text
siaas install personal
siaas install personal --version <version>
siaas update personal
siaas rollback personal
siaas uninstall personal
siaas start personal
siaas stop personal
siaas restart personal
siaas status personal
siaas doctor personal
siaas list
```

They do not report success or perform partial lifecycle operations. The intended
[product manifest/API contract](docs/product-manifest-contract.md) is documentation
only. The source-release verifier does not implement that future execution contract.

A successful `--check` includes `personal.compatible: false` and
`runtimeProvisioned: false`; it is not an update applicability check. `--dry-run`
verifies the same release but returns exit 3 because execution is unavailable.
Latest-channel lookup selects the highest stable/RC version without silently
falling back. Future compatible-release negotiation is still blocked. See the
[implementation and acceptance report](docs/m2-lifecycle-progress.md).

## Develop and validate locally

Use the Go version in [.go-version](.go-version), currently **1.27.3**. There are
no third-party Go modules and no package installation step. Bash and standard
Unix utilities are needed for Unix installer tests. Windows installer tests use
Windows PowerShell 5.1 on Windows. The owner-authorized
[native acceptance workflow](docs/native-acceptance.md) uses only free standard
GitHub-hosted runners in this public repository, without artifact uploads or caching.

From the repository root on Unix:

```sh
export GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOTELEMETRY=off
go test -count=1 -timeout=8m ./...
go vet ./...
bash -n install.sh
go run ./tools/build
./dist/siaas-cli_0.1.0_linux_amd64 --version --json
```

Select the filename matching the build host. On Windows PowerShell:

```powershell
$env:GOTOOLCHAIN='local'
$env:GOPROXY='off'
$env:GOSUMDB='off'
$env:GOTELEMETRY='off'
go test -count=1 -timeout=8m ./...
go vet ./...
go run ./tools/build
.\dist\siaas-cli_0.1.0_windows_amd64.exe --version --json
```

Tests use temporary directories, a built CLI executable and local release fixtures.
Unix tests inject fake `uname`, `curl` and a rename barrier; they never contact
external services. They cover parsing/output/exits, standalone execution with an
empty PATH, artifact integrity, missing metadata/artifacts, unsupported platforms,
path safety, Unicode/spaces, repeat installs, backups, locking, SIGTERM/SIGKILL
interruption and secret redaction. Native Windows tests are selected only on
Windows; a Linux test pass does not include those tests. The Unix installer suite
now runs on both Linux and macOS. Windows checks require PowerShell 5.1 Desktop
and exercise exclusive file locks and running executable replacement.

## Standalone executable generation

```sh
go run ./tools/build -out dist
go run ./tools/build -targets linux/amd64,linux/arm64,darwin/amd64,darwin/arm64,windows/amd64,windows/arm64 -out dist/cross
```

The build tool pins the compiler, disables cgo, automatic toolchain downloads and
module network access, and uses `-trimpath -buildvcs=false -ldflags="-s -w
-buildid="`. The version comes from `internal/cli/cli.go`; change it intentionally
before preparing a new release. Each requested target yields a raw executable,
`cli-release-v1.txt` with version/target/name/SHA-256/size records, and `SHA256SUMS`.
There are no archives to unpack during bootstrap. Build only prepares local
artifacts; it does not sign, upload or publish them. `dist/` is ignored by Git.

To check repeatability, build the same target into two fresh output directories
and compare the binaries and metadata. The validation report records an actual
byte-for-byte comparison, binary size and SHA-256. Unsigned repeatability is not
the same as reproducibility after timestamped platform signing.

## Install the CLI using the bootstrap

**There are no published CLI releases as of the 2026-11-08 public-security review.**
The repository is public, and its page was accessible anonymously. Release asset
downloads and remote bootstrap remain unverified until publication. No release,
tag or administrative setting was created or changed for this work. See the
[current public-security verdict](docs/public-release-security-review.md) and
[public installation instructions](docs/public-installation.md).

For an offline, locally built installation from this checkout:

```sh
go run ./tools/build
bash ./install.sh --from-dir "$PWD/dist" --install-dir "$HOME/.local/bin"
"$HOME/.local/bin/siaas" --version
```

On Windows, after building on Windows:

```powershell
go run ./tools/build
.\install.ps1 -FromDir (Join-Path $PWD 'dist')
& (Join-Path $env:LOCALAPPDATA 'SIAAS CLI\bin\siaas.exe') --version
```

Review scripts and run them under your existing execution policy; do not disable
PowerShell policy, TLS validation, Gatekeeper or antivirus. `--from-dir` /
`-FromDir` **explicitly trusts that local directory's metadata**. Checksums detect
corruption, not a malicious local publisher who can change both files. Only use
artifacts built or obtained from a source you trust.

Once public releases are separately authorized and published using the
[CLI release contract](docs/cli-release-contract.md), a reviewed bootstrap script
can be run without `--from-dir` / `-FromDir`. It uses the fixed repository's latest
stable release metadata over HTTPS and derives an immutable versioned artifact
path. No application version argument or existing language runtime is required.
Both bootstraps validate every redirect before connecting: HTTPS on port 443,
at most five hops, and only `github.com`, `release-assets.githubusercontent.com`
or `objects.githubusercontent.com`. Userinfo, fragments, control characters,
backslashes and other hosts/ports are refused. Bash ignores ambient curl config.
Private release authentication is not implemented; the scripts fail clearly
instead of requesting or exposing tokens.

| Bootstrap option | Unix | Windows |
| --- | --- | --- |
| Help | `--help` | `-Help` |
| Installation directory | `--install-dir <absolute-path>` | `-InstallDir <absolute-local-drive-path>` |
| Explicit local fixture/release | `--from-dir <directory>` | `-FromDir <directory>` |
| Default destination | `~/.local/bin/siaas` | `%LOCALAPPDATA%\SIAAS CLI\bin\siaas.exe` |
| Environment override | `SIAAS_CLI_INSTALL_DIR` | `SIAAS_CLI_INSTALL_DIR` |

An explicit option overrides the environment. Installers do not edit PATH or shell
profiles, elevate privileges, download products, execute downloaded binaries,
remove models or access application profiles. Add the chosen directory to your
user PATH yourself, or invoke the installed executable by its path. Unix requires
Bash, standard file utilities, `sha256sum` or `shasum`, and curl only for remote
downloads. Windows needs PowerShell 5.1+ and a local filesystem supporting the
required move/replace operations and NTFS alternate data streams for remote
Internet-origin marking. macOS remote installation additionally requires `xattr`.
Use a private directory under your own account;
shared-writable destinations and concurrent hostile local users are outside the
bootstrap threat model.

Remote installation marks new Windows executables with Internet Zone 3 and new
macOS executables with quarantine, failing safely if marking is unavailable.
Offline copies retain existing origin metadata. Identical-byte installs leave the
existing file and its metadata unchanged. SHA-256 covers file bytes, not ACLs or
origin metadata. These markers preserve OS assessment opportunities; they do not
provide publisher signing or promise that the OS will allow execution.

The installer locks its destination, verifies metadata, stages in that same
directory, checks size and SHA-256, and atomically replaces where supported.
Unix uses same-filesystem rename; Windows uses `File.Replace` with a backup when
a target exists, and refuses delete-then-move fallback. Identical bytes are a
no-op (Unix also ensures executable permissions). A replaced file is retained as
`siaas.backup.*` / `siaas.exe.backup.*`; there is no automatic backup pruning.
Symlink/reparse-point paths and targets, relative paths, traversal and volume
roots are refused. Spaces and Unicode are covered by native offline acceptance.

Installer exits: 0 installed/unchanged, 2 usage/unsupported OS or architecture,
4 locked, 6 filesystem/metadata/integrity failure, 7 missing artifact/download or
prerequisite failure. Unix handles INT/TERM as 130/143 and cleans its own stage.
Native PowerShell interruption codes remain OS/host dependent.

After an error, preserve the current executable and backups. If a force-killed
installer left `.siaas-cli-install.lock` or `.siaas-cli-stage.*`, first confirm that
no installer is active, inspect those exact entries in the selected CLI directory,
then remove only abandoned installer state and retry. Never recursively delete
the install directory or an application data directory. Disk durability through
power loss and recovery from hostile filesystem changes are not guaranteed.

## Platform evidence and boundaries

| Target | Build | Native CLI/bootstrap acceptance in M1 |
| --- | --- | --- |
| Linux x64 | Native PASS | PASS on Debian 13.6 and `ubuntu-24.04`; offline fixtures |
| Linux arm64 | Native PASS | PASS on `ubuntu-24.04-arm`; offline fixtures |
| macOS x64 / arm64 | Native PASS | PASS on `macos-15-intel` / `macos-15`; offline fixtures |
| Windows x64 / arm64 | Native PASS | PASS on `windows-2022` / `windows-11-arm`, PowerShell 5.1 Desktop; offline fixtures, exclusive locks and running-image replacement |

The [native acceptance report](docs/native-acceptance.md) links all six successful
jobs, exact tested commit, 378 passing test/subtest events and Windows PowerShell
versions. No real remote Release installation was tested. The follow-up [public-security review](docs/public-release-security-review.md)
records the latest six-platform PASS, 450 passing events including both genuine
standard-user Windows suites, and OS-policy probes.
Hosted administrator tests alone do not prove standard-user acceptance.
Downloaded-file SmartScreen/Gatekeeper behavior, signing/notarization decisions
and support for older OS versions remain outside this acceptance. No signing
credentials, artifact uploads or paid services were introduced.

See the [architecture decision](docs/adr/0001-standalone-runtime.md),
[validation report](docs/validation-m1.md), and [future product contract](docs/product-manifest-contract.md).

For a complete **local unsigned candidate** on a Linux development host with GNU
utilities and the pinned Go compiler, use a fresh output directory:

```sh
bash tools/prepare-release.sh dist/v0.1.0
```

This assembles all six target binaries, `SHA256SUMS`, `cli-release-v1.txt`, both
bootstrap scripts, `RELEASE_NOTES.md` and `GO-LICENSE.txt`. It performs no upload,
tagging or publication. All six targets now have native offline acceptance. The
[security review](docs/security-review.md) records fixes, trust assumptions and
remaining historical release gates. The current [public-security review](docs/public-release-security-review.md)
is authoritative for the latest candidate and distribution decision. No release
has been published. See the [post-publication smoke and incident procedure](docs/post-publication-smoke-test.md).

Current decision: **READY WITH EXPLICIT RISK ACCEPTANCE** for unsigned M1
distribution. This acceptance has not yet been granted. Internet-marked unsigned
PowerShell scripts can be blocked, and macOS Gatekeeper assessment rejected the
candidates. Public downloads and interactive OS acceptance remain unverified.
See the current security review before authorizing any release.
