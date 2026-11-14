// Package cli implements standalone commands and read-only Personal release verification.
// Product execution remains unavailable until a verified lifecycle backend exists.
package cli

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"github.com/siaas-software/siaas-cli/internal/personal"
	"github.com/siaas-software/siaas-cli/internal/productcommand"
	"io"
	"runtime"
	"time"
)

const Version = "0.1.0"

//go:embed NOTICE.txt
var runtimeNotice string

const help = `SIAAS CLI — standalone product management foundation

Usage: siaas [--json] [--help | --version | help]

  --help, help  Show this help
  --version     Show the CLI version
  --json        Write one schema-versioned JSON result to stdout

Personal release verification (read-only):
  install personal --check [--rc | --version VERSION]
  update personal --check [--rc | --version VERSION]
  --dry-run performs the same verification but fails if execution is unavailable.
Checks verify published source integrity, not installation compatibility or updates
to local state. Current published releases have no first-party M2 entry point. Future verified
Linux x64 first-party install/update/uninstall requires --yes. Rollback is unsupported.
No legacy adoption, GUI startup, profile access or --purge is performed.
No Node.js, npm, Electron, Docker or Ollama installation is needed to run this CLI.
Exit codes: 0 success; 1 output/internal failure; 2 invalid usage;
            3 unsupported capability; 4 release/network/integrity verification failed.
`

type Problem struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
type Result struct {
	SchemaVersion int              `json:"schemaVersion"`
	OK            bool             `json:"ok"`
	Command       string           `json:"command"`
	Version       string           `json:"version"`
	Message       string           `json:"message,omitempty"`
	RuntimeNotice string           `json:"runtimeNotice,omitempty"`
	Error         *Problem         `json:"error,omitempty"`
	Personal      *personal.Report `json:"personal,omitempty"`
}

func Run(args []string, stdout, stderr io.Writer) int {
	client := personal.NewClient()
	return runLifecycle(args, stdout, stderr, client.Audit, client.Execute)
}

type executor func(context.Context, string, *personal.Report, string, io.Writer) error

func run(args []string, stdout, stderr io.Writer, audit func(context.Context, personal.Selection) (*personal.Report, error)) int {
	return runLifecycle(args, stdout, stderr, audit, nil)
}
func runLifecycle(args []string, stdout, stderr io.Writer, audit func(context.Context, personal.Selection) (*personal.Report, error), execute executor) int {
	jsonMode := false
	duplicateJSON := false
	var rest []string
	for _, arg := range args {
		if arg == "--json" {
			duplicateJSON = duplicateJSON || jsonMode
			jsonMode = true
		} else {
			rest = append(rest, arg)
		}
	}
	r := Result{SchemaVersion: 1, Version: Version}
	code := 0
	fail := func(exit int, name, message string) {
		code = exit
		r.Error = &Problem{Code: name, Message: message}
	}
	// Never echo arbitrary arguments, URLs, paths or environment values in errors.
	// An unsupported invocation can contain secrets; omission is the redaction policy.
	if duplicateJSON {
		fail(2, "invalid_usage", "Use --json only once. Run siaas --help.")
	} else if len(rest) == 0 || len(rest) == 1 && (rest[0] == "--help" || rest[0] == "help") {
		r.Command, r.Message, r.RuntimeNotice = "help", help, runtimeNotice
	} else if len(rest) == 1 && rest[0] == "--version" {
		r.Command, r.Message = "version", "siaas "+Version
	} else {
		switch rest[0] {
		case "install", "update", "rollback", "uninstall":
			r.Command = rest[0]
			request, err := productcommand.Parse(rest)
			if err != nil {
				fail(2, "invalid_usage", "Invalid Personal command or conflicting options; --purge is unsupported. Run siaas --help.")
			} else if request.Command == "rollback" || execute == nil && (!request.Check && !request.DryRun || request.Command == "uninstall") {
				fail(3, "unsupported_capability", "No verified Personal lifecycle backend is available. No installation, update, adoption or removal was attempted.")
			} else {
				platform, arch := runtime.GOOS, runtime.GOARCH
				if platform == "windows" {
					platform = "win32"
				}
				if arch == "amd64" {
					arch = "x64"
				}
				ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
				var report *personal.Report
				var err error
				if request.Command != "uninstall" {
					report, err = audit(ctx, personal.Selection{Version: request.Version, Channel: request.Channel, Platform: platform, Arch: arch})
				} else {
					report = &personal.Report{Channel: request.Channel, Compatible: true}
				}
				cancel()
				if err != nil {
					if problem, ok := err.(*personal.Failure); ok {
						fail(4, problem.Code, problem.Message)
					} else {
						fail(4, "verification_failed", "Personal release verification failed. No changes were made.")
					}
				} else if report == nil {
					fail(4, "verification_failed", "Personal release verification produced no evidence.")
				} else {
					r.Personal = report
					if execute != nil && report.Compatible && !request.Check && !request.DryRun {
						if !request.Yes {
							fail(2, "confirmation_required", "Official source builds run as your user and download dependencies; rerun with --yes after review.")
						} else {
							root, e := personal.ManagedRoot(request.Channel)
							if e == nil {
								e = execute(context.Background(), request.Command, report, root, io.Discard)
							}
							if e != nil {
								fail(5, "lifecycle_failed", "Personal lifecycle operation failed. Inspect build diagnostics and retained locks; user data is preserved.")
							} else {
								r.Message = "Personal " + request.Command + " completed."
							}
						}
					} else if !request.Check {
						fail(3, "unsupported_capability", "Verified source bytes are not an accepted lifecycle adapter. No changes were made.")
					} else {
						r.Message = "Verified Personal " + report.Version + " source integrity; lifecycle execution remains unsupported. No installation was performed."
					}
				}
			}
		case "start", "stop", "restart", "status", "doctor", "list":
			r.Command = rest[0] // Fixed allowlist only; never user-provided text.
			fail(3, "not_implemented", "Product lifecycle commands are not implemented in M1. No changes were made.")
		default:
			fail(2, "invalid_usage", "Unsupported command or options. Run siaas --help.")
		}
	}
	r.OK = code == 0
	var err error
	if jsonMode {
		err = json.NewEncoder(stdout).Encode(r)
	} else if r.Error != nil {
		_, err = fmt.Fprintln(stderr, "siaas:", r.Error.Message)
	} else {
		_, err = fmt.Fprintln(stdout, r.Message)
		if err == nil && r.RuntimeNotice != "" {
			_, err = fmt.Fprintln(stdout, "Go runtime notice (BSD-3-Clause):\n"+r.RuntimeNotice)
		}
	}
	if err != nil {
		return 1
	}
	return code
}
