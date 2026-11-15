package personal

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

type transport func(*http.Request) (*http.Response, error)

func (f transport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func encoded(v any) []byte {
	b, e := json.Marshal(v)
	if e != nil {
		panic(e)
	}
	return b
}
func sourceZip(t *testing.T, commit, version string, extra map[string]string) []byte {
	t.Helper()
	var b bytes.Buffer
	w := zip.NewWriter(&b)
	w.SetComment(commit)
	entries := map[string]string{"package.json": `{"version":"` + version + `"}`, "runtime.json": `{"nodeVersion":"24.22.0"}`}
	for k, v := range extra {
		entries[k] = v
	}
	for k, v := range entries {
		f, e := w.Create(k)
		if e != nil {
			t.Fatal(e)
		}
		f.Write([]byte(v))
	}
	if e := w.Close(); e != nil {
		t.Fatal(e)
	}
	return b.Bytes()
}
func fixtures(t *testing.T, versions ...string) (*Client, map[string][]byte, []release) {
	t.Helper()
	content := map[string][]byte{}
	var releases []release
	id := int64(0)
	for _, v := range versions {
		commit := strings.Repeat("a", 40)
		r := release{ID: 1, Tag: "v" + v, URL: releaseBase + "tag/v" + v, Immutable: true, Prerelease: strings.Contains(v, "-rc.")}
		add := func(name string, b []byte) Asset {
			id++
			a := Asset{ID: id, Name: name, URL: releaseBase + "download/" + r.Tag + "/" + name, Size: int64(len(b)), Digest: fmt.Sprintf("sha256:%x", sha256.Sum256(b))}
			r.Assets = append(r.Assets, a)
			content[a.URL] = b
			return a
		}
		prefix := "SIAAS-Personal-" + v
		source := add(prefix+"-source.zip", sourceZip(t, commit, v, nil))
		add(prefix+"-install.json", encoded(sourceIndex{1, v, "24.22.0"}))
		if r.Prerelease {
			all := map[string]manifest{}
			for _, p := range []string{"linux", "win32", "darwin"} {
				for _, arch := range []string{"x64", "arm64"} {
					m := manifest{Schema: 1, Version: v, Platform: p, Arch: arch, Name: source.Name, Size: source.Size, SHA256: source.Digest[7:], URL: source.URL}
					osname, ext := p, ".tar.gz"
					if p == "win32" {
						osname = "win"
						ext = ".zip"
					}
					m.Runtime.Version = "24.22.0"
					m.Runtime.Name = "node-v24.22.0-" + osname + "-" + arch + ext
					m.Runtime.URL = "https://nodejs.org/dist/v24.22.0/" + m.Runtime.Name
					m.Runtime.SHA256 = strings.Repeat("b", 64)
					all[p+"-"+arch] = m
				}
			}
			add("rc-manifests.json", encoded(all))
		}
		var sums strings.Builder
		for _, a := range r.Assets {
			fmt.Fprintf(&sums, "%s  %s\n", a.Digest[7:], a.Name)
		}
		add("SHA256SUMS.txt", []byte(sums.String()))
		content[api+"/releases/tags/"+r.Tag] = encoded(r)
		content[api+"/git/ref/tags/"+r.Tag] = []byte(`{"ref":"refs/tags/` + r.Tag + `","object":{"type":"commit","sha":"` + commit + `"}}`)
		releases = append(releases, r)
	}
	content[api+"/releases?per_page=100&page=1"] = encoded(releases)
	c := NewClient()
	c.http.Transport = transport(func(req *http.Request) (*http.Response, error) {
		if req.Header.Get("Authorization") != "" {
			t.Fatal("unexpected credential")
		}
		status := 200
		b, ok := content[req.URL.String()]
		if !ok {
			status = 404
		}
		return &http.Response{StatusCode: status, Body: io.NopCloser(bytes.NewReader(b)), Header: make(http.Header), Request: req}, nil
	})
	return c, content, releases
}
func assertCode(t *testing.T, err error, code string) {
	t.Helper()
	if e, ok := err.(*Failure); !ok || e.Code != code {
		t.Fatalf("want %s got %v", code, err)
	}
}
func TestPublishedSourceAudit(t *testing.T) {
	c, _, _ := fixtures(t, "1.0.0", "1.1.0-rc.2", "1.1.0-rc.10")
	for _, tc := range []struct{ version, channel, want string }{{"", "stable", "1.0.0"}, {"", "candidate", "1.1.0-rc.10"}, {"1.1.0-rc.2", "candidate", "1.1.0-rc.2"}} {
		for _, os := range []string{"linux", "win32", "darwin"} {
			for _, arch := range []string{"x64", "arm64"} {
				t.Run(tc.want+os+arch, func(t *testing.T) {
					r, e := c.Audit(context.Background(), Selection{tc.version, tc.channel, os, arch})
					if e != nil {
						t.Fatal(e)
					}
					if r.Version != tc.want || !r.SourceVerified || r.Compatible || r.RuntimeProvisioned || r.Blocker != "unsupported_capability" {
						t.Fatalf("bad report %+v", r)
					}
				})
			}
		}
	}
}
func TestCatalogAndExactFailures(t *testing.T) {
	c, content, releases := fixtures(t, "2.0.0", "1.9.0", "2.1.0-rc.1")
	_, e := c.Audit(context.Background(), Selection{"9.0.0", "stable", "linux", "x64"})
	assertCode(t, e, "release_not_found")
	_, e = c.Audit(context.Background(), Selection{"", "stable", "linux", "riscv64"})
	assertCode(t, e, "unsupported_platform")
	releases[0].Immutable = false
	content[api+"/releases?per_page=100&page=1"] = encoded(releases)
	_, e = c.Audit(context.Background(), Selection{"", "stable", "linux", "x64"})
	assertCode(t, e, "invalid_release") // no downgrade fallback
	releases[0].Immutable = true
	releases[0].Prerelease = true
	content[api+"/releases?per_page=100&page=1"] = encoded(releases)
	_, e = c.Audit(context.Background(), Selection{"", "stable", "linux", "x64"})
	assertCode(t, e, "invalid_release")
}
func TestTamperedRelease(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(map[string][]byte, *release)
		code   string
	}{
		{"foreign origin", func(_ map[string][]byte, r *release) { r.URL = "https://github.com/other/repo/releases/tag/" + r.Tag }, "invalid_release"},
		{"mutable", func(_ map[string][]byte, r *release) { r.Immutable = false }, "invalid_release"},
		{"duplicate assets", func(_ map[string][]byte, r *release) { r.Assets = append(r.Assets, r.Assets[0]) }, "invalid_release"},
		{"asset credentials", func(_ map[string][]byte, r *release) { r.Assets[0].URL = "https://secret@github.com/other" }, "invalid_release"},
		{"corrupt source", func(m map[string][]byte, r *release) { m[r.Assets[0].URL][0] ^= 1 }, "integrity_error"},
		{"truncated checksum", func(m map[string][]byte, r *release) { a := r.Assets[len(r.Assets)-1]; m[a.URL] = []byte("x") }, "integrity_error"},
		{"missing runtime", func(m map[string][]byte, r *release) {
			a := r.Assets[1]
			m[a.URL] = []byte(`{"schemaVersion":1,"version":"1.0.0"}`)
		}, "integrity_error"},
		{"bad size", func(_ map[string][]byte, r *release) { r.Assets[0].Size++ }, "integrity_error"},
		{"missing hash", func(_ map[string][]byte, r *release) { r.Assets[0].Digest = "" }, "integrity_error"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c, m, rs := fixtures(t, "1.0.0")
			r := rs[0]
			tc.mutate(m, &r)
			m[api+"/releases/tags/"+r.Tag] = encoded(r)
			_, e := c.Audit(context.Background(), Selection{"1.0.0", "stable", "linux", "x64"})
			assertCode(t, e, tc.code)
		})
	}
}
func TestJSONStrictness(t *testing.T) {
	var raw any
	if decode([]byte{34, 255, 34}, &raw, false) == nil {
		t.Fatal("invalid UTF-8 accepted")
	}
	for _, data := range []string{`{"schemaVersion":1,"schemaVersion":1}`, `{"schemaVersion":1,"SchemaVersion":2}`, `{"SCHEMAVERSION":1}`, `{"a":{"b":1,"b":2}}`, `{} {}`, `{"version":`, strings.Repeat("[", 40) + strings.Repeat("]", 40), `{"unknown":1}`} {
		var v sourceIndex
		if decode([]byte(data), &v, true) == nil {
			t.Fatalf("accepted %s", data)
		}
	}
}
func TestArchiveChecks(t *testing.T) {
	commit := strings.Repeat("a", 40)
	index := sourceIndex{1, "1.0.0", "24.22.0"}
	for _, entry := range []string{"../escape", "/absolute", "dir/../../escape", "C:/escape", "dir\\escape", "dir/NUL", "name. ", "PACKAGE.JSON"} {
		t.Run(entry, func(t *testing.T) {
			assertCode(t, verifyArchive(sourceZip(t, commit, "1.0.0", map[string]string{entry: "x"}), commit, index), "integrity_error")
		})
	}
	assertCode(t, verifyArchive(sourceZip(t, commit, "1.0.0", nil), strings.Repeat("b", 40), index), "integrity_error")
	assertCode(t, verifyArchive(sourceZip(t, commit, "2.0.0", nil), commit, index), "integrity_error")
	assertCode(t, verifyArchive([]byte("not a zip"), commit, index), "integrity_error")
}
func TestNetworkAndRedirectPolicy(t *testing.T) {
	for _, raw := range []string{"http://github.com/a", "https://github.com:444/a", "https://user:secret@github.com/a", "https://evil.invalid/a", "https://github.com/a#secret"} {
		if secureURL(raw, false) {
			t.Fatal("allowed untrusted URL")
		}
	}
	c := NewClient()
	c.http.Transport = transport(func(r *http.Request) (*http.Response, error) { return nil, fmt.Errorf("private-secret") })
	_, e := c.get(context.Background(), api+"/releases", 32, false)
	assertCode(t, e, "network_error")
	if strings.Contains(e.Error(), "private-secret") {
		t.Fatal("leak")
	}
	c.http.Transport = transport(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 302, Header: http.Header{"Location": []string{"http://evil.invalid"}}, Body: io.NopCloser(strings.NewReader("")), Request: r}, nil
	})
	_, e = c.get(context.Background(), releaseBase+"download/v1.0.0/a", 32, true)
	assertCode(t, e, "invalid_release")
	_, e = c.get(context.Background(), api+"/releases", 32, false)
	assertCode(t, e, "invalid_release")
}

// Refresh all integrity bindings so these exercise semantic checks, not just hashes.
func replaceAsset(content map[string][]byte, r *release, name string, data []byte) {
	for i, a := range r.Assets {
		if a.Name == name {
			r.Assets[i].Size = int64(len(data))
			r.Assets[i].Digest = fmt.Sprintf("sha256:%x", sha256.Sum256(data))
			content[a.URL] = data
		}
	}
	var sums strings.Builder
	for _, a := range r.Assets {
		if a.Name != "SHA256SUMS.txt" {
			fmt.Fprintf(&sums, "%s  %s\n", a.Digest[7:], a.Name)
		}
	}
	for i, a := range r.Assets {
		if a.Name == "SHA256SUMS.txt" {
			b := []byte(sums.String())
			r.Assets[i].Size = int64(len(b))
			r.Assets[i].Digest = fmt.Sprintf("sha256:%x", sha256.Sum256(b))
			content[a.URL] = b
		}
	}
	content[api+"/releases/tags/"+r.Tag] = encoded(r)
}
func TestSemanticallyInvalidManifestWithValidHashes(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*manifest)
	}{
		{"runtime", func(m *manifest) { m.Runtime.Version = "24.23.0" }},
		{"origin", func(m *manifest) { m.Runtime.URL = "https://evil.invalid/node" }},
		{"architecture", func(m *manifest) { m.Arch = "arm64" }},
		{"source digest", func(m *manifest) { m.SHA256 = strings.Repeat("f", 64) }},
		{"schema", func(m *manifest) { m.Schema = 2 }},
		{"size", func(m *manifest) { m.Size++ }},
		{"version", func(m *manifest) { m.Version = "9.0.0" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, content, releases := fixtures(t, "1.1.0-rc.3")
			r := releases[0]
			var all map[string]manifest
			json.Unmarshal(content[releaseBase+"download/"+r.Tag+"/rc-manifests.json"], &all)
			m := all["linux-x64"]
			tc.mutate(&m)
			all["linux-x64"] = m
			replaceAsset(content, &r, "rc-manifests.json", encoded(all))
			_, err := c.Audit(context.Background(), Selection{"1.1.0-rc.3", "candidate", "linux", "x64"})
			assertCode(t, err, "invalid_release")
		})
	}
	c, content, releases := fixtures(t, "1.0.0")
	r := releases[0]
	replaceAsset(content, &r, "SIAAS-Personal-1.0.0-install.json", []byte(`{"schemaVersion":1,"version":"1.0.0","nodeVersion":"24.22.0","capabilities":["install-only"]}`))
	_, err := c.Audit(context.Background(), Selection{"1.0.0", "stable", "linux", "x64"})
	assertCode(t, err, "invalid_release")
}
func TestTagIdentityAndPagination(t *testing.T) {
	c, m, rs := fixtures(t, "1.0.0")
	ref := api + "/git/ref/tags/v1.0.0"
	commit := strings.Repeat("a", 40)
	tag := strings.Repeat("b", 40)
	m[ref] = []byte(`{"ref":"refs/tags/v1.0.0","object":{"type":"tag","sha":"` + tag + `"}}`)
	m[api+"/git/tags/"+tag] = []byte(`{"object":{"type":"commit","sha":"` + commit + `"}}`)
	got, e := c.commit(context.Background(), "v1.0.0")
	if e != nil || got != commit {
		t.Fatal(got, e)
	}
	m[api+"/git/tags/"+tag] = m[ref]
	_, e = c.commit(context.Background(), "v1.0.0")
	assertCode(t, e, "invalid_release")
	m[ref] = []byte(`{"ref":"refs/tags/v2.0.0","object":{"type":"commit","sha":"` + commit + `"}}`)
	_, e = c.commit(context.Background(), "v1.0.0")
	assertCode(t, e, "invalid_release")
	many := make([]release, 100)
	for i := range many {
		many[i] = rs[0]
		many[i].Draft = true
	}
	for p := 1; p <= 10; p++ {
		m[api+fmt.Sprintf("/releases?per_page=100&page=%d", p)] = encoded(many)
	}
	_, e = c.selectRelease(context.Background(), Selection{Channel: "stable"})
	assertCode(t, e, "catalog_limit")
	m[api+"/releases?per_page=100&page=2"] = encoded(rs)
	gotRelease, e := c.selectRelease(context.Background(), Selection{Channel: "stable"})
	if e != nil || gotRelease.Tag != "v1.0.0" {
		t.Fatal(gotRelease, e)
	}
}
func TestCancellationAndBounds(t *testing.T) {
	c := NewClient()
	c.http.Transport = transport(func(r *http.Request) (*http.Response, error) {
		if e := r.Context().Err(); e != nil {
			return nil, e
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader("oversized")), Request: r}, nil
	})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, e := c.get(ctx, api+"/releases", 4, false)
	assertCode(t, e, "network_error")
	_, e = c.get(context.Background(), api+"/releases", 4, false)
	assertCode(t, e, "integrity_error")
}
