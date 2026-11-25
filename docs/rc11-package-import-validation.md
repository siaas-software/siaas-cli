# RC11 — Personal package import and validation through SIAAS CLI

**Status:** planning only, implementation and executable acceptance NOT RUN. Decision: 2026-11-25. Tracks [CLI #15](https://github.com/siaas-software/siaas-cli/issues/15) and [Personal #50](https://github.com/siaas-software/siaas-personal/issues/50). Start after the evolved Personal stable release, RC11 scope review and an agreed versioned Personal importer/lifecycle interface. This is not part of RC9 [CLI #9](https://github.com/siaas-software/siaas-cli/issues/9), does not change current CLI v0.2.1 availability, and does not authorize release or implementation now.

## Responsibility

CLI remains a standalone Go shortcut/controller. Personal owns archive validation, preview, authorization, import, canonical asset readback, persistence and execution trace. Reuse one typed first-party operation contract for UI/CLI; no mandatory MCP listener, second extractor, database writes, arbitrary installer, workflow executor or AI dependency in CLI. For external AI-generated packages, a compatible execution host can run the CLI; no dependency on a particular AI provider, GitHub account, Actions runner or hosted validation service. The host's access and execution permissions must be available and authorized. A command is not an execution environment.

## Proposed commands — not implemented

```text
siaas import personal --package <zip>
siaas validate personal --package <zip> --version <exact-personal-version> --json
```

Syntax, consent flag, capability negotiation, exit/status mapping and transport remain to finalize jointly with Personal. Retain current parser and JSON envelope compatibility; see [lifecycle reporting](m2-json-reporting.md) and [product contract](product-manifest-contract.md). An exact version cannot silently fall back to a different release. Existing lifecycle mutations are currently Linux x64 only; cross-platform CLI binaries do not establish Personal lifecycle/validation support. Unsupported target/host must fail closed, not pretend that a check or static schema result is an installation.

### Import into an existing installation

Resolve a compatible authorized installed Personal runtime; use its actual preview and scoped approval flow, including UI approval where required. Do not start or enable MCP to import. Deny wins, no overwrite/delete by default, no privilege escalation, no auto-run of Actions/Skills or arbitrary scripts. Preserve normal source lifecycle after authorized creation; no URL retrieval or connector activation during preview. Report actual completed postconditions, not just dispatch success.

### Isolated package validation before delivery

1. Resolve exact official released Personal version, OS/architecture and required capabilities using trusted first-party release policy. Verify release identity, artifact sizes and checksums, runtime and compatibility; only use trusted executable/install assets. Cache verified immutable downloads if supported, never reuse contaminated installed state.
2. With explicit authorization, install from those verified assets into a fresh, owned, isolated test environment/profile. No production install, user profile, launcher/PATH changes, unrelated processes or data. No privilege escalation or OS protection bypass. If a verified isolated install is unavailable, return BLOCKED/unsupported. Do not redirect a normal installer into an unproven root.
3. Invoke the same Personal-owned real import service and approved test-profile policy; validate archive bounds, paths, symlinks, duplicates, version/schema, ownership, conflicts, permission decisions and dependencies. Never run ZIP-bundled scripts or imported instructions as code.
4. Read back expected vs observed normalized supported assets through the authoritative Personal service: projects, assistants, templates, prompts, settings, Actions, Skills, orchestration, allowed URL references and logical MCP dependencies where supported. Use stable portable identity/remapping and verify values/associations, not just counts. Compare inventory derived by trusted Personal validation from the actual package; fail for dropped/unexpected resources or silently altered settings.
5. Stop/restart the owned runtime and verify persistence by fresh readback. Prove cleanup is restricted to the validation environment; handle timeout, cancellation, partial failure and interrupted child process lifecycle. If cleanup or mandatory postconditions cannot be verified, no PASS.
6. Return actual machine-readable evidence (separate report bound to SHA-256 of exact ZIP, no mutation of tested bytes). Package may be delivered as validated only on full PASS. Regeneration changes digest and requires retest.

No actual external URL fetching, execution of imported Actions/Skills, model inference/downloads or active MCP connections is needed for package validation; no hidden permission grants. Knowledge portability is URL references only, never RAG files/folders/local paths, contents, snapshots, chunks, embeddings, indexes or provider settings. Dependencies on local inputs or existing MCP clients remain explicit user setup requirements. Full workflow instructions may describe external tools/manual stages, but the CLI validates only supported Personal assets. Package PASS is not evidence of model quality, external service integration, end-to-end business outcome or compatibility on untested platforms.

## Proposed machine-readable evidence

Extend the established CLI JSON envelope compatibly with a versioned validation receipt only after implementation. Define exact schema jointly. Include requested/resolved Personal version, CLI version, official release identity/digest, tested OS/architecture, ZIP SHA-256, validation service version, install/import/readback/restart/persistence/cleanup results, sanitized expected/observed inventory comparison, unresolved user prerequisites, and status PASS/FAIL/BLOCKED/NOT_RUN. Distinguish a successful read-only check from an actual completed validation. No secrets, raw prompts, private filesystem paths or user data in logs or report metadata by default. A non-PASS result must not return exit zero or claim verified delivery. Preserve current error semantics where applicable, document any extension explicitly. Bound output, retries and timeout; no uncontrolled background workers.

## Work packages and gates

| Package | Scope | Evidence |
| --- | --- | --- |
| WP01 | Inspect final Personal RC11 shared service, isolation/lifecycle capabilities and supported schema; agree CLI grammar, consent, error/result contract | Reviewed exact refs and interface compatibility; no unsupported flags advertised |
| WP02 | Thin Go import/validate adapters and version selection, status/JSON and cancellation | Table-driven parser, fixture/mocked protocol, ownership, unsupported host/version, malformed result, denial and timeout negative tests |
| WP03 | Real isolated install/import/readback/restart/cleanup with representative packages and forbidden inputs | Exact-version real installation and actual receipts; UI/CLI parity; no production-profile effects; compare expected records |
| WP04 | Integration, security/native regression and release preparation | Pinned Go format, vet, unit/race tests, build, required native targets, real supported lifecycle acceptance, exact commit and review; unsupported targets explicitly blocked |

Apply resource-aware development: scoped packages, compact exact-HEAD checkpoint, fast L1 and affected L2 tests before complete mandatory L3 gate. Prefer existing local/executor runs; remote jobs only when authorized and within actual budget. No new CI workflows or paid services; do not bypass required checks. Test malicious ZIP/path/symlink/duplicate/oversize, version/hash mismatch, untrusted results, missing/ambiguous dependencies, permission denial/revocation, cancellation/restart/cleanup, repeated import/conflicts, production sentinel untouched, no external network/permission effects, and preservation of existing v0.2.1 commands. Actual native OS support requires actual test evidence, not cross-builds. Post-RC11 plugin development and publication remain separate.

No code, test execution, release, publication, dependency or workflow changes in this planning PR.
