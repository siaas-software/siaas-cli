# M1 reproducible validation report

Date: 2026-11-07. Repository: `siaas-software/siaas-cli` only.
Default branch inspected: `main`, baseline
`541de994375ea7bc79a87f68422566e7c4f10ed2` (`Initial commit`).
Branch: `feature/foundation-standalone-bootstrap`.
The baseline contained only `README.md`; no `AGENTS.md` or other repository
instructions were present. No files in SIAAS Personal or another repository were
changed. No product was installed; PRs #20/#21 were not modified.

**Outcome:** M1 source, standalone build and Linux fixture bootstrap validated.
Windows/macOS/native ARM acceptance and remote bootstrap remain **UNVERIFIED**.
Product lifecycle commands remain explicitly unimplemented. This is a draft-review
result, not release publication or all-platform production acceptance.

## Environment and acquisition

- Debian GNU/Linux 13.6 (trixie), Linux 6.18.57, x86_64.
- Go `go1.27.3 linux/amd64`, pinned in `.go-version` and `go.mod`.
- Existing Node `v24.19.0` inspected only for the SEA comparison; not used to build,
  test or execute SIAAS CLI.
- Optional portable PowerShell `7.6.9` used on Linux for parser/rejection checks.
  Windows PowerShell 5.1 and native Windows were not available.
- `/usr/bin/go` was an unrelated executable, not the Go language compiler. A
  separate user-local toolchain was verified under `/tmp/siaas-cli-toolchain/go`;
  no global PATH or system installation was changed.

Development tool acquisition used TLS-verified official downloads, separate from
the offline baseline tests. The official Go download metadata selected
`go1.27.3.linux-amd64.tar.gz` (70,588,678 bytes), verified SHA-256:

```text
65efbc80fdc1fa2cd2631cad5b481777a68e8725f34e89f50fc6ac714e379b40
```

Portable `powershell-7.6.9-linux-x64.tar.gz` was verified against the official
GitHub release asset digest before extraction to `/tmp/siaas-cli-powershell`:

```text
1da6d6d6035d651f20e70ddee7c4c15ac6ab8b9fb40ee92240d91263f28a884b
```

No dependencies were installed into another repository. There are no third-party
Go module dependencies, and no npm files or lockfiles. Tool acquisition required
network access; normal build/test commands below did not.

## Commands and actual outcomes

Commands ran from the repository root. In this environment `go` below was invoked
as `/tmp/siaas-cli-toolchain/go/bin/go`. Each build/test used
`GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOTELEMETRY=off` and the user-owned cache
`GOCACHE=/tmp/siaas-cli-go-cache`. The build tool also disables cgo and restricts
the child build environment. Race tests use the host C compiler; it is not a CLI
runtime or ordinary build requirement.

| Command/check | Actual result | Scope/limitations |
| --- | --- | --- |
| `git clone --branch main …`; inspect tree/README; `git switch -c feature/foundation-standalone-bootstrap` | PASS, default branch confirmed using GitHub metadata and checkout | No SIAAS Personal branch used |
| `go test -count=1 -timeout=90s -v ./...` | PASS on first baseline run | Offline CLI, Linux bootstrap and standalone integration tests |
| Final `go test -race -count=1 -timeout=90s -json ./...` | **PASS, exit 0: 12 top-level tests, 46 passing test/subtest events, 0 failures, 0 skipped tests** | [Retained output](evidence/m1-tests.txt). Two command packages have no test files; Windows test file is excluded by OS build tag, not executed or counted as passed |
| `go vet ./...` | PASS, exit 0 | Static Go checks |
| `gofmt -l cmd internal tools tests` | PASS, no output | Formatting |
| `bash -n install.sh` | PASS, exit 0 | Bash syntax |
| `go run ./tools/build` | PASS, exit 0 | Host Linux x64 raw executable, manifest and checksums |
| `go run ./tools/build -targets linux/amd64,linux/arm64,darwin/amd64,darwin/arm64,windows/amd64,windows/arm64 -out dist/cross` | PASS, exit 0 | Six cross-built artifacts; only Linux x64 run |
| `GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go test -c ./tests -o dist/windows-installer-tests.exe` | PASS, exit 0 | Windows test code compiles; **tests not executed** |
| `go run ./tools/build -out dist/repro-a`; same with `dist/repro-b`; `cmp` binaries and manifests | **PASS, both comparisons exit 0** | Same source/compiler/target, separate output directories; independent clean-room builder reproduction not claimed |
| `file dist/siaas-cli_0.1.0_linux_amd64` | ELF x86-64, statically linked, stripped | Structural evidence |
| `ldd dist/siaas-cli_0.1.0_linux_amd64` | `not a dynamic executable` (expected nonzero) | Not treated as test failure; confirms no dynamic runtime dependency on this Linux artifact |
| `readelf -l …` inspected for `INTERP` | No interpreter segment | No external ELF loader needed |
| `env -i PATH=/nonexistent ./dist/siaas-cli_0.1.0_linux_amd64 --version --json` | PASS, exit 0, expected version JSON | No discoverable Node/Go/npm/Electron/Docker/Ollama runtime; integration test additionally copies only the executable into a fresh directory |
| Build target `freebsd/amd64` and duplicate `linux/amd64,linux/amd64` | Expected exit 1; no output directory created | Unsupported/duplicate targets refused before writes |
| PowerShell parser `Language.Parser.ParseFile` on `install.ps1` | PASS, zero parser errors | PowerShell 7.6.9 on Linux, not native Windows 5.1 acceptance |
| PowerShell script `-Help`, no args, missing `-InstallDir` value, unknown secret argument, explicit path on Linux | PASS: help exit 0; four safe rejections exit 2; sentinel absent from all output | Exercises manual parser, redaction and unsupported OS path; does not execute Windows filesystem operations |
| GitHub `GET /repos/siaas-software/siaas-cli/releases?per_page=100` via connected read tool | `[]` | No release artifacts published; repository metadata reports private. Anonymous remote installation was not attempted |
| Documentation links; `git diff --check`; changed-file review | PASS | Only `siaas-cli` M1 files; no workflows, releases, product implementation or unrelated repository edits |

The test runner's local fixtures generate checksums from the built executable;
they do not download a pre-existing release or rely on any local Personal data.
Fixture transport is explicit; no environment switch silently disables integrity
checking or adds test behavior to production code.

## Artifact evidence

Raw unsigned local outputs from the six-target build (not published release assets):

| Target | Bytes | SHA-256 |
| --- | ---: | --- |
| Linux amd64 | 2,580,604 | `1d1295d05d04bc43c5f2ca4d6a0881e656062db712125f08b4cf8e2b17fed1b0` |
| Linux arm64 | 2,556,028 | `52fc5c478afb1f6a13f67b00d5c01e5d99097030fb5d2d93f637d62f50c59712` |
| Darwin amd64 | 2,630,976 | `7fa9e34caaeb123a54d1507bea1834926cb4ad1b07375f94a2a02f11bf00acc9` |
| Darwin arm64 | 2,494,818 | `e8635ce6f6acccc502a72a74edba588c90ea2e38707c7a335b1bb3abd0bee844` |
| Windows amd64 | 2,744,832 | `fb01f18e7e4faf07655e55371ec6d4ed5796fb1d59e780ceed2de17a30e55b18` |
| Windows arm64 | 2,541,056 | `98fdb5876b136b3bc0a45f2f8359f334d498307a0fccf26644d02718a7c205de` |

Go's target-specific linker may apply platform-native binary metadata; no publisher
code-signing credentials were used. Checksums cover exactly these local bytes.
Do not reuse them after signing or rebuilding with a different toolchain/source.

## Failure and safety evidence

- Unsupported lifecycle invocations, including `install personal --version …`,
  return 3 and `not_implemented`; malformed/unknown invocations return 2. JSON and
  human stream/exit behavior are asserted, including output-write failures.
- Installer fixtures corrupt/truncate artifacts, remove metadata/assets, alter
  schema/version/name/hash and provide missing/duplicate platform entries. Existing
  CLI bytes are unchanged on rejection.
- Unix tests cover Unicode/spaces, root/relative/control/traversal paths, symlink
  parent/target/artifact, foreign data sentinels, repeat no-op and backup retention.
- A fixture PATH adapter pauses the **real installer at its final rename**.
  SIGTERM and SIGKILL target only the test-created process group. The old binary
  survives. SIGTERM cleans its own stage/lock; SIGKILL leaves a lock and retry is
  refused until the test simulates explicit operator inspection/removal in that
  temporary directory. Recovery then succeeds. No general process-name kill is used.
- A fake curl proves the unpublished-release failure path uses the fixed repository
  and HTTPS-only options without network. This is not a remote download test.
- CLI errors omit arbitrary arguments/environment data. Unix tests also inject a
  filesystem utility error containing a secret sentinel and assert that only the
  installer's fixed actionable message is emitted. PowerShell safe rejections were
  independently checked for the same omission behavior.

## Remaining acceptance and reproducibility limits

Native Windows and macOS, Linux arm64, PowerShell 5.1, Windows ACL/file-lock/replace
semantics, macOS launch policy, minimum OS baselines, signed artifacts and remote
public bootstrap require separate native/distribution validation. The Windows
test source is ready for its host, but is not marked passed. The Unix suite is
currently Linux-tagged. Checksums trusted through release metadata are not an
independent signature against publisher compromise.

No power-loss/fsync durability guarantee, malicious same-user filesystem race
protection, automatic abandoned-lock recovery, PATH mutation, auto-updater,
product lifecycle operation or release publication is included. Repeatability
was checked for Linux unsigned output on this host, not all targets on independent
builders. Existing files are backed up, not automatically pruned.

To reproduce the essential result with the pinned Go compiler, run the README's
offline tests/build, build twice into fresh output directories, compare bytes,
then run the resulting executable with a clean environment and empty PATH.
Use only temporary install directories for fixture installation tests. Keep
product data and released artifacts outside validation.
