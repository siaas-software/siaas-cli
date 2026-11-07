# Intended product manifest/API contract v1 (design only)

The CLI version and a product version are independent. Bootstrap installs the CLI
without selecting a product. A later `siaas install personal` may resolve the latest
stable compatible Personal release; `--version <version>` must resolve that exact
version or fail. There is no fallback to a different product/version and no use of
the CLI's release number as a Personal version. **No product resolver, downloader,
installer or lifecycle operation is implemented in M1.**

## Logical read API

These are proposed resource names, not deployed URLs. No API host is selected:

| Logical request | Intended response |
| --- | --- |
| `GET /v1/products` | Versioned list of supported product IDs and display names |
| `GET /v1/products/{id}/releases/latest?channel=stable` | Exact immutable version and manifest reference; excludes drafts/prereleases |
| `GET /v1/products/{id}/releases/{version}` | Immutable product manifest for the exact version, or not found |

Equivalent static release assets are acceptable; no hosted service or paid API is
required. Separate resolution from verification: mutable latest metadata selects a
version; the immutable manifest then describes artifacts. Cache immutable metadata
locally with verified identity. Offline cached behavior and freshness policy need
an explicit later decision, not a silent network fallback.

## Manifest fields

| Field | Required semantics |
| --- | --- |
| `schemaVersion` | Integer `1`; reject unsupported major schema versions |
| `product.id` | Allowlisted identifier, initially `personal`; never an executable path |
| `product.version` | Exact release version, matching the requested immutable resource/tag |
| `channel` | Explicit stable/candidate channel; no implicit promotion or downgrade |
| `minimumCliVersion` | Minimum CLI capable of interpreting the complete manifest |
| `artifacts[]` | One unambiguous artifact per OS/architecture, with canonical filename, approved immutable HTTPS URL, exact byte size and SHA-256 |
| `artifacts[].format` | Explicit raw/archive/package type; future extraction needs bounded paths/symlinks/entry count and size checks |
| `runtimeRequirements` | Product-owned runtime versions and provisioning responsibility; never require global Node/npm merely to run the CLI |
| `dataCompatibility` | Supported schema range, migration requirement, downgrade constraints and backup policy identifier |
| `lifecycleProtocol` | Versioned local authenticated readiness/control protocol and capability identifiers; no arbitrary shell command strings |
| `publisher` | Expected repository/authority and, if adopted, signature key identifier with rotation policy |

Reject duplicate targets, invalid digests/sizes, path traversal, unknown required
capabilities, mismatched identity/version and unapproved redirect destinations.
URLs must not contain credentials. An artifact checksum validates bytes but does
not replace publisher authentication. Do not execute manifest-provided scripts
merely because a checksum matches; later product adapters must define allowed
operations and validate archive/application layout.

## Future local state and operation boundaries

An installation record should separately track product/version, artifact digest,
runtime, canonical install root, data profile and previous compatible activation.
It must not store secrets in portable diagnostics. Product discovery/list/status
must be read-only; live health and installed metadata are distinct states.

Future start/stop/restart require authenticated ownership and bounded cleanup;
update/rollback require quiescence, validated snapshots and schema compatibility;
uninstall preserves user data unless a separately reviewed explicit data-removal
operation is authorized. Optional Ollama/Docker availability is a product concern,
not a CLI runtime prerequisite. These invariants guide later milestones without
reusing or modifying SIAAS Personal or its outstanding PRs during M1.
