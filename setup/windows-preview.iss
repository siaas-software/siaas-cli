; Windows x64 single-file QA installer for the CLI ONLY.
; Intentionally distinct from the official SIAAS CLI installation directory:
; preview uninstall cannot silently remove production CLI or Personal.
#define PreviewName "SIAAS CLI Preview"
[Setup]
AppId={{C27C3750-3987-4A21-9A4C-070C499DC4D1}
AppName={#PreviewName}
AppVersion=0.1.0-preview
AppVerName=SIAAS CLI Preview (M2 source build)
AppPublisher=SIAAS Software
DefaultDirName={localappdata}\SIAAS CLI Preview
DefaultGroupName=SIAAS CLI Preview
PrivilegesRequired=lowest
DisableProgramGroupPage=yes
UsePreviousAppDir=no
OutputDir=..\dist\setup
OutputBaseFilename=SIAAS-CLI-Setup-preview-windows-amd64
Compression=lzma
SolidCompression=yes
WizardStyle=modern
UninstallDisplayIcon={app}\bin\siaas.exe
ArchitecturesAllowed=x64compatible
ChangesEnvironment=no
CloseApplications=yes
RestartApplications=no
SetupLogging=no
[Files]
Source: "..\dist\siaas.exe"; DestDir: "{app}\bin"; DestName: "siaas.exe"; Flags: ignoreversion
[Icons]
Name: "{autoprograms}\SIAAS CLI Preview"; Filename: "{cmd}"; Parameters: "/K {#33}{app}\bin\siaas.exe{#33} --help"; WorkingDir: "{app}\bin"
