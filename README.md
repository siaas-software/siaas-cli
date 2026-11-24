# SIAAS CLI

A standalone, local command-line application for managing SIAAS products.

## 1. Installation

No administrator privileges or Node.js, npm, Electron, Docker, Ollama or Go installation is required to run the CLI.

### Windows (PowerShell)

```powershell
Invoke-WebRequest -Uri "https://github.com/siaas-software/siaas-cli/releases/download/v0.2.0/install.ps1" -OutFile install-siaas-cli.ps1
```

Review the script and compare its SHA-256 against the release's SHA256SUMS. Then run under your existing execution policy:

```powershell
.\install-siaas-cli.ps1
$env:Path = "$env:LOCALAPPDATA\SIAAS CLI\bin;$env:Path"
siaas --version
```

### Linux and macOS (Bash)

```bash
curl -fsSL https://github.com/siaas-software/siaas-cli/releases/download/v0.2.0/install.sh -o install-siaas-cli.sh
```

Review the script and compare its SHA-256 against the release's SHA256SUMS. Then run:

```bash
bash install-siaas-cli.sh
export PATH="$HOME/.local/bin:$PATH"
siaas --version
```

See [official releases](https://github.com/siaas-software/siaas-cli/releases). The bootstrap selects the latest stable CLI release and verifies executable size and SHA-256. The versioned URL pins the script, not the installed CLI version. PATH changes above apply only to the current terminal; add the corresponding directory to your user PATH for future sessions. For v0.2.0, avoid evaluating the script with `irm | iex` in your current PowerShell: its final `exit` closes that session. Run the downloaded script as shown or in a separate PowerShell process. Artifacts are unsigned; respect OS security policies.

| OS | Architectures | Default executable |
| --- | --- | --- |
| Windows | x64, ARM64 | `%LOCALAPPDATA%\SIAAS CLI\bin\siaas.exe` |
| Linux | x64, ARM64 | `~/.local/bin/siaas` |
| macOS | Intel, Apple Silicon | `~/.local/bin/siaas` |

## 2. Usage

```text
siaas --help
siaas --version
siaas --version --json
```

Installing the CLI does not install SIAAS Personal. The published v0.2.0 release supports standalone CLI commands, read-only Personal release verification and first-party Personal lifecycle on Linux x64. Run the installed CLI's help to discover its capabilities.

## 3. Commands

| Command | Purpose | Availability |
| --- | --- | --- |
| `siaas`, `siaas help`, `siaas --help` | Show help | v0.1.0+ |
| `siaas --version` | Show version | v0.1.0+ |
| `siaas install personal --rc --check` | Verify release without installation | v0.2.0 |
| `siaas update personal --rc --check` | Verify update candidate | v0.2.0 |
| `siaas install personal --rc --yes` | Install Personal | v0.2.0 (Linux x64) |
| `siaas update personal --rc --yes` | Update Personal | v0.2.0 (Linux x64; RC4 → RC5 verified) |
| `siaas uninstall personal --rc --yes` | Uninstall Personal | v0.2.0 (Linux x64) |

Use `--json` for machine-readable output.

### Published lifecycle details

```text
siaas --help
siaas help
siaas --version
siaas --version --json
siaas --help --json
siaas install personal --check
siaas install personal --rc --check
siaas install personal --version 1.1.0-rc.4 --check --json
siaas install personal --rc --yes --json
siaas update personal --rc --check
siaas update personal --rc --yes --json
siaas uninstall personal --rc --yes --json
```

No arguments displays help. `--json` may precede or follow the command. Duplicate
`--json`, unknown options, ambiguous combinations and unsupported commands fail.
Arguments are never echoed in errors, avoiding accidental disclosure of tokens,
credential-bearing URLs and private paths. Read-only release checks contact
fixed public GitHub HTTPS endpoints and verify source archives in bounded memory.
They do not extract source, provision Node or write installations.
Mutating first-party commands require explicit `--yes`, verified compatible
immutable releases and safe CLI ownership; they execute the reviewed Personal
engine under ordinary user permissions. No arbitrary repository, token, publisher
or endpoint override is accepted.

| Exit | Meaning |
| --- | --- |
| 0 | Help/version, read-only verification or verified completed lifecycle operation |
| 1 | Output/internal failure |
| 2 | Invalid usage / unknown command or options |
| 3 | Unsupported lifecycle capability or reserved command |
| 4 | Release, network or integrity verification failed |
| 5 | Lifecycle operation or postcondition verification failed |

These codes describe completed invocations. OS signals or a closed output pipe
can terminate a process before it writes a result.

Human successes go to stdout and errors to stderr. JSON mode writes one JSON
object to stdout, including errors, with no human prose on stderr. Its stable
envelope has `schemaVersion: 1`, `ok`, `command`, `version`, and either `message`
or `error: {code, message}`. Help also includes the embedded Go runtime copyright
notice; `--version` remains short.

```json
{"schemaVersion":1,"ok":true,"command":"version","version":"0.2.0","message":"siaas 0.2.0"}
```

Mutating Personal commands verify the official first-party capability and
require `--yes` on supported Linux x64 targets. Unsupported legacy releases or
platforms return exit **3** (`unsupported_capability`); invalid options return
**2**. `--yes` never bypasses verification. `--purge` is explicitly rejected.
Other lifecycle commands remain reserved or unsupported without the required options:

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

The supported first-party Linux x64 installation/update/uninstall commands
do not imply availability of the reserved start/stop/rollback/doctor actions.
The [product manifest/API contract](docs/product-manifest-contract.md) remains
reference architecture rather than a generic third-party execution adapter.

A successful `--check` reports verified source integrity and the selected
release's actual `personal.compatible` value. It performs no installation;
`runtimeProvisioned: false` on the audit does not establish that no previous
installation exists. `--dry-run` verifies release availability but currently
returns exit 3 because execution planning is not implemented. A completed
mutating action must additionally verify its local operation receipt, including
the private runtime's persisted checksum after installation. See
[JSON lifecycle reporting](docs/m2-json-reporting.md) for the tested reporting
contract and evidence. Public RC4 → RC5 update acceptance passed; other changed-data-code transitions require independent review. RC3 is not an eligible baseline.

## Personal lifecycle validation status

Linux x64 installation, repeat installation, RC4 → RC5 update, refusal of downgrade and repeated uninstall passed [public lifecycle acceptance](https://github.com/siaas-software/siaas-cli/actions/runs/45373227144). All six native CLI/bootstrap targets passed. The [published v0.2.0 release](https://github.com/siaas-software/siaas-cli/releases/tag/v0.2.0) passed [12 anonymous downloads and Linux bootstrap](https://github.com/siaas-software/siaas-cli/actions/runs/45382863100). A real Windows 11 / PowerShell 7.6.9 installation and version check also passed, with manual PATH configuration. Native GUI and other platform Personal mutations remain unsupported/unverified. See [native lifecycle evidence](docs/m2-native-success.md), [historical blocked attempt](docs/m2-working-lifecycle.md) and [issue #2](https://github.com/siaas-software/siaas-cli/issues/2).

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
./dist/siaas-cli_0.2.0_linux_amd64 --version --json
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
.\dist\siaas-cli_0.2.0_windows_amd64.exe --version --json
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

## Advanced bootstrap and offline installation

The [public-security review](docs/public-release-security-review.md) and [original installation instructions](docs/public-installation.md) describe the historical pre-publication review.

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

For published releases following the
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

| Target | Build | Native CLI/bootstrap acceptance (v0.1.0) |
| --- | --- | --- |
| Linux x64 | Native PASS | PASS on Debian 13.6 and `ubuntu-24.04`; offline fixtures |
| Linux arm64 | Native PASS | PASS on `ubuntu-24.04-arm`; offline fixtures |
| macOS x64 / arm64 | Native PASS | PASS on `macos-15-intel` / `macos-15`; offline fixtures |
| Windows x64 / arm64 | Native PASS | PASS on `windows-2022` / `windows-11-arm`, PowerShell 5.1 Desktop; offline fixtures, exclusive locks and running-image replacement |

The [native acceptance report](docs/native-acceptance.md) links all six successful
jobs, exact tested commit, 378 passing test/subtest events and Windows PowerShell
versions. No real remote Release installation was tested in that historical acceptance. The historical follow-up [public-security review](docs/public-release-security-review.md)
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
bash tools/prepare-release.sh dist/v0.2.0
```

This assembles all six target binaries, `SHA256SUMS`, `cli-release-v1.txt`, both
bootstrap scripts, `RELEASE_NOTES.md` and `GO-LICENSE.txt`. It performs no upload,
tagging or publication. All six targets now have native offline acceptance. The
[security review](docs/security-review.md) records fixes, trust assumptions and
remaining historical release gates. The current [public-security review](docs/public-release-security-review.md)
records the historical unsigned distribution decision. The v0.2.0 publication evidence above supersedes its pre-publication status. See the [post-publication smoke and incident procedure](docs/post-publication-smoke-test.md).

Historical review decision: **READY WITH EXPLICIT RISK ACCEPTANCE** for unsigned v0.1.0
distribution. Internet-marked unsigned
PowerShell scripts can be blocked, and macOS Gatekeeper assessment rejected the
candidates. Public downloads and interactive OS acceptance were unverified at the time of that review.
See the security review before authorizing any future release.
