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
	"path/filepath"
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
	root := filepath.Join(t.TempDir(), "Personal è space")
	c := NewClient()
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
		runtimeHashes, e := os.ReadFile("/workspace/rc3-preserved/assets/rc-manifests.json")
		if e != nil {
			t.Fatal(e)
		}
		var all map[string]manifest
		json.Unmarshal(runtimeHashes, &all)
		m := all["linux-x64"]
		m.Version = version
		m.Size = int64(len(data))
		m.SHA256 = hex.EncodeToString(sum[:])
		m.Name = "SIAAS-Personal-" + version + "-source.zip"
		m.URL = releaseBase + "download/v" + version + "/" + m.Name
		if e := verifyArchive(data, strings.Repeat("a", 40), sourceIndex{1, version, "24.22.0"}); e != nil {
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
	run("uninstall", nil)
	run("uninstall", nil)
	b, e := os.ReadFile(sentinel)
	if e != nil || string(b) != "unchanged" {
		t.Fatal("data changed", e)
	}
	if _, e = os.Stat(filepath.Join(root, "launch-rc.sh")); !os.IsNotExist(e) {
		t.Fatal("launcher not removed")
	}
	t.Log("Actual first-party engine install, repeat, update, downgrade refusal, uninstall and repeated uninstall; profile sentinel preserved")
}
