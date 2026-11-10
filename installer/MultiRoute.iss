; Per-user installer for the MultiRoute window. Does not request administrator.
; ISCC /DVersion=0.5.0 installer\MultiRoute.iss
#ifndef Version
  #define Version "0.0.0"
#endif

[Setup]
AppId={{A9E4C3D2-1B70-4F58-8C6A-5D2E9B7A0146}
AppName=MultiRoute
AppVersion={#Version}
AppVerName=MultiRoute {#Version}
AppPublisher=MultiRouteNetwork contributors
VersionInfoVersion={#Version}
DefaultDirName={localappdata}\MultiRoute
DefaultGroupName=MultiRoute
DisableProgramGroupPage=yes
PrivilegesRequired=lowest
ArchitecturesAllowed=x64compatible
ArchitecturesInstallIn64BitMode=x64compatible
MinVersion=10.0.17763
Compression=lzma2
SolidCompression=yes
WizardStyle=modern
LicenseFile=LICENSE
SetupIconFile=..\multiroute\MultiRoute.ico
SourceDir=..\dist\MultiRoute-{#Version}-windows
; OutputDir is relative to SourceDir, so this is the repository dist folder.
OutputDir=..\..\dist
OutputBaseFilename=MultiRoute-{#Version}-windows-setup
UninstallDisplayIcon={app}\MultiRoute.exe
CloseApplications=yes
CloseApplicationsFilter=MultiRoute.exe

[Languages]
Name: "english"; MessagesFile: "compiler:Default.isl"

[Tasks]
Name: "desktopicon"; Description: "Create a &desktop shortcut"; GroupDescription: "Shortcuts:"; Flags: unchecked

[Files]
Source: "MultiRoute.exe"; DestDir: "{app}"; Flags: ignoreversion
Source: "MultiRouteNetwork.exe"; DestDir: "{app}"; Flags: ignoreversion
Source: "wintun.dll"; DestDir: "{app}"; Flags: ignoreversion
Source: "wintun-LICENSE.txt"; DestDir: "{app}"; Flags: ignoreversion
Source: "go-winio-LICENSE.txt"; DestDir: "{app}"; Flags: ignoreversion
Source: "README.txt"; DestDir: "{app}"; Flags: ignoreversion
Source: "LICENSE"; DestDir: "{app}"; Flags: ignoreversion
Source: "THIRD-PARTY-NOTICES.md"; DestDir: "{app}"; Flags: ignoreversion

[Icons]
Name: "{autoprograms}\MultiRoute"; Filename: "{app}\MultiRoute.exe"; IconFilename: "{app}\MultiRoute.exe"; Comment: "Route applications across network adapters"
Name: "{autodesktop}\MultiRoute"; Filename: "{app}\MultiRoute.exe"; IconFilename: "{app}\MultiRoute.exe"; Tasks: desktopicon

[Run]
Filename: "{app}\MultiRoute.exe"; Description: "Start MultiRoute"; Flags: nowait postinstall skipifsilent

[Code]
procedure StopMultiRoute;
var
  rc: Integer;
begin
  if FileExists(ExpandConstant('{app}\MultiRouteNetwork.exe')) then
    Exec(ExpandConstant('{app}\MultiRouteNetwork.exe'), 'quit', '', SW_HIDE, ewWaitUntilTerminated, rc);
  Exec(ExpandConstant('{sys}\taskkill.exe'), '/IM MultiRoute.exe', '', SW_HIDE, ewWaitUntilTerminated, rc);
end;

function PrepareToInstall(var NeedsRestart: Boolean): String;
begin
  StopMultiRoute;
  Result := '';
end;

procedure CurUninstallStepChanged(CurUninstallStep: TUninstallStep);
begin
  if CurUninstallStep = usUninstall then
    StopMultiRoute;
end;
