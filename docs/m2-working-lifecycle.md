# First-party M2 integration attempt — 2026-11-14

**NO-GO: real install/update/uninstall acceptance is BLOCKED.** This branch builds
on PR4 (546e5632f2bc539e390b5e1a0f1388dcd50fbf42) without changing that PR.
Personal integration builds on PR29 and reuses the manager change from PR28.
Issue #2's approved November 14 first-party model supersedes mandatory namespace,
cgroup and hardened Adapter v1 prerequisites for this path. No sandbox bypass is
introduced or required. Official scripts execute as the ordinary user; they are
trusted with that user's filesystem/network access. This is not arbitrary-code support.

## Code delivered for review

- Source audit recognizes an exact, source-integrity-covered first-party marker at
  scripts/lib/cli-first-party.json; legacy published releases remain incompatible.
  The proposed target is Linux x64 only. The marker is an interface declaration,
  not native acceptance or release authorization.
- Go provisions a private runtime from canonical nodejs.org archives, checks official
  checksums and release runtime bindings, uses safe ZIP/tar paths and argument arrays,
  then calls the actual Personal manager's install-cli/update-cli/uninstall-cli.
  No duplicate application build/activation engine is introduced.
- Separate stable/candidate managed roots, ownership record, no legacy adoption,
  operation lock, version/channel checks and explicit --yes gate. Uninstall --rc
  selects the candidate root. The engine preserves profiles and previous generations.
- Child environment excludes ambient application/release credentials; only explicit
  proxy/CA configuration is forwarded. Build diagnostics are bounded to 10 MiB and
  retained under the owned root. Go and Personal locks are conservative; no automatic
  stale-lock cleanup or destructive recovery is implemented.
- Managed Personal launch acquires its existing operation lock. Mutations scan the
  same user's native processes under that lock and reject a running owned executable
  or profile. No process or external Ollama is signalled.

## Actual attempted acceptance

Pinned Go 1.27.3 and Node 24.22.0. On Linux x64, the opt-in test constructs trusted
local future-version source ZIPs from the reviewed Personal working tree, changes
package/lock versions to 1.1.1-rc.1/.2, validates the ZIP, and invokes Execute with
actual runtime provisioning and actual Personal build scripts. No npm/builder/GUI
mock is installed. Local provenance/commit comment are test fixture identities,
not a published GitHub release; production URLs cannot be overridden through CLI.

```sh
SIAAS_M2_TRUSTED_SOURCE_TEST=/workspace/siaas-personal-m2 \
SIAAS_M2_EVIDENCE_ROOT=/workspace/m2-evidence/working \
go test -v -run '^TestFirstPartyRealLifecycle$' -timeout=20m ./internal/personal
```

Observed: runtime provisioning completed and Personal manager was invoked. The
manager refused before build/activation with `EACCES: permission denied, readlink
'/proc/1/exe'`. PID1 is owned by UID1000; cmdline is readable as `tail -f /dev/null`,
but the executable identity is not. An argv string is not trusted executable identity.
No exception for PID1 or permission downgrade was added. Test temp root is removed
by testing.T cleanup; no application was installed/launched or real profile touched.

[Actual test log](evidence/m2-working/real-test.log) and [engine diagnostic](evidence/m2-working/engine-real.log).
This is a BLOCKED native test represented by a failing Go assertion, not a PASS.
The test also specifies repeat installation, genuine newer version, retained previous,
downgrade refusal, unregister/repeated unregister and data sentinel assertions;
those later steps are **NOT RUN**, since installation failed first. It calls the real
Go lifecycle backend; it is not a successful standalone CLI binary acceptance.

## Validation and gaps

`go test -count=1 -timeout=3m ./...` PASS (native opt-in test skipped by default),
`go vet ./...` PASS, and final internal tests PASS after runtime-hash propagation fix.
[Full suite log](evidence/m2-working/go-tests.log), [final internal suite](evidence/m2-working/go-internal-final.log).
Personal targeted 20/20 tests and typecheck PASS. The actual Next desktop build PASS;
packaging required a writable cache and Node's explicit proxy support. These are
build checks, not lifecycle acceptance. Earlier PR4/29 evidence remains historical.

| Operation / target | Result |
| --- | --- |
| Real Linux x64 install | BLOCKED by incomplete process visibility |
| Real repeat/update/uninstall/data preservation | NOT RUN after install refusal |
| Standalone binary successful lifecycle | NOT RUN |
| Windows/macOS/all ARM64 native lifecycle | NOT RUN; unavailable interface targets |
| Deterministic Go tests / existing Personal regressions | PASS, narrow scope only |

Remaining safety/acceptance gaps before support: demonstrate native quiescence,
launcher/process race handling, installed-tree/ownership integrity, injected failure
recovery, profile cold-backup/data-compatibility policy and guarded updates under
interruption. Current CommandContext cancellation is direct-child only; it does not
claim complete descendant cleanup. No forced cleanup should be added to hide this.
Build/data/metadata/receipt recovery is not power-loss durable. The production CLI
still requires future published first-party capabilities; no current legacy release
is silently executed. Linux x64 is a proposed target, not signed-off support.

The exact remaining host prerequisite is an ordinary-user Linux runner on which
same-user executable identity can be inspected (or an independently reviewed
alternative instance-ownership mechanism). No namespace/cgroup requirement is
reintroduced. No other authorized native runner is configured in this workspace;
Actions dispatch/enablement, merge and publication were not performed.
