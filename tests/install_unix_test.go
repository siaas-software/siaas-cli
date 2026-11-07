//go:build linux

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
	"syscall"
	"testing"
	"time"
)

func fixture(t *testing.T) (string, string) {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "release café space")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(binary)
	if err != nil {
		t.Fatal(err)
	}
	name := "siaas-cli_0.1.0_linux_" + runtime.GOARCH
	write(t, filepath.Join(dir, name), data, 0755)
	manifest := fmt.Sprintf("siaas-cli-release-v1 0.1.0\nlinux %s %s %x %d\n", runtime.GOARCH, name, sha256.Sum256(data), len(data))
	write(t, filepath.Join(dir, "cli-release-v1.txt"), []byte(manifest), 0600)
	return dir, name
}
func write(t *testing.T, path string, data []byte, mode os.FileMode) {
	t.Helper()
	if err := os.WriteFile(path, data, mode); err != nil {
		t.Fatal(err)
	}
}
func runInstaller(t *testing.T, dir, source string, extraEnv []string, args ...string) (int, string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	all := []string{filepath.Join(root, "install.sh"), "--install-dir", dir, "--from-dir", source}
	all = append(all, args...)
	cmd := exec.CommandContext(ctx, "/bin/bash", all...)
	cmd.Env = append(os.Environ(), "SIAAS_TOKEN=environment-secret-sentinel")
	cmd.Env = append(cmd.Env, extraEnv...)
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
func TestUnixInstallRepeatAndReplacement(t *testing.T) {
	source, _ := fixture(t)
	dir := filepath.Join(t.TempDir(), "CLI 空間 café")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	sentinel := filepath.Join(dir, "personal.sqlite")
	write(t, sentinel, []byte("preserve application data"), 0600)
	target := filepath.Join(dir, "siaas")
	write(t, target, []byte("previous CLI"), 0755)
	for i := 0; i < 2; i++ {
		if code, out := runInstaller(t, dir, source, nil); code != 0 {
			t.Fatalf("%d: %s", code, out)
		}
	}
	backups, _ := filepath.Glob(filepath.Join(dir, "siaas.backup.*"))
	if len(backups) != 1 {
		t.Fatalf("repeat created extra backups: %v", backups)
	}
	old, _ := os.ReadFile(backups[0])
	if string(old) != "previous CLI" {
		t.Fatal("old CLI not preserved")
	}
	saved, _ := os.ReadFile(sentinel)
	if string(saved) != "preserve application data" {
		t.Fatal("application data changed")
	}
	cmd := exec.Command(target, "--version")
	if out, err := cmd.Output(); err != nil || string(out) != "siaas 0.1.0\n" {
		t.Fatalf("installed binary: %s %v", out, err)
	}
	stages, _ := filepath.Glob(filepath.Join(dir, ".siaas-cli-*"))
	if len(stages) != 0 {
		t.Fatalf("owned staging leaked: %v", stages)
	}
}
func TestUnixInstallerFailuresPreserveExisting(t *testing.T) {
	for _, mode := range []string{"corruption", "truncated", "missing binary", "missing manifest", "wrong schema", "wrong version", "missing platform", "duplicate platform", "traversal", "invalid hash", "symlink artifact"} {
		t.Run(mode, func(t *testing.T) {
			source, name := fixture(t)
			path := filepath.Join(source, name)
			meta := filepath.Join(source, "cli-release-v1.txt")
			data, _ := os.ReadFile(meta)
			text := string(data)
			want := 6
			switch mode {
			case "corruption":
				b, _ := os.ReadFile(path)
				b[0] ^= 255
				write(t, path, b, 0755)
			case "truncated":
				write(t, path, []byte("short"), 0755)
			case "missing binary":
				os.Remove(path)
				want = 7
			case "missing manifest":
				os.Remove(meta)
				want = 7
			case "wrong schema":
				write(t, meta, []byte(strings.Replace(text, "release-v1", "release-v2", 1)), 0600)
			case "wrong version":
				write(t, meta, []byte(strings.Replace(text, "0.1.0", "../escape", 1)), 0600)
			case "missing platform":
				write(t, meta, []byte(strings.ReplaceAll(text, "linux", "darwin")), 0600)
				want = 7
			case "duplicate platform":
				write(t, meta, []byte(text+strings.Split(text, "\n")[1]+"\n"), 0600)
				want = 7
			case "traversal":
				write(t, meta, []byte(strings.Replace(text, name, "../escape", 1)), 0600)
			case "invalid hash":
				fields := strings.Fields(text)
				fields[5] = "invalid"
				write(t, meta, []byte(strings.Join(fields[:2], " ")+"\n"+strings.Join(fields[2:], " ")+"\n"), 0600)
			case "symlink artifact":
				os.Rename(path, path+".real")
				os.Symlink(path+".real", path)
				want = 7
			}
			dir := t.TempDir()
			target := filepath.Join(dir, "siaas")
			write(t, target, []byte("old sentinel"), 0755)
			code, out := runInstaller(t, dir, source, nil)
			if code != want {
				t.Fatalf("exit %d want %d: %s", code, want, out)
			}
			old, _ := os.ReadFile(target)
			if string(old) != "old sentinel" {
				t.Fatal("existing target changed")
			}
		})
	}
}
func TestUnixPathSafetyAndRedaction(t *testing.T) {
	source, _ := fixture(t)
	for _, dir := range []string{"/", "relative", "/tmp/../secret-sentinel", "/tmp/secret-sentinel\nnew"} {
		code, out := runInstaller(t, dir, source, nil)
		if code != 2 || strings.Contains(out, "secret-sentinel") {
			t.Fatalf("unsafe path response %d %s", code, out)
		}
	}
	base := t.TempDir()
	real := filepath.Join(base, "real")
	os.Mkdir(real, 0700)
	link := filepath.Join(base, "link")
	os.Symlink(real, link)
	if code, out := runInstaller(t, filepath.Join(link, "child"), source, nil); code != 6 {
		t.Fatalf("symlink path: %d %s", code, out)
	}
	dir := t.TempDir()
	outside := filepath.Join(t.TempDir(), "personal.sqlite")
	write(t, outside, []byte("private"), 0600)
	os.Symlink(outside, filepath.Join(dir, "siaas"))
	if code, out := runInstaller(t, dir, source, nil); code != 6 {
		t.Fatalf("symlink target: %d %s", code, out)
	}
	if b, _ := os.ReadFile(outside); string(b) != "private" {
		t.Fatal("symlink destination changed")
	}
	code, out := runInstaller(t, t.TempDir(), source, nil, "--token=secret-sentinel")
	if code != 2 || strings.Contains(out, "secret-sentinel") {
		t.Fatalf("secret leaked %s", out)
	}
}
func TestUnixUnsupportedPlatformAndBusyLock(t *testing.T) {
	source, _ := fixture(t)
	for _, value := range []string{"FreeBSD", "riscv64"} {
		bin := t.TempDir()
		script := "#!/bin/sh\ncase \"$1\" in -s) echo Linux;; -m) echo " + value + ";; esac\n"
		if value == "FreeBSD" {
			script = "#!/bin/sh\necho FreeBSD\n"
		}
		write(t, filepath.Join(bin, "uname"), []byte(script), 0755)
		code, out := runInstaller(t, t.TempDir(), source, []string{"PATH=" + bin + ":/usr/bin:/bin"})
		if code != 2 {
			t.Fatalf("unsupported: %d %s", code, out)
		}
	}
	dir := t.TempDir()
	os.Mkdir(filepath.Join(dir, ".siaas-cli-install.lock"), 0700)
	code, out := runInstaller(t, dir, source, nil)
	if code != 4 {
		t.Fatalf("lock: %d %s", code, out)
	}
}

func TestUnixUnpublishedReleaseUsesTrustedHTTPS(t *testing.T) {
	bin := t.TempDir()
	record := filepath.Join(bin, "arguments")
	write(t, filepath.Join(bin, "curl"), []byte("#!/bin/sh\nprintf '%s\\n' \"$@\" > \"$SIAAS_TEST_RECORD\"\nexit 22\n"), 0755)
	dir := t.TempDir()
	cmd := exec.Command("/bin/bash", filepath.Join(root, "install.sh"), "--install-dir", dir)
	cmd.Env = append(os.Environ(), "PATH="+bin+":/usr/bin:/bin", "SIAAS_TEST_RECORD="+record)
	out, err := cmd.CombinedOutput()
	if e, ok := err.(*exec.ExitError); !ok || e.ExitCode() != 7 {
		t.Fatalf("unpublished release: %s %v", out, err)
	}
	args, _ := os.ReadFile(record)
	text := string(args)
	for _, required := range []string{"--proto\n=https", "--proto-redir\n=https", "--max-filesize", "https://github.com/siaas-software/siaas-cli/releases/latest/download/cli-release-v1.txt"} {
		if !strings.Contains(text, required) {
			t.Fatalf("missing transport policy %s", required)
		}
	}
	if strings.Contains(text, "--insecure") || strings.Contains(text, "\n-k\n") {
		t.Fatal("TLS bypass")
	}
	if _, err := os.Stat(filepath.Join(dir, "siaas")); !os.IsNotExist(err) {
		t.Fatal("failed download installed a target")
	}
}

func TestUnixFilesystemErrorRedaction(t *testing.T) {
	source, _ := fixture(t)
	bin := t.TempDir()
	write(t, filepath.Join(bin, "chmod"), []byte("#!/bin/sh\necho 'private-secret-sentinel' >&2\nexit 1\n"), 0755)
	code, out := runInstaller(t, t.TempDir(), source, []string{"PATH=" + bin + ":/usr/bin:/bin"})
	if code != 6 || strings.Contains(out, "secret-sentinel") {
		t.Fatalf("filesystem diagnostic leaked: %d %s", code, out)
	}
}
func TestUnixInterruptedInstall(t *testing.T) {
	for _, signal := range []syscall.Signal{syscall.SIGTERM, syscall.SIGKILL} {
		t.Run(signal.String(), func(t *testing.T) {
			source, _ := fixture(t)
			dir := t.TempDir()
			target := filepath.Join(dir, "siaas")
			write(t, target, []byte("old CLI sentinel"), 0755)
			bin := t.TempDir()
			marker := filepath.Join(bin, "ready")
			// Pause the real installer's final rename via a fixture-only PATH adapter.
			write(t, filepath.Join(bin, "mv"), []byte("#!/bin/sh\nprintf ready > \"$SIAAS_TEST_MARKER\"\nexec /bin/sleep 30\n"), 0755)
			cmd := exec.Command("/bin/bash", filepath.Join(root, "install.sh"), "--install-dir", dir, "--from-dir", source)
			cmd.Env = append(os.Environ(), "PATH="+bin+":/usr/bin:/bin", "SIAAS_TEST_MARKER="+marker)
			cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
			if err := cmd.Start(); err != nil {
				t.Fatal(err)
			}
			waited := false
			defer func() {
				if !waited {
					_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
					_ = cmd.Wait()
				}
			}()
			deadline := time.Now().Add(5 * time.Second)
			for {
				if _, err := os.Stat(marker); err == nil {
					break
				}
				if time.Now().After(deadline) {
					t.Fatal("commit barrier not reached")
				}
				time.Sleep(10 * time.Millisecond)
			}
			if err := syscall.Kill(-cmd.Process.Pid, signal); err != nil {
				t.Fatal(err)
			}
			waitErr := cmd.Wait()
			waited = true
			exit, ok := waitErr.(*exec.ExitError)
			if !ok || signal == syscall.SIGTERM && exit.ExitCode() != 143 {
				t.Fatalf("unexpected interruption result: %v", waitErr)
			}
			old, _ := os.ReadFile(target)
			if string(old) != "old CLI sentinel" {
				t.Fatal("interruption replaced old CLI")
			}
			if signal == syscall.SIGKILL {
				if code, _ := runInstaller(t, dir, source, nil); code != 4 {
					t.Fatal("stale lock must require inspection")
				}
				// Test operator recovery only in this owned temporary fixture after process exit.
				os.Remove(filepath.Join(dir, ".siaas-cli-install.lock"))
				stages, _ := filepath.Glob(filepath.Join(dir, ".siaas-cli-stage.*"))
				for _, s := range stages {
					os.RemoveAll(s)
				}
			}
			if code, out := runInstaller(t, dir, source, nil); code != 0 {
				t.Fatalf("recovery %d %s", code, out)
			}
		})
	}
}
