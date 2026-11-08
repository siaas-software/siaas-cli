# M1 native acceptance on GitHub Actions

The owner explicitly authorized this workflow after the initial M1 readiness
review. Earlier reports saying Actions were not introduced describe those earlier
checks. This workflow does not merge, create tags, publish releases or change
repository settings.

## Execution contract

`.github/workflows/native-acceptance.yml` runs on PRs to `main`, including drafts,
and declares `workflow_dispatch`. GitHub requires the workflow on the default
branch before manual dispatch is available; PR #1 is exercised through its
`pull_request` event without merging it to enable dispatch.

Only the requested standard GitHub-hosted labels are used:

| Runner | Native target |
| --- | --- |
| `ubuntu-24.04` | linux/amd64 |
| `ubuntu-24.04-arm` | linux/arm64 |
| `windows-2022` | windows/amd64 |
| `windows-11-arm` | windows/arm64 |
| `macos-15-intel` | darwin/amd64 |
| `macos-15` | darwin/arm64 |

[GitHub's runner documentation](https://docs.github.com/en/actions/reference/runners/github-hosted-runners#standard-github-hosted-runners-for-public-repositories)
lists all six as standard runners, free for public repositories. A job-level
condition refuses a private repository or a repository other than
`siaas-software/siaas-cli`. There are no larger-runner labels, self-hosted runner
groups, artifact uploads, cache actions, schedules, external signing services or
paid integrations. Go setup caching is explicitly disabled. Jobs have read-only
contents permission, checkout credential persistence disabled, 30-minute timeouts (20 in the original recorded run),
at most three matrix jobs concurrently, and cancellation of superseded runs.
Official checkout/setup-go actions are pinned to inspected v6 commit SHAs.

## Checks and evidence boundaries

- Pinned native Go compiler; both host OS and CPU architecture must match the
  matrix. Go formatting and `go vet ./...` must pass.
- `go test -v -count=1 -timeout=8m ./...` (5m in the original run) builds the native artifact using the real
  release build helper, verifies its canonical manifest, SHA-256 and exact size,
  then executes CLI/help/version/error/JSON and bootstrap tests.
- CLI execution with an empty runtime search path proves no separately installed
  Go, Node, npm, Electron, Docker or Ollama is needed by the executable.
- Linux and macOS execute the Unix fixture suite, including integrity failures,
  missing assets, unsafe paths, Unicode/spaces, replacement/backup retention,
  idempotency, concurrent exclusion, signals and recovery. Only macOS test temp
  directories are canonicalized to avoid the OS's `/var` symlink; production
  symlink rejection is unchanged. Curl adapters never contact a release server.
- Windows uses `powershell.exe` and explicitly asserts **5.1 Desktop**, both in a
  workflow step and a Go test. It tests actual installed CLI execution, spaces and
  Unicode, same-size corruption, truncation, oversize, repeat installation, safe
  replacement, retained backups, exclusive file locks, an actual running native
  executable, stale locks, ambiguous paths and redaction.
- A running Windows image may either be safely replaced with a retained backup
  or fail safely with exit 6, depending on filesystem semantics. Tests record the
  result, verify old bytes on failure, stop the fixture process, retry, and verify
  both the new CLI and the retained old executable. Exclusive file-handle tests
  require failure with old bytes and prior backups intact.

Bootstrap tests use only disposable local fixtures. GitHub checkout and toolchain
setup use network access; the acceptance tests do not install from GitHub Releases.
Native acceptance does not establish signed-distribution, SmartScreen/Gatekeeper
or remote-download acceptance. Hosted Windows runners may have administrator
rights; tests request no elevation, but do not prove standard-user Windows ACL
behavior. Previous Linux tests ran as UID 1000. Full host/compiler/PowerShell
details and test names are retained in workflow logs, not uploaded artifacts.

## Release tag protection

Read-only API verification on 2026-11-07:
[`protect-release-tags`, ID 26770131](https://api.github.com/repos/siaas-software/siaas-cli/rulesets/26770131)
is active, targets `refs/tags/v*` with no exclusions, and now contains `deletion`,
`non_fast_forward` **and `update`**. `bypass_actors` is empty and
`current_user_can_bypass` is `never`. This resolves the previously reported tag
update restriction gap; no setting was modified by this work.

## Recorded run

**READY for the requested M1 native offline acceptance.** All six completed jobs
and their test logs were inspected in
[run 42411030838](https://github.com/siaas-software/siaas-cli/actions/runs/42411030838),
triggered by PR #1 at tested commit
`b3d6abf328c0f41cdbcee44a2d3db76f0782d72d`. Run conclusion: `success`.
The subsequent documentation-only commit records these results without rerunning
unchanged tests. It does not extend the tested scope.

| Runner / target | Job | Test/subtest passes | Result |
| --- | --- | --- | --- |
| ubuntu-24.04 / linux/amd64 | [128519002046](https://github.com/siaas-software/siaas-cli/actions/runs/42411030838/job/128519002046) | 72 | PASS |
| ubuntu-24.04-arm / linux/arm64 | [128519001852](https://github.com/siaas-software/siaas-cli/actions/runs/42411030838/job/128519001852) | 72 | PASS |
| windows-2022 / windows/amd64 | [128519002189](https://github.com/siaas-software/siaas-cli/actions/runs/42411030838/job/128519002189) | 45 | PASS |
| windows-11-arm / windows/arm64 | [128519001918](https://github.com/siaas-software/siaas-cli/actions/runs/42411030838/job/128519001918) | 45 | PASS |
| macos-15-intel / darwin/amd64 | [128519001937](https://github.com/siaas-software/siaas-cli/actions/runs/42411030838/job/128519001937) | 72 | PASS |
| macos-15 / darwin/arm64 | [128519001969](https://github.com/siaas-software/siaas-cli/actions/runs/42411030838/job/128519001969) | 72 | PASS |

Total: **378 passing test/subtest events**, no failing or skipped test cases.
Platform-inapplicable workflow steps are skipped by design (for example, the
PowerShell step on Linux); they are not counted as tests. The API artifact list
is empty. All six native release SHA-256 values and sizes match the earlier Linux
cross-build inventory in `docs/evidence/v0.1.0-artifacts.json`.

Windows evidence from the actual Go test subprocesses:

```text
windows-2022:   Windows PowerShell 5.1.20348.5773 Desktop
windows-11-arm: Windows PowerShell 5.1.26100.9711 Desktop
```

Both Windows jobs passed `TestWindowsLockedTargetPreservesBackup`,
`TestWindowsRunningExecutableReplacement` and `TestWindowsBootstrap`. Replacement
while the fixture executable was running returned **0** on both runners, retaining
the previous executable as a backup; after process exit, retry was safe. Exclusive
file handles caused the expected **6**, leaving existing bytes and prior backups
intact. Space/Unicode destinations, repeat installation, actual installed binary
execution with an empty PATH, and same-size checksum corruption all passed.

The initial [run 42410808788](https://github.com/siaas-software/siaas-cli/actions/runs/42410808788)
at `805d239a75ddd7c7e5fb2adacb333bcbf9e106cc` passed both Linux jobs and failed both
Windows and both macOS jobs. The Windows toolchain/formatting gate failed; the
correction enforces LF checkout endings through `.gitattributes`, normalizes
command-output line endings and prints explicit failure diagnostics. On macOS,
the fake curl compared padded BSD `wc` output as a string, and a traversal fixture
encountered the `/tmp` symlink before the intended dot-component check. Numeric
comparison and a canonical temporary traversal path corrected the fixtures.
The successful run retains the same security assertions. No installer or CLI
runtime code was changed to make these tests pass.

Local checks also passed: `go test -count=1 -timeout=5m ./...`,
`go test -race -count=1 -timeout=5m ./...`,
`go vet ./...`, Windows ARM64/macOS ARM64 test compilation, workflow YAML
validation, `git diff --check` and documentation link checks. Remote native
results are not inferred from those local checks.

API results and selected log evidence are committed in
[`native-acceptance-2026-11-07.json`](evidence/native-acceptance-2026-11-07.json).

## Remaining release boundaries

There is no failing job or Actions-policy blocker for this native acceptance
scope. `protect-main` is active and `protect-release-tags` includes `update`;
those earlier protection gaps are resolved. No administrative setting was changed
by this work.

**Public v0.1.0 publication remains NOT READY for authorization:** independent
release/security review and an explicit unsigned-distribution/support decision
remain open. Standard-user Windows ACL/reparse-point behavior, downloaded-file
SmartScreen/Gatekeeper behavior and older OS versions are not established by
these hosted tests. Administrative security settings outside the readable
rulesets still need owner inspection. A real anonymous remote bootstrap is a
post-publication verification gate, not a claimed pass. Signing is not provided
or assumed to require paid infrastructure. PR #1 stays draft; merge, release-tag
creation and publication require separate explicit approval.

## Public-security follow-up

The historical results above remain unchanged. The 2026-11-08 follow-up adds
origin metadata preservation, default paths, Windows junction/ACL recovery tests,
a disposable genuine non-administrator account with a loaded user profile, and
read-only OS signing/policy probes. Windows standard-user tests have a separate
10-minute test timeout; jobs remain bounded at 30 minutes. The workflow does not
weaken execution policy or OS protection. See
[public-release-security-review.md](public-release-security-review.md) for the
new exact commits, failed attempts, completed results and release decision.
