package personal

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
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

// A completed real installation gets a receipt only if persisted active state
// and the private Node binary are present and independently hash-verified.
func TestVerifiedLifecycleOperationReceipts(t *testing.T) {
	root := filepath.Join(t.TempDir(), "candidate")
	node := filepath.Join(root, "runtimes", "node")
	if err := os.MkdirAll(filepath.Dir(node), 0700); err != nil {
		t.Fatal(err)
	}
	binary := []byte("disposable-private-node-binary")
	if err := os.WriteFile(node, binary, 0700); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(binary)
	version := "1.1.0-rc.4"
	state := map[string]any{
		"version":    version,
		"folder":     "1.1.0-rc.4-verified",
		"cliManaged": 1,
		"runtime": map[string]any{
			"node":         node,
			"binarySha256": hex.EncodeToString(sum[:]),
		},
	}
	data, err := json.Marshal(state)
	if err != nil {
		t.Fatal(err)
	}
	current := filepath.Join(root, "current.json")
	if err := os.WriteFile(current, data, 0600); err != nil {
		t.Fatal(err)
	}
	for _, command := range []string{"install", "update"} {
		report := &Report{Version: version, Channel: "candidate", Compatible: true}
		if err := finishOperation(root, command, report); err != nil {
			t.Fatalf("%s verification: %v", command, err)
		}
		if !report.RuntimeProvisioned || report.Operation == nil ||
			report.Operation.Command != command || report.Operation.Status != "completed" ||
			!report.Operation.Installed || !report.Operation.RuntimeProvisioned {
			t.Fatalf("%s missing truthful receipt: %+v", command, report)
		}
	}
	// An installed generation under a different release cannot be reported
	// as a successful operation merely because a source audit passed.
	wrong := &Report{Version: "1.1.0-rc.3"}
	if err := finishOperation(root, "install", wrong); err == nil || wrong.Operation != nil || wrong.RuntimeProvisioned {
		t.Fatal("wrong installed version was accepted")
	}
	if err := os.WriteFile(node, []byte("tampered"), 0700); err != nil {
		t.Fatal(err)
	}
	tampered := &Report{Version: version}
	if err := finishOperation(root, "install", tampered); err == nil || tampered.Operation != nil || tampered.RuntimeProvisioned {
		t.Fatal("tampered private runtime was accepted")
	}
	// Uninstall cannot claim completion while active metadata still exists.
	stillActive := &Report{Channel: "candidate"}
	if err := finishOperation(root, "uninstall", stillActive); err == nil || stillActive.Operation != nil {
		t.Fatal("active install was incorrectly reported removed")
	}
	if err := os.Remove(current); err != nil {
		t.Fatal(err)
	}
	// Missing active registration alone is insufficient: retained runtime
	// files must prevent a false completed-uninstall receipt.
	partial := &Report{Channel: "candidate"}
	if err := finishOperation(root, "uninstall", partial); err == nil || partial.Operation != nil {
		t.Fatal("incomplete owned-runtime cleanup accepted")
	}
	if err := os.RemoveAll(filepath.Join(root, "runtimes")); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		uninstalled := &Report{Channel: "candidate"}
		if err := finishOperation(root, "uninstall", uninstalled); err != nil {
			t.Fatal(err)
		}
		if uninstalled.Operation == nil || uninstalled.Operation.Status != "completed" ||
			uninstalled.Operation.Command != "uninstall" || uninstalled.Operation.Installed ||
			uninstalled.Operation.RuntimeProvisioned || uninstalled.RuntimeProvisioned {
			t.Fatal("repeated uninstall reported an installed runtime")
		}
	}
}

func TestVerifiedReceiptRejectsRuntimeLinks(t *testing.T) {
	root := filepath.Join(t.TempDir(), "candidate")
	if err := os.MkdirAll(root, 0700); err != nil {
		t.Fatal(err)
	}
	external := t.TempDir()
	if err := os.WriteFile(filepath.Join(external, "node"), []byte("external"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(external, filepath.Join(root, "runtimes")); err != nil {
		t.Skip("host cannot create test symlinks")
	}
	b := sha256.Sum256([]byte("external"))
	state := map[string]any{
		"version":    "1.1.0-rc.4",
		"folder":     "1.1.0-rc.4-verified",
		"cliManaged": 1,
		"runtime": map[string]any{
			"node":         filepath.Join(root, "runtimes", "node"),
			"binarySha256": hex.EncodeToString(b[:]),
		},
	}
	data, err := json.Marshal(state)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "current.json"), data, 0600); err != nil {
		t.Fatal(err)
	}
	r := &Report{Version: "1.1.0-rc.4"}
	if err := finishOperation(root, "install", r); err == nil || r.Operation != nil {
		t.Fatal("linked runtime ancestry accepted")
	}
}
