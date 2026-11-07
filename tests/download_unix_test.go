//go:build linux || darwin

package tests

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// The adapter models curl's status/redirect_url output; it never opens a socket.
// Logs prove rejected redirect destinations are never passed to the transport.
func TestUnixDownloadPolicy(t *testing.T) {
	for _, tc := range []struct {
		name, redirect, mode string
		code, calls          int
	}{
		{"github-assets", "https://release-assets.githubusercontent.com/asset?signature=secret-sentinel", "success", 0, 4},
		{"legacy-assets", "https://objects.githubusercontent.com/asset?signature=secret-sentinel", "success", 0, 4},
		{"github-relative-resolved-by-curl", "https://github.com/siaas-software/siaas-cli/releases/download/v0.1.0/asset", "success", 0, 4},
		{"explicit-tls-port", "https://release-assets.githubusercontent.com:443/asset", "success", 0, 4},
		{"arbitrary-host", "https://evil.example/secret-sentinel", "redirect", 6, 1},
		{"suffix-spoof", "https://github.com.evil.example/secret-sentinel", "redirect", 6, 1},
		{"userinfo", "https://github.com@evil.example/secret-sentinel", "redirect", 6, 1},
		{"credentials", "https://secret-sentinel@github.com/asset", "redirect", 6, 1},
		{"downgrade", "http://github.com/secret-sentinel", "redirect", 6, 1},
		{"file-url", "file:///secret-sentinel", "redirect", 6, 1},
		{"port", "https://github.com:444/secret-sentinel", "redirect", 6, 1},
		{"fragment", "https://github.com/asset#secret-sentinel", "redirect", 6, 1},
		{"backslash", "https://github.com/\\secret-sentinel", "redirect", 6, 1},
		{"newline", "https://github.com/\nsecret-sentinel", "redirect", 6, 1},
		{"empty-location", "", "redirect", 6, 1},
		{"redirect-loop", "https://github.com/loop", "redirect", 7, 6},
		{"tls-failure", "", "tls", 7, 1},
		{"http-failure", "", "http", 7, 1},
		{"partial-binary", "https://release-assets.githubusercontent.com/asset", "partial", 7, 4},
		{"short-binary-success", "https://release-assets.githubusercontent.com/asset", "short", 6, 4},
		{"unexpected-status", "", "status", 7, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source, name := fixture(t)
			bin, dir := t.TempDir(), t.TempDir()
			record := filepath.Join(bin, "calls")
			write(t, filepath.Join(dir, "siaas"), []byte("old CLI"), 0755)
			write(t, filepath.Join(bin, "curl"), []byte(`#!/bin/bash
set -eu
[[ "$1" == --disable ]] || exit 90
printf 'call\n' >> "$SIAAS_TEST_RECORD"
destination=''
while [[ $# -gt 0 ]]; do
  case "$1" in
    --location|-L|--insecure|-k) exit 91 ;;
    -o) destination="$2"; shift ;;
  esac
  shift
done
count=$(wc -l < "$SIAAS_TEST_RECORD")
case "$SIAAS_TEST_MODE" in
  tls) echo secret-sentinel >&2; exit 60 ;;
  http) echo secret-sentinel >&2; exit 22 ;;
  status) printf '204\n'; exit 0 ;;
  redirect) printf '302\n%s' "$SIAAS_TEST_REDIRECT"; exit 0 ;;
esac
if [[ "$count" == 1 || "$count" == 3 ]]; then
  printf '302\n%s' "$SIAAS_TEST_REDIRECT"
elif [[ "$count" == 2 ]]; then
  cp "$SIAAS_TEST_SOURCE/cli-release-v1.txt" "$destination"
  printf '200\n'
else
  case "$SIAAS_TEST_MODE" in
    partial) printf partial > "$destination"; echo secret-sentinel >&2; exit 18 ;;
    short) printf partial > "$destination" ;;
    *) cp "$SIAAS_TEST_SOURCE/$SIAAS_TEST_NAME" "$destination" ;;
  esac
  printf '200\n'
fi
`), 0755)
			cmd := exec.Command("/bin/bash", filepath.Join(root, "install.sh"), "--install-dir", dir)
			cmd.Env = append(os.Environ(), "PATH="+bin+":/usr/bin:/bin", "SIAAS_TEST_RECORD="+record,
				"SIAAS_TEST_SOURCE="+source, "SIAAS_TEST_NAME="+name, "SIAAS_TEST_MODE="+tc.mode, "SIAAS_TEST_REDIRECT="+tc.redirect)
			out, err := cmd.CombinedOutput()
			code := 0
			if err != nil {
				if e, ok := err.(*exec.ExitError); ok {
					code = e.ExitCode()
				} else {
					t.Fatal(err)
				}
			}
			if code != tc.code || strings.Contains(string(out), "secret-sentinel") {
				t.Fatalf("exit %d want %d: %s", code, tc.code, out)
			}
			calls, _ := os.ReadFile(record)
			if strings.Count(string(calls), "call\n") != tc.calls {
				t.Fatalf("unexpected transport calls: %s", calls)
			}
			if tc.code != 0 {
				old, _ := os.ReadFile(filepath.Join(dir, "siaas"))
				if string(old) != "old CLI" {
					t.Fatal("failed download replaced CLI")
				}
			}
			stages, _ := filepath.Glob(filepath.Join(dir, ".siaas-cli-*"))
			if len(stages) != 0 {
				t.Fatalf("failed download leaked staging/lock: %v", stages)
			}
		})
	}
}
