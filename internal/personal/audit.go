package personal

import (
	"archive/zip"
	"bytes"
	"context"
	"io"
	"path"
	"strings"
	"unicode/utf8"
)

type sourceIndex struct {
	Schema  int    `json:"schemaVersion"`
	Version string `json:"version"`
	Node    string `json:"nodeVersion"`
}
type manifest struct {
	Schema   int    `json:"schemaVersion"`
	Version  string `json:"version"`
	Platform string `json:"platform"`
	Arch     string `json:"arch"`
	Name     string `json:"name"`
	Size     int64  `json:"size"`
	SHA256   string `json:"sha256"`
	URL      string `json:"downloadUrl"`
	Runtime  struct {
		Version string `json:"version"`
		Name    string `json:"name"`
		URL     string `json:"url"`
		SHA256  string `json:"sha256"`
	} `json:"runtime"`
}

// Audit downloads and verifies the selected published source bytes in bounded
// memory, never extracting, provisioning a runtime or executing release code.
// It does NOT choose an older version to hide incompatible latest metadata.
func (c *Client) Audit(ctx context.Context, s Selection) (*Report, error) {
	if (s.Platform != "linux" && s.Platform != "darwin" && s.Platform != "win32") || (s.Arch != "x64" && s.Arch != "arm64") {
		return nil, fail("unsupported_platform")
	}
	if s.Channel != "stable" && s.Channel != "candidate" {
		return nil, fail("invalid_release")
	}
	r, err := c.selectRelease(ctx, s)
	if err != nil {
		return nil, err
	}
	commit, err := c.commit(ctx, r.Tag)
	if err != nil {
		return nil, err
	}
	assets := map[string]Asset{}
	ids := map[int64]bool{}
	for _, a := range r.Assets {
		if a.ID <= 0 || ids[a.ID] || a.Name == "" || strings.ContainsAny(a.Name, "/\\\x00\r\n\t ") || a.Name == ".." || a.URL != releaseBase+"download/"+r.Tag+"/"+a.Name {
			return nil, fail("invalid_release")
		}
		if _, ok := assets[a.Name]; ok {
			return nil, fail("invalid_release")
		}
		assets[a.Name] = a
		ids[a.ID] = true
	}
	checksums, err := c.bytes(ctx, assets["SHA256SUMS.txt"], 65536)
	if err != nil {
		return nil, err
	}
	sums := map[string]string{}
	for _, line := range strings.Split(strings.TrimSuffix(string(checksums), "\n"), "\n") {
		if len(line) < 67 || line[64:66] != "  " || !hashRE.MatchString(line[:64]) {
			return nil, fail("invalid_release")
		}
		name := line[66:]
		a, ok := assets[name]
		if !ok || sums[name] != "" || a.Digest != "sha256:"+line[:64] {
			return nil, fail("integrity_error")
		}
		sums[name] = line[:64]
	}
	load := func(name string, limit int64) ([]byte, error) {
		a, ok := assets[name]
		if !ok || sums[name] == "" {
			return nil, fail("invalid_release")
		}
		return c.bytes(ctx, a, limit)
	}
	version := r.Tag[1:]
	prefix := "SIAAS-Personal-" + version
	data, err := load(prefix+"-install.json", 65536)
	if err != nil {
		return nil, err
	}
	var index sourceIndex
	if err := decode(data, &index, true); err != nil {
		return nil, err
	}
	if index.Schema != 1 || index.Version != version || !validVersion(index.Node) || strings.Contains(index.Node, "-") {
		return nil, fail("invalid_release")
	}
	source, ok := assets[prefix+"-source.zip"]
	if !ok {
		return nil, fail("invalid_release")
	}
	if r.Prerelease {
		data, err = load("rc-manifests.json", 65536)
		if err != nil {
			return nil, err
		}
		var manifests map[string]manifest
		if err := decode(data, &manifests, true); err != nil {
			return nil, err
		}
		if len(manifests) != 6 {
			return nil, fail("invalid_release")
		}
		for _, platform := range []string{"linux", "win32", "darwin"} {
			for _, arch := range []string{"x64", "arm64"} {
				m, ok := manifests[platform+"-"+arch]
				osname, extension := platform, ".tar.gz"
				if platform == "win32" {
					osname = "win"
					extension = ".zip"
				}
				nodeName := "node-v" + index.Node + "-" + osname + "-" + arch + extension
				if !ok || m.Schema != 1 || m.Version != version || m.Platform != platform || m.Arch != arch || m.Name != source.Name || m.Size != source.Size || "sha256:"+m.SHA256 != source.Digest || m.URL != source.URL || m.Runtime.Version != index.Node || m.Runtime.Name != nodeName || m.Runtime.URL != "https://nodejs.org/dist/v"+index.Node+"/"+nodeName || !hashRE.MatchString(m.Runtime.SHA256) {
					return nil, fail("invalid_release")
				}
			}
		}
	}
	data, err = load(source.Name, maxSource)
	if err != nil {
		return nil, err
	}
	if err := verifyArchive(data, commit, index); err != nil {
		return nil, err
	}
	// The published source format, including PR28's proposed bootstrap hint, is
	// not the normative adapter. There is intentionally no execution fallback.
	return &Report{Repository: repository, Version: version, Channel: s.Channel, Platform: s.Platform, Arch: s.Arch, ReleaseID: r.ID, Commit: commit,
		Immutable: r.Immutable, SourceSHA256: strings.TrimPrefix(source.Digest, "sha256:"), SourceSize: source.Size, SourceVerified: true,
		RuntimeVersion: index.Node, RuntimeProvisioned: false, Compatible: false, Blocker: "unsupported_capability"}, nil
}

func safeEntry(name string) bool {
	if len(name) > 4096 || !utf8.ValidString(name) || strings.ContainsAny(name, "\\:\x00\r\n\t") || strings.HasPrefix(name, "/") {
		return false
	}
	for _, c := range name {
		if c < 32 || c == 127 {
			return false
		}
	}
	name = strings.TrimSuffix(name, "/")
	if name == "" || path.Clean(name) != name || name == ".." || strings.HasPrefix(name, "../") {
		return false
	}
	for _, part := range strings.Split(name, "/") {
		if strings.TrimRight(part, ". ") != part {
			return false
		}
		stem := strings.ToUpper(strings.SplitN(part, ".", 2)[0])
		if stem == "CON" || stem == "PRN" || stem == "AUX" || stem == "NUL" || (len(stem) == 4 && (strings.HasPrefix(stem, "COM") || strings.HasPrefix(stem, "LPT")) && stem[3] >= '0' && stem[3] <= '9') {
			return false
		}
	}
	return true
}
func verifyArchive(data []byte, commit string, index sourceIndex) error {
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return fail("integrity_error")
	}
	if r.Comment != commit || len(r.File) == 0 || len(r.File) > 20000 {
		return fail("integrity_error")
	}
	seen := map[string]bool{}
	var expanded uint64
	required := map[string][]byte{}
	for _, f := range r.File {
		key := strings.ToLower(strings.TrimSuffix(f.Name, "/"))
		if !safeEntry(f.Name) || seen[key] || (!f.Mode().IsRegular() && !f.Mode().IsDir()) || f.Flags&1 != 0 {
			return fail("integrity_error")
		}
		seen[key] = true
		if f.UncompressedSize64 > 128<<20 {
			return fail("integrity_error")
		}
		expanded += f.UncompressedSize64
		if expanded > 512<<20 {
			return fail("integrity_error")
		}
		if f.Mode().IsDir() {
			continue
		}
		file, err := f.Open()
		if err != nil {
			return fail("integrity_error")
		}
		var target bytes.Buffer
		var writer io.Writer = io.Discard
		if f.Name == "package.json" || f.Name == "runtime.json" {
			if f.UncompressedSize64 > 65536 {
				file.Close()
				return fail("integrity_error")
			}
			writer = &target
		}
		n, err := io.Copy(writer, io.LimitReader(file, int64(f.UncompressedSize64)+1))
		file.Close()
		if err != nil || uint64(n) != f.UncompressedSize64 {
			return fail("integrity_error")
		}
		if writer != io.Discard {
			required[f.Name] = target.Bytes()
		}
	}
	var pkg struct {
		Version string `json:"version"`
	}
	var runtime struct {
		Node string `json:"nodeVersion"`
	}
	if decode(required["package.json"], &pkg, false) != nil || decode(required["runtime.json"], &runtime, false) != nil || pkg.Version != index.Version || runtime.Node != index.Node {
		return fail("integrity_error")
	}
	return nil
}
