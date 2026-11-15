package personal

import (
	"archive/zip"
	"bytes"
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
func requireSameDataCode(root, folder string, source []byte) error {
	base := filepath.Join(root, "versions", folder, "src", "lib", "local")
	old := map[string]string{}
	next := map[string]string{}
	err := filepath.WalkDir(base, func(p string, d fs.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if d.IsDir() {
			return nil
		}
		b, e := os.ReadFile(p)
		if e != nil {
			return e
		}
		sum := sha256.Sum256(b)
		rel, _ := filepath.Rel(base, p)
		old[filepath.ToSlash(rel)] = hex.EncodeToString(sum[:])
		return nil
	})
	if err != nil {
		return err
	}
	z, e := zip.NewReader(bytes.NewReader(source), int64(len(source)))
	if e != nil {
		return e
	}
	for _, f := range z.File {
		if strings.HasPrefix(f.Name, "src/lib/local/") && !f.FileInfo().IsDir() {
			r, e := f.Open()
			if e != nil {
				return e
			}
			h := sha256.New()
			_, e = io.Copy(h, r)
			r.Close()
			if e != nil {
				return e
			}
			next[strings.TrimPrefix(f.Name, "src/lib/local/")] = hex.EncodeToString(h.Sum(nil))
		}
	}
	if !reflect.DeepEqual(old, next) {
		return fmt.Errorf("local data code changed; reviewed migration compatibility is required")
	}
	return nil
}
