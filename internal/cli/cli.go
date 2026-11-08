// Package cli implements the offline M1 command boundary. It never starts products.
package cli

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
)

const Version = "0.1.0"

//go:embed NOTICE.txt
var runtimeNotice string

const help = `SIAAS CLI — standalone product management foundation

Usage: siaas [--json] [--help | --version | help]

  --help, help  Show this help
  --version     Show the CLI version
  --json        Write one schema-versioned JSON result to stdout

Product installation and lifecycle commands are not implemented in M1.
No Node.js, npm, Electron, Docker or Ollama installation is needed to run this CLI.
Exit codes: 0 success; 1 output/internal failure; 2 invalid usage;
            3 command not implemented.
`

type Problem struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
type Result struct {
	SchemaVersion int      `json:"schemaVersion"`
	OK            bool     `json:"ok"`
	Command       string   `json:"command"`
	Version       string   `json:"version"`
	Message       string   `json:"message,omitempty"`
	RuntimeNotice string   `json:"runtimeNotice,omitempty"`
	Error         *Problem `json:"error,omitempty"`
}

func Run(args []string, stdout, stderr io.Writer) int {
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
		case "install", "update", "rollback", "uninstall", "start", "stop", "restart", "status", "doctor", "list":
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
