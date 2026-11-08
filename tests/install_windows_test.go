//go:build windows

package tests

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestWindowsPowerShell51(t *testing.T) {
	cmd := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-Command", `$v=$PSVersionTable.PSVersion; if ($v.Major -ne 5 -or $v.Minor -ne 1 -or $PSVersionTable.PSEdition -ne 'Desktop') { exit 1 }; Write-Output ('Windows PowerShell '+$v.ToString()+' '+$PSVersionTable.PSEdition)`)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("PowerShell 5.1 required: %s %v", out, err)
	}
	t.Log(strings.TrimSpace(string(out)))
}

func TestWindowsRunningExecutableReplacement(t *testing.T) {
	source, _ := winFixture(t)
	dir := filepath.Join(t.TempDir(), "Running CLI space")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(dir, "siaas.exe")
	helper := filepath.Join(t.TempDir(), "main.go")
	winWrite(t, helper, []byte("package main\nimport(\"fmt\";\"os\")\nfunc main(){fmt.Println(\"ready\"); var b [1]byte; os.Stdin.Read(b[:])}\n"))
	build := exec.Command(filepath.Join(runtime.GOROOT(), "bin", "go.exe"), "build", "-o", target, helper)
	build.Env = append(os.Environ(), "CGO_ENABLED=0", "GOTOOLCHAIN=local", "GOPROXY=off", "GOSUMDB=off")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("running-image fixture: %s %v", out, err)
	}
	old, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	process := exec.CommandContext(ctx, target)
	input, err := process.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	output, err := process.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err = process.Start(); err != nil {
		t.Fatal(err)
	}
	waited := false
	defer func() {
		if !waited {
			_ = process.Process.Kill()
			_ = process.Wait()
		}
	}()
	line, err := bufio.NewReader(output).ReadString('\n')
	if err != nil || strings.TrimSpace(line) != "ready" {
		t.Fatalf("running-image barrier: %q %v", line, err)
	}
	code, out := winInstall(t, dir, source)
	t.Logf("Running executable replacement exit=%d: %s", code, strings.TrimSpace(out))
	if code != 0 && code != 6 {
		t.Fatalf("unexpected running-image result: %d %s", code, out)
	}
	if code == 6 {
		current, err := os.ReadFile(target)
		if err != nil || !bytes.Equal(current, old) {
			t.Fatal("failed running-image replacement changed old bytes")
		}
	}
	_ = input.Close()
	if err = process.Wait(); err != nil {
		t.Fatal(err)
	}
	waited = true
	if code, out = winInstall(t, dir, source); code != 0 {
		t.Fatalf("retry after process exit: %d %s", code, out)
	}
	want, err := os.ReadFile(binary)
	if err != nil {
		t.Fatal(err)
	}
	current, err := os.ReadFile(target)
	if err != nil || !bytes.Equal(current, want) {
		t.Fatal("retry did not install verified CLI")
	}
	backups, _ := filepath.Glob(filepath.Join(dir, "siaas.exe.backup.*"))
	retained := false
	for _, backup := range backups {
		b, err := os.ReadFile(backup)
		if err == nil && bytes.Equal(b, old) {
			retained = true
		}
	}
	if !retained {
		t.Fatal("previous running executable not retained for recovery")
	}
}

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

func TestWindowsLockedTargetPreservesBackup(t *testing.T) {
	source, _ := winFixture(t)
	dir := t.TempDir()
	target := filepath.Join(dir, "siaas.exe")
	backup := filepath.Join(dir, "siaas.exe.backup.previous")
	winWrite(t, target, []byte("old CLI"))
	winWrite(t, backup, []byte("older CLI"))
	p, err := syscall.UTF16PtrFromString(target)
	if err != nil {
		t.Fatal(err)
	}
	h, err := syscall.CreateFile(p, syscall.GENERIC_READ, 0, nil, syscall.OPEN_EXISTING, syscall.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if h != syscall.InvalidHandle {
			_ = syscall.CloseHandle(h)
		}
	}()
	if code, out := winInstall(t, dir, source); code != 6 {
		t.Fatalf("locked target %d %s", code, out)
	}
	if err := syscall.CloseHandle(h); err != nil {
		t.Fatal(err)
	}
	h = syscall.InvalidHandle
	old, _ := os.ReadFile(target)
	retained, _ := os.ReadFile(backup)
	if string(old) != "old CLI" || string(retained) != "older CLI" {
		t.Fatal("locked replacement lost previous bytes")
	}
	if code, out := winInstall(t, dir, source); code != 0 {
		t.Fatalf("retry %d %s", code, out)
	}
	backups, _ := filepath.Glob(filepath.Join(dir, "siaas.exe.backup.*"))
	if len(backups) != 2 {
		t.Fatal("retry pruned or lost a backup")
	}
}

func TestWindowsAmbiguousPaths(t *testing.T) {
	source, _ := winFixture(t)
	for _, suffix := range []string{"stream:secret-sentinel", "trailing. ", "NUL", "COM1.txt", "parent/../child"} {
		if code, out := winInstall(t, t.TempDir()+`\`+suffix, source); code != 2 || strings.Contains(out, "secret-sentinel") {
			t.Fatalf("ambiguous path %d %s", code, out)
		}
	}
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
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	all := []string{"-NoProfile", "-NonInteractive", "-File", filepath.Join(root, "install.ps1"), "-FromDir", source}
	if dir != "" {
		all = append(all, "-InstallDir", dir)
	}
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
	installed := exec.Command(filepath.Join(dir, "siaas.exe"), "--version", "--json")
	installed.Env = []string{"PATH=" + filepath.Join(dir, "empty"), "SystemRoot=" + os.Getenv("SystemRoot")}
	if out, err := installed.CombinedOutput(); err != nil || !strings.Contains(string(out), `"version":"0.1.0"`) {
		t.Fatalf("installed standalone CLI: %s %v", out, err)
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
	for _, kind := range []string{"corrupt", "truncated", "oversized", "missing artifact", "missing metadata", "bad schema", "unsafe filename", "missing platform", "duplicate platform", "stale lock", "unsafe path", "secret argument"} {
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
				b, err := os.ReadFile(filepath.Join(source, name))
				if err != nil {
					t.Fatal(err)
				}
				b[0] ^= 255
				winWrite(t, filepath.Join(source, name), b)
			case "truncated":
				winWrite(t, filepath.Join(source, name), []byte("short"))
			case "oversized":
				b, err := os.ReadFile(filepath.Join(source, name))
				if err != nil {
					t.Fatal(err)
				}
				winWrite(t, filepath.Join(source, name), append(b, 0))
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
