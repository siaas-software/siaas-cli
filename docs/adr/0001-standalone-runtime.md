# ADR 0001 — Go standard library for the standalone CLI

Status: accepted for M1 implementation, pending review. Date: 2026-11-07.

## Context and decision

The default `main` branch (`541de99`, initial commit) contained only a one-line
README; there was no existing JavaScript codebase or repository agent policy to
preserve. The user requires an independent CLI that runs without separately
installed Node.js, npm, Electron, Docker or Ollama. A bootstrap must install the
CLI before any product is selected. Choose **Go with only the standard library**,
`CGO_ENABLED=0`, a pinned compiler and a small explicit command parser.

This choice does not reuse SIAAS Personal's Node scripts or imply that Go will
implement product internals. Future product manifests can describe a managed
runtime where a product needs one. That is separate from the CLI's own runtime.

## Options and evidence

| Approach | Practical benefits | Costs / trade-offs | M1 decision |
| --- | --- | --- | --- |
| Node.js SEA | Embeds Node, so end users need no Node installation; natural fit for an existing JS/TS codebase | SEA remains active development in the current official docs. Newer Node has built-in generation; available Node 24.19.0 exposes preparation blobs but not `--build-sea`, adding injection/tooling steps. Bundling, target runtimes and platform signatures need maintenance; Node/V8 runtime size and vulnerability updates ship with the app | Viable, not selected for this empty repository |
| Bun standalone compiler | Official standalone compilation and target selection; supports JS/TS | Also bundles a JS runtime; introduces Bun/compiler compatibility and target artifact maintenance. No local Bun size, reproducibility or native acceptance measurement was performed | Practical alternative; no benefit here that offsets another runtime |
| Go, cgo disabled | Standard parser/JSON/hash/process/test facilities; direct cross-target executables; no application dependencies or injection step | Go runtime is embedded and must be patched/rebuilt with the compiler; requires Go expertise for maintainers; native OS behavior/signing is still separate | Selected; actual build and offline test evidence below |
| Rust | Native compilation and documented platform tiers; strong compile-time checks | New toolchain and target/linker decisions; static linking differs by target/libc. No Rust implementation or local size comparison was attempted | Viable but no clear M1 advantage over dependency-free Go |
| Legacy `vercel/pkg` | Familiar Node executable packaging | Official repository is archived and package deprecated; adopting forks would require a separate maintenance review | Reject original deprecated tool |
| Runtime-plus-script archive | Straightforward for JS development | Not the requested standalone executable; more files and runtime path management | Reject for CLI distribution |

Primary sources inspected (2026-11-07):

- [Node single executable applications](https://nodejs.org/api/single-executable-applications.html): embedded runtime, current generation/signing and active-development status. The current docs describe Node 26.13.0 and record built-in `--build-sea` as added in 25.5.0. Do not project that workflow onto the installed 24.19.0 host.
- [Bun single-file executables](https://bun.sh/docs/compiler/executables): standalone compilation and cross-target selection.
- [Go build command](https://pkg.go.dev/cmd/go#hdr-Compile_packages_and_dependencies): build flags and environment controls.
- [Go installation](https://go.dev/doc/install) and [official download metadata](https://go.dev/dl/?mode=json): toolchain acquisition and published SHA-256.
- [Rust platform support](https://doc.rust-lang.org/rustc/platform-support.html): target/toolchain requirements and tier distinctions.
- [Archived vercel/pkg](https://github.com/vercel/pkg): deprecation and archival evidence.

These are capability comparisons, not benchmark claims. On this host, the selected
Linux CLI is **2,580,604 bytes**; the existing unmodified Node 24.19.0 executable is
125,989,464 bytes. The latter is a runtime-file measurement, **not a built SEA
artifact size**. Bun/Rust binaries were not produced or measured.

## Build and distribution

Pin Go 1.27.3 in `.go-version` and `go.mod`. The developer build tool verifies its
running compiler version. Disable cgo and automatic toolchain/module downloads;
use trimpath, omit VCS metadata and remove build IDs/debug tables. Do not embed
timestamps, local paths, environment credentials or Git dirty-state into output.
The same source, target and compiler produced byte-identical Linux binaries and
manifests in two output directories. Independent-builder reproducibility is not
claimed. All six intended targets cross-compiled; only Linux x64 executed.

Each artifact is a raw binary, avoiding archive traversal/extraction logic in the
bootstrap. Publish SHA-256/size metadata only after all transformations, including
signing, are complete. Retain a source commit/compiler record and unsigned hashes
for reproducibility comparisons. Signing may add timestamps and change bytes.

macOS distribution requires native codesign/Gatekeeper assessment and a decision
on Developer ID/notarization. Cross-compiled/ad-hoc-signed output alone does not
establish public distribution trust. Windows requires native testing and an
Authenticode/reputation policy; an unsigned build is not signed acceptance.
Linux release provenance can later use detached signatures/attestations. M1 does
not acquire certificates, disable protections, publish releases or introduce a
paid signing service. These are explicit pre-publication decisions, not hidden
bootstrap prerequisites.

## Security and maintenance

M1 command handlers perform no network, filesystem or subprocess operations; the
standalone integration tests prove output/exits without a discoverable runtime.
Untrusted argument values and environment content are omitted from diagnostics.
The embedded Go copyright/license is displayed by help and included in help JSON.
No additional Go module dependency or package-manager lifecycle script exists.

The bootstraps trust the fixed GitHub repository's HTTPS-served release manifest;
artifact checksums provide integrity within that authority. This is **not an
independent publisher signature** and cannot protect against compromised release
publishing access or a malicious explicitly trusted local fixture. Keep publishing
rights constrained and decide signing/key rotation before wider distribution.
No remote URL override, token prompt or certificate bypass is provided.

Compiler/standard-library vulnerabilities still require tracking official Go
security releases, reviewing the pin, rebuilding every artifact and repeating
tests. Avoid claiming that zero module dependencies means zero security maintenance.
If future requirements demand substantial JS reuse or native product bindings,
revisit this ADR with actual measurements; M1 does not pre-commit product lifecycle
implementation to a new service framework.
