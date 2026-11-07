# M1 bootstrap security review — v0.1.0 candidate

Reviewed 2026-11-07 against foundation commit
`0e695cb36b4f130460ed39c8174e0a8688bd9eef`, with corrections in PR #1.
Scope: this repository's CLI, build tooling and bootstrap. No product lifecycle
code, other repository changes, Actions, release publication or signing services.

## Findings and corrections

| Finding | Severity | Correction and evidence | Remaining boundary |
| --- | --- | --- | --- |
| Bash followed arbitrary HTTPS redirects | High | `install.sh:get` now checks each destination before curl; exactly three GitHub hosts, HTTPS/443, five-hop limit. `tests/download_unix_test.go` checks accepted hosts and rejected host spoofing, userinfo, downgrade, alternate port, fragment, backslash, newline and missing Location | HTTP fixtures model curl's resolved `redirect_url`; no real published asset download has been tested |
| Ambient `.curlrc` could alter transport security | High | curl's first argument is `--disable`; no `--location`, credentials or certificate bypass. Adapter regression requires this first argument and refuses insecure/follow options | Trust the installed OS curl, CA store, proxy settings and local process environment |
| PowerShell allowlist accepted alternate ports and URI normalization ambiguities | Medium | `Test-TrustedReleaseUri` checks raw and parsed URI; raw Location validated before resolving. Portable parser/predicate tests include relative GitHub redirects | PowerShell 7/Linux policy tests are not Windows 5.1 transport acceptance |
| Duplicate manifest rows for other platforms were accepted | Low | Both parsers reject every duplicate target (exit 7), preserving existing CLI. Linux regression covers a duplicate non-selected target | Selected filename/version/hash/size checks and schema remain v1; all canonical existing manifests remain compatible |
| Windows path aliases / NTFS alternate streams were not explicitly rejected | Medium | Reject colon beyond drive prefix, invalid filename characters, trailing dot/space and reserved DOS device components before path normalization; Windows test cases added | Native execution UNVERIFIED; private user-owned local volume remains required |
| Oversized local Unix fixtures were copied before checking size | Low | Check local size before copying and again after staging | Local fixture directories are explicitly trusted; hostile concurrent mutation is outside scope |
| A reused build directory can retain previous artifacts on build failure | Medium, packaging | `tools/prepare-release.sh` uses fresh staging, validates all checksums and moves a completed candidate to a new destination; refuses an existing output | Low-level build helper remains a developer tool; use fresh candidate assembly for releases, not a reused `dist` directory |

Neither CLI arguments, signed redirect query strings nor native exception paths
are printed by installer errors. Download regressions check secret sentinels,
transport failures and short/partial artifacts. CLI argument and environment
redaction tests remain in the normal Go suite.

## Cross-platform review

| Concern | Evidence / behavior | Assessment |
| --- | --- | --- |
| Integrity | `install.sh:get`, selected candidate comparison; `install.ps1:Get-Artifact`, `Get-FileHash`; manifest contains SHA-256 plus exact bytes, metadata limit 64 KiB, selected artifact limit 100 MB | Verified in Linux offline tests; Windows native UNVERIFIED |
| Metadata / traversal | Fixed repository, numeric version, canonical OS/architecture/filename, lowercase 64-character SHA-256, bounded positive size; no URLs or arbitrary output paths from metadata | No product URL, shell evaluation or archive extraction |
| Locking | Unix atomic `mkdir`; Windows `CreateNew` with `FileShare.None`. Only lock owner cleans up | Linux concurrent invocation and stale-lock tests pass; no automatic stale-lock removal |
| Commit point | Same-filesystem Unix rename after private staging and retained copy; Windows `File.Replace` with backup or `File.Move` for first install | Never delete current CLI to make replacement possible; Windows filesystems and locked image behavior require native tests |
| Backup / rollback safety | Unique names, no pruning, previous CLI preserved on integrity/transport failures | Backups are CLI files, not product backups. No automated rollback command. Linux SIGTERM/SIGKILL before rename preserves old executable; Windows exclusive-handle test added but not executed |
| Path safety | Reject relative/root/dot components, symlink/reparse ancestors and target; quote paths / use literal filesystem APIs; no metadata-selected output path | Linux spaces/Unicode and symlink tests pass. Windows ACLs, junctions/reparse points and normalization need native acceptance |
| Interruption | Normal exit/INT/TERM cleans owned Unix staging and lock; hard termination leaves inspectable state. PowerShell uses `finally` for ordinary failure | No power-loss durability guarantee or automatic recovery after hard termination; no fsync protocol |
| Download boundaries | Connect/read deadlines, bounded bytes, bounded redirects, HTTPS certificate validation | Deadline is per hop (up to six requests), not a whole-install SLA; controlled failure preserves old CLI |
| End-user runtime | Pure Go executable with bundled runtime; bootstrap uses OS tools | No Node/npm/Electron/Docker/Ollama/Go runtime dependency; Linux static/no-interpreter evidence in readiness report |

## Trust and residual risks

- HTTPS/repository publisher authority is the trust anchor. A compromised publisher
  can replace both artifact and checksum; SHA-256 is integrity, not publisher
  authentication. Release assets are unsigned. Account protection, scoped release
  credentials and protected branches/tags need owner review before publication.
- Latest-release selection has no independent freshness or anti-downgrade proof.
  Bootstraps do not run downloaded executables; users explicitly run installed CLI.
- Use a private directory owned by the installing user. Shared writable paths,
  hostile same-user filesystem races, compromised OS tools/proxies/CA stores and
  administrator compromise are outside this bootstrap's boundary. Windows ACL
  ownership is not enforced by the script; default per-user location is preferred.
- Windows PowerShell 5.1 syntax is intentionally used (no PS7-only language APIs),
  but only PowerShell 7.6.9/Linux parsing and pure URI tests have executed. Native
  5.1 transport, .NET/NTFS replacement, locked/running executables, junctions and
  non-administrator execution remain UNVERIFIED.
- macOS signing/notarization/Gatekeeper and Windows Authenticode/SmartScreen
  decisions remain open. Cross-compilation establishes no native acceptance.
  No bypass instructions, signing infrastructure or paid service was introduced.
- `main` is unprotected and CLI rulesets are empty. Security/analysis switches
  were not exposed to the integration. These are release governance gaps; no
  administrative changes were made. See the settings evidence and readiness report.

The CLI interface and canonical v1 bootstrap contract are preserved. The only
new rejections are unsafe transport/path inputs and duplicate manifest targets.
No M2 functionality or `siaas install personal` implementation is included.
