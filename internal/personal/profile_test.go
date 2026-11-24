package personal

import (
	"context"
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
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
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

func TestReviewedMigrationIdentityRejectsOtherReleases(t *testing.T) {
	valid := Report{Repository: repository, Version: "1.1.0-rc.5", Channel: "candidate", Platform: "linux", Arch: "x64", Compatible: true, Immutable: true, SourceVerified: true, Commit: rc5Commit, SourceSHA256: rc5Source}
	if !reviewedIdentity(&valid, "1.1.0-rc.5", rc5Commit, rc5Source) {
		t.Fatal("exact reviewed identity rejected")
	}
	for _, change := range []func(*Report){
		func(r *Report) { r.Version = "1.1.0-rc.6" },
		func(r *Report) { r.Commit = rc4Commit },
		func(r *Report) { r.SourceSHA256 = rc4Source },
		func(r *Report) { r.Immutable = false },
		func(r *Report) { r.SourceVerified = false },
		func(r *Report) { r.Compatible = false },
		func(r *Report) { r.Repository = "other/repo" },
		func(r *Report) { r.Channel = "stable" },
		func(r *Report) { r.Platform = "win32" },
		func(r *Report) { r.Arch = "arm64" },
	} {
		candidate := valid
		change(&candidate)
		if reviewedIdentity(&candidate, "1.1.0-rc.5", rc5Commit, rc5Source) {
			t.Fatal("unreviewed release identity accepted")
		}
	}
}

func TestCompatibilityGuardRetainsDefaultRefusal(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(root, "versions", "fixture", "src", "lib", "local")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "database.ts"), []byte("old"), 0600); err != nil {
		t.Fatal(err)
	}
	changed := sourceZip(t, "commit", "1.1.0-rc.6", map[string]string{"src/lib/local/database.ts": "new"})
	r := &Report{Version: "1.1.0-rc.6", source: changed}
	if err := NewClient().requireCompatibleDataCode(context.Background(), root, "fixture", "1.1.0-rc.5", r); err == nil {
		t.Fatal("unknown data-code transition accepted")
	}
	same := sourceZip(t, "commit", "1.1.0-rc.6", map[string]string{"src/lib/local/database.ts": "old"})
	r.source = same
	if err := NewClient().requireCompatibleDataCode(context.Background(), root, "fixture", "1.1.0-rc.5", r); err != nil {
		t.Fatal(err)
	}
}
