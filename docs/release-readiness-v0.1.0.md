# M1 release readiness — v0.1.0

**Verdict: NOT READY for the proposed public cross-platform release.** Local Linux
acceptance and candidate assembly pass. Repository governance and native platform
acceptance remain open. No release, tag, merge, Actions workflow, administrative
setting change, paid service or product installation was performed.

Review date: 2026-11-07. Repository: `siaas-software/siaas-cli`; default/base branch:
`main`. Work remains on `feature/foundation-standalone-bootstrap` in
[draft PR #1](https://github.com/siaas-software/siaas-cli/pull/1). Starting source:
`0e695cb36b4f130460ed39c8174e0a8688bd9eef`. The review did not change CLI handlers,
version or the product manifest proposal. `siaas install personal` still returns
exit 3 and performs no operation. No repository instructions file was present.

## Implemented corrections

- Bash checks every redirect before connecting, permits only HTTPS/443 and three
  GitHub release hosts, limits redirects to five, ignores `.curlrc`, and retains
  certificate checks, download bounds and final SHA-256/size verification.
- PowerShell applies the same URI policy, rejects unsafe raw Location values and
  Windows path aliases/alternate data streams. No PS7-only syntax was added.
- Both installers reject duplicate targets anywhere in a manifest. Unix checks
  local fixture size before copying. Existing CLI bytes survive tested failures.
- Offline regression coverage now includes malicious redirects, download/partial
  failures, all allowed hosts, concurrent installs and additional metadata bounds.
  Native Windows locked-target/backup and ambiguous-path tests were added and
  cross-compiled, **not executed**.
- A local candidate assembly helper stages a fresh complete asset set, checks
  hashes, refuses existing output and performs no publication. Documentation now
  records public visibility, exact commands, publisher trust and native limits.

See [security review](security-review.md) for source/test references and residual
risks, including unsigned artifacts, publisher compromise, no freshness proof,
private-directory assumptions and lack of power-loss durability guarantees.

## Repository visibility and protections

Read-only GitHub evidence is preserved in
[`repository-settings-2026-11-07.json`](evidence/repository-settings-2026-11-07.json).

| Setting | SIAAS CLI | SIAAS Personal (read-only comparison) |
| --- | --- | --- |
| Visibility / default branch | Public / `main` | Public / `main` |
| Branch API `protected` | `main: false` | `main: true`, `develop: true` |
| Rulesets including parent rules | Empty list | Two active repository rulesets |
| Branch safeguards | No observed PR/deletion/force-push rules | `main` and `develop`: deletion and non-fast-forward blocked; PR and resolved review threads required |
| Review approvals | No observed requirement | Required approving review count 0; no stale dismissal, code-owner or last-push approval requirement |
| Tag safeguards | No rulesets | `v*`: deletion, non-fast-forward and update blocked |
| Required checks | Branch summary reports none | Branch summary reports none; ruleset contains no required-status-check rule |
| Auto-merge / merge methods | Auto-merge off; merge/squash/rebase allowed | Same |
| Security/analysis settings | UNVERIFIED: omitted from repository response | UNVERIFIED: omitted from repository response |

Classic branch-protection endpoint reads returned **403, Resource not accessible
by integration** for both repositories. That failure alone is not evidence of
absent protection; the CLI conclusion uses `protected: false` plus the empty
ruleset list. Vulnerability-alert and Actions-permissions endpoint requests were
rejected by the connector's URL policy (400). Secret scanning, push protection,
Dependabot alerts/security updates, private vulnerability reporting, code-scanning
configuration and Actions permissions therefore remain **UNVERIFIED**, not disabled.
No workflow files exist in this change; no Actions were enabled.

The owner should resolve CLI branch/tag protection before release: require PRs
and resolved review threads, prevent deletion/force pushes on `main`, protect
release tags from update/deletion, and decide whether at least one independent
approval is required. Inspect the security settings with administrative access.
Do not add mandatory CI checks that do not exist or activate Actions implicitly.
No administrative change was attempted.

Anonymous network checks (no Authorization header, token, netrc or curl config):

```sh
curl --disable --silent --show-error --proto '=https' --output /dev/null --write-out 'Repository page HTTP %{http_code}\n' https://github.com/siaas-software/siaas-cli
curl --disable --silent --show-error --fail --proto '=https' --output /dev/null --write-out 'Latest manifest HTTP %{http_code}\n' https://github.com/siaas-software/siaas-cli/releases/latest/download/cli-release-v1.txt
```

Results: repository page **200 / exit 0**; latest manifest **404 / exit 22**.
Authenticated read-only GitHub release listing returned `[]`. An anonymous API
probe using `curl --disable --silent --show-error --proto '=https' --write-out
'\nHTTP %{http_code}\n' https://api.github.com/repos/siaas-software/siaas-cli`
failed at the environment proxy's CONNECT layer (403, curl 56, HTTP 000); an earlier
`--fail` probe returned curl 22/403. This does not contradict public web visibility.
No actual release-asset delivery can be certified before assets are published.

## Executed validation

Host: Debian 13.6, Linux 6.18.57, x86_64; **UID 1000, no administrator elevation**
for builds/tests/install fixtures. Go 1.27.3 at
`/tmp/siaas-cli-toolchain/go/bin/go`; supplemental PowerShell 7.6.9/Linux at
`/tmp/siaas-cli-powershell/pwsh`. Compiler matches `.go-version`.

From repository root, the Go commands used:

```sh
export PATH=/tmp/siaas-cli-toolchain/go/bin:$PATH
export GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOTELEMETRY=off
export GOCACHE=/tmp/siaas-cli-go-cache
```

| Actual command | Exit | Result / limitation |
| --- | --- | --- |
| `go test -count=1 -timeout=90s -json ./...` | 0 | 13 top-level tests, 71 total test/subtest passes, no failures or skipped test cases; two packages have no tests |
| `go test -race -count=1 -timeout=90s -json ./...` | 0 | Same 71 passes; no race report |
| `go vet ./...` | 0 | No diagnostics |
| `test -z "$(gofmt -l cmd internal tools tests)"` | 0 | All Go files formatted |
| `bash -n install.sh tools/prepare-release.sh` | 0 | Bash syntax passes |
| `git diff --check` | 0 | No whitespace errors |
| `go run ./tools/build -targets linux/amd64,linux/arm64,darwin/amd64,darwin/arm64,windows/amd64,windows/arm64 -out dist/readiness-cross` | 0 | Six targets built |
| `GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go test -c -o dist/readiness-windows-tests.exe ./tests` | 0 | Windows tests compile only; no native execution |
| `bash tools/prepare-release.sh dist/v0.1.0` | 0 | Twelve local assets; eleven checksum verifications passed |
| `bash tools/prepare-release.sh dist/readiness-repro` | 0 | Second complete six-target candidate |
| `diff -rq dist/v0.1.0 dist/readiness-repro` | 0 | Every binary, script, notice, notes and metadata file byte-identical |
| `bash tools/prepare-release.sh dist/v0.1.0` (repeat) | 2, expected | Refuses existing candidate directory; preserves prepared files |
| `env -i PATH=/nonexistent HOME=/tmp ./dist/v0.1.0/siaas-cli_0.1.0_linux_amd64 --version --json` | 0 | Valid version JSON; no external runtime available via PATH |
| `file dist/v0.1.0/siaas-cli_0.1.0_linux_amd64` | 0 | Statically linked, stripped x86-64 ELF |
| `readelf -l dist/v0.1.0/siaas-cli_0.1.0_linux_amd64 \| rg INTERP` | 1, expected | No dynamic interpreter entry |
| `ldd dist/v0.1.0/siaas-cli_0.1.0_linux_amd64` | 1, expected | `not a dynamic executable` |
| `/tmp/siaas-cli-powershell/pwsh -NoProfile -NonInteractive -File tests/powershell-policy.ps1` | 0 | Syntax plus 4 accepted/10 rejected URI cases and a relative GitHub redirect; pure policy only |
| `/tmp/siaas-cli-powershell/pwsh -NoProfile -NonInteractive -File install.ps1 -Help` | 0 | Help on portable PowerShell |
| `/tmp/siaas-cli-powershell/pwsh -NoProfile -NonInteractive -File install.ps1` | 2, expected | Safe rejection of non-Windows host |

Detailed pass names and build/checksum output:
[`release-readiness-tests.txt`](evidence/release-readiness-tests.txt). Raw local
JSON logs are in ignored `test-results/unit.jsonl` and `test-results/race.jsonl`.
These tests use built binaries, local release fixtures and a fake curl, with no
external service or network requirement. They test real Bash staging/replacement,
integrity failures, secret redaction, Unicode/spaces, symlinks, idempotency,
backups, concurrent exclusion, SIGTERM/SIGKILL and inspected stale-lock recovery.
The additional Windows tests are excluded by build tags on Linux, not counted as
passing or skipped native tests. ShellCheck was unavailable and was not run.

Repeatability is demonstrated for unsigned artifacts from the same compiler,
source and host in separate output directories; it is not independent-builder
reproducibility, native acceptance, or reproducibility after platform signing.

## Native platform matrix

| Target | Cross/native build | Native CLI | Native bootstrap | Release status |
| --- | --- | --- | --- | --- |
| Linux amd64 | PASS | PASS, no external runtime | PASS offline, unprivileged Debian 13.6 | Validated only on this host; remote bootstrap pending |
| Linux arm64 | PASS | UNVERIFIED | UNVERIFIED | Candidate only |
| macOS amd64 | PASS | UNVERIFIED | UNVERIFIED | Candidate only |
| macOS arm64 | PASS | UNVERIFIED | UNVERIFIED | Candidate only |
| Windows amd64 | PASS | UNVERIFIED | UNVERIFIED, including PowerShell 5.1 | Candidate only |
| Windows arm64 | PASS | UNVERIFIED | UNVERIFIED, including PowerShell 5.1 | Candidate only |

No native Windows/macOS/ARM64 environment was available. Their minimum supported
OS versions, filesystem behavior, CPU baseline and distribution policy must be
established before claiming support. Native acceptance must include:

1. Run `go test -count=1 -timeout=90s ./...`, `go vet ./...` and available race
   tests on the native host. On Windows record `$PSVersionTable` from
   `powershell.exe`, confirming 5.1, and run under an ordinary user account.
2. Run the exact distributed binary from a machine/account without Go/Node/npm/
   Electron/Docker/Ollama. Check help/version, JSON, unknown exit 2 and reserved
   `install personal` exit 3. Cross-built executable format is insufficient.
3. Install from local fixtures into a private Unicode/space path; repeat; replace
   older bytes; verify old and pre-existing backups and unrelated data survive.
   Test corruption, truncation, missing assets, concurrent installers, interruption
   and explicit stale-lock recovery. Linux suite currently has a Linux build tag;
   macOS must receive/run equivalent native fixture tests, not count Linux tests.
4. On Windows run the new exclusive-file-handle test, plus actual running-image,
   read-only/ACL and junction/reparse-point scenarios. Confirm replacement fails
   without deleting current CLI/backups, then retry after unlocking. Exercise
   manual restoration with verified CLI backups in a disposable directory.
5. Assess macOS quarantine/Gatekeeper and Windows execution-policy/SmartScreen
   behavior without bypasses. Decide unsigned-distribution limitations or an
   approved signing process; no paid signing infrastructure is assumed.

## Candidate artifacts and contract verification

Prepared locally in ignored `dist/v0.1.0/`; no binaries are committed or uploaded.
Complete inventory (twelve assets):

```text
siaas-cli_0.1.0_linux_amd64
siaas-cli_0.1.0_linux_arm64
siaas-cli_0.1.0_darwin_amd64
siaas-cli_0.1.0_darwin_arm64
siaas-cli_0.1.0_windows_amd64.exe
siaas-cli_0.1.0_windows_arm64.exe
SHA256SUMS
cli-release-v1.txt
install.sh
install.ps1
RELEASE_NOTES.md
GO-LICENSE.txt
```

Every asset's exact size and SHA-256 is recorded in
[`v0.1.0-artifacts.json`](evidence/v0.1.0-artifacts.json). Six binary hashes are
unchanged from the M1 foundation validation because CLI runtime source is unchanged.
`SHA256SUMS` covers all eleven other files; it cannot hash itself. Bootstrap
consumes only the version/target/name/hash/size manifest, whose six entries agree
with `SHA256SUMS` and actual bytes. The copied installers equal repository source.

The following equivalent assertion check reproduces the executed manifest/name/
checksum/path verification (Python 3 was used as an audit tool, not a CLI or
bootstrap prerequisite):

```sh
python3 - <<'PY'
from pathlib import Path
import hashlib
p = Path('dist/v0.1.0')
lines = (p / 'cli-release-v1.txt').read_text().splitlines()
assert lines[0] == 'siaas-cli-release-v1 0.1.0'
sums = {line.split()[1]: line.split()[0]
        for line in (p / 'SHA256SUMS').read_text().splitlines()}
seen = set()
for line in lines[1:]:
    os, arch, name, digest, size = line.split(' ')
    assert (os, arch) not in seen
    seen.add((os, arch))
    assert name == f'siaas-cli_0.1.0_{os}_{arch}' + ('.exe' if os == 'windows' else '')
    data = (p / name).read_bytes()
    assert len(data) == int(size)
    assert hashlib.sha256(data).hexdigest() == digest == sums[name]
assert seen == {(os, arch) for os in ('linux', 'darwin', 'windows')
                for arch in ('amd64', 'arm64')}
assert set(sums) == {f.name for f in p.iterdir()} - {'SHA256SUMS'}
for name, digest in sums.items():
    assert hashlib.sha256((p / name).read_bytes()).hexdigest() == digest
for name in ('install.sh', 'install.ps1'):
    assert (p / name).read_bytes() == Path(name).read_bytes()
    text = Path(name).read_text()
    assert 'https://github.com/siaas-software/siaas-cli/releases' in text
    assert '/latest/download/cli-release-v1.txt' in text
    assert '/download/v$version/$artifact' in text
print('PASS: candidate bytes, names, manifest, checksums and bootstrap paths')
PY
```

Public visibility and credential-free bootstrap requests satisfy the repository
access precondition. After separate publication approval, upload all assets to
stable `v0.1.0`, publish it as latest, and verify every manifest/versioned asset
URL anonymously, including legitimate GitHub asset redirects. Draft/prerelease
visibility and missing assets must never be treated as successful installation.
The current 404 is expected, not an executed remote acceptance test.

## Explicit release blockers

1. Owner review/resolution of unprotected `main` and absent release-tag rules;
   administrative verification of security settings inaccessible to this review.
2. Native Windows 5.1, macOS and ARM64 CLI/bootstrap acceptance and OS/distribution
   decisions for those proposed targets. Alternatively, the owner must explicitly
   approve a narrower release scope before claiming any such release is ready.
3. Final independent technical/security review and explicit authorization to
   merge, tag or publish. PR #1 remains draft; none of those actions is authorized
   by this preparation task.

Anonymous remote install is a required **post-publication verification gate**,
not a claimed pre-publication pass. Until the blockers are resolved, artifacts are
reviewable local candidates only. No M2 or SIAAS Personal work was started.
