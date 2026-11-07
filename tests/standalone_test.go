package tests

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

var binary, root string

func TestMain(m *testing.M) {
	var err error
	root, err = filepath.Abs("..")
	if err != nil {
		panic(err)
	}
	dir, err := os.MkdirTemp("", "siaas-cli-standalone-")
	if err != nil {
		panic(err)
	}
	binary = filepath.Join(dir, "siaas")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	goTool := filepath.Join(runtime.GOROOT(), "bin", "go")
	if runtime.GOOS == "windows" {
		goTool += ".exe"
	}
	cmd := exec.Command(goTool, "build", "-trimpath", "-buildvcs=false", "-ldflags=-s -w -buildid=", "-o", binary, "./cmd/siaas")
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
