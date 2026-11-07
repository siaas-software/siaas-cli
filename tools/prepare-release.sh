#!/usr/bin/env bash
# Local unsigned candidate assembly only. Never tags, uploads or publishes.
set -euo pipefail
[[ $# -eq 1 && -n "$1" ]] || { echo 'Usage: bash tools/prepare-release.sh NEW_OUTPUT_DIRECTORY' >&2; exit 2; }
destination="$1"
[[ ! -e "$destination" && ! -L "$destination" ]] || { echo 'Output already exists; use a fresh directory.' >&2; exit 2; }
mkdir -p -- "$(dirname -- "$destination")"
stage=$(mktemp -d "$(dirname -- "$destination")/.siaas-release.XXXXXXXX")
trap 'rm -rf -- "$stage"' EXIT
trap 'exit 130' INT
trap 'exit 143' TERM
go run ./tools/build -targets linux/amd64,linux/arm64,darwin/amd64,darwin/arm64,windows/amd64,windows/arm64 -out "$stage"
read -r schema version < "$stage/cli-release-v1.txt"
[[ "$schema" == siaas-cli-release-v1 && "$version" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]
cp -- install.sh install.ps1 "$stage/"
cp -- "docs/releases/v$version.md" "$stage/RELEASE_NOTES.md"
cp -- internal/cli/NOTICE.txt "$stage/GO-LICENSE.txt"
(
  cd "$stage"
  # Build metadata already contains the six executable checksums. Include every
  # supplemental asset, including the bootstrap's trusted manifest, in the list.
  sha256sum cli-release-v1.txt install.sh install.ps1 RELEASE_NOTES.md GO-LICENSE.txt >> SHA256SUMS
  sha256sum --check SHA256SUMS
)
# Output is intentionally fresh: no stale release metadata or artifacts survive.
[[ ! -e "$destination" && ! -L "$destination" ]]
mv -T -- "$stage" "$destination"
printf 'Prepared unsigned local candidate: %s\n' "$destination"
