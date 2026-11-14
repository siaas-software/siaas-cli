// Package personal verifies published Personal source releases without executing
// them. Release integrity is not adapter conformance or permission to install.
package personal

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

const repository = "siaas-software/siaas-personal"
const api = "https://api.github.com/repos/" + repository
const releaseBase = "https://github.com/" + repository + "/releases/"
const maxMetadata = 4 << 20
const maxSource = 150_000_000

var versionRE = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(?:-rc\.(0|[1-9][0-9]*))?$`)
var hashRE = regexp.MustCompile(`^[a-f0-9]{64}$`)
var commitRE = regexp.MustCompile(`^[a-f0-9]{40}$`)

// Failure messages are fixed: no server body, URL query, token or local path.
type Failure struct{ Code, Message string }

func (e *Failure) Error() string { return e.Message }
func fail(code string) error {
	messages := map[string]string{
		"network_error":          "Personal release metadata could not be retrieved securely; retry with network access.",
		"release_not_found":      "No published Personal release matches the requested channel or exact version.",
		"invalid_release":        "Personal release identity or integrity metadata is invalid; no changes were made.",
		"unsupported_platform":   "Personal does not declare this operating system and architecture.",
		"integrity_error":        "Personal release bytes do not match their verified size, digest or source identity.",
		"unsupported_capability": "A verified, supported Personal lifecycle adapter is unavailable. Legacy installers were not executed; no changes were made.",
		"catalog_limit":          "The release catalog exceeds the bounded discovery limit; use an exact version.",
	}
	return &Failure{code, messages[code]}
}

type Asset struct {
	ID     int64  `json:"id"`
	Name   string `json:"name"`
	Size   int64  `json:"size"`
	Digest string `json:"digest"`
	URL    string `json:"browser_download_url"`
}
type release struct {
	ID         int64   `json:"id"`
	Tag        string  `json:"tag_name"`
	URL        string  `json:"html_url"`
	Draft      bool    `json:"draft"`
	Prerelease bool    `json:"prerelease"`
	Immutable  bool    `json:"immutable"`
	Assets     []Asset `json:"assets"`
}
type Selection struct{ Version, Channel, Platform, Arch string }
type Report struct {
	Repository         string `json:"repository"`
	Version            string `json:"version"`
	Channel            string `json:"channel"`
	Platform           string `json:"platform"`
	Arch               string `json:"arch"`
	ReleaseID          int64  `json:"releaseId"`
	Commit             string `json:"commit"`
	Immutable          bool   `json:"immutable"`
	SourceSHA256       string `json:"sourceSha256"`
	SourceSize         int64  `json:"sourceSize"`
	SourceVerified     bool   `json:"sourceVerified"`
	RuntimeVersion     string `json:"runtimeVersion"`
	RuntimeProvisioned bool   `json:"runtimeProvisioned"`
	Compatible         bool   `json:"compatible"`
	Blocker            string `json:"blocker"`
}

// Client has no public endpoint override or credential option. Tests replace only
// the HTTP transport; canonical request URLs and redirect checks stay enabled.
type Client struct{ http *http.Client }

func NewClient() *Client {
	return &Client{http: &http.Client{Timeout: 30 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}
}

func secureURL(raw string, initial bool) bool {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.User != nil || u.Fragment != "" || u.Opaque != "" || (u.Port() != "" && u.Port() != "443") {
		return false
	}
	if initial {
		return u.RawQuery == "" && strings.HasPrefix(raw, releaseBase+"download/")
	}
	return u.Hostname() == "release-assets.githubusercontent.com" || u.Hostname() == "github.com"
}

func (c *Client) get(ctx context.Context, raw string, limit int64, asset bool) ([]byte, error) {
	if asset && !secureURL(raw, true) {
		return nil, fail("invalid_release")
	}
	if !asset && !strings.HasPrefix(raw, api+"/") {
		return nil, fail("invalid_release")
	}
	for redirects := 0; redirects <= 5; redirects++ {
		req, err := http.NewRequestWithContext(ctx, "GET", raw, nil)
		if err != nil {
			return nil, fail("invalid_release")
		}
		req.Header.Set("User-Agent", "siaas-cli-release-verifier")
		req.Header.Set("Accept", "application/json")
		if asset {
			req.Header.Set("Accept", "application/octet-stream")
		}
		response, err := c.http.Do(req)
		if err != nil {
			return nil, fail("network_error")
		}
		if response.StatusCode >= 300 && response.StatusCode <= 399 {
			location, e := response.Location()
			response.Body.Close()
			if !asset || e != nil || !secureURL(location.String(), false) {
				return nil, fail("invalid_release")
			}
			raw = location.String()
			continue
		}
		if response.StatusCode != 200 {
			response.Body.Close()
			if response.StatusCode == 404 {
				return nil, fail("release_not_found")
			}
			return nil, fail("network_error")
		}
		data, err := io.ReadAll(io.LimitReader(response.Body, limit+1))
		response.Body.Close()
		if err != nil {
			return nil, fail("network_error")
		}
		if int64(len(data)) > limit {
			return nil, fail("integrity_error")
		}
		return data, nil
	}
	return nil, fail("invalid_release")
}

// Reject duplicate keys (including nested keys), trailing JSON and excess depth
// before Go's normal decoder can silently accept last-key-wins ambiguity.
func decode(data []byte, value any, closed bool) error {
	if !utf8.Valid(data) {
		return fail("invalid_release")
	}
	d := json.NewDecoder(bytes.NewReader(data))
	d.UseNumber()
	var walk func(int) error
	walk = func(depth int) error {
		if depth > 32 {
			return errors.New("depth")
		}
		t, err := d.Token()
		if err != nil {
			return err
		}
		if delimiter, ok := t.(json.Delim); ok {
			switch delimiter {
			case '{':
				seen := map[string]bool{}
				for d.More() {
					key, err := d.Token()
					if err != nil {
						return err
					}
					k, ok := key.(string)
					if !ok || seen[strings.ToLower(k)] {
						return errors.New("key")
					}
					// encoding/json otherwise accepts case-insensitive field aliases.
					// These are the exact keys of the existing source index/RC manifests.
					if closed {
						switch k {
						case "schemaVersion", "version", "nodeVersion", "platform", "arch", "name", "size", "sha256", "downloadUrl", "runtime", "url", "linux-x64", "linux-arm64", "win32-x64", "win32-arm64", "darwin-x64", "darwin-arm64":
						default:
							return errors.New("unknown manifest key")
						}
					}
					seen[strings.ToLower(k)] = true
					if err := walk(depth + 1); err != nil {
						return err
					}
				}
			case '[':
				for d.More() {
					if err := walk(depth + 1); err != nil {
						return err
					}
				}
			default:
				return errors.New("delimiter")
			}
			_, err = d.Token()
			return err
		}
		return nil
	}
	if err := walk(0); err != nil {
		return fail("invalid_release")
	}
	if _, err := d.Token(); err != io.EOF {
		return fail("invalid_release")
	}
	d = json.NewDecoder(bytes.NewReader(data))
	if closed {
		d.DisallowUnknownFields()
	}
	if err := d.Decode(value); err != nil {
		return fail("invalid_release")
	}
	return nil
}

func validVersion(v string) bool { return len(v) <= 80 && versionRE.MatchString(v) }
func newer(a, b string) bool {
	x, y := versionRE.FindStringSubmatch(a), versionRE.FindStringSubmatch(b)
	for i := 1; i <= 3; i++ {
		if len(x[i]) != len(y[i]) {
			return len(x[i]) > len(y[i])
		}
		if x[i] != y[i] {
			return x[i] > y[i]
		}
	}
	if x[4] == "" || y[4] == "" {
		return x[4] == "" && y[4] != ""
	}
	if len(x[4]) != len(y[4]) {
		return len(x[4]) > len(y[4])
	}
	return x[4] > y[4]
}

func (c *Client) selectRelease(ctx context.Context, s Selection) (release, error) {
	var candidates []release
	if s.Version != "" {
		if !validVersion(s.Version) {
			return release{}, fail("invalid_release")
		}
		data, err := c.get(ctx, api+"/releases/tags/v"+s.Version, maxMetadata, false)
		if err != nil {
			return release{}, err
		}
		var r release
		if err := decode(data, &r, false); err != nil {
			return r, err
		}
		if r.Tag != "v"+s.Version {
			return r, fail("invalid_release")
		}
		candidates = append(candidates, r)
	} else {
		for page := 1; page <= 10; page++ {
			data, err := c.get(ctx, api+"/releases?per_page=100&page="+strconv.Itoa(page), maxMetadata, false)
			if err != nil {
				return release{}, err
			}
			var rows []release
			if err := decode(data, &rows, false); err != nil {
				return release{}, err
			}
			if len(rows) > 100 {
				return release{}, fail("invalid_release")
			}
			candidates = append(candidates, rows...)
			if len(rows) < 100 {
				break
			}
			if page == 10 {
				return release{}, fail("catalog_limit")
			}
		}
	}
	var eligible []release
	seen := map[string]bool{}
	for _, r := range candidates {
		v := strings.TrimPrefix(r.Tag, "v")
		if r.Draft || r.Tag != "v"+v || !validVersion(v) {
			continue
		}
		if seen[r.Tag] {
			return release{}, fail("invalid_release")
		}
		seen[r.Tag] = true
		rc := strings.Contains(v, "-rc.")
		if rc != r.Prerelease {
			return release{}, fail("invalid_release")
		}
		if rc != (s.Channel == "candidate") {
			continue
		}
		eligible = append(eligible, r)
	}
	if len(eligible) == 0 {
		return release{}, fail("release_not_found")
	}
	sort.Slice(eligible, func(i, j int) bool { return newer(eligible[i].Tag[1:], eligible[j].Tag[1:]) })
	r := eligible[0]
	if r.ID <= 0 || !r.Immutable || r.URL != releaseBase+"tag/"+r.Tag || len(r.Assets) > 200 {
		return r, fail("invalid_release")
	}
	return r, nil
}

func (c *Client) commit(ctx context.Context, tag string) (string, error) {
	raw := api + "/git/ref/tags/" + tag
	for i := 0; i < 5; i++ {
		data, err := c.get(ctx, raw, maxMetadata, false)
		if err != nil {
			return "", err
		}
		var ref struct {
			Ref    string `json:"ref"`
			Object struct {
				Type string `json:"type"`
				SHA  string `json:"sha"`
			} `json:"object"`
		}
		if err := decode(data, &ref, false); err != nil {
			return "", err
		}
		if i == 0 && ref.Ref != "refs/tags/"+tag {
			return "", fail("invalid_release")
		}
		if !commitRE.MatchString(ref.Object.SHA) {
			return "", fail("invalid_release")
		}
		if ref.Object.Type == "commit" {
			return ref.Object.SHA, nil
		}
		if ref.Object.Type != "tag" {
			return "", fail("invalid_release")
		}
		raw = api + "/git/tags/" + ref.Object.SHA
	}
	return "", fail("invalid_release")
}

func (c *Client) bytes(ctx context.Context, a Asset, limit int64) ([]byte, error) {
	if a.Size <= 0 || a.Size > limit || !strings.HasPrefix(a.Digest, "sha256:") || !hashRE.MatchString(strings.TrimPrefix(a.Digest, "sha256:")) {
		return nil, fail("invalid_release")
	}
	data, err := c.get(ctx, a.URL, a.Size, true)
	if err != nil {
		return nil, err
	}
	digest := sha256.Sum256(data)
	if int64(len(data)) != a.Size || "sha256:"+hex.EncodeToString(digest[:]) != a.Digest {
		return nil, fail("integrity_error")
	}
	return data, nil
}
