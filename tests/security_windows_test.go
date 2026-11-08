//go:build windows

package tests

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func winCommand(t *testing.T, code string, values ...string) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-Command", "$ErrorActionPreference='Stop'; "+code)
	cmd.Env = append(os.Environ(), values...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Windows filesystem fixture failed: %s %v", out, err)
	}
	return strings.TrimSpace(string(out))
}

func TestWindowsDefaultPath(t *testing.T) {
	source, _ := winFixture(t)
	t.Setenv("SIAAS_CLI_INSTALL_DIR", "")
	if os.Getenv("SIAAS_STANDARD_USER_ACCEPTANCE") != "1" {
		t.Setenv("LOCALAPPDATA", t.TempDir())
	}
	dir := filepath.Join(os.Getenv("LOCALAPPDATA"), "SIAAS CLI", "bin")
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatal("default-path fixture must start absent")
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	for i := 0; i < 2; i++ {
		if code, out := winInstall(t, "", source); code != 0 {
			t.Fatalf("default install: %d %s", code, out)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "siaas.exe")); err != nil {
		t.Fatal(err)
	}
	t.Logf("Default installation validated at %s; genuine standard-user context=%s", dir, os.Getenv("SIAAS_STANDARD_USER_ACCEPTANCE"))
}

func TestWindowsReparseRejection(t *testing.T) {
	for _, dangling := range []bool{false, true} {
		t.Run(map[bool]string{false: "junction", true: "dangling junction"}[dangling], func(t *testing.T) {
			source, _ := winFixture(t)
			dir := t.TempDir()
			destination := filepath.Join(t.TempDir(), "real")
			if err := os.Mkdir(destination, 0700); err != nil {
				t.Fatal(err)
			}
			link := filepath.Join(dir, "junction")
			winCommand(t, `New-Item -ItemType Junction -Path $env:SIAAS_TEST_PATH -Target $env:SIAAS_TEST_OTHER | Out-Null`, "SIAAS_TEST_PATH="+link, "SIAAS_TEST_OTHER="+destination)
			// Remove only the junction entry, never traverse to its destination.
			defer os.Remove(link)
			if dangling {
				if err := os.Remove(destination); err != nil {
					t.Fatal(err)
				}
			}
			for _, target := range []string{link, filepath.Join(link, "child")} {
				if code, out := winInstall(t, target, source); code != 6 {
					t.Fatalf("reparse path: %d %s", code, out)
				}
			}
			if dangling {
				if _, err := os.Stat(destination); !os.IsNotExist(err) {
					t.Fatal("dangling target recreated")
				}
			}
		})
	}
}

func TestWindowsACLFailureAndRecovery(t *testing.T) {
	source, _ := winFixture(t)
	dir := t.TempDir()
	target := filepath.Join(dir, "siaas.exe")
	winWrite(t, target, []byte("previous CLI"))
	// Deny writes to this owned disposable directory for the actual token SID.
	restore := `$a=Get-Acl -LiteralPath $env:SIAAS_TEST_PATH; $a.SetSecurityDescriptorSddlForm([IO.File]::ReadAllText($env:SIAAS_TEST_ACL)); Set-Acl -LiteralPath $env:SIAAS_TEST_PATH -AclObject $a`
	acl := filepath.Join(t.TempDir(), "acl.txt")
	env := []string{"SIAAS_TEST_PATH=" + dir, "SIAAS_TEST_ACL=" + acl}
	winCommand(t, `$a=Get-Acl -LiteralPath $env:SIAAS_TEST_PATH; [IO.File]::WriteAllText($env:SIAAS_TEST_ACL,$a.Sddl); $sid=[Security.Principal.WindowsIdentity]::GetCurrent().User; $r=New-Object Security.AccessControl.FileSystemAccessRule($sid,'Write','Deny'); $a.AddAccessRule($r); Set-Acl -LiteralPath $env:SIAAS_TEST_PATH -AclObject $a`, env...)
	restored := false
	defer func() {
		if !restored {
			winCommand(t, restore, env...)
		}
	}()
	code, out := winInstall(t, dir, source)
	if code != 4 && code != 6 {
		t.Fatalf("denied ACL: %d %s", code, out)
	}
	old, err := os.ReadFile(target)
	if err != nil || string(old) != "previous CLI" {
		t.Fatal("ACL failure changed old CLI")
	}
	winCommand(t, restore, env...)
	restored = true
	if code, out = winInstall(t, dir, source); code != 0 {
		t.Fatalf("restored ACL retry: %d %s", code, out)
	}
	t.Log("Denied-write ACL preserved old CLI; restoring only the fixture ACL allowed recovery.")
}

func TestWindowsInternetZonePreservation(t *testing.T) {
	source, name := winFixture(t)
	winWrite(t, filepath.Join(source, name)+":Zone.Identifier", []byte("[ZoneTransfer]\r\nZoneId=3\r\n"))
	dir := t.TempDir()
	for _, replace := range []bool{false, true} {
		if replace {
			winWrite(t, filepath.Join(dir, "siaas.exe"), []byte("older CLI"))
		}
		if code, out := winInstall(t, dir, source); code != 0 {
			t.Fatalf("marked local fixture: %d %s", code, out)
		}
		zone, err := os.ReadFile(filepath.Join(dir, "siaas.exe") + ":Zone.Identifier")
		if err != nil || !strings.Contains(string(zone), "ZoneId=3") {
			t.Fatalf("Internet origin lost: %s %v", zone, err)
		}
	}
	t.Log("Internet-origin ADS retained on first install and replacement; marked executable was not launched.")
}
