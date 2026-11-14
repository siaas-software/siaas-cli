// Package productcommand validates the proposed M2 Personal command syntax.
// This package is pure: parsing neither discovers nor installs a product.
// Mutating requests still require independently verified Personal execution support.
package productcommand

import (
	"errors"
	"regexp"
	"strings"
)

var ErrInvalidRequest = errors.New("invalid product command or option; see siaas --help")

// The public release convention accepts stable x.y.z and candidate x.y.z-rc.N.
// This validates syntax, not whether any release actually exists or is trusted.
var versionPattern = regexp.MustCompile("^(0|[1-9][0-9]*)\\.(0|[1-9][0-9]*)\\.(0|[1-9][0-9]*)(-rc\\.(0|[1-9][0-9]*))?$")

type Request struct {
	Command string
	Product string
	Channel string // stable or candidate; exact --version overrides channel inference
	Version string
	Check   bool
	DryRun  bool
	Yes     bool
	Purge   bool
}

// Parse performs only deterministic input validation. It deliberately does not
// resolve releases, contact GitHub, read profiles or return a successful install.
// Errors never contain untrusted option values, paths or credentials.
func Parse(args []string) (Request, error) {
	var r Request
	if len(args) < 2 || args[1] != "personal" {
		return Request{}, ErrInvalidRequest
	}
	r.Command, r.Product, r.Channel = args[0], "personal", "stable"
	switch r.Command {
	case "install", "update", "rollback", "uninstall":
	default:
		return Request{}, ErrInvalidRequest
	}
	seen := map[string]bool{}
	for i := 2; i < len(args); i++ {
		flag := args[i]
		if seen[flag] {
			return Request{}, ErrInvalidRequest
		}
		seen[flag] = true
		switch flag {
		case "--version":
			if r.Command != "install" && r.Command != "update" || i+1 >= len(args) {
				return Request{}, ErrInvalidRequest
			}
			i++
			if len(args[i]) > 80 || !versionPattern.MatchString(args[i]) {
				return Request{}, ErrInvalidRequest
			}
			r.Version = args[i]
			if containsRC(r.Version) {
				r.Channel = "candidate"
			}
		case "--rc":
			if r.Command != "install" && r.Command != "update" && r.Command != "uninstall" {
				return Request{}, ErrInvalidRequest
			}
			r.Channel = "candidate"
		case "--check":
			if r.Command != "install" && r.Command != "update" {
				return Request{}, ErrInvalidRequest
			}
			r.Check = true
		case "--dry-run":
			r.DryRun = true
		case "--yes":
			r.Yes = true
		case "--purge":
			if r.Command != "uninstall" {
				return Request{}, ErrInvalidRequest
			}
			r.Purge = true
		default:
			return Request{}, ErrInvalidRequest
		}
	}
	if seen["--rc"] && r.Version != "" {
		// An exact version already determines its channel: no ambiguous selectors.
		return Request{}, ErrInvalidRequest
	}
	if (r.Check && r.DryRun) || (r.Yes && (r.Check || r.DryRun)) || r.Purge {
		return Request{}, ErrInvalidRequest
	}
	return r, nil
}

func containsRC(version string) bool {
	return strings.Contains(version, "-rc.")
}
