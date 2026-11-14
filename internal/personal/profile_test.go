package personal

import (
	"os"
	"path/filepath"
	"testing"
)

func TestProfileSnapshotsRejectLinksAndPreserveBytes(t *testing.T) {
	root := t.TempDir()
	profile := filepath.Join(root, "profile")
	os.Mkdir(profile, 0700)
	file := filepath.Join(profile, "data")
	os.WriteFile(file, []byte("preserve"), 0600)
	before, e := profileSnapshot(root)
	if e != nil || len(before) != 1 {
		t.Fatal(before, e)
	}
	b, _ := os.ReadFile(file)
	if string(b) != "preserve" {
		t.Fatal("snapshot edited data")
	}
	if e = os.Symlink(file, filepath.Join(profile, "alias")); e != nil {
		t.Skip("host cannot create fixture symlink")
	}
	if _, e = profileSnapshot(root); e == nil {
		t.Fatal("symlink accepted")
	}
}

func TestPersistenceCodeChangeNeedsMigrationReview(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "versions", "fixture", "src", "lib", "local")
	os.MkdirAll(dir, 0700)
	os.WriteFile(filepath.Join(dir, "database.ts"), []byte("old schema"), 0600)
	same := sourceZip(t, "commit", "1.1.1-rc.2", map[string]string{"src/lib/local/database.ts": "old schema"})
	if e := requireSameDataCode(root, "fixture", same); e != nil {
		t.Fatal(e)
	}
	changed := sourceZip(t, "commit", "1.1.1-rc.2", map[string]string{"src/lib/local/database.ts": "new schema"})
	if e := requireSameDataCode(root, "fixture", changed); e == nil {
		t.Fatal("unreviewed data code accepted")
	}
}
