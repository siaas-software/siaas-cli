#!/usr/bin/env bash
# Install SIAAS CLI only. No product installation, shell-profile edits or elevation.
set -euo pipefail
umask 077
fail() { printf 'siaas installer: %s\n' "$2" >&2; exit "$1"; }
usage() {
  printf '%s\n' 'Usage: bash install.sh [--install-dir ABSOLUTE_PATH] [--from-dir LOCAL_RELEASE_DIRECTORY]' \
    'Installs the CLI itself; no product version is required.' \
    '--from-dir explicitly trusts local release metadata for offline testing.'
}
install_dir="${SIAAS_CLI_INSTALL_DIR:-${HOME:?HOME is required}/.local/bin}"
fixture=''
while [[ $# -gt 0 ]]; do
  case "$1" in
    --help) [[ $# -eq 1 ]] || fail 2 'Invalid options. Use --help.'; usage; exit 0 ;;
    --install-dir|--from-dir)
      [[ $# -ge 2 && -n "$2" ]] || fail 2 'An option value is missing. Use --help.'
      if [[ "$1" == --install-dir ]]; then install_dir="$2"; else fixture="$2"; fi
      shift 2 ;;
    *) fail 2 'Unsupported option. Use --help. No product argument is accepted.' ;;
  esac
done
case "$(uname -s)" in Linux) platform=linux ;; Darwin) platform=darwin ;; *) fail 2 'Unsupported OS. Use Windows PowerShell on Windows; Unix targets are Linux and macOS.' ;; esac
case "$(uname -m)" in x86_64|amd64) arch=amd64 ;; arm64|aarch64) arch=arm64 ;; *) fail 2 'Unsupported architecture. CLI artifacts target x64 and arm64.' ;; esac
# Require a canonical absolute path and refuse symlink components, including target.
[[ "$install_dir" == /* && "$install_dir" != / && "$install_dir" != *'//'*
   && ! "$install_dir" =~ [[:cntrl:]] ]] || fail 2 'Use an absolute, non-root install directory without control characters.'
install_dir="${install_dir%/}"
IFS=/ read -r -a components <<< "$install_dir"
cursor=''
for component in "${components[@]}"; do
  [[ -n "$component" ]] || continue
  [[ "$component" != . && "$component" != .. ]] || fail 2 'Dot path components are not allowed.'
  cursor="$cursor/$component"
  [[ ! -L "$cursor" ]] || fail 6 'Symlink install paths are refused. Choose a private real directory.'
done
if command -v sha256sum >/dev/null 2>&1; then
  hash() { sha256sum "$1" 2>/dev/null | awk '{print $1}'; }
elif command -v shasum >/dev/null 2>&1; then
  hash() { shasum -a 256 "$1" 2>/dev/null | awk '{print $1}'; }
else fail 7 'SHA-256 verification needs sha256sum or shasum from your OS.'; fi
if [[ -z "$fixture" ]]; then
  command -v curl >/dev/null 2>&1 || fail 7 'Install curl using your OS tools, or use a verified local release directory.'
else
  [[ -d "$fixture" ]] || fail 7 'Local release directory is missing. Build CLI artifacts first.'
fi
mkdir -p "$install_dir" 2>/dev/null || fail 6 'Cannot create install directory. Choose a writable user-owned location.'
[[ -O "$install_dir" && -w "$install_dir" ]] || fail 6 'Install directory must be writable and owned by the current user.'
lock="$install_dir/.siaas-cli-install.lock"
mkdir "$lock" 2>/dev/null || fail 4 'Installation locked. Wait for the active installer. After interruption, confirm it has exited before removing only .siaas-cli-install.lock.'
stage=''
cleanup() {
  local result=$?
  trap - EXIT
  if [[ -n "$stage" ]] && ! rm -rf -- "$stage" 2>/dev/null; then
    printf '%s\n' 'siaas installer: staging cleanup failed. Inspect only abandoned .siaas-cli-stage entries after installer exit.' >&2
    [[ "$result" -ne 0 ]] || result=6
  fi
  if ! rmdir "$lock" 2>/dev/null; then
    printf '%s\n' 'siaas installer: lock cleanup failed. Inspect .siaas-cli-install.lock before retrying.' >&2
    [[ "$result" -ne 0 ]] || result=6
  fi
  exit "$result"
}
trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM
stage=$(mktemp -d "$install_dir/.siaas-cli-stage.XXXXXXXX" 2>/dev/null) || fail 6 'Cannot create private staging directory.'
target="$install_dir/siaas"
[[ ! -L "$target" && ( ! -e "$target" || -f "$target" ) ]] || fail 6 'Existing siaas target is not a regular file. Inspect it without deleting application data.'
get() {
  local name="$1" destination="$2" limit="$3" url="$4" info status redirect
  if [[ -n "$fixture" ]]; then
    [[ -f "$fixture/$name" && ! -L "$fixture/$name" ]] || fail 7 'Required local release artifact is missing or is a symlink. Rebuild the release directory.'
    [[ $(wc -c < "$fixture/$name") -le "$limit" ]] || fail 6 'Release artifact exceeds the permitted size.'
    cp -- "$fixture/$name" "$destination" 2>/dev/null || fail 6 'Cannot stage local artifact. Check permissions and free space.'
  else
    for ((redirect=0; redirect<=5; redirect++)); do
      # Validate every hop BEFORE connecting. No userinfo, alternate ports,
      # fragments, backslashes or control characters; only GitHub release hosts.
      [[ "$url" =~ ^https://(github\.com|release-assets\.githubusercontent\.com|objects\.githubusercontent\.com)(:443)?/ \
         && ! "$url" =~ [[:cntrl:][:space:]] && "$url" != *\\* && "$url" != *'#'* ]] || fail 6 'Untrusted release redirect refused.'
      # --disable must be first: an ambient .curlrc cannot enable insecure TLS,
      # automatic redirects, credentials or an alternate output destination.
      # curl resolves relative Location values through redirect_url without -L.
      info=$(curl --disable --fail --silent --proto '=https' --proto-redir '=https' \
        --connect-timeout 20 --max-time 180 --max-filesize "$limit" \
        --write-out '%{http_code}\n%{redirect_url}' -o "$destination" "$url" 2>/dev/null) || fail 7 'CLI release unavailable or download failed. Check network/release access or use local fixtures; existing CLI preserved.'
      status="${info%%$'\n'*}"
      case "$status" in
        200) break ;;
        301|302|303|307|308)
          [[ "$redirect" -lt 5 ]] || fail 7 'Too many release redirects. Existing CLI preserved.'
          url="${info#*$'\n'}" ;;
        *) fail 7 'Release server returned an unexpected response. Existing CLI preserved.' ;;
      esac
    done
  fi
  [[ $(wc -c < "$destination") -le "$limit" ]] || fail 6 'Release artifact exceeds the permitted size.'
}
base='https://github.com/siaas-software/siaas-cli/releases'
get cli-release-v1.txt "$stage/manifest" 65536 "$base/latest/download/cli-release-v1.txt"
IFS=' ' read -r schema version extra < "$stage/manifest" || fail 6 'Release metadata is empty or incomplete.'
[[ "$schema" == siaas-cli-release-v1 && -z "${extra:-}" && "$version" =~ ^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$ ]] || fail 6 'Invalid release metadata schema/version. Obtain a verified CLI release.'
found=0
seen=' '
artifact='' expected='' bytes=''
while IFS=' ' read -r row_os row_arch row_name row_hash row_bytes row_extra || [[ -n "${row_os:-}" ]]; do
  [[ -n "${row_os:-}" ]] || fail 6 'Invalid blank metadata entry.'
  case "$row_os/$row_arch" in linux/amd64|linux/arm64|darwin/amd64|darwin/arm64|windows/amd64|windows/arm64) ;; *) fail 6 'Invalid platform in release metadata.' ;; esac
  [[ "$seen" != *" $row_os/$row_arch "* ]] || fail 7 'Release contains duplicate platform entries. Existing CLI preserved.'
  seen="$seen$row_os/$row_arch "
  suffix=''; [[ "$row_os" != windows ]] || suffix='.exe'
  [[ -z "${row_extra:-}" && "$row_name" == "siaas-cli_${version}_${row_os}_${row_arch}${suffix}"
     && "$row_hash" =~ ^[a-f0-9]{64}$ && "$row_bytes" =~ ^[1-9][0-9]{0,8}$ ]] || fail 6 'Invalid release artifact metadata.'
  if [[ "$row_os/$row_arch" == "$platform/$arch" ]]; then
    found=$((found + 1)); artifact="$row_name"; expected="$row_hash"; bytes="$row_bytes"
  fi
done < <(tail -n +2 "$stage/manifest")
[[ "$found" -eq 1 ]] || fail 7 'Release has no unique artifact for this platform. No existing executable was replaced.'
[[ "$bytes" -le 100000000 ]] || fail 6 'CLI artifact is unexpectedly large.'
get "$artifact" "$stage/candidate" "$bytes" "$base/download/v$version/$artifact"
[[ $(wc -c < "$stage/candidate") -eq "$bytes" && "$(hash "$stage/candidate")" == "$expected" ]] || fail 6 'Artifact checksum/size verification failed. Existing CLI preserved; obtain a fresh verified release.'
if [[ "$platform" == darwin && -z "$fixture" ]]; then
  command -v xattr >/dev/null 2>&1 || fail 7 'macOS Internet downloads require xattr to retain quarantine metadata.'
  xattr -w com.apple.quarantine "0081;$(printf '%x' "$(date +%s)");SIAAS CLI;" "$stage/candidate" 2>/dev/null || fail 6 'Cannot retain macOS quarantine metadata. Choose a private filesystem supporting extended attributes.'
fi
if [[ -f "$target" && "$(hash "$target")" == "$expected" && -x "$target" ]]; then
  printf '%s\n' 'SIAAS CLI is already installed with identical verified bytes.'; exit 0
fi
chmod 755 "$stage/candidate" 2>/dev/null || fail 6 'Cannot set executable permissions.'
if [[ -f "$target" ]]; then
  backup="$install_dir/siaas.backup.${stage##*.}"
  [[ ! -e "$backup" ]] || fail 6 'Backup name conflict. Retry installation.'
  cp -p -- "$target" "$backup" 2>/dev/null || fail 6 'Cannot preserve existing CLI. Check permissions/free space; existing CLI remains active.'
fi
# Same-filesystem rename is the commit point. The old executable is backed up.
mv -f -- "$stage/candidate" "$target" 2>/dev/null || fail 6 'Atomic replacement failed. Existing CLI/backup preserved; close CLI processes, inspect permissions and retry.'
printf '%s\n' "Installed SIAAS CLI $version. No SIAAS product was installed." \
  'Use the selected directory on PATH, or invoke its siaas executable directly.' \
  'Any replaced executable is retained as siaas.backup.*; inspect it before manual removal.'
