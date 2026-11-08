# M1 public release security review — 2026-11-08

**Verdict: READY WITH EXPLICIT RISK ACCEPTANCE for an unsigned M1 release.**
All six native offline jobs, both genuine Windows standard-user suites and the
OS-policy probes passed. This is not a claim of universal downloaded-file
acceptance: the release owner has not yet accepted the distribution risks below,
and merge, tagging and publication remain unauthorized. No release was published.

Scope: `siaas-software/siaas-cli`, draft PR #1, v0.1.0 candidate. Baseline
`6e3afa3cde3f1057da184dc770427d2e4b2b1156`. No product lifecycle implementation,
other repository writes, administrative setting changes, merge, tag, release,
paid runner, signing service or artifact upload is part of this review.

The CLI remains standalone: end users need no Node.js, npm, Go, Electron, Docker
or Ollama. The bootstrap installs only the CLI. `siaas install personal [--version
VERSION]` remains reserved, returns 3, and performs no installation. Help/version
and JSON behavior and exit codes are unchanged.

## Findings and corrections

| Finding | Severity / disposition | Evidence and correction |
| --- | --- | --- |
| Raw HTTP writes did not attach Internet-origin security metadata | Medium; corrected | `install.ps1:Set-InternetZone` now writes NTFS ZoneId 3 before remote candidate commit; `install.sh` adds macOS quarantine after hash/size verification. Failure aborts without replacing old CLI. Windows uses PowerShell's stream provider for .NET Framework/5.1 compatibility. |
| Windows path inspection depended on target-following Exists checks | Low; defense in depth corrected | `Assert-NoReparsePoint` uses GetAttributes on every ancestor and executable. Native regression covers ordinary and dangling junctions and confirms the missing destination is not recreated. No successful pre-fix exploit was demonstrated. |
| Unsigned distribution may be refused by OS policy | High distribution limitation; unresolved support decision | Windows policy probe marks the real bootstrap Internet Zone 3 and checks refusal under RemoteSigned or a stricter existing policy. macOS `spctl --assess` rejects the local candidate. No bypass or publisher-signing claim. |
| Publisher controls bootstrap, manifest, checksums and binaries | Medium residual trust assumption | HTTPS with normal certificate validation and fixed GitHub repository/allowlisted hosts; SHA-256 detects corruption but cannot authenticate a compromised publisher. No separate signatures, transparency record, freshness or downgrade protection. |
| Private destination and trusted local account required | Medium residual limitation | Ancestor/target symlink/reparse refusal and destination locking are not a sandbox against hostile same-user races or a shared-writable parent. Windows inherits directory ACLs; installer does not claim to repair arbitrary existing ACLs. |
| Power loss and force-kill can leave installer state | Low residual limitation | Same-directory staging and atomic replacement where supported; stale locks fail closed. No fsync/power-loss durability guarantee, automatic backup pruning or automatic rollback. |
| Real public distribution and interactive OS decisions untested | UNVERIFIED | No release exists. Offline native tests and synthetic origin markers do not validate GitHub asset downloads, SmartScreen reputation, browser/Finder/Explorer behavior or antivirus clearance. |

All installer paths were reviewed: manual HTTPS redirect handling permits only
`github.com`, `release-assets.githubusercontent.com` and
`objects.githubusercontent.com`, port 443, at most five hops. No credentials,
fragments, controls or backslashes are accepted; Bash ignores curl configuration.
Manifest parsing validates stable version, known OS/CPU, canonical filename,
unique rows, lowercase SHA-256 and bounded positive size. Artifact transfer is
bounded; exact size and SHA-256 must match before commit. Published SHA256SUMS is
for manual verification; bootstrap reads the same hash/size from the manifest.

Lock acquisition precedes staging. Existing locks are never stolen. Unix keeps a
unique preserved copy as backup then renames on the same filesystem; Windows uses
File.Replace with a unique backup and never delete-then-move fallback. Native
exclusive-handle and running-image tests verify preservation/retry. Identical-byte
installs preserve existing file metadata; they do not retrospectively mark an
older unmarked installation. Offline copies retain origin metadata. New remote
Windows installation now requires NTFS alternate streams; remote macOS requires
xattr/extended attributes. This is a documented security correction to filesystem
requirements, with no change to the v1 manifest or bootstrap arguments.

Windows two-installer process races and forced process termination are not newly
claimed as native passes: the Windows suite checks stale/exclusive locks and
recovery, while active concurrency and signal interruption are exercised by the
Unix suite. There is no real HTTP transport integration before publication;
Bash uses deterministic download adapters and Windows tests the URI predicate
and local copy path. These limits are not hidden by the six-platform job result.

Error paths use fixed actionable messages rather than native exceptions or
attacker-supplied arguments. Existing tests cover secret-bearing arguments,
malformed metadata, wrong sizes/hashes, partial/missing downloads, unsafe paths,
interruption, concurrency and backup retention. No download is executed by an
installer. Windows ACL tests deny writes only on disposable owned fixtures and
restore those fixture ACLs for recovery; no system ACL or OS policy is weakened.

## Native and OS evidence

The original six-platform success remains preserved in
[native-acceptance.md](native-acceptance.md) and its 2026-11-07 JSON evidence. New
changes require their own run; historical passes are not substituted for them.

[Run 42456877849](https://github.com/siaas-software/siaas-cli/actions/runs/42456877849)
completed successfully at tested commit
`751694759669c44ea71ef44313c3d6246eb45233`. All six job logs were inspected:
**450 passing test/subtest events, no failing or skipped test cases**. Windows
counts include 27 repeated integration events under each genuine standard user;
these are not 450 distinct tests. Platform-inapplicable workflow steps are skipped
by design. The API artifact list is empty.

| Native runner / target | Result | Passing events | Genuine standard-user result |
| --- | --- | --- | --- |
| [ubuntu-24.04 / linux amd64](https://github.com/siaas-software/siaas-cli/actions/runs/42456877849/job/128657742270) | PASS | 73 | Not part of Windows-specific probe |
| [ubuntu-24.04-arm / linux arm64](https://github.com/siaas-software/siaas-cli/actions/runs/42456877849/job/128657742265) | PASS | 73 | Not part of Windows-specific probe |
| [windows-2022 / windows amd64](https://github.com/siaas-software/siaas-cli/actions/runs/42456877849/job/128657742146) | PASS | 78 | PASS, 27 integration events |
| [windows-11-arm / windows arm64](https://github.com/siaas-software/siaas-cli/actions/runs/42456877849/job/128657742276) | PASS | 78 | PASS, 27 integration events |
| [macos-15-intel / darwin amd64](https://github.com/siaas-software/siaas-cli/actions/runs/42456877849/job/128657742003) | PASS | 74 | Not part of Windows-specific probe |
| [macos-15 / darwin arm64](https://github.com/siaas-software/siaas-cli/actions/runs/42456877849/job/128657742292) | PASS | 74 | Not part of Windows-specific probe |

Windows subprocesses reported **5.1.20348.5773 Desktop** (x64) and
**5.1.26100.9711 Desktop** (ARM64), including under the standard-user accounts.
Both recorded `AdministratorsMember=False`, real user profiles/default install
paths and `STANDARD_USER_ACCEPTANCE=PASS`. Both passed normal/dangling junction
rejection, denied-write ACL preservation/recovery, Unicode/spaces, repeated
installation and retained backups. Running-image replacement returned 0 with
backup retention in both account contexts; exclusive-handle tests required exit
6 with old bytes/prior backups intact, followed by safe retry.

Both Windows OS probes recorded Authenticode=NotSigned, ZoneId=3, unchanged
SHA-256, and exit 1 plus a recognized policy-refusal error from the Internet-marked
unsigned script under the explicitly stronger RemoteSigned process policy.
Both macOS runners recorded Gatekeeper enabled and assessment exit 3 (rejected).
Intel codesign inspection returned 1 (not signed); Apple Silicon returned 0 with
`Signature=adhoc`. Quarantine remained intact and byte hashes did not change.
Marked executables were not launched by these OS probes. Interactive SmartScreen,
Explorer/Finder and browser-downloaded execution remain **UNVERIFIED**.

All six native binary hashes/sizes match the independently cross-built local
candidate. The final documentation-only commit records this tested revision and
uses `[skip ci]` to avoid rerunning unchanged implementation/tests. See
[API and selected log evidence](evidence/public-security-native-2026-11-08.json).


The Windows standard-user harness creates a disposable local account, adds only
Users membership, loads its real profile and runs the native integration binary
with that identity. It verifies the SID and absence of Administrators membership,
uses the account's real LOCALAPPDATA/default path, private Go cache and temporary
files, and exercises Unicode/spaces, denied ACL recovery, junctions, repeat
installation, replacement, backups, running images and exclusive handles. The
parent uses the hosted administrator account only to provision/remove this test
account. It does not prove the installer needs administrator rights. The child's
identity and test results, not the parent job identity, establish acceptance.
The hosted Windows image has LocalMachine=Unrestricted; the standard-user
result establishes permissions and filesystem behavior under that existing image
configuration, not compatibility with a clean Windows client default of
Restricted. The separate Internet-script probe strengthens to RemoteSigned.
No execution-policy override is supplied to the child; inherited process override
is removed. Any policy refusal is reported, never bypassed. The ephemeral runner
disposes of the loaded profile; the harness removes only its account/staging.

Windows origin probes verify NotSigned Authenticode, ZoneId 3 and unchanged bytes.
The marked executable is not launched. Script policy is tested with RemoteSigned
only when it strengthens or equals the current policy; stricter policy is retained.
The negative test must return nonzero for the marked unsigned script. This does
not establish a SmartScreen user-interface result. macOS probes record codesign,
Gatekeeper status and assessment exits, retain quarantine and never execute the
quarantined binary. A Go-generated ad-hoc signature is not Developer ID signing
or notarization. Linux tests verify native standalone execution, installed mode
0755 and newly created default bin directory mode 0700.

Expected OS behavior is supported by Microsoft's
[execution-policy documentation](https://learn.microsoft.com/en-us/powershell/module/microsoft.powershell.core/about/about_execution_policies?view=powershell-5.1)
and [SmartScreen documentation](https://learn.microsoft.com/en-us/windows/security/operating-system-security/virus-and-threat-protection/microsoft-defender-smartscreen/).
Restricted blocks scripts; RemoteSigned can block Internet-marked unsigned
scripts. Execution policy is defense in depth, not an absolute security boundary.
Headless fixture success cannot establish browser download reputation. CI's
preinstalled tool setup may use its own process policy; acceptance probes remove
inherited overrides and never recommend such a setup policy to end users.

## Protection and public authority

Read-only API evidence is retained in
[public-security-protections-2026-11-08.json](evidence/public-security-protections-2026-11-08.json).

| Check | Result | API evidence |
| --- | --- | --- |
| Public repository / default branch | PASS | GET `/repos/siaas-software/siaas-cli`: private=false, visibility=public, default_branch=main |
| protect-main active and matches main | PASS | GET `/repos/siaas-software/siaas-cli/rulesets/26770075`: active; refs/heads/main; no exclusions |
| Main PR/deletion/force-push restrictions | PASS for configured rules | pull_request, deletion, non_fast_forward, required_linear_history; resolved review threads required |
| Mandatory approving reviewer / required CI gate | NOT CONFIGURED | required_approving_review_count=0; no required_status_checks rule. A PR is required, an approving reviewer and successful CI are not enforced by this ruleset. Owner risk decision; no automatic settings change. |
| protect-release-tags active, v* and update restricted | PASS | GET `/repos/siaas-software/siaas-cli/rulesets/26770131`: active; refs/tags/v*; deletion, non_fast_forward, update; no exclusions |
| Tag creation approval / signed tags | NOT CONFIGURED by these rules | No creation or signature rule is listed. Ordinary authorized publisher permissions still determine who can create a new release tag; this review did not exercise them. |
| Unintended listed bypass actors | PASS | Both bypass_actors=[] and current_user_can_bypass=never |
| Enforcement inferred from plan/settings | PASS for active public-repository ruleset configuration; destructive probe NOT PERFORMED | No disabled/evaluate enforcement shown. No attempt to push main or mutate a protected tag. Administrators with settings rights can still change rules. |
| Effective main rule aggregation / additional admin settings | UNVERIFIED | Connector rejected `/rules/branches/main` URL; previous administrative endpoints lacked integration scope. No plan/security-setting claim beyond readable evidence. |
| Public release assets | UNVERIFIED | GET `/repos/siaas-software/siaas-cli/releases` returned []; no live release installation possible |

No new comparison writes or administrative actions were made in any other SIAAS
repository. Earlier protection comparisons remain historical evidence.

## Candidate and bootstrap compatibility

The current candidate inventory is recorded separately with hashes and sizes.

| Prepared asset | Bytes |
| --- | ---: |
| `GO-LICENSE.txt` | 1453 |
| `RELEASE_NOTES.md` | 2747 |
| `SHA256SUMS` | 982 |
| `cli-release-v1.txt` | 725 |
| `install.ps1` | 12917 |
| `install.sh` | 9415 |
| `siaas-cli_0.1.0_darwin_amd64` | 2630976 |
| `siaas-cli_0.1.0_darwin_arm64` | 2494818 |
| `siaas-cli_0.1.0_linux_amd64` | 2580604 |
| `siaas-cli_0.1.0_linux_arm64` | 2556028 |
| `siaas-cli_0.1.0_windows_amd64.exe` | 2744832 |
| `siaas-cli_0.1.0_windows_arm64.exe` | 2541056 |

The twelve assets are six raw native executables, cli-release-v1.txt, SHA256SUMS,
install.sh, install.ps1, RELEASE_NOTES.md and GO-LICENSE.txt. The manifest has six
rows; SHA256SUMS covers the other eleven files. Scripts copied into the candidate
must match repository bytes. Version 0.1.0 and tag path v0.1.0 are consistent.

Commands in [public-installation.md](public-installation.md) invoke Bash explicitly
and PowerShell 5.1 with existing policy; no `curl | sh`, application version,
external language runtime or default elevation is required. Defaults are
~/.local/bin/siaas and %LOCALAPPDATA%\SIAAS CLI\bin\siaas.exe; paths are quoted.
Host architecture selects amd64 or arm64 and the exact canonical artifact name.
Latest stable manifest download is fixed to this public repository; executable
URLs are derived under `/releases/download/v0.1.0/`, not supplied by metadata.
A downloaded older bootstrap still selects latest stable, not a pinned CLI.

Anonymous accessibility is expected only after a complete non-draft stable public
release is published. It is not a live test result. The
[post-publication smoke-test and incident procedure](post-publication-smoke-test.md)
covers anonymous access, normal OS security, CLI exits, repeat installation,
verified backup recovery and a defective/compromised release. Protected tags must
not be moved; fixes require a new reviewed release version and separate approval.

## Local validation commands and limits

Host: Linux x64; pinned Go 1.27.3, no third-party modules. Environment:
`PATH=/tmp/siaas-cli-toolchain/go/bin:$PATH`, `GOTOOLCHAIN=local GOPROXY=off
GOSUMDB=off GOTELEMETRY=off GOCACHE=/tmp/siaas-cli-go-cache`. Temporary tools were
missing after session resumption; the first attempted check did not run and is
not counted as a pass. The official Go 1.27.3 and PowerShell 7.6.9 distributions
were restored with HTTPS/certificate validation and their published SHA-256
verified. PowerShell 7 on Linux is only a portable syntax/URI check.

| Actual command / check | Result |
| --- | --- |
| `go test -count=1 -timeout=8m ./...` | Exit 0; CLI unit and Linux native integration tests |
| `go test -race -count=1 -timeout=8m ./...` | Exit 0; CLI unit and Linux native integration tests |
| `go vet ./...` | Exit 0 |
| `test -z "$(gofmt -l cmd internal tools tests)"` | Exit 0 |
| `bash -n install.sh tests/macos-os-security.sh tools/prepare-release.sh` | Exit 0 |
| `GOOS=windows GOARCH=arm64 CGO_ENABLED=0 go test -c -o dist/public-security-windows-tests.exe ./tests` | Exit 0; compilation only |
| `GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go test -c -o dist/public-security-macos-tests ./tests` | Exit 0; compilation only |
| `/tmp/siaas-cli-powershell/pwsh -NoProfile -File tests/powershell-policy.ps1` | Exit 0; 4 accepted and 10 rejected URI cases, relative redirect and syntax |
| PowerShell Parser.ParseFile on install.ps1, windows-os-security.ps1, windows-standard-user.ps1 | Exit 0; no parse errors; not Windows acceptance |
| `bash tools/prepare-release.sh dist/public-security-v0.1.0` | Exit 0; all six executable targets and eleven checksum checks |
| `bash tools/prepare-release.sh dist/public-security-v0.1.0-repeat` | Exit 0; independent fresh output, same twelve assets |
| `cmp dist/public-security-v0.1.0/SHA256SUMS dist/public-security-v0.1.0-repeat/SHA256SUMS` | Exit 0 |
| Python comparison of every asset's bytes, size/hash, manifest rows, checksum list and copied scripts | Exit 0; 12 identical files, 6 canonical rows, 11 verified checksum entries |
| PyYAML workflow parse/assertions; `git diff --check` | Exit 0; six targets, contents read, 30-minute timeout; no whitespace errors |

Native workflow also runs pinned host/architecture assertions, gofmt, go vet,
`go test -v -count=1 -timeout=8m ./...`, PowerShell 5.1 Desktop URI checks on
Windows, `./tests/windows-os-security.ps1`,
`./tests/windows-standard-user.ps1`, and `bash tests/macos-os-security.sh` on
applicable hosts. The standard-user child invokes the compiled integration suite
with `-test.v -test.timeout=10m`. Tests use local release fixtures; native binary
hashes and sizes are compared with the local candidate inventory separately.

The complete candidate is in ignored local `dist/public-security-v0.1.0`; inventory
and reproducibility evidence are committed in
[public-release-candidate-2026-11-08.json](evidence/public-release-candidate-2026-11-08.json).
No binaries were uploaded to Actions or Releases. These local files are prepared
artifacts, not publicly downloadable assets.

## Follow-up attempts retained for review

These failed/cancelled attempts are not counted as successful acceptance:

| Run / commit | Observed issue | Correction |
| --- | --- | --- |
| [42456149023](https://github.com/siaas-software/siaas-cli/actions/runs/42456149023), `3b3731f666e012562c17b3e8f93013a71503570c` | Windows x64 OS-origin probe failed to write ADS using .NET Framework File.WriteAllText; superseded run cancelled | Use PowerShell 5.1 Set-Content -Stream and Get-Content -Stream; installer still fails closed if marking fails |
| [42456513757](https://github.com/siaas-software/siaas-cli/actions/runs/42456513757), `ae4fa2796ec604d8d289917f5c3e2f11ad734944` | Expected script-policy refusal left LASTEXITCODE=1, making Actions report the probe as failed; superseded run cancelled | Explicit success exit only after all assertions pass; negative result is still required |
| [42456663768](https://github.com/siaas-software/siaas-cli/actions/runs/42456663768), `ebb1affb0c47ce33b4280203c1059004d400e44d` | Genuine standard-user identity established, but PS5.1 parsed unquoted Go `-test.v` flags incorrectly; no standard-user acceptance pass | Quote test flags, retain process handle to read child exit, and verify policy refusal has a recognized security-error cause |

No policy, certificate validation, signature check or path restriction was relaxed
to turn these failures into a pass. Superseded runs are cancelled by the existing
concurrency policy to avoid unnecessary runner use.

## Risks requiring explicit release-owner acceptance

1. **Unsigned distribution and limited default-policy compatibility.** The
   Windows script is not Authenticode-signed and can be refused by Restricted or
   RemoteSigned when downloaded. macOS candidates lack Developer ID/notarization
   and were rejected by Gatekeeper assessment. Approve only a clearly scoped
   unsigned release for environments whose existing policy permits it, or defer
   publication until an approved signing/distribution approach is implemented.
   Do not promise frictionless public installation across all six OS targets.
2. **Publisher trust and governance.** SHA-256 is not a signature. Compromise of
   the publisher/bootstrap/manifest can compromise installation. Rulesets prevent
   listed destructive updates, but do not mandate an approving reviewer or CI
   success, and do not restrict new tag creation. Administrative security
   settings outside the available API scope still need owner review.
3. **Filesystem and support boundaries.** Private account-controlled destinations,
   trusted OS utilities/CA/proxy configuration and supported rename/replace/origin
   metadata operations are required. No hostile same-user race, power-loss
   durability, older OS acceptance or comprehensive Windows interruption/concurrent
   process proof is claimed.
4. **Post-publication gate.** Anonymous release asset access and live remote
   installation have not been tested because v0.1.0 does not exist. The owner must
   arrange the documented smoke tests after separately authorized publication and
   treat a failure as a distribution incident. Interactive OS checks remain
   explicitly unverified until executed on clean native systems.

There is no remaining failed native job or confirmed uncorrected implementation
bug from this review. The blockers to an unconditional READY FOR PUBLIC RELEASE
are the unsigned support/risk decision and outstanding public/interactive
acceptance, not a lack of native compilation or offline tests. Acceptance of
these risks has not been inferred from authorization to conduct this review.
PR #1 remains draft. Stop here: no merge, release tag or publication without
separate explicit authorization.
