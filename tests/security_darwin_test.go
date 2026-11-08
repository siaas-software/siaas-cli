package tests

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestMacLocalQuarantinePreservation(t *testing.T) {
	source, name := fixture(t)
	if out, err := exec.Command("/usr/bin/xattr", "-w", "com.apple.quarantine", "0081;00000000;SIAAS fixture;", filepath.Join(source, name)).CombinedOutput(); err != nil {
		t.Fatalf("quarantine fixture: %s %v", out, err)
	}
	dir := t.TempDir()
	for _, replace := range []bool{false, true} {
		if replace {
			write(t, filepath.Join(dir, "siaas"), []byte("previous CLI"), 0755)
		}
		if code, out := runInstaller(t, dir, source, nil); code != 0 {
			t.Fatalf("quarantined fixture: %d %s", code, out)
		}
		q, err := exec.Command("/usr/bin/xattr", "-p", "com.apple.quarantine", filepath.Join(dir, "siaas")).CombinedOutput()
		if err != nil || !strings.HasPrefix(string(q), "0081;") {
			t.Fatalf("quarantine lost: %s %v", q, err)
		}
	}
	t.Log("Local quarantine preserved on installation/replacement; marked executable was not launched.")
}
