# CLI bootstrap release contract v1

This contract describes **CLI releases**, not product releases. No releases exist
at M1 validation time; remote installation was not tested or claimed. Local fixture
builds exercise this format without GitHub, credentials or external services.

## Assets and trust

For CLI version `X.Y.Z`, a future stable release in
`siaas-software/siaas-cli` must use tag `vX.Y.Z`. Bootstrap first reads the fixed
repository's `releases/latest/download/cli-release-v1.txt`. It derives the selected
artifact's `releases/download/vX.Y.Z/<canonical-filename>` path from validated
fields, never from an arbitrary URL in metadata. These are **prospective routing
rules**, not assertions that any version is currently downloadable. The repository
was verified public on 2026-11-07; no release assets exist yet. Publication must use
a public, non-draft stable release, designated latest, with all referenced assets
uploaded before publication. The scripts send no GitHub credentials. Actual
anonymous asset accessibility still needs a post-publication smoke test.

The manifest is a UTF-8, LF-terminated text file with no BOM, blank lines or comments:

```text
siaas-cli-release-v1 X.Y.Z
<os> <arch> siaas-cli_X.Y.Z_<os>_<arch>[.exe] <64-lowercase-hex-sha256> <positive-byte-count>
```

Space-separated fields avoid a Unix dependency on Node, Python or jq. The parser
only accepts stable numeric versions (no leading-zero components) and the six
known target pairs: Linux/macOS/Windows (`linux`, `darwin`, `windows`) with
`amd64`/`arm64`. Windows filenames end in `.exe`. All names must exactly match
version/target, so traversal, flags and URL injection cannot become artifact paths.
Exactly one matching platform row is required. Metadata is limited to 64 KiB and
the selected raw binary to 100,000,000 bytes. The release publisher must not include
duplicate rows, even for a different target than the current installer selects.
Both installers now enforce that rule (exit 7), correcting the earlier acceptance
of duplicates for non-selected platforms. Canonical v1 manifests remain compatible.

The build also emits conventional `SHA256SUMS` for manual verification. Bootstrap
uses the SHA-256 and byte count published in the manifest; those checksums must
agree with `SHA256SUMS`. Hashes do not independently authenticate a publisher who
controls both artifact and manifest. HTTPS/repository authority is the M1 trust
anchor; stronger signing is a documented follow-up, not silently claimed here.
Every transport hop is HTTPS, default port 443, through exactly `github.com`,
`release-assets.githubusercontent.com` or `objects.githubusercontent.com`. At most
five redirects are followed. Relative redirects are resolved by curl/.NET before
the next URI is checked; userinfo, backslashes, fragments, control characters and
other destinations are refused. Certificate validation stays enabled. Bash
disables curl configuration files and never enables automatic redirect following.

`tools/prepare-release.sh` assembles a fresh local candidate, adding `install.sh`,
`install.ps1`, `RELEASE_NOTES.md` and `GO-LICENSE.txt` to the six raw binaries and
two metadata files. `SHA256SUMS` covers all eleven other assets, including the
manifest; the manifest contains only the six binary rows. The supplemental hashes
do not change the bootstrap v1 format. A failed assembly never renames its staging
directory to the destination. The developer helper requires GNU `sha256sum` and
`mv`; this is not an end-user runtime requirement.

## Local fixture path

`go run ./tools/build` writes the actual manifest, checksums and host binary to
`dist/`. `install.sh --from-dir <dir>` / `install.ps1 -FromDir <dir>` replace only
the transport with local file copies. Parsing, selected target, size/hash checks,
staging, locking, backup and replacement stay on the production code path. This is
an explicit trust choice; never use unreviewed local metadata as authentication.
Installer tests create malformed/corrupt fixtures in temporary directories and
prove failures preserve an existing executable and unrelated data.

## Pre-publication requirements (not performed by M1)

1. Review and pin the source commit/compiler and CLI version; run baseline tests.
2. Build each intended target, execute native CLI and bootstrap tests on that OS,
   and define its minimum supported OS/CPU/filesystem requirements.
3. Complete the platform signing/notarization decision. Sign final executable
   bytes before computing the manifest and `SHA256SUMS`; the current build tool
   describes unsigned local artifacts and does not implement a signing pipeline.
4. Package the exact final artifacts, manifest, checksums, source/compiler record,
   reviewed installers and appropriate third-party notices in a separately
   authorized release. Go's notice is also embedded in CLI help.
5. Verify public visibility, final metadata/artifact agreement and remote bootstrap
   independently. Do not call an artifact available based on a constructed URL.

M1 creates no release, tag, hosting service or publication automation. The separately
owner-authorized native acceptance workflow runs offline tests on free standard runners.
Failure to find a CLI release is an actionable installer error (7), not a reason
to install SIAAS Personal or require a product version.

## OS origin metadata (security correction, same v1 byte contract)

Remote candidates are marked as Internet-sourced before commit: Windows NTFS
`Zone.Identifier` ZoneId 3, macOS `com.apple.quarantine`. Failure to retain the
marker aborts installation (6; missing macOS xattr utility is 7). Offline copies
preserve existing origin metadata; identical-byte installs preserve the existing
file. SHA-256 and byte counts remain unchanged and do not attest ACLs or extended
attributes. This correction preserves CLI arguments, canonical asset names and
manifest v1 while allowing normal OS policy to assess downloaded executables.
No publisher signature or notarization is asserted. See the current public
security review before representing these unsigned assets as generally usable.
