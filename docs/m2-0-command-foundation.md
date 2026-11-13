# M2-0 — Personal product request grammar (inactive foundation)

Status: Draft, parser-only. Tracking: [M2 issue #2](https://github.com/siaas-software/siaas-cli/issues/2).

## Recovered contract

The intended user flow is CLI bootstrap first, then `siaas install personal`. CLI and Personal versions are independent; the CLI must not be updated just because a new Personal release exists. Product release discovery and artifact verification must be dynamic, not hardcoded. Stable is the default; `--rc` requests the latest candidate; `--version X` chooses an exact release. The exact-version example `1.1.0-rc.3` is **syntax evidence only**, not permission to invoke RC3's legacy installer.

The `internal/productcommand` package validates future request syntax for `install`, `update`, `rollback`, and `uninstall`. It rejects unknown products, invalid/duplicate options, invalid version strings, ambiguous selectors, and inappropriate `--purge` or `--check`. It never echoes untrusted values in errors. `--check` and `--dry-run` are intended to remain read-only when integrated. Options remain proposed API and may require contract review before activation.

## Deliberate non-capabilities

- The package is **not imported by the existing M1 CLI entry point**. The existing M1 command behavior and exit 3 for reserved product commands remain unchanged.
- No HTTP calls, mutable release resolution, manifest/download verification, install, update, uninstall, rollback, live status, launcher, database, Electron, Ollama, profile or filesystem operations are introduced by this package.
- No support is claimed for Personal RC3 via CLI: published RC3 has no verified install-only Adapter v1. Do not call the legacy `source-manager.mjs` as a fallback or fabricate an Adapter receipt.
- The normative [Personal Adapter design PR #23](https://github.com/siaas-software/siaas-personal/pull/23) remains unimplemented; its seven operations and native containment/recovery blockers must be resolved or explicitly re-reviewed before any platform is advertised.
- Application rollback, uninstall and purge are **not capabilities of Adapter v1**; they need separately reviewed contracts.
- No workflow, release, tag, dependency, product installation or existing PR changes.

## Verification

The new parser and two table-driven test functions were exercised in a disposable source-only local module with Go 1.23.2, standard library only: `go test -count=1 ./...` and `go vet ./...` PASS. This is a syntax/unit check, **not** the repository's pinned Go 1.27.3 validation or six-platform acceptance.

Before merging, run the full repository checks on the pinned Go toolchain, including `go test -count=1 -timeout=8m ./...`, `go vet ./...`, `gofmt -l`, M1 bootstrap regressions and the existing native acceptance where available. No native PASS is inferred here.

## Next independent increments

1. Review parser grammar and error/output/exit compatibility with the current M1 `--json` envelope.
2. Implement a **read-only, trusted** product release resolver and immutable manifest/adapter descriptor verifier (no product execution).
3. Implement Personal's verified install-only Adapter separately and only then wire CLI orchestration, with fail-closed capabilities and documented platform gates.

The goal is to progress without a second installation engine and without repeating disproportionate native-test infrastructure work.
