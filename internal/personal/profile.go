package personal

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
)

// Read-only profile integrity snapshot; the Personal engine copies its cold backup
// under the operation lock. Refuse links/devices and oversized data.
func profileSnapshot(root string) (map[string]string, error) {
	result := map[string]string{}
	base := filepath.Join(root, "profile")
	var total int64
	if _, e := os.Lstat(base); os.IsNotExist(e) {
		return result, nil
	}
	err := filepath.WalkDir(base, func(p string, d fs.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if d.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("profile link refused")
		}
		if d.IsDir() {
			return nil
		}
		info, e := d.Info()
		if e != nil {
			return e
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("non-regular profile file refused")
		}
		total += info.Size()
		if total > 1<<30 {
			return fmt.Errorf("profile exceeds cold-backup bound")
		}
		b, e := os.ReadFile(p)
		if e != nil {
			return e
		}
		sum := sha256.Sum256(b)
		rel, _ := filepath.Rel(base, p)
		result[rel] = hex.EncodeToString(sum[:])
		return nil
	})
	return result, err
}
// A reviewed legacy transition is bound to immutable release identities, not
// merely the SQLite user_version. Future Personal-owned declarations require
// their own versioned protocol and acceptance; unknown changes remain refused.
const (
	rc4Commit = "d8bf6e61a4d47920a00cc4e069a90cf26ac2d80f"
	rc4Source = "8f78f698d1a0447b862703c1f674810d416164f232fa1684a9b5695338e34615"
	rc5Commit = "853f35563b4b86ee355d276acd1a96c3e98f1007"
	rc5Source = "71e43b4ddce6d39317af3dabb73ed783a89d3a04bbe010c6f33654a306ef2e6b"
)

func installedDataCode(root, folder string) (map[string]string, error) {
	if !safeEntry(folder) || strings.Contains(folder, "/") {
		return nil, fmt.Errorf("unsafe installed metadata")
	}
	base := filepath.Join(root, "versions", folder, "src", "lib", "local")
	if err := privatePath(base); err != nil {
		return nil, err
	}
	result := map[string]string{}
	err := filepath.WalkDir(base, func(p string, d fs.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if d.IsDir() {
			return nil
		}
		if !d.Type().IsRegular() {
			return fmt.Errorf("non-regular installed data code refused")
		}
		b, e := os.ReadFile(p)
		if e != nil {
			return e
		}
		sum := sha256.Sum256(b)
		rel, _ := filepath.Rel(base, p)
		result[filepath.ToSlash(rel)] = hex.EncodeToString(sum[:])
		return nil
	})
	if err != nil || len(result) == 0 {
		return nil, fmt.Errorf("installed data code unavailable")
	}
	return result, nil
}

func archiveDataCode(source []byte) (map[string]string, error) {
	result := map[string]string{}
	z, e := zip.NewReader(bytes.NewReader(source), int64(len(source)))
	if e != nil {
		return nil, e
	}
	for _, f := range z.File {
		if strings.HasPrefix(f.Name, "src/lib/local/") && !f.FileInfo().IsDir() {
			if !safeEntry(f.Name) || !f.Mode().IsRegular() {
				return nil, fmt.Errorf("unsafe data code entry")
			}
			name := strings.TrimPrefix(f.Name, "src/lib/local/")
			if _, exists := result[name]; exists {
				return nil, fmt.Errorf("duplicate data code entry")
			}
			r, e := f.Open()
			if e != nil {
				return nil, e
			}
			h := sha256.New()
			_, e = io.Copy(h, r)
			r.Close()
			if e != nil {
				return nil, e
			}
			result[name] = hex.EncodeToString(h.Sum(nil))
		}
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("release data code unavailable")
	}
	return result, nil
}

func migrationReviewRequired() error {
	return fmt.Errorf("local data code changed; reviewed migration compatibility is required")
}

func requireSameDataCode(root, folder string, source []byte) error {
	old, err := installedDataCode(root, folder)
	if err != nil {
		return err
	}
	next, err := archiveDataCode(source)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(old, next) {
		return migrationReviewRequired()
	}
	return nil
}

func reviewedIdentity(r *Report, version, commit, digest string) bool {
	return r != nil && r.Repository == repository && r.Version == version &&
		r.Channel == "candidate" && r.Platform == "linux" && r.Arch == "x64" &&
		r.Compatible && r.Immutable && r.SourceVerified && r.Commit == commit &&
		r.SourceSHA256 == digest
}

func (c *Client) requireCompatibleDataCode(ctx context.Context, root, folder, from string, to *Report) error {
	old, err := installedDataCode(root, folder)
	if err != nil {
		return err
	}
	next, err := archiveDataCode(to.source)
	if err != nil {
		return err
	}
	if reflect.DeepEqual(old, next) {
		return nil
	}
	// RC4 and RC5 predate a product-owned compatibility declaration.
	// Only this exact reviewed pair may use the legacy exception.
	if from != "1.1.0-rc.4" || !reviewedIdentity(to, "1.1.0-rc.5", rc5Commit, rc5Source) {
		return migrationReviewRequired()
	}
	previous, err := c.Audit(ctx, Selection{Version: from, Channel: "candidate", Platform: "linux", Arch: "x64"})
	if err != nil {
		return err
	}
	if !reviewedIdentity(previous, from, rc4Commit, rc4Source) {
		return migrationReviewRequired()
	}
	expected, err := archiveDataCode(previous.source)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(old, expected) {
		return fmt.Errorf("installed data code differs from reviewed predecessor")
	}
	return nil
}
