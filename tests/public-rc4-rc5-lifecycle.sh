#!/usr/bin/env bash
# Real compiled CLI lifecycle against published immutable RC4 and RC5.
set -euo pipefail
test "$(id -u)" -ne 0
test "$(go env GOHOSTOS)/$(go env GOHOSTARCH)" = linux/amd64
test "$(go version | cut -d ' ' -f 3)" = go1.27.3
root=$(mktemp -d)
trap 'rm -rf -- "$root"' EXIT
go build -o "$root/siaas" ./cmd/siaas
sha256sum "$root/siaas"
export HOME="$root/home"
mkdir -m 700 "$HOME"
base="$HOME/.local/share/siaas-cli/personal"
candidate="$base/candidate"
mkdir -p "$base/stable" "$HOME/.ollama"
printf stable > "$base/stable/sentinel"
printf ollama > "$HOME/.ollama/sentinel"
run() {
  local label=$1
  shift
  if ! "$root/siaas" "$@" --json > "$root/$label.json"; then
    cat "$root/$label.json"
    if [[ -f "$candidate/cli-build.log" ]]; then tail -c 6000 "$candidate/cli-build.log"; fi
    return 1
  fi
  cat "$root/$label.json"
}
receipt() {
  python3 - "$root/$1.json" "$2" "$3" <<'PY'
import json, sys
with open(sys.argv[1]) as f: r=json.load(f)
assert r["ok"] and r["version"]=="0.2.1"
p=r["personal"]
if sys.argv[2]=="check":
    assert p["sourceVerified"] and p["compatible"] and p["immutable"] and "operation" not in p
else:
    op=p["operation"]
    assert op["command"]==sys.argv[2] and op["status"]=="completed"
    assert op["installed"]==(sys.argv[2]!="uninstall")
    assert op["runtimeProvisioned"]==op["installed"]==p["runtimeProvisioned"]
if sys.argv[3]!="-": assert p["version"]==sys.argv[3]
PY
}
run check4 install personal --version 1.1.0-rc.4 --check
receipt check4 check 1.1.0-rc.4
run install4 install personal --version 1.1.0-rc.4 --yes
receipt install4 install 1.1.0-rc.4
mkdir -p "$candidate/profile"
printf preserved > "$candidate/profile/sentinel"
run repeat4 install personal --version 1.1.0-rc.4 --yes
receipt repeat4 install 1.1.0-rc.4
run check5 update personal --version 1.1.0-rc.5 --check
receipt check5 check 1.1.0-rc.5
cp "$candidate/current.json" "$root/before.json"
if "$root/siaas" install personal --version 1.1.0-rc.5 --yes --json > "$root/refused.json"; then
  echo "Newer install unexpectedly succeeded"; exit 1
fi
cmp "$root/before.json" "$candidate/current.json"
run update5 update personal --version 1.1.0-rc.5 --yes
receipt update5 update 1.1.0-rc.5
python3 - "$candidate" <<'PY'
import json, pathlib, sys
p=pathlib.Path(sys.argv[1])
r=json.loads((p/"current.json").read_text())
assert r["version"]=="1.1.0-rc.5" and r["previous"]["version"]=="1.1.0-rc.4"
assert (p/"versions"/r["previous"]["folder"]).is_dir()
assert (p/"profile/sentinel").read_text()=="preserved"
assert any(f.read_text()=="preserved" for f in (p/"backups").rglob("sentinel"))
PY
cp "$candidate/current.json" "$root/after.json"
if "$root/siaas" update personal --version 1.1.0-rc.4 --yes --json > "$root/downgrade.json"; then
  echo "Downgrade unexpectedly succeeded"; exit 1
fi
cmp "$root/after.json" "$candidate/current.json"
for i in 1 2; do
  run "uninstall$i" uninstall personal --rc --yes
  receipt "uninstall$i" uninstall -
done
for name in current.json versions runtimes build-home launch-rc.sh; do
  test ! -e "$candidate/$name"
done
test "$(cat "$candidate/profile/sentinel")" = preserved
test "$(cat "$base/stable/sentinel")" = stable
test "$(cat "$HOME/.ollama/sentinel")" = ollama
echo "Real public compiled CLI RC4 -> RC5 lifecycle PASS"
