#!/usr/bin/env bash
# Review exact immutable release commits on a disposable Linux host.
set -euo pipefail
test "$(node --version)" = "v24.22.0"
here=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
root=$(mktemp -d)
trap 'rm -rf -- "$root"' EXIT
export HOME="$root/home" npm_config_cache="$root/npm-cache" NEXT_TELEMETRY_DISABLED=1
mkdir -m 700 "$HOME"
export GIT_CONFIG_NOSYSTEM=1 GIT_TERMINAL_PROMPT=0
for version in 4 5; do
  case "$version" in
    4) commit=d8bf6e61a4d47920a00cc4e069a90cf26ac2d80f ;;
    5) commit=853f35563b4b86ee355d276acd1a96c3e98f1007 ;;
  esac
  target="$root/rc$version"
  git clone --depth=1 --branch "v1.1.0-rc.$version" https://github.com/siaas-software/siaas-personal.git "$target"
  test "$(git -C "$target" rev-parse HEAD)" = "$commit"
  cp -- "$here/rc4-rc5-profile.mjs" "$target/.cli-profile-review.mjs"
  (cd "$target" && npm ci --ignore-scripts --no-audit --no-fund)
done
(cd "$root/rc4" && node --import tsx .cli-profile-review.mjs seed "$root/old-profile" "$root/expected.json")
cp -a -- "$root/old-profile" "$root/new-profile"
(cd "$root/rc5" && node --import tsx .cli-profile-review.mjs verify "$root/new-profile" "$root/expected.json")
echo "RC4 -> RC5 copied profile compatibility fixture PASS; not full lifecycle acceptance."
