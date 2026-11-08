//go:build windows

package tests

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Audit-only: native HTTP delivery of the exact reviewed website/CLI script.
// The HTTPS artifact transport is an in-process fixture WebRequest factory;
// production scripts, allowlists, certificate callbacks and OS policy are unchanged.
func TestWebsitePowerShellPipeline(t *testing.T) {
	script, err := os.ReadFile(filepath.Join(root, "install.ps1"))
	if err != nil {
		t.Fatal(err)
	}
	const digest = "97992151229052d5d2f4b86a3a1822a2a48a03ad5f120b696e8c471868a678f2"
	if len(script) != 12917 || fmt.Sprintf("%x", sha256.Sum256(script)) != digest {
		t.Fatal("reviewed website/CLI script mismatch")
	}
	for _, mode := range []string{"install", "missing-release", "transport-failure", "bad-hash", "bad-manifest", "untrusted-redirect", "endpoint-unavailable"} {
		t.Run(mode, func(t *testing.T) {
			source, name := winFixture(t)
			dir := filepath.Join(t.TempDir(), "CLI space Ω")
			if err := os.Mkdir(dir, 0700); err != nil {
				t.Fatal(err)
			}
			target := filepath.Join(dir, "siaas.exe")
			winWrite(t, target, []byte("previous CLI"))
			personal := filepath.Join(t.TempDir(), "Personal data sentinel")
			winWrite(t, personal, []byte("unchanged product data"))
			if mode == "bad-hash" {
				b, e := os.ReadFile(filepath.Join(source, name))
				if e != nil {
					t.Fatal(e)
				}
				b[0] ^= 1
				winWrite(t, filepath.Join(source, name), b)
			}
			if mode == "bad-manifest" {
				winWrite(t, filepath.Join(source, "cli-release-v1.txt"), []byte("<html>not metadata</html>\n"))
			}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/plain; charset=utf-8")
				w.Header().Set("X-Content-Type-Options", "nosniff")
				if r.URL.Path != "/cli/install.ps1" {
					w.WriteHeader(404)
					return
				}
				if mode == "endpoint-unavailable" {
					w.WriteHeader(503)
					fmt.Fprint(w, "# Bootstrap unavailable\n")
					return
				}
				w.Write(script)
			}))
			defer server.Close()
			run := func() (int, string) {
				ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
				defer cancel()
				cmd := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-Command", pipelineSetup)
				cmd.Env = append(os.Environ(), "SIAAS_TEST_FIXTURE="+source, "SIAAS_TEST_ENDPOINT="+server.URL+"/cli/install.ps1", "SIAAS_TEST_MODE="+mode, "SIAAS_CLI_INSTALL_DIR="+dir)
				out, e := cmd.CombinedOutput()
				if e == nil {
					return 0, string(out)
				}
				if x, ok := e.(*exec.ExitError); ok {
					return x.ExitCode(), string(out)
				}
				t.Fatalf("PowerShell execution: %v %s", e, out)
				return -1, string(out)
			}
			code, out := run()
			expected := map[string]int{"install": 0, "missing-release": 7, "transport-failure": 7, "bad-hash": 6, "bad-manifest": 6, "untrusted-redirect": 6, "endpoint-unavailable": 1}[mode]
			if code != expected {
				t.Fatalf("pipeline exit=%d want=%d: %s", code, expected, out)
			}
			if !strings.Contains(out, "PIPELINE_PS=5.1") {
				t.Fatalf("missing 5.1 evidence: %s", out)
			}
			if strings.Contains(out, "secret-sentinel") || strings.Contains(out, "Cannot bind") {
				t.Fatalf("unsafe output: %s", out)
			}
			if strings.Contains(out, "PIPELINE_RETURNED") {
				t.Fatal("installer exit did not terminate the dedicated PowerShell process")
			}
			got, e := os.ReadFile(target)
			if e != nil {
				t.Fatal(e)
			}
			if mode == "install" {
				wanted, e := os.ReadFile(filepath.Join(source, name))
				if e != nil {
					t.Fatal(e)
				}
				if !bytes.Equal(got, wanted) {
					t.Fatal("installed bytes differ")
				}
				backups, _ := filepath.Glob(filepath.Join(dir, "siaas.exe.backup.*"))
				if len(backups) != 1 {
					t.Fatal("missing unique backup")
				}
				old, e := os.ReadFile(backups[0])
				if e != nil || string(old) != "previous CLI" {
					t.Fatal("old CLI not preserved")
				}
				if code, out = run(); code != 0 {
					t.Fatalf("repeat pipeline: %d %s", code, out)
				}
				repeated, _ := filepath.Glob(filepath.Join(dir, "siaas.exe.backup.*"))
				if len(repeated) != 1 {
					t.Fatal("repeat created another backup")
				}
				// Do not execute Internet-marked candidate: OS assessment stays intact.
				zone, e := os.ReadFile(target + ":Zone.Identifier")
				if e != nil || !bytes.Contains(zone, []byte("ZoneId=3")) {
					t.Fatal("Internet origin marker missing")
				}
			} else if string(got) != "previous CLI" {
				t.Fatal("failure replaced existing CLI")
			}
			sentinel, e := os.ReadFile(personal)
			if e != nil || string(sentinel) != "unchanged product data" {
				t.Fatal("product sentinel changed")
			}
			if _, e := os.Stat(filepath.Join(dir, ".siaas-cli-install.lock")); !os.IsNotExist(e) {
				t.Fatal("lock not cleaned")
			}
			t.Logf("HTTP_IRM_IEX mode=%s exit=%d native=%s standard_user=%s; prior CLI/data preserved; no marked executable launched", mode, code, os.Getenv("SIAAS_EXPECTED_ARCH"), os.Getenv("SIAAS_STANDARD_USER_ACCEPTANCE"))
			t.Log(strings.TrimSpace(out))
		})
	}
}

const pipelineSetup = `
Remove-Item Env:PSExecutionPolicyPreference -ErrorAction SilentlyContinue
if ($PSVersionTable.PSVersion.Major -ne 5 -or $PSVersionTable.PSVersion.Minor -ne 1 -or $PSVersionTable.PSEdition -ne 'Desktop') { throw 'Native Windows PowerShell 5.1 Desktop required.' }
Write-Output ('PIPELINE_PS=' + $PSVersionTable.PSVersion + '; policy=' + (Get-ExecutionPolicy))
if ($env:SIAAS_STANDARD_USER_ACCEPTANCE -eq '1') {
 $id=[Security.Principal.WindowsIdentity]::GetCurrent()
 if ($id.Groups.Value -contains 'S-1-5-32-544') { throw 'Administrator token refused for standard-user acceptance.' }
 Write-Output 'PIPELINE_STANDARD_USER=True'
}
Add-Type -TypeDefinition @'
using System;
using System.IO;
using System.Net;
public sealed class SiaasAuditFactory : IWebRequestCreate {
 public WebRequest Create(Uri uri) { return new SiaasAuditRequest(uri); }
}
public sealed class SiaasAuditRequest : WebRequest {
 readonly Uri uri;
 public SiaasAuditRequest(Uri value) { uri=value; }
 public bool AllowAutoRedirect {get;set;}
 public int ReadWriteTimeout {get;set;}
 public string UserAgent {get;set;}
 public override int Timeout {get;set;}
 public override WebResponse GetResponse() {
  string mode=Environment.GetEnvironmentVariable("SIAAS_TEST_MODE");
  if (mode=="missing-release" || mode=="transport-failure") throw new WebException("secret-sentinel",mode=="missing-release" ? WebExceptionStatus.ProtocolError : WebExceptionStatus.ConnectFailure);
  if (mode=="untrusted-redirect") return new SiaasAuditResponse(new byte[0],true);
  string directory=Environment.GetEnvironmentVariable("SIAAS_TEST_FIXTURE");
  string name=Path.GetFileName(uri.AbsolutePath);
  if (name!="cli-release-v1.txt" && !name.StartsWith("siaas-cli_0.1.0_windows_")) throw new WebException("Unexpected fixture request");
  return new SiaasAuditResponse(File.ReadAllBytes(Path.Combine(directory,name)),false);
 }
}
public sealed class SiaasAuditResponse : WebResponse {
 readonly byte[] bytes;
 readonly WebHeaderCollection headers=new WebHeaderCollection();
 public HttpStatusCode StatusCode {get;private set;}
 public SiaasAuditResponse(byte[] value,bool redirect) {bytes=value;StatusCode=redirect?HttpStatusCode.Found:HttpStatusCode.OK;if(redirect)headers["Location"]="http://evil.example/secret-sentinel";}
 public override long ContentLength {get{return bytes.Length;}set{throw new NotSupportedException();}}
 public override WebHeaderCollection Headers {get{return headers;}}
 public override Stream GetResponseStream(){return new MemoryStream(bytes,false);}
 public override void Close(){}
}
'@
if (-not [Net.WebRequest]::RegisterPrefix('https://github.com/siaas-software/siaas-cli/releases/',(New-Object SiaasAuditFactory))) { throw 'Cannot register isolated release fixture.' }
# The real website command form; no file invocation, arguments or policy bypass.
irm $env:SIAAS_TEST_ENDPOINT | iex
if ($env:SIAAS_TEST_MODE -ne 'endpoint-unavailable') { Write-Output 'PIPELINE_RETURNED'; exit 99 }
exit 1
`
