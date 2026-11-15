package personal

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestExecutionFailsBeforeWritesWithoutCapability(t *testing.T) {
	root := filepath.Join(t.TempDir(), "absent")
	for _, command := range []string{"install", "update", "rollback"} {
		if NewClient().Execute(context.Background(), command, &Report{}, root, io.Discard) == nil {
			t.Fatal("missing capability accepted")
		}
	}
	if _, e := os.Stat(root); !os.IsNotExist(e) {
		t.Fatal("root written")
	}
}
func TestUninstallDoesNotAdoptUnknownRoot(t *testing.T) {
	root := t.TempDir()
	p := filepath.Join(root, "unrelated")
	os.WriteFile(p, []byte("preserve"), 0600)
	if NewClient().Execute(context.Background(), "uninstall", nil, root, io.Discard) == nil {
		t.Fatal("adopted unknown root")
	}
	b, e := os.ReadFile(p)
	if e != nil || string(b) != "preserve" {
		t.Fatal("data lost")
	}
}
func TestSymlinkAncestorRejected(t *testing.T) {
	base := t.TempDir()
	link := filepath.Join(base, "link")
	if e := os.Symlink(t.TempDir(), link); e != nil {
		t.Skip("host cannot create test symlink")
	}
	if privatePath(filepath.Join(link, "child")) == nil {
		t.Fatal("symlink accepted")
	}
}
func TestBoundedDiagnostics(t *testing.T) {
	b := &boundedLog{writer: io.Discard, remaining: 1}
	if n, e := b.Write([]byte("abc")); n != 3 || e != nil || b.remaining != 0 {
		t.Fatal(n, e)
	}
}

func TestUninstallDoesNotCompareAnAbsentReleaseVersion(t *testing.T) {
	root := t.TempDir()
	owner, _ := json.Marshal(ownership{1, "siaas.personal.first-party-m2", root, "candidate"})
	os.WriteFile(filepath.Join(root, "cli-owner.json"), owner, 0600)
	os.WriteFile(filepath.Join(root, "current.json"), []byte(`{"version":"1.1.1-rc.2","folder":"1.1.1-rc.2-1","cliManaged":1,"runtime":{"node":"/outside"}}`), 0600)
	// A malformed installed runtime must return an error, never panic in semver.
	if e := NewClient().Execute(context.Background(), "uninstall", &Report{Channel: "candidate"}, root, io.Discard); e == nil {
		t.Fatal("invalid runtime accepted")
	}
}
