package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/siaas-software/siaas-cli/internal/personal"
	"io"
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
		{"future install", []string{"install", "personal"}, 3, "install", "unsupported_capability"},
		{"future pin", []string{"install", "personal", "--version", "1.0.0"}, 3, "install", "unsupported_capability"},
		{"future help", []string{"install", "personal", "--help"}, 2, "install", "invalid_usage"},
	}
	for _, verb := range []string{"update", "rollback", "uninstall", "start", "stop", "restart", "status", "doctor", "list"} {
		cases = append(cases, struct {
			name             string
			args             []string
			exit             int
			command, problem string
		}{verb, []string{verb, "personal"}, 3, verb, map[bool]string{true: "unsupported_capability", false: "not_implemented"}[verb == "update" || verb == "rollback" || verb == "uninstall"]})
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
				if got := run(args, &out, &diagnostic, personal.NewClient().Audit); got != want {
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

func TestPersonalChecksAndNoMutation(t *testing.T) {
	for _, tc := range []struct {
		args  []string
		exit  int
		calls int
	}{
		{[]string{"install", "personal", "--check", "--rc"}, 0, 1},
		{[]string{"update", "personal", "--dry-run", "--version", "1.1.0-rc.3"}, 3, 1},
		{[]string{"install", "personal", "--yes"}, 3, 0},
		{[]string{"update", "personal"}, 3, 0},
		{[]string{"uninstall", "personal"}, 3, 0},
		{[]string{"uninstall", "personal", "--purge"}, 2, 0},
		{[]string{"install", "personal", "--check", "--yes"}, 2, 0},
	} {
		var out, diagnostic bytes.Buffer
		calls := 0
		code := run(append([]string{"--json"}, tc.args...), &out, &diagnostic, func(ctx context.Context, s personal.Selection) (*personal.Report, error) {
			calls++
			if s.Channel != "candidate" {
				t.Fatalf("wrong selector %+v", s)
			}
			if _, ok := ctx.Deadline(); !ok {
				t.Fatal("unbounded lookup")
			}
			return &personal.Report{Version: "1.1.0-rc.3", SourceVerified: true, Compatible: false, Blocker: "unsupported_capability"}, nil
		})
		if code != tc.exit || calls != tc.calls || diagnostic.Len() != 0 {
			t.Fatalf("%v code=%d calls=%d err=%s", tc.args, code, calls, diagnostic.String())
		}
		var r Result
		if err := json.Unmarshal(out.Bytes(), &r); err != nil {
			t.Fatal(err)
		}
		if tc.calls > 0 && (r.Personal == nil || r.Personal.Compatible) {
			t.Fatal("false compatibility")
		}
	}
}
func TestReadOnlyErrorRedaction(t *testing.T) {
	var out, diagnostic bytes.Buffer
	code := run([]string{"install", "personal", "--check", "--json"}, &out, &diagnostic, func(context.Context, personal.Selection) (*personal.Report, error) {
		return nil, errors.New("private-secret")
	})
	if code != 4 || strings.Contains(out.String()+diagnostic.String(), "private-secret") {
		t.Fatal("unredacted error")
	}
}

func TestFirstPartyExecutionRouting(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	for _, tc := range []struct {
		args       []string
		compatible bool
		calls      int
		exit       int
	}{
		{[]string{"install", "personal", "--yes"}, true, 1, 0},
		{[]string{"update", "personal", "--yes", "--rc"}, true, 1, 0},
		{[]string{"install", "personal", "--yes"}, false, 0, 3},
		{[]string{"install", "personal"}, true, 0, 2},
		{[]string{"install", "personal", "--check"}, true, 0, 0},
		{[]string{"uninstall", "personal", "--yes", "--rc"}, true, 1, 0},
	} {
		calls := 0
		var out, diagnostics bytes.Buffer
		got := runLifecycle(append(tc.args, "--json"), &out, &diagnostics,
			func(context.Context, personal.Selection) (*personal.Report, error) {
				return &personal.Report{Version: "1.1.1-rc.1", Channel: "candidate", Compatible: tc.compatible}, nil
			},
			func(ctx context.Context, command string, r *personal.Report, root string, logs io.Writer) error {
				calls++
				return nil
			})
		if got != tc.exit || calls != tc.calls {
			t.Fatalf("%v exit %d calls %d", tc.args, got, calls)
		}
		var result Result
		if json.Unmarshal(out.Bytes(), &result) != nil {
			t.Fatal("invalid JSON")
		}
	}
}
