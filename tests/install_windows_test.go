//go:build windows

package tests

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// These tests run only on a genuine Windows host. Cross-compilation is not a pass.
func winFixture(t *testing.T) (string, string) {
	t.Helper()
	dir := t.TempDir()
	data, err := os.ReadFile(binary)
	if err != nil {
		t.Fatal(err)
	}
	name := "siaas-cli_0.1.0_windows_" + runtime.GOARCH + ".exe"
	winWrite(t, filepath.Join(dir, name), data)
	winWrite(t, filepath.Join(dir, "cli-release-v1.txt"), []byte(fmt.Sprintf("siaas-cli-release-v1 0.1.0\nwindows %s %s %x %d\n", runtime.GOARCH, name, sha256.Sum256(data), len(data))))
	return dir, name
}
func winWrite(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
}
func winInstall(t *testing.T, dir, source string, args ...string) (int, string) {
	t.Helper()
	shell, err := exec.LookPath("powershell.exe")
	if err != nil {
		t.Fatal("Windows PowerShell 5.1 required for native installer acceptance")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	all := []string{"-NoProfile", "-NonInteractive", "-File", filepath.Join(root, "install.ps1"), "-InstallDir", dir, "-FromDir", source}
	all = append(all, args...)
	cmd := exec.CommandContext(ctx, shell, all...)
	cmd.Env = append(os.Environ(), "SIAAS_TOKEN=environment-secret-sentinel")
	out, err := cmd.CombinedOutput()
	if ctx.Err() != nil {
		t.Fatal(ctx.Err())
	}
	code := 0
	if err != nil {
		if e, ok := err.(*exec.ExitError); ok {
			code = e.ExitCode()
		} else {
			t.Fatal(err)
		}
	}
	return code, string(out)
}
func TestWindowsBootstrap(t *testing.T) {
	source, _ := winFixture(t)
	dir := filepath.Join(t.TempDir(), "CLI café 空間")
	if code, out := winInstall(t, dir, source); code != 0 {
		t.Fatalf("install %d %s", code, out)
	}
	if code, out := winInstall(t, dir, source); code != 0 || !strings.Contains(out, "already installed") {
		t.Fatalf("repeat %d %s", code, out)
	}
	target := filepath.Join(dir, "siaas.exe")
	winWrite(t, target, []byte("previous CLI"))
	sentinel := filepath.Join(dir, "personal.sqlite")
	winWrite(t, sentinel, []byte("preserved"))
	if code, out := winInstall(t, dir, source); code != 0 {
		t.Fatalf("replace %d %s", code, out)
	}
	backups, _ := filepath.Glob(filepath.Join(dir, "siaas.exe.backup.*"))
	if len(backups) != 1 {
		t.Fatal("backup missing")
	}
	old, _ := os.ReadFile(backups[0])
	if string(old) != "previous CLI" {
		t.Fatal("old CLI lost")
	}
	saved, _ := os.ReadFile(sentinel)
	if string(saved) != "preserved" {
		t.Fatal("product data changed")
	}
}
func TestWindowsInstallerFailures(t *testing.T) {
	for _, kind := range []string{"corrupt", "missing artifact", "missing metadata", "bad schema", "unsafe filename", "missing platform", "duplicate platform", "stale lock", "unsafe path", "secret argument"} {
		t.Run(kind, func(t *testing.T) {
			source, name := winFixture(t)
			dir := t.TempDir()
			target := filepath.Join(dir, "siaas.exe")
			winWrite(t, target, []byte("old sentinel"))
			meta := filepath.Join(source, "cli-release-v1.txt")
			data, _ := os.ReadFile(meta)
			text := string(data)
			want := 6
			var args []string
			switch kind {
			case "corrupt":
				winWrite(t, filepath.Join(source, name), []byte("corrupt"))
			case "missing artifact":
				os.Remove(filepath.Join(source, name))
				want = 7
			case "missing metadata":
				os.Remove(meta)
				want = 7
			case "bad schema":
				winWrite(t, meta, []byte(strings.Replace(text, "release-v1", "release-v2", 1)))
			case "unsafe filename":
				winWrite(t, meta, []byte(strings.Replace(text, name, "../secret-sentinel", 1)))
			case "missing platform":
				winWrite(t, meta, []byte(strings.ReplaceAll(strings.ReplaceAll(text, "windows", "linux"), ".exe", "")))
				want = 7
			case "duplicate platform":
				winWrite(t, meta, []byte(text+strings.Split(text, "\n")[1]+"\n"))
				want = 7
			case "stale lock":
				winWrite(t, filepath.Join(dir, ".siaas-cli-install.lock"), []byte("interrupted owner"))
				want = 4
			case "unsafe path":
				args = []string{"-InstallDir", "relative-secret-sentinel"}
				want = 2
			case "secret argument":
				args = []string{"-secret-sentinel"}
				want = 2
			}
			code, out := winInstall(t, dir, source, args...)
			if code != want || strings.Contains(out, "secret-sentinel") {
				t.Fatalf("exit %d want %d: %s", code, want, out)
			}
			old, _ := os.ReadFile(target)
			if string(old) != "old sentinel" {
				t.Fatal("old CLI changed on failure")
			}
			if kind == "stale lock" {
				os.Remove(filepath.Join(dir, ".siaas-cli-install.lock"))
				if code, out := winInstall(t, dir, source); code != 0 {
					t.Fatalf("inspected-lock recovery %d %s", code, out)
				}
			}
		})
	}
}
