# SIAAS CLI

A standalone, local command-line application that will manage SIAAS products.
This repository is independent of `siaas-personal`. **M1 implements only the
CLI foundation and installation of the CLI itself.** It does not install, start,
stop, update or inspect SIAAS Personal or any other product.

End users run one executable. **Node.js, npm, Electron, Docker, Ollama and Go are
not runtime prerequisites.** Developers use Go to build and test it.

## Commands

```text
siaas --help
siaas help
siaas --version
siaas --version --json
siaas --help --json
```

No arguments displays help. `--json` may precede or follow the command. Duplicate
`--json`, unknown options, ambiguous combinations and unsupported commands fail.
Arguments are never echoed in errors, avoiding accidental disclosure of tokens,
credential-bearing URLs and private paths. No command reads credentials, opens a
network connection, creates configuration or changes application data.

| Exit | Meaning |
| --- | --- |
| 0 | Help or version successfully written |
| 1 | Output/internal failure |
| 2 | Invalid usage / unknown command or options |
| 3 | Recognized future product command is not implemented |

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

Future commands are reserved and currently return exit **3**, including when
additional arguments or `--help` follow them:

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
only, not a product client or a dependency on the Personal repository.

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
go test -count=1 -timeout=90s ./...
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
go test -count=1 -timeout=90s ./...
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

**There are no published CLI releases as of the 2026-11-07 readiness review.**
The repository is public, and its page was accessible anonymously. Release asset
downloads and remote bootstrap remain unverified until publication. No release,
tag or administrative setting was created or changed for this work. See the
[release-readiness verdict](docs/release-readiness-v0.1.0.md) and
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
required move/replace operations. Use a private directory under your own account;
shared-writable destinations and concurrent hostile local users are outside the
bootstrap threat model.

The installer locks its destination, verifies metadata, stages in that same
directory, checks size and SHA-256, and atomically replaces where supported.
Unix uses same-filesystem rename; Windows uses `File.Replace` with a backup when
a target exists, and refuses delete-then-move fallback. Identical bytes are a
no-op (Unix also ensures executable permissions). A replaced file is retained as
`siaas.backup.*` / `siaas.exe.backup.*`; there is no automatic backup pruning.
Symlink/reparse-point paths and targets, relative paths, traversal and volume
roots are refused. Spaces and Unicode are supported by design and tested on Linux.

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
versions. No real remote Release installation was tested. Hosted Windows accounts
may be administrators; standard-user ACL/reparse-point behavior remains unverified.
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
remaining release gates. **Native acceptance: READY. Public release: NOT READY
for authorization pending the documented review/distribution decisions.**
