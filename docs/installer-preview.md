# SIAAS CLI Windows x64 installer (temporary preview)

This scoped test produces **one self-contained Windows installer EXE for CLI only**
from the exact pull-request head using pinned Go 1.27.3 and the native Windows
`Inno Setup 6` packaging tool.

## Guarantees and limitations

- **CLI-only:** no Personal source, Electron, Node, Ollama, models or profile
  installed; no attempts to update or remove SIAAS Personal.
- Dedicated per-user `%LOCALAPPDATA%\\SIAAS CLI Preview` directory, distinct
  from the public CLI v0.1.0 `%LOCALAPPDATA%\\SIAAS CLI` and from Personal.
- Temporary **unsigned test artifact**, downloadable from the PR's successful
  GitHub Actions run ZIP by a signed-in reviewer, not a stable GitHub Release.
  Windows may block unsigned software; do not bypass SmartScreen or security
  policy to continue a blocked test.
- Source CLI still reports version 0.1.0 internally. **It contains later M2
  source code that is not present in the public v0.1.0 release**. The installer
  UI explicitly identifies it as a preview, not as the published stable CLI.
- Per-user Inno Setup provides native install/reinstall/uninstall UI and a
  Start-menu command prompt shortcut, without editing system/user PATH or
  shell profiles. Uninstall deletes owned application files but preserves
  unrelated data.
- Build does not create tags, publish releases, or deploy the website.

## Native Windows acceptance

Existing pull request GitHub Actions (Windows 2022 x64):
1. Verify Go 1.27.3 and `go test ./...`; build Windows binary using `go
   build -trimpath`.
2. Verify locally present Inno Setup 6; compile the single setup EXE and
   record its SHA-256/size.
3. On isolated temporary directory: install silently, execute `siaas
   --version`, reinstall, execute again, uninstall, confirm CLI removed,
   and confirm a separate sentinel remained unchanged.
4. Upload the verified setup EXE as a GitHub Actions artifact only on PASS.

Manual review on a non-production Windows 11 account must still verify:
download provenance and SHA-256, double-click installer UI, Start-menu shortcut,
OS signing/security policy, uninstall, optional backup and **no impact on Stable
Personal/CLI or Ollama**. Record Windows version/CPU and any blocking dialog.

This preview only targets Windows x64. Windows ARM64/macOS/Linux packaging and
signed public installers require separate implementation and acceptance.
The public Get Started site must only display download links after actual
official release assets have been published and verified.
