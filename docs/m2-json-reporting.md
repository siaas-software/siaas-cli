# M2 JSON reporting: release audit versus verified lifecycle receipt

**Status:** JSON reporting acceptance **GO** on the reviewed CLI PR #6 code (2026-11-16); merge and broader M2 acceptance remain separate.  
**Scope:** first-party SIAAS Personal lifecycle on Linux x64.  
**Reference:** [M2 acceptance issue #2](https://github.com/siaas-software/siaas-cli/issues/2)
and [Personal v1.1.0-rc.4](https://github.com/siaas-software/siaas-personal/releases/tag/v1.1.0-rc.4).

## Reason for the change

The first public M2 Linux x64 test reported all five actual CLI operations as
functional PASS: immutable-release check, installation, repeat installation,
uninstallation and repeat uninstallation. Two JSON discrepancies remained:

1. A successful `--check` reported that lifecycle execution was unsupported
   despite `personal.compatible=true`.
2. Successful installation reused the read-only source audit's original
   `runtimeProvisioned=false` even after the private runtime was installed.

The release itself is immutable. This change affects only the standalone Go CLI
and does **not** modify Personal RC4, its assets or any stable distribution.

## Output contract

The existing top-level `schemaVersion: 1`, `ok`, `command`, `version`,
`message`, `error` and `personal` fields remain. The existing
`personal` release fields remain backward compatible.

A new optional **`personal.operation`** object records a completed, verified
local lifecycle operation:

| Field | Meaning |
| --- | --- |
| `command` | `install`, `update` or `uninstall` |
| `status` | `completed` only after the engine returns successfully and local state is checked |
| `installed` | Whether an owned active application registration was verified after the operation |
| `runtimeProvisioned` | Whether the owned private Node binary was hash-verified as installed, or confirmed cleaned up after uninstall |

This object is **absent** for `--check`, `--dry-run`, unsupported operations,
audit/network failures and any operation whose postcondition cannot be verified.

For installation/update, the CLI verifies the persisted `current.json`
ownership marker, exact selected version, safe runtime path and SHA-256 of the
private Node executable before producing a receipt. The source manager's
activation checks still apply. Uninstall receipts require absence of
`current.json`, owned versions, private runtimes and owned build-home/cache.
This does **not** prove GUI startup, user-data migration or integrity of all
historical backups.

The existing `personal.runtimeProvisioned` remains false after a read-only
source audit; on verified install/update it becomes true, and after successful
uninstall it is false. Its interpretation for a read-only check is **not
provisioned by this invocation**, **not** proof that no prior installation
exists on disk.

**Illustrative JSON fragments, not executed results:**

Read-only compatible check (no mutation, no `operation`):

```json
{
  "schemaVersion": 1,
  "ok": true,
  "command": "install",
  "personal": {
    "version": "1.1.0-rc.4",
    "compatible": true,
    "sourceVerified": true,
    "runtimeProvisioned": false
  }
}
```

Successful verified installation (only after real operation and persisted
receipt verification):

```json
{
  "schemaVersion": 1,
  "ok": true,
  "command": "install",
  "personal": {
    "version": "1.1.0-rc.4",
    "compatible": true,
    "runtimeProvisioned": true,
    "operation": {
      "command": "install",
      "status": "completed",
      "installed": true,
      "runtimeProvisioned": true
    }
  }
}
```

Successful verified uninstall (no new release download or source audit):

```json
{
  "schemaVersion": 1,
  "ok": true,
  "command": "uninstall",
  "personal": {
    "runtimeProvisioned": false,
    "operation": {
      "command": "uninstall",
      "status": "completed",
      "installed": false,
      "runtimeProvisioned": false
    }
  }
}
```

The old `personal` metadata is an audit/eligibility view, **not** a complete
installation-state API. For uninstall, only the operation receipt asserts a
completed local result: no current release is independently audited during
uninstall. Do not infer an installed version from its placeholder fields.

## Error and read-only semantics

- Compatible `--check` reports that the verified published release **declares
  first-party lifecycle compatibility**; it explicitly says no installation
  occurred. An incompatible source audit reports that lifecycle support is
  unavailable despite successfully verified source integrity.
- `--dry-run` remains unsupported for execution planning (exit 3); it does not
  run installation or fabricate a receipt.
- The CLI requires an actual completed operation receipt for success (exit 0).
  Failed engine operations or unverified postconditions return exit 5, clear any
  stale receipt, and do not claim successful runtime provisioning.
- Diagnostics continue to use fixed sanitized messages. No credentials,
  private paths or untrusted request arguments are echoed.

## Test and release scope

The fix includes regression tests for compatible/incompatible checks, read-only
and dry-run paths, missing/wrong/stale receipts, installed runtime checksum
verification, symlink rejection, incomplete uninstall cleanup and repeated
uninstall.

**Validated on 2026-11-16 (exact tested CLI source commit
`af0239c9836095af3bcef6ebfdc4ad831da1dcd4`):**

- **Six-platform native CI PASS:** [GitHub workflow run #43837647182](https://github.com/siaas-software/siaas-cli/actions/runs/43837647182)
  finished SUCCESS on Linux x64/ARM64, Windows x64/ARM64 and macOS
  Intel/Apple Silicon with Go **1.27.3**. This includes `gofmt`,
  `go vet ./...`, `go test -v -count=1 -timeout=8m ./...` and
  platform-native CLI/offline-bootstrap checks.
- **Race detector PASS:** `GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off
  go test -race -count=1 ./...` returned exit **0** under Go 1.27.3
  on Linux x64. The executing Codex workspace's exact command and result
  are recorded in [PR #6 acceptance evidence](https://github.com/siaas-software/siaas-cli/pull/6#issuecomment-6913336628).
- **Public RC4 JSON lifecycle PASS:** the real CLI compiled from that same
  commit ran as UID 1000 with disposable HOME/cache against the published
  immutable Personal v1.1.0-rc.4. All five commands exited 0: read-only
  compatibility check (no receipt), initial/repeated install (completed
  receipts, installed=true, runtimeProvisioned=true), initial/repeated
  uninstall (completed removal receipts, both flags false). Private Node
  binary SHA-256 and preservation of candidate profile/backup and unrelated
  Stable/Ollama sentinels were checked. Full command JSON and evidence:
  [PR #6 acceptance](https://github.com/siaas-software/siaas-cli/pull/6#issuecomment-6913336628).

**Evidence scope:** GitHub's six CI conclusions were read back independently;
race and public-smoke results came from the executing Codex workspace and
were not rerun in the connector session. The source files underlying the
accepted CLI tests were not changed in this documentation-only follow-up.

**Limitations:** This is GO for the **JSON reporting fix** and for the
previously missing public Linux x64 install/reinstall/uninstall JSON acceptance,
**not** a completion of M2 as a whole. Public compatible RC-to-RC update
remains NOT RUN until a second compatible published Personal RC exists
(historical RC3 is incompatible). Actual Electron GUI startup, real Ollama,
first-party Personal lifecycle on Windows/macOS/ARM64, and migration/power-loss
acceptance remain distinct.

Integration into protected `main` must use the repository's PR and review
rules. No changes to immutable Personal RC4 or release protection are implied.
