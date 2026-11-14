package personal

import (
	"context"
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
