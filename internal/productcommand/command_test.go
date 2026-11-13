package productcommand

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestParseAccepted(t *testing.T) {
	cases := []struct {
		args []string
		want Request
	}{
		{[]string{"install", "personal"}, Request{Command: "install", Product: "personal", Channel: "stable"}},
		{[]string{"install", "personal", "--rc"}, Request{Command: "install", Product: "personal", Channel: "candidate"}},
		{[]string{"install", "personal", "--version", "1.1.0-rc.3"}, Request{Command: "install", Product: "personal", Channel: "candidate", Version: "1.1.0-rc.3"}},
		{[]string{"install", "personal", "--version", "1.0.0", "--dry-run"}, Request{Command: "install", Product: "personal", Channel: "stable", Version: "1.0.0", DryRun: true}},
		{[]string{"install", "personal", "--check", "--yes"}, Request{Command: "install", Product: "personal", Channel: "stable", Check: true, Yes: true}},
		{[]string{"update", "personal", "--rc", "--check"}, Request{Command: "update", Product: "personal", Channel: "candidate", Check: true}},
		{[]string{"update", "personal", "--version", "2.5.17"}, Request{Command: "update", Product: "personal", Channel: "stable", Version: "2.5.17"}},
		{[]string{"rollback", "personal", "--dry-run", "--yes"}, Request{Command: "rollback", Product: "personal", Channel: "stable", DryRun: true, Yes: true}},
		{[]string{"uninstall", "personal", "--purge", "--dry-run"}, Request{Command: "uninstall", Product: "personal", Channel: "stable", Purge: true, DryRun: true}},
	}
	for _, tc := range cases {
		got, err := Parse(tc.args)
		if err != nil || !reflect.DeepEqual(got, tc.want) {
			t.Errorf("Parse(%v) = %+v, %v; want %+v", tc.args, got, err, tc.want)
		}
	}
}

func TestParseRejectsUnsafeAndAmbiguous(t *testing.T) {
	cases := [][]string{
		nil, {"install"}, {"install", "other"}, {"unknown", "personal"},
		{"install", "personal", "--version"}, {"install", "personal", "--version", "1.0.0", "--version", "2.0.0"},
		{"install", "personal", "--version", "1.0.0", "--rc"},
		{"install", "personal", "--rc", "--version", "1.1.0-rc.3"},
		{"install", "personal", "--rc", "--rc"},
		{"install", "personal", "--version", "../secret"},
		{"install", "personal", "--version", "1.2.3;rm"},
		{"install", "personal", "--version", "1.2.3-rc.00"},
		{"install", "personal", "--version", "01.2.3"},
		{"rollback", "personal", "--version", "1.0.0"},
		{"uninstall", "personal", "--rc"},
		{"install", "personal", "--purge"},
		{"uninstall", "personal", "--check"},
		{"rollback", "personal", "--check"},
		{"install", "personal", "--yes", "--yes"},
		{"install", "personal", "--secret=canary-secret-token"},
		{"install", "personal", "--version", "https://user:canary-secret-token@example.invalid"},
	}
	for _, args := range cases {
		got, err := Parse(args)
		if !errors.Is(err, ErrInvalidRequest) || got != (Request{}) {
			t.Errorf("expected invalid with empty result; got %+v, %v", got, err)
		}
		if err != nil && strings.Contains(err.Error(), "canary-secret-token") {
			t.Fatal("secret leaked")
		}
	}
}
