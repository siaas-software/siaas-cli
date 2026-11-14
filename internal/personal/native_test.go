package personal

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// Explicitly opt-in: actual official source npm build/packaging, no fake builder.
func TestFirstPartyRealLifecycle(t *testing.T) {
	source := os.Getenv("SIAAS_M2_TRUSTED_SOURCE_TEST")
	if source == "" {
		t.Skip("requires explicit authorized first-party source fixture")
	}
	if !filepath.IsAbs(source) {
		t.Fatal("absolute trusted fixture required")
	}
	home := filepath.Join(t.TempDir(), "Home è space")
	if e := os.MkdirAll(home, 0700); e != nil {
		t.Fatal(e)
	}
	t.Setenv("HOME", home)
	root, e := ManagedRoot("candidate")
	if e != nil {
		t.Fatal(e)
	}
	c := NewClient()
	// Use the source fixture's pinned runtime, not a workspace-specific
	// previously generated RC manifest. The real test already requires HTTPS
	// for official Node provisioning; obtain its checksum from the same issuer.
	runtimeJSON, e := os.ReadFile(filepath.Join(source, "runtime.json"))
	if e != nil {
		t.Fatal(e)
	}
	var runtimeConfig struct {
		NodeVersion string `json:"nodeVersion"`
	}
	if e := json.Unmarshal(runtimeJSON, &runtimeConfig); e != nil || !validVersion(runtimeConfig.NodeVersion) || strings.Contains(runtimeConfig.NodeVersion, "-") {
		t.Fatal("invalid trusted source runtime.json")
	}
	nodeVersion := runtimeConfig.NodeVersion
	nodeName := "node-v" + nodeVersion + "-linux-x64.tar.gz"
	sums, e := c.nodeBytes(context.Background(), "https://nodejs.org/dist/v"+nodeVersion+"/SHASUMS256.txt", 65536)
	if e != nil {
		t.Fatal(e)
	}
	var nodeHash string
	for _, line := range strings.Split(string(sums), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[1] == nodeName {
			if nodeHash != "" {
				t.Fatal("duplicate official Node checksum entry")
			}
			nodeHash = fields[0]
		}
	}
	if !hashRE.MatchString(nodeHash) {
		t.Fatal("official Node runtime checksum missing or invalid")
	}
	makeRelease := func(version string) *Report {
		var b bytes.Buffer
		w := zip.NewWriter(&b)
		w.SetComment(strings.Repeat("a", 40))
		err := filepath.WalkDir(source, func(p string, d os.DirEntry, e error) error {
			if e != nil {
				return e
			}
			rel, _ := filepath.Rel(source, p)
			rel = filepath.ToSlash(rel)
			if d.IsDir() {
				if rel == ".git" || rel == "node_modules" || rel == ".next" || rel == ".desktop" || rel == "dist" || rel == "dist-source" {
					return filepath.SkipDir
				}
				return nil
			}
			if rel == ".git" {
				return nil
			}
			if strings.HasPrefix(rel, ".env") || strings.HasSuffix(rel, ".tsbuildinfo") {
				return nil
			}
			info, e := d.Info()
			if e != nil {
				return e
			}
			if !info.Mode().IsRegular() {
				return nil
			}
			data, e := os.ReadFile(p)
			if e != nil {
				return e
			}
			if rel == "package.json" || rel == "package-lock.json" {
				var pkg map[string]any
				json.Unmarshal(data, &pkg)
				pkg["version"] = version
				if pkgs, ok := pkg["packages"].(map[string]any); ok {
					pkgs[""].(map[string]any)["version"] = version
				}
				if version == "1.1.1-rc.3" && rel == "package.json" {
					pkg["scripts"].(map[string]any)["desktop:build"] = "node -e 'process.exit(17)'"
				}
				data, _ = json.Marshal(pkg)
			}
			h, e := zip.FileInfoHeader(info)
			if e != nil {
				return e
			}
			h.Name = rel
			h.Method = zip.Deflate
			out, e := w.CreateHeader(h)
			if e != nil {
				return e
			}
			_, e = out.Write(data)
			return e
		})
		if err != nil {
			t.Fatal(err)
		}
		w.Close()
		data := b.Bytes()
		sum := sha256.Sum256(data)
		t.Logf("trusted local source %s: bytes=%d sha256=%x", version, len(data), sum)
		m := manifest{
			Schema: 1, Version: version, Platform: "linux", Arch: "x64",
			Name: "SIAAS-Personal-" + version + "-source.zip",
			Size: int64(len(data)), SHA256: hex.EncodeToString(sum[:]),
		}
		m.URL = releaseBase + "download/v" + version + "/" + m.Name
		m.Runtime.Version = nodeVersion
		m.Runtime.Name = nodeName
		m.Runtime.URL = "https://nodejs.org/dist/v" + nodeVersion + "/" + nodeName
		m.Runtime.SHA256 = nodeHash
		if e := verifyArchive(data, strings.Repeat("a", 40), sourceIndex{1, version, nodeVersion}); e != nil {
			t.Fatal(e)
		}
		return &Report{Version: version, Channel: "candidate", Compatible: true, source: data, runtimeSpec: m}
	}
	logPath := filepath.Join(os.Getenv("SIAAS_M2_EVIDENCE_ROOT"), "engine-real.log")
	log, e := os.Create(logPath)
	if e != nil {
		t.Fatal(e)
	}
	defer log.Close()
	run := func(command string, r *Report) {
		t.Helper()
		if e := c.Execute(context.Background(), command, r, root, log); e != nil {
			log.Sync()
			b, _ := os.ReadFile(logPath)
			tail := len(b) - 6000
			if tail < 0 {
				tail = 0
			}
			t.Fatalf("%s: %v\n%s", command, e, b[tail:])
		}
	}
	binary := filepath.Join(t.TempDir(), "siaas")
	build := exec.Command(filepath.Join(runtime.GOROOT(), "bin", "go"), "build", "-o", binary, "./cmd/siaas")
	build.Dir = "../.."
	if b, e := build.CombinedOutput(); e != nil {
		t.Fatalf("CLI build: %v %s", e, b)
	}
	binaryBytes, e := os.ReadFile(binary)
	if e != nil {
		t.Fatal(e)
	}
	t.Logf("compiled CLI sha256=%x", sha256.Sum256(binaryBytes))
	stableSibling, _ := ManagedRoot("stable")
	if e = os.MkdirAll(stableSibling, 0700); e != nil {
		t.Fatal(e)
	}
	stableSentinel := filepath.Join(stableSibling, "unrelated-stable")
	os.WriteFile(stableSentinel, []byte("stable preserved"), 0600)
	first := makeRelease("1.1.1-rc.1")
	run("install", first)
	profile := filepath.Join(root, "profile")
	os.MkdirAll(profile, 0700)
	sentinel := filepath.Join(profile, "preserved.txt")
	os.WriteFile(sentinel, []byte("unchanged"), 0600)
	state := func() map[string]any {
		b, e := os.ReadFile(filepath.Join(root, "current.json"))
		if e != nil {
			t.Fatal(e)
		}
		var v map[string]any
		json.Unmarshal(b, &v)
		return v
	}
	if state()["version"] != first.Version {
		t.Fatal("wrong installed version")
	}
	if _, e := os.Stat(filepath.Join(root, "launch-rc.sh")); e != nil {
		t.Fatal(e)
	}
	run("install", first)
	second := makeRelease("1.1.1-rc.2")
	run("update", second)
	if state()["version"] != second.Version || state()["previous"].(map[string]any)["version"] != first.Version {
		t.Fatal("wrong update retention")
	}
	if e := c.Execute(context.Background(), "update", first, root, io.Discard); e == nil {
		t.Fatal("downgrade accepted")
	}
	// Fail a genuine newer build and prove current metadata/data remain unchanged.
	previousBytes, _ := os.ReadFile(filepath.Join(root, "current.json"))
	failed := makeRelease("1.1.1-rc.3")
	if e := c.Execute(context.Background(), "update", failed, root, log); e == nil {
		t.Fatal("injected build failure accepted")
	}
	afterFailure, _ := os.ReadFile(filepath.Join(root, "current.json"))
	if !bytes.Equal(previousBytes, afterFailure) {
		t.Fatal("failed update changed active generation")
	}
	if data, e := os.ReadFile(sentinel); e != nil || string(data) != "unchanged" {
		t.Fatal("failed update changed data")
	}
	for i := 0; i < 2; i++ {
		command := exec.Command(binary, "uninstall", "personal", "--rc", "--yes", "--json")
		output, e := command.CombinedOutput()
		if e != nil {
			t.Fatalf("compiled CLI uninstall: %v %s", e, output)
		}
		var receipt struct {
			OK bool `json:"ok"`
		}
		if json.Unmarshal(output, &receipt) != nil || !receipt.OK {
			t.Fatal("invalid CLI receipt", string(output))
		}
		t.Logf("compiled CLI uninstall attempt %d: %s", i+1, output)
	}
	for _, name := range []string{"versions", "runtimes", "build-home"} {
		if _, e := os.Stat(filepath.Join(root, name)); !os.IsNotExist(e) {
			t.Fatalf("owned software retained: %s", name)
		}
	}
	if entries, e := os.ReadDir(filepath.Join(root, "backups")); e != nil || len(entries) == 0 {
		t.Fatal("cold backup missing", e)
	}
	b, e := os.ReadFile(sentinel)
	if e != nil || string(b) != "unchanged" {
		t.Fatal("data changed", e)
	}
	if _, e = os.Stat(filepath.Join(root, "launch-rc.sh")); !os.IsNotExist(e) {
		t.Fatal("launcher not removed")
	}
	if data, e := os.ReadFile(stableSentinel); e != nil || string(data) != "stable preserved" {
		t.Fatal("unrelated stable data modified")
	}
	t.Log("Actual first-party engine install, repeat, update, downgrade refusal, failed-update preservation; compiled CLI uninstall/repeat removes owned software; profile and cold backups preserved")
}
