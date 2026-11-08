package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestContract(t *testing.T) {
	cases := []struct {
		name             string
		args             []string
		exit             int
		command, problem string
	}{
		{"empty", nil, 0, "help", ""}, {"help flag", []string{"--help"}, 0, "help", ""},
		{"help command", []string{"help"}, 0, "help", ""}, {"version", []string{"--version"}, 0, "version", ""},
		{"unknown", []string{"unknown"}, 2, "", "invalid_usage"}, {"unknown option", []string{"--token=SECRET"}, 2, "", "invalid_usage"},
		{"conflict", []string{"--help", "--version"}, 2, "", "invalid_usage"},
		{"extra help", []string{"help", "personal"}, 2, "", "invalid_usage"},
		{"duplicate JSON", []string{"--json", "--json"}, 2, "", "invalid_usage"},
		{"future install", []string{"install", "personal"}, 3, "install", "not_implemented"},
		{"future pin", []string{"install", "personal", "--version", "1.0.0"}, 3, "install", "not_implemented"},
		{"future help", []string{"install", "personal", "--help"}, 3, "install", "not_implemented"},
	}
	for _, verb := range []string{"update", "rollback", "uninstall", "start", "stop", "restart", "status", "doctor", "list"} {
		cases = append(cases, struct {
			name             string
			args             []string
			exit             int
			command, problem string
		}{verb, []string{verb, "personal"}, 3, verb, "not_implemented"})
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for _, machine := range []bool{false, true} {
				args := append([]string{}, tc.args...)
				if machine {
					args = append([]string{"--json"}, args...)
				}
				var out, diagnostic bytes.Buffer
				want := tc.exit
				if tc.name == "duplicate JSON" {
					want = 2
				}
				if got := Run(args, &out, &diagnostic); got != want {
					t.Fatalf("exit=%d want=%d", got, want)
				}
				jsonMode := machine || tc.name == "duplicate JSON"
				if jsonMode {
					var r Result
					if err := json.Unmarshal(out.Bytes(), &r); err != nil {
						t.Fatal(err)
					}
					if diagnostic.Len() != 0 || r.SchemaVersion != 1 || r.Version != Version || r.OK != (want == 0) {
						t.Fatalf("invalid envelope: %+v", r)
					}
					if r.Command != tc.command {
						t.Fatalf("command %q", r.Command)
					}
					if want != 0 && (r.Error == nil || r.Error.Code != tc.problem) {
						t.Fatalf("invalid error: %+v", r)
					}
				} else if want != 0 {
					if out.Len() != 0 || diagnostic.Len() == 0 {
						t.Fatal("human failure stream contract")
					}
				} else if out.Len() == 0 || diagnostic.Len() != 0 {
					t.Fatal("human success stream contract")
				}
			}
		})
	}
}

func TestJSONOrderAndRedaction(t *testing.T) {
	t.Setenv("SIAAS_TOKEN", "environment-secret-sentinel")
	for _, args := range [][]string{{"--version", "--json"}, {"help", "--json"}, {"--secret=argument-secret-sentinel", "--json"}, {"install", "https://user:argument-secret-sentinel@host/private", "--json"}, {"argument-secret-sentinel"}} {
		var out, diagnostic bytes.Buffer
		Run(args, &out, &diagnostic)
		if strings.Contains(out.String()+diagnostic.String(), "secret-sentinel") {
			t.Fatal("secret leaked")
		}
	}
}

type brokenWriter struct{}

func (brokenWriter) Write([]byte) (int, error) {
	return 0, errors.New("private error must not be printed")
}
func TestOutputFailure(t *testing.T) {
	for _, args := range [][]string{{"--help"}, {"--version", "--json"}, {"unknown"}} {
		if Run(args, brokenWriter{}, brokenWriter{}) != 1 {
			t.Fatal("output failure must exit 1")
		}
	}
}
