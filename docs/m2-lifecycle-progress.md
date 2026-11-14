> Historical PR4 snapshot. The revised November 14 first-party model and current
> integration evidence are in [the current report](m2-working-lifecycle.md).
> Its namespace/Adapter prerequisites below are superseded for official M2 builds.

# M2 lifecycle implementation and acceptance

Verdict: **NO-GO for complete M2**. This work delivers real read-only source-release
verification and explicit safe command failures, not an operational Personal
install/update/uninstall client. No product runtime or application was installed.

## Inspected state and integration decision

- CLI main: `57a08696bd82c07192b45be21a0e83ee14217238`; PR1 merged.
- Parser reused from CLI PR3 `8695a2795e4884513e155d58ce8da6cc29366511`, still Draft.
  It is copied with provenance here, not assumed merged. Purge and conflicting
  check/dry-run/yes flags are now rejected; release-version input is bounded.
- Personal develop: `da8988061b275444f4a02c869b7b22bc4df983a7` includes PR25.
- Personal PR28 `9e1374637c3fb074537f14cc7955a684c1e5fd9e` is Draft. Its
  install-cli/update-cli reuse the engine and omit its explicit launch call, but
  npm lifecycle code remains uncontained. Its marker is not Adapter v1 conformance.
- Personal PR21 `9ea36d30c7b5adcfad0ce5c53ec5a3aee7d7c647`, PR23
  `c3567cc0b574d77a95e13b48f19fde42045cab8d`, PR24
  `6a15b7f067d0e44106763d9e90561498449e3e33` were inspected. The mandatory build
  containment, quiescence and transaction gates remain unresolved. Adapter v1
  explicitly excludes uninstall and application rollback; do not infer them.

The shortest safe available increment is the existing Go parser plus public source
verification, and an independently tested Personal engine uninstall correction.
There is no second installer, new native framework, silently weakened protocol,
legacy adoption, alternative publisher, runtime prerequisite or release modification.

## Delivered CLI behavior

`internal/productcommand` validates the existing grammar. `internal/personal`
resolves exact versions or the highest version in the explicitly chosen channel
from a bounded, paginated GitHub catalog. Drafts and wrong channels are excluded;
prerelease/tag inconsistencies and mutable releases fail. Semver comparison handles
RC numeric ordering without integer overflow. There is no downgrade fallback.

`install personal --check` and `update personal --check` verify release origin,
immutability, tag commit (including annotated tags), canonical asset URLs, GitHub
SHA-256/size metadata, SHA256SUMS, source index and all six RC manifest identities.
The source ZIP is hashed, checked for bounded entries/expansion, unsafe names,
links/duplicates and CRC corruption, and its package/runtime versions are checked.
Its git-archive comment must match the tag commit. This is **not independent Git
source-tree equivalence or a publisher signature**. HTTPS/account/release trust
remains essential; a compromised publisher is not solved by checksums.

No files are extracted or executed. Runtime metadata is checked, but runtime
archives are not downloaded/provisioned or claimed verified. Metadata is limited
to 4 MiB, sidecars to 64 KiB, source download to 150 MB, expanded ZIP to 512 MiB,
20,000 entries and 128 MiB per entry. API redirects are rejected; asset redirects
are HTTPS/443-only to GitHub or its release-assets host, at most five. Responses,
JSON depth, duplicate keys, UTF-8, unknown manifest fields and total query time
are bounded. No credential option or environment endpoint override is exposed.

The output preserves the M1 envelope and adds a `personal` report. `--check`
success means read-only verification completed; `compatible` remains false and
`runtimeProvisioned` false. `--dry-run` performs verification then exits 3.
Mutating requests fail before network, filesystem or process operations, even
with `--yes`. Invalid input exits 2; verification errors exit 4. CLI version stays
independent at 0.1.0; no release is being prepared by this change.

The resolver currently selects the latest **channel release**, not the latest
compatible executable release: no accepted execution backend exists. Future
versions using this source format need no hardcoded version change for auditing,
but dynamic adapter discovery/negotiation remains unimplemented. Neither a checksum
nor PR28's bootstrap hint can turn `compatible` true.

## Executed validation and native limits

Pinned user-local Go 1.27.3 was downloaded with official size/SHA-256 verification.
The existing `/usr/bin/go` is an unrelated executable. GOCACHE/GOPATH are under
`/workspace/m2-toolchain`; no global installation or security setting was changed.

```sh
export GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOTELEMETRY=off
go test -json -count=1 -timeout=8m ./...
go test -race -count=1 -timeout=8m ./...
go vet ./...
gofmt -l internal/cli internal/personal internal/productcommand
git diff --check
```

[Local validation and source hashes](evidence/m2-local-validation.json) record the
executed suite. Baseline suite PASS; final suite has 128 passing test/subtest events, no failures
or skips. Race/vet PASS. HTTP fixtures are **MOCK ONLY** for release selection,
malformed/duplicate JSON, manifests, digests, size bounds, redirects, cancellation,
archive corruption/unsafe names, pagination and tag failures. They are not native
installation tests. Six target builds prove compilation only.

[Live command evidence](evidence/m2-live-verification.json) records actual standalone
Linux x64 invocations with empty PATH, isolated home containing a preservation
sentinel, real GitHub HTTPS and exact exit/JSON results. Stable and RC checks PASS;
nonexistent exact version exits 4; mutating and dry-run requests correctly refuse.
The home stays unchanged and is removed after commands return. These are **safe
refusal/read-only PASS results**, not successful lifecycle operations. No Personal,
Electron, backend or external Ollama process is started or stopped.

| Required acceptance | Result | Evidence / missing prerequisite |
| --- | --- | --- |
| Fresh/repeated install, RC/exact installation | BLOCKED | No released compatible adapter or contained execution backend |
| Genuine older-to-newer update | BLOCKED | No safe quiescence/data/recovery contract or executable adapter |
| Downgrade/channel-switch prevention | BLOCKED | Mutating operations refuse universally; specific installed-state checks absent |
| Update failure preserving generation | MOCK ONLY | Existing Personal characterization; no CLI-to-real-engine acceptance |
| Uninstall preserving data / repeated uninstall | BLOCKED for CLI | Personal engine fix tested with disposable build/GUI doubles separately |
| No autolaunch / external Ollama untouched | PASS for read-only/refusal path only | No product execution exists; successful-install guarantee untested |
| JSON/human errors/exit/version | PASS | CLI unit/integration plus live commands |
| Corruption, archive paths, network/manifest failures | MOCK ONLY | Deterministic Go transport/archive fixtures |
| Install locks, interruptions, spaces/Unicode, coexistence | MOCK ONLY | Existing Personal filesystem/process characterization, not CLI native lifecycle |
| Linux x64 complete lifecycle | BLOCKED | Native source-build containment unavailable |
| Linux arm64, Windows x64/arm64, macOS x64/arm64 | NOT RUN | No native acceptance on those targets; cross-builds only |

## Concrete blockers and minimum safe next step

1. **Execution/security:** the real Linux command
   `timeout 10 unshare --user --map-root-user --mount --net --pid --fork /bin/true`
   fails with `cannot open /proc/self/uid_map: Read-only file system`.
   Cgroup hierarchy is not writable; no user session bus is available. This
   re-confirms PR24's host limitation. These facts do not prove all Linux hosts
   unsuitable. No sudo, namespace-policy modification or sandbox bypass was used.
2. **Product contract:** PR28 lacks enforced build containment, robust process
   quiescence, native path ownership, installed-integrity recovery and durable
   transactions. No safe lifecycle invocation can be inferred from no-autolaunch.
   Update needs data compatibility/backup acceptance; uninstall needs a negotiated
   ownership-safe operation beyond minimum Adapter v1.
3. **Distribution:** published stable 1.0.0 and RC1/2/3 have no accepted adapter.
   RC3 must remain unsupported. A future reviewed compatible release is necessary
   for real public install commands; publication is not authorized here.
4. **Native evidence:** all successful-install/update/uninstall cases remain
   BLOCKED/NOT RUN, regardless of parser, hash or cross-compilation results.

Minimum next step for explicit review: use an authorized disposable native Linux
host with a demonstrated private build/process/filesystem boundary, finish the
Personal backend and lifecycle capabilities reusing the extracted engine, and
verify actual temporary installation/update/uninstall before advertising support.
An alternative prebuilt distribution or reduced trust model requires an explicit
contract/security decision; it is not silently substituted in this PR.

No merge, release/tag, Actions dispatch/enablement or repository setting change.
Existing CLI PR3 and Personal PR21/23/24/25/27/28 remain unchanged. Existing CLI
Actions are not modified; the commit uses `[skip ci]` for preparation-only delivery.
