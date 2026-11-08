// Build raw, self-contained CLI artifacts and their versioned checksum manifest.
// No release publication or network operation is performed.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/siaas-software/siaas-cli/internal/cli"
)

func main() {
	if err := build(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func build() error {
	targets := flag.String("targets", runtime.GOOS+"/"+runtime.GOARCH, "comma-separated OS/architecture targets")
	out := flag.String("out", "dist", "output directory")
	flag.Parse()
	if flag.NArg() != 0 {
		return fmt.Errorf("unexpected build arguments")
	}
	pin, err := os.ReadFile(".go-version")
	if err != nil {
		return err
	}
	if runtime.Version() != "go"+strings.TrimSpace(string(pin)) {
		return fmt.Errorf("use the Go version pinned in .go-version")
	}
	allowed := map[string]bool{"linux/amd64": true, "linux/arm64": true, "darwin/amd64": true, "darwin/arm64": true, "windows/amd64": true, "windows/arm64": true}
	seen := map[string]bool{}
	list := strings.Split(*targets, ",")
	for _, target := range list {
		if !allowed[target] || seen[target] {
			return fmt.Errorf("unsupported or duplicate build target")
		}
		seen[target] = true
	}
	if err := os.MkdirAll(*out, 0755); err != nil {
		return err
	}
	manifest := "siaas-cli-release-v1 " + cli.Version + "\n"
	checksums := ""
	for _, target := range list {
		parts := strings.Split(target, "/")
		name := "siaas-cli_" + cli.Version + "_" + parts[0] + "_" + parts[1]
		if parts[0] == "windows" {
			name += ".exe"
		}
		path := filepath.Join(*out, name)
		goTool := filepath.Join(runtime.GOROOT(), "bin", "go")
		if runtime.GOOS == "windows" {
			goTool += ".exe"
		}
		cmd := exec.Command(goTool, "build", "-trimpath", "-buildvcs=false", "-ldflags=-s -w -buildid=", "-o", path, "./cmd/siaas")
		// A closed build environment prevents ambient GOFLAGS or alternate toolchains.
		for _, key := range []string{"HOME", "USERPROFILE", "LOCALAPPDATA", "TMPDIR", "TMP", "TEMP", "SystemRoot", "PATH", "GOCACHE"} {
			if value, ok := os.LookupEnv(key); ok {
				cmd.Env = append(cmd.Env, key+"="+value)
			}
		}
		cmd.Env = append(cmd.Env, "GOOS="+parts[0], "GOARCH="+parts[1], "CGO_ENABLED=0", "GOTOOLCHAIN=local", "GOPROXY=off", "GOSUMDB=off", "GOENV=off", "GOTELEMETRY=off")
		cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
		if err := cmd.Run(); err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		digest := sha256.Sum256(data)
		sum := hex.EncodeToString(digest[:])
		manifest += fmt.Sprintf("%s %s %s %s %d\n", parts[0], parts[1], name, sum, len(data))
		checksums += sum + "  " + name + "\n"
		fmt.Printf("Built %s (%d bytes)\n", name, len(data))
	}
	if err := os.WriteFile(filepath.Join(*out, "SHA256SUMS"), []byte(checksums), 0644); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(*out, "cli-release-v1.txt"), []byte(manifest), 0644)
}
