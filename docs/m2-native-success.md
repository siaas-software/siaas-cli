# Linux native lifecycle success — 2026-11-14

The unrelated `/proc/1/exe` failure is fixed. **Real backend lifecycle PASS on
Linux x64** using trusted, locally prepared future Personal releases. This does
not certify public release availability or six-platform support. Both PRs remain Draft.

## Changed behavior

Personal now checks only persisted managed-instance leases. A new root requires
CLI ownership, the exclusive CLI lock and no retained generation; partial roots
are refused. Existing managed generations must identify the lease protocol. Known
leases bind PID, raw kernel start ticks, boot UUID and exact owned executable.
Reused/dead PIDs are stale; inaccessible or mismatched known candidates fail closed.
Unrelated processes are never enumerated or queried. No process is signalled.

Managed launch holds the operation lock through child exit, rather than releasing
it after spawn. Failure to capture a known child leaves the lock in place. A killed
supervisor leaves a stale operation lock requiring explicit inspection. This is
cooperative managed-launch protection, not adversarial process containment; raw
binary execution outside the managed launcher is not adopted. Actual Electron
GUI/descendant shutdown acceptance remains NOT RUN.

Personal uninstall-cli removes only the selected root's versions, runtimes,
build-home/cache, dispatcher, runtime-path, launcher and active metadata. Profile,
cold backups and owner record remain. Completed repeated uninstall is idempotent.
No global launcher, external Ollama or sibling root is removed. An interrupted
partial uninstall needs explicit recovery rather than guessed cleanup.

Go bootstrapping now creates npm/npx links to the verified private runtime's bundled
npm, so nested npm scripts require no global Node/npm. Uninstall no longer attempts
a semver comparison with an absent target release. Both defects were discovered
by the real integration test, not hidden with test doubles.

Updates conservatively require unchanged `src/lib/local` source bytes between
retained and verified incoming source. Changed persistence code requires separate
migration review. The engine copies the stopped profile under its operation lock
before building; the Go backend hashes the profile before/after successful update.
Links/devices and backups beyond 1 GiB fail. This is a cold file-copy policy, not
online SQLite backup, automatic restoration, schema downgrade or power-loss durability.

## Executed evidence

Go 1.27.3, Node 24.22.0, Linux x64. The same existing opt-in test constructs local
source ZIPs from the reviewed working tree, using future versions 1.1.1-rc.1/.2.
It verifies ZIP bytes and uses actual managed Node, npm ci, Next desktop build,
packaging, Personal activation, metadata and launcher creation. No builder/GUI
executable is replaced with a test double. Release acquisition is a local trusted
fixture, not a published-release/provenance acceptance claim. Synthetic archive
commit comments identify fixtures; hashes in the log bind their actual bytes.

```sh
SIAAS_M2_TRUSTED_SOURCE_TEST=/workspace/siaas-personal-m2 \
SIAAS_M2_EVIDENCE_ROOT=/workspace/m2-evidence/working \
go test -v -run '^TestFirstPartyRealLifecycle$' -timeout=20m ./internal/personal
```

[Native log, including CLI/source hashes](evidence/m2-native-success/real-test.log).
[Actual build/engine output](evidence/m2-native-success/engine-real.log).
The opt-in test passed in **93.08 seconds**. Earlier failed attempts remain in the
prior evidence directory. The first failure after correcting process scope exposed
missing npm links; the compiled-CLI attempt then exposed the uninstall semver panic.

| Scenario | Actual result / scope |
| --- | --- |
| Fresh RC installation | PASS: real Go backend → Personal engine, version and launcher asserted |
| Repeat same-version install | PASS: real backend/engine |
| RC.1 → RC.2 update | PASS: actual newer build; previous RC.1 retained |
| Downgrade | PASS: rejected |
| Injected RC.3 build failure | PASS: intentionally exits 17; current metadata and sentinel unchanged |
| Uninstall | PASS: actual compiled CLI, --rc --yes --json, exit 0 |
| Repeated uninstall | PASS: same compiled CLI, exit 0 |
| Software/runtime removal | PASS: versions/runtimes/build-home absent |
| Data, backup and sibling preservation | PASS: profile sentinel, cold-backup presence and unrelated stable sentinel asserted |
| No automatic GUI launch | Engine emits launched:false and never calls launch in install-only; no GUI acceptance claim |
| Running-instance primitive | Native owned /bin/sleep lease blocks, then becomes stale after exit; deterministic reuse/denial/mismatch tests PASS |
| Standalone CLI install/update against published compatible release | NOT RUN: no such release is published; no fallback to legacy installers |
| Windows/macOS/ARM64 native lifecycle | NOT RUN |

The real installation/update calls use the backend directly. Do not report them
as a full published-network CLI-binary install/update test. The actual compiled CLI
entry point was used for both software-removing uninstall calls. Temporary homes
and installations are removed by testing.T cleanup after assertions. External
Ollama is never signalled, started or uninstalled by these code paths; real Ollama
service/inference acceptance is not claimed.

Full Go suite, race suite and vet PASS; 28 targeted Personal tests and typecheck
PASS. Final lease/backup tests: 10/10 PASS (overlap with those 28), including native
owned-child and launch-lock checks. Persistence-change and profile-link negative
fixtures PASS. [Machine-readable evidence](evidence/m2-native-success/validation.json).
The last nonbehavioral Go struct-tag formatting correction was followed by full
Go/race/vet checks; the log retains the actual native-tested binary hash.

**Verdict: CONDITIONAL GO for reviewing the demonstrated Linux backend path.**
M2 public distribution remains incomplete until a compatible reviewed Personal
release exists and CLI install/update via its real release resolution are accepted.
GUI lifecycle/descendant acceptance, hostile same-user tampering, automatic stale
lock/partial uninstall recovery and power-loss guarantees remain outside these
passes. No merge, Actions run, OS policy change, release or tag operation occurred.

## Reproducibility maintenance after the recorded native PASS

The 93.08-second PASS in this report was executed on commit
`82d89c7eda59b160d6cabb5d7b69ddc94f01ef2e`. Later review-only changes to
`internal/personal/native_test.go` eliminate its absolute
`/workspace/rc3-preserved/assets/rc-manifests.json` dependency: the opted-in
real test now reads the pinned Node version from the trusted source fixture's
`runtime.json` and retrieves the corresponding SHA-256 from official Node
`SHASUMS256.txt` over HTTPS (same origin used by real provisioning).
`SIAAS_M2_EVIDENCE_ROOT` is optional: without it the log goes to a disposable
test directory instead of the checkout. `SIAAS_M2_TRUSTED_SOURCE_TEST` remains an
explicit absolute local path to the reviewed Personal source fixture.

These changes passed static source/manifest consistency checks performed through
the GitHub connector. **The 93.08-second native suite was NOT rerun after these
test-harness edits.** Prior native PASS must not be misrepresented as a fresh
execution of the latest commit. Re-run on an authorized capable Linux x64 host
before using the latest revision as new native acceptance evidence.

Proposed future compatible candidate: Personal `v1.1.0-rc.4` (not created or
published). Its preparation documents live in Personal PR #30. The current public
RC3 is incompatible and cannot be used as a CLI update source. A published
install/uninstall verification can follow an authorized RC4 publication; an
actual public RC-to-RC update requires a second compatible candidate.
