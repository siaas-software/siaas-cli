package tests

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

var binary, root string

func TestMain(m *testing.M) {
	var err error
	// macOS exposes /var through a symlink. Canonicalize only test temp storage;
	// the production installer's refusal of symlink paths stays unchanged.
	if runtime.GOOS == "darwin" {
		resolved, e := filepath.EvalSymlinks(os.TempDir())
		if e != nil {
			panic(e)
		}
		if e = os.Setenv("TMPDIR", resolved); e != nil {
			panic(e)
		}
	}
	root, err = filepath.Abs("..")
	if err != nil {
		panic(err)
	}
	dir, err := os.MkdirTemp("", "siaas-cli-standalone-")
	if err != nil {
		panic(err)
	}
	binary = filepath.Join(dir, "siaas-cli_0.1.0_"+runtime.GOOS+"_"+runtime.GOARCH)
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	goTool := filepath.Join(runtime.GOROOT(), "bin", "go")
	if runtime.GOOS == "windows" {
		goTool += ".exe"
	}
	cmd := exec.Command(goTool, "run", "./tools/build", "-out", dir)
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0", "GOTOOLCHAIN=local", "GOPROXY=off", "GOSUMDB=off", "GOTELEMETRY=off")
	if out, err := cmd.CombinedOutput(); err != nil {
		fmt.Fprintln(os.Stderr, string(out), err)
		os.RemoveAll(dir)
		os.Exit(1)
	}
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}

func TestNativePlatformAndReleaseContract(t *testing.T) {
	t.Logf("Native test process: %s/%s; compiler %s", runtime.GOOS, runtime.GOARCH, runtime.Version())
	for key, actual := range map[string]string{"SIAAS_EXPECTED_OS": runtime.GOOS, "SIAAS_EXPECTED_ARCH": runtime.GOARCH} {
		if expected := os.Getenv(key); expected != "" && expected != actual {
			t.Fatalf("%s: expected %s, got %s", key, expected, actual)
		}
	}
	data, err := os.ReadFile(binary)
	if err != nil {
		t.Fatal(err)
	}
	name := filepath.Base(binary)
	manifest, err := os.ReadFile(filepath.Join(filepath.Dir(binary), "cli-release-v1.txt"))
	if err != nil {
		t.Fatal(err)
	}
	expected := fmt.Sprintf("siaas-cli-release-v1 0.1.0\n%s %s %s %x %d\n", runtime.GOOS, runtime.GOARCH, name, sha256.Sum256(data), len(data))
	if string(manifest) != expected {
		t.Fatalf("release manifest mismatch: %s", manifest)
	}
	sums, err := os.ReadFile(filepath.Join(filepath.Dir(binary), "SHA256SUMS"))
	if err != nil {
		t.Fatal(err)
	}
	if string(sums) != fmt.Sprintf("%x  %s\n", sha256.Sum256(data), name) {
		t.Fatal("checksum list mismatch")
	}
	t.Logf("Verified native release: %s; bytes=%d; SHA256=%x", name, len(data), sha256.Sum256(data))
}

func TestStandaloneWithoutRuntime(t *testing.T) {
	// Copy only the binary into a new directory, and give it an empty PATH/home.
	dir := t.TempDir()
	copy := filepath.Join(dir, filepath.Base(binary))
	data, err := os.ReadFile(binary)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(copy, data, 0755); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(copy, "--version", "--json")
	cmd.Dir = dir
	cmd.Env = []string{"PATH=" + filepath.Join(dir, "empty"), "HOME=" + dir, "USERPROFILE=" + dir, "NODE_OPTIONS=--require=secret-sentinel"}
	if runtime.GOOS == "windows" {
		cmd.Env = append(cmd.Env, "SystemRoot="+os.Getenv("SystemRoot"))
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("standalone: %s %v", out, err)
	}
	var r struct {
		OK      bool
		Version string
	}
	if err = json.Unmarshal(out, &r); err != nil || !r.OK || r.Version != "0.1.0" {
		t.Fatalf("invalid standalone output: %s", out)
	}
	if strings.Contains(string(out), "secret-sentinel") {
		t.Fatal("environment secret leaked")
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Fatal("CLI unexpectedly wrote files")
	}
}

func TestStandaloneExitCodes(t *testing.T) {
	for _, tc := range []struct {
		args []string
		exit int
	}{{[]string{"--help"}, 0}, {[]string{"help"}, 0}, {[]string{"--json", "unknown"}, 2}, {[]string{"--json", "install", "personal"}, 3}} {
		cmd := exec.Command(binary, tc.args...)
		out, err := cmd.CombinedOutput()
		code := 0
		if err != nil {
			if e, ok := err.(*exec.ExitError); ok {
				code = e.ExitCode()
			} else {
				t.Fatal(err)
			}
		}
		if code != tc.exit || len(out) == 0 {
			t.Fatalf("args %v exit %d output %s", tc.args, code, out)
		}
	}
}
