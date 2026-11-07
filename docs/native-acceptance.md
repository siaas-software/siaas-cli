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
contents permission, checkout credential persistence disabled, 20-minute timeouts,
at most three matrix jobs concurrently, and cancellation of superseded runs.
Official checkout/setup-go actions are pinned to inspected v6 commit SHAs.

## Checks and evidence boundaries

- Pinned native Go compiler; both host OS and CPU architecture must match the
  matrix. Go formatting and `go vet ./...` must pass.
- `go test -v -count=1 -timeout=5m ./...` builds the native artifact using the real
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

Pending execution. No native job is considered passed until its completed result
and test log have been inspected. The existing release verdict remains NOT READY
until the actual matrix and remaining distribution/review gates are assessed.
