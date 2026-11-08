#!/usr/bin/env bash
# Observe signing/quarantine/Gatekeeper without executing a quarantined file.
set -euo pipefail
stage=$(mktemp -d "${TMPDIR:-/tmp}/siaas-os-security.XXXXXXXX")
trap 'rm -rf -- "$stage"' EXIT
go run ./tools/build -out "$stage"
artifact=$(find "$stage" -type f -name 'siaas-cli_*' -print)
before=$(shasum -a 256 "$artifact" | awk '{print $1}')
xattr -w com.apple.quarantine '0081;00000000;SIAAS offline security fixture;' "$artifact"
xattr -p com.apple.quarantine "$artifact"
test "$(shasum -a 256 "$artifact" | awk '{print $1}')" = "$before"
set +e
codesign -dv --verbose=4 "$artifact" 2>&1
printf 'OS_SECURITY codesign inspection exit=%s\n' "$?"
spctl --status
printf 'OS_SECURITY Gatekeeper status exit=%s\n' "$?"
spctl --assess --type execute --verbose=4 "$artifact" 2>&1
printf 'OS_SECURITY Gatekeeper assessment exit=%s\n' "$?"
set -e
printf '%s\n' 'OS_SECURITY quarantine retained and SHA256 unchanged; quarantined executable not launched.' \
  'OS_SECURITY browser/Finder launch and notarized distribution acceptance=UNVERIFIED.'
