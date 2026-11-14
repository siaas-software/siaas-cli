package personal

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

type ownership struct {
	Schema   int    `json:"schemaVersion"`
	Protocol string `json:"protocol"`
	Root     string `json:"root"`
	Channel  string `json:"channel"`
}

func privatePath(root string) error {
	if !filepath.IsAbs(root) || filepath.Clean(root) != root {
		return fmt.Errorf("invalid installation root")
	}
	for p := root; ; p = filepath.Dir(p) {
		s, e := os.Lstat(p)
		if e != nil && !os.IsNotExist(e) {
			return e
		}
		if e == nil && (s.Mode()&os.ModeSymlink != 0 || !s.IsDir()) {
			return fmt.Errorf("unsafe installation ancestor")
		}
		if p == filepath.Dir(p) {
			break
		}
	}
	return nil
}
func readBound(path string, v any) error {
	b, e := os.ReadFile(path)
	if e != nil {
		return e
	}
	if len(b) > 65536 {
		return fmt.Errorf("metadata too large")
	}
	return decode(b, v, true)
}

// ManagedRoot creates a distinct channel root; no environment-based adoption.
func ManagedRoot(channel string) (string, error) {
	h, e := os.UserHomeDir()
	if e != nil {
		return "", e
	}
	return filepath.Join(h, ".local", "share", "siaas-cli", "personal", channel), nil
}
func (c *Client) Execute(ctx context.Context, command string, r *Report, root string, logs io.Writer) error {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		return fail("unsupported_platform")
	}
	if command != "install" && command != "update" && command != "uninstall" {
		return fail("unsupported_capability")
	}
	if command != "uninstall" && (r == nil || !r.Compatible || len(r.source) == 0) {
		return fail("unsupported_capability")
	}
	if e := privatePath(root); e != nil {
		return e
	}
	ownerFile := filepath.Join(root, "cli-owner.json")
	var owner ownership
	if b, e := os.ReadFile(ownerFile); e == nil {
		if len(b) > 65536 || json.Unmarshal(b, &owner) != nil || owner.Schema != 1 || owner.Protocol != "siaas.personal.first-party-m2" || owner.Root != root {
			return fmt.Errorf("unverified installation ownership")
		}
	} else if !os.IsNotExist(e) {
		return e
	} else {
		if command == "uninstall" {
			if _, e := os.Stat(root); os.IsNotExist(e) {
				return nil
			}
			return fmt.Errorf("unmanaged installation; adoption refused")
		}
		if command == "update" {
			return fmt.Errorf("no managed installation to update")
		}
		if _, e := os.Stat(root); e == nil {
			return fmt.Errorf("existing root; adoption refused")
		}
		if e := os.MkdirAll(root, 0700); e != nil {
			return e
		}
		owner = ownership{1, "siaas.personal.first-party-m2", root, r.Channel}
		b, _ := json.Marshal(owner)
		f, e := os.OpenFile(ownerFile, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if e != nil {
			return e
		}
		_, e = f.Write(b)
		f.Close()
		if e != nil {
			return e
		}
	}
	// Keep state on error; never guess that another process's stale lock is removable.
	lock := filepath.Join(root, ".cli-lock")
	if e := os.Mkdir(lock, 0700); e != nil {
		return fmt.Errorf("installation operation locked")
	}
	defer os.Remove(lock)
	var current struct {
		Version string `json:"version"`
		Folder  string `json:"folder"`
		CLI     int    `json:"cliManaged"`
		Runtime struct {
			Node   string `json:"node"`
			Binary string `json:"binarySha256"`
		} `json:"runtime"`
	}
	b, e := os.ReadFile(filepath.Join(root, "current.json"))
	active := e == nil
	if e != nil && !os.IsNotExist(e) {
		return e
	}
	if active {
		if len(b) > 65536 || json.Unmarshal(b, &current) != nil || current.CLI != 1 || !validVersion(current.Version) {
			return fmt.Errorf("unverified active installation")
		}
	}
	if command == "update" && !active {
		return fmt.Errorf("no managed installation to update")
	}
	if command == "uninstall" && !active {
		return nil
	}
	if r != nil && owner.Channel != r.Channel {
		return fmt.Errorf("channel switch refused")
	}
	if active && r != nil && newer(current.Version, r.Version) {
		return fmt.Errorf("downgrade refused")
	}
	var node, entry string
	args := []string{}
	if command == "uninstall" {
		node = current.Runtime.Node
		if !strings.HasPrefix(node, root+string(os.PathSeparator)) || !safeEntry(current.Folder) || strings.Contains(current.Folder, "/") {
			return fmt.Errorf("unsafe installed metadata")
		}
		nb, e := os.ReadFile(node)
		if e != nil {
			return e
		}
		sum := sha256.Sum256(nb)
		if hex.EncodeToString(sum[:]) != current.Runtime.Binary {
			return fmt.Errorf("runtime integrity failure")
		}
		entry = filepath.Join(root, "versions", current.Folder, "scripts", "source-manager.mjs")
		args = []string{"uninstall-cli"}
	} else {
		node, e = c.bootstrap(ctx, root, &r.runtimeSpec)
		if e != nil {
			return e
		}
		stage, e := os.MkdirTemp(root, ".cli-prepared-")
		if e != nil {
			return e
		}
		defer os.RemoveAll(stage)
		if e = os.WriteFile(filepath.Join(stage, "source.zip"), r.source, 0600); e != nil {
			return e
		}
		reader, e := zip.NewReader(bytes.NewReader(r.source), int64(len(r.source)))
		if e != nil {
			return e
		}
		source := filepath.Join(stage, "source")
		if e = os.Mkdir(source, 0700); e != nil {
			return e
		}
		for _, f := range reader.File {
			target := filepath.Join(source, filepath.FromSlash(f.Name))
			if !safeEntry(f.Name) {
				return fmt.Errorf("unsafe source entry")
			}
			if f.FileInfo().IsDir() {
				e = os.MkdirAll(target, 0700)
			} else {
				e = os.MkdirAll(filepath.Dir(target), 0700)
				if e == nil {
					var in io.ReadCloser
					in, e = f.Open()
					if e == nil {
						e = writeExclusive(target, in, f.Mode().Perm())
						in.Close()
					}
				}
			}
			if e != nil {
				return e
			}
		}
		m := r.runtimeSpec
		mb, _ := json.Marshal(m)
		if e = os.WriteFile(filepath.Join(stage, "manifest.json"), mb, 0600); e != nil {
			return e
		}
		entry = filepath.Join(source, "scripts", "source-manager.mjs")
		action := "install-cli"
		if command == "update" {
			action = "update-cli"
		}
		args = []string{action, "--prepared", stage, "--manifest", filepath.Join(stage, "manifest.json")}
	}
	env := []string{"PATH=" + filepath.Dir(node) + ":/usr/bin:/bin", "HOME=" + filepath.Join(root, "build-home"), "SIAAS_INSTALL_DIR=" + root, "NEXT_TELEMETRY_DISABLED=1", "NODE_USE_ENV_PROXY=1", "ELECTRON_CACHE=" + filepath.Join(root, "build-home", "electron-cache"), "ELECTRON_BUILDER_CACHE=" + filepath.Join(root, "build-home", "builder-cache")}
	// Trusted build needs configured TLS/proxy, never GitHub/release credentials.
	for _, k := range []string{"HTTPS_PROXY", "HTTP_PROXY", "NO_PROXY", "https_proxy", "http_proxy", "no_proxy", "NODE_EXTRA_CA_CERTS", "SSL_CERT_FILE"} {
		if v := os.Getenv(k); v != "" {
			env = append(env, k+"="+v)
		}
	}
	if e = os.MkdirAll(filepath.Join(root, "build-home"), 0700); e != nil {
		return e
	}
	cmd := exec.CommandContext(ctx, node, append([]string{entry}, args...)...)
	cmd.Env = env
	diagnostics, e := os.OpenFile(filepath.Join(root, "cli-build.log"), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if e != nil {
		return e
	}
	defer diagnostics.Close()
	output := &boundedLog{writer: io.MultiWriter(diagnostics, logs), remaining: 10 << 20}
	cmd.Stdout = output
	cmd.Stderr = output
	// Cancellation is deliberately not advertised: CommandContext kills only its
	// direct child; interrupted operations retain locks for explicit inspection.
	if e = cmd.Run(); e != nil {
		return fmt.Errorf("Personal engine failed; inspect retained logs and locks: %w", e)
	}
	if command == "uninstall" {
		if _, e = os.Stat(filepath.Join(root, "current.json")); !os.IsNotExist(e) {
			return fmt.Errorf("unregister verification failed")
		}
		return nil
	}
	b, e = os.ReadFile(filepath.Join(root, "current.json"))
	if e != nil {
		return e
	}
	if json.Unmarshal(b, &current) != nil || current.Version != r.Version || current.CLI != 1 || !validVersion(current.Version) {
		return fmt.Errorf("activation verification failed")
	}
	return nil
}
func writeExclusive(path string, in io.Reader, mode os.FileMode) error {
	f, e := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
	if e != nil {
		return e
	}
	_, e = io.Copy(f, in)
	ce := f.Close()
	if e != nil {
		return e
	}
	return ce
}
func (c *Client) nodeBytes(ctx context.Context, url string, limit int64) ([]byte, error) {
	if !strings.HasPrefix(url, "https://nodejs.org/dist/v") {
		return nil, fail("invalid_release")
	}
	req, e := httpRequest(ctx, url)
	if e != nil {
		return nil, e
	}
	response, e := c.http.Do(req)
	if e != nil {
		return nil, fail("network_error")
	}
	defer response.Body.Close()
	if response.StatusCode != 200 {
		return nil, fail("network_error")
	}
	b, e := io.ReadAll(io.LimitReader(response.Body, limit+1))
	if e != nil || int64(len(b)) > limit {
		return nil, fail("integrity_error")
	}
	return b, nil
}
func (c *Client) bootstrap(ctx context.Context, root string, m *manifest) (string, error) {
	spec := m.Runtime
	sums, e := c.nodeBytes(ctx, "https://nodejs.org/dist/v"+spec.Version+"/SHASUMS256.txt", 65536)
	if e != nil {
		return "", e
	}
	hash := ""
	for _, line := range strings.Split(string(sums), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[1] == spec.Name {
			if hash != "" {
				return "", fail("invalid_release")
			}
			hash = fields[0]
		}
	}
	if !hashRE.MatchString(hash) || (spec.SHA256 != "" && spec.SHA256 != hash) {
		return "", fail("integrity_error")
	}
	m.Runtime.SHA256 = hash
	dest := filepath.Join(root, "runtimes", "node-v"+spec.Version+"-linux-x64")
	node := filepath.Join(dest, "bin", "node")
	if _, e := os.Stat(dest); e == nil {
		return node, nil
	}
	data, e := c.nodeBytes(ctx, spec.URL, 150_000_000)
	if e != nil {
		return "", e
	}
	sum := sha256.Sum256(data)
	if hex.EncodeToString(sum[:]) != hash {
		return "", fail("integrity_error")
	}
	if e = os.MkdirAll(filepath.Dir(dest), 0700); e != nil {
		return "", e
	}
	stage, e := os.MkdirTemp(filepath.Dir(dest), ".bootstrap-")
	if e != nil {
		return "", e
	}
	defer os.RemoveAll(stage)
	gz, e := gzip.NewReader(bytes.NewReader(data))
	if e != nil {
		return "", e
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	var total int64
	prefix := strings.TrimSuffix(spec.Name, ".tar.gz") + "/"
	for {
		h, e := tr.Next()
		if e == io.EOF {
			break
		}
		if e != nil {
			return "", e
		}
		if !strings.HasPrefix(h.Name, prefix) {
			return "", fail("integrity_error")
		}
		name := strings.TrimPrefix(h.Name, prefix)
		if name == "" {
			continue
		}
		if !safeEntry(name) {
			return "", fail("integrity_error")
		}
		p := filepath.Join(stage, name)
		total += h.Size
		if total > 500_000_000 {
			return "", fail("integrity_error")
		}
		switch h.Typeflag {
		case tar.TypeDir:
			e = os.MkdirAll(p, 0700)
		case tar.TypeReg:
			e = os.MkdirAll(filepath.Dir(p), 0700)
			if e == nil {
				e = writeExclusive(p, tr, os.FileMode(h.Mode)&0755)
			}
		case tar.TypeSymlink:
			continue
		default:
			return "", fail("integrity_error")
		}
		if e != nil {
			return "", e
		}
	}
	if e = os.Rename(stage, dest); e != nil {
		return "", e
	}
	nb, e := os.ReadFile(node)
	if e != nil {
		return "", e
	}
	binary := sha256.Sum256(nb)
	record := map[string]string{"folder": dest, "node": node, "npm": filepath.Join(dest, "lib/node_modules/npm/bin/npm-cli.js"), "version": spec.Version, "binarySha256": hex.EncodeToString(binary[:])}
	b, _ := json.Marshal(record)
	e = os.WriteFile(filepath.Join(dest, "runtime.json"), b, 0600)
	return node, e
}

func httpRequest(ctx context.Context, url string) (*http.Request, error) {
	return http.NewRequestWithContext(ctx, "GET", url, nil)
}

type boundedLog struct {
	writer    io.Writer
	remaining int
}

func (b *boundedLog) Write(p []byte) (int, error) {
	n := len(p)
	if b.remaining > 0 {
		part := p
		if len(part) > b.remaining {
			part = part[:b.remaining]
		}
		_, e := b.writer.Write(part)
		b.remaining -= len(part)
		if e != nil {
			return 0, e
		}
	}
	return n, nil
}
