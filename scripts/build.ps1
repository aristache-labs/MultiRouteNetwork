# Builds everything into dist\: the two exes, wintun.dll, the installer, and a portable zip.
#   scripts\build.ps1 -Version 0.1.0
# Needs Go and Inno Setup 6 (ISCC.exe on PATH, in a default install dir, or in $env:ISCC).
param(
    [string]$Version = '',
    [string]$RepoUrl = ''
)
$ErrorActionPreference = 'Stop'
Set-Location (Split-Path $PSScriptRoot -Parent)
if (-not $Version) { $Version = (Get-Content -LiteralPath 'VERSION' -Raw).Trim() }

function Invoke-Native {
    param([string]$What, [scriptblock]$Cmd)
    & $Cmd
    if ($LASTEXITCODE -ne 0) { throw "$What failed (exit $LASTEXITCODE)" }
}

if ($Version -notmatch '^\d+\.\d+\.\d+$') { throw "Version must look like 1.2.3, got '$Version'" }

if (Test-Path dist) { Remove-Item dist -Recurse -Force }
New-Item -ItemType Directory dist | Out-Null

Invoke-Native 'go vet'  { go vet ./... }
Invoke-Native 'go test' { go test ./... }

# Version info, icon and manifest for each exe: antivirus ML models flag bare unsigned Go exes
# (Microsoft's Trojan:Win32/Wacatac.B!ml).
function New-Resources([string]$Exe, [string]$Description) {
    Invoke-Native "resources $Exe" { go run github.com/tc-hib/go-winres@v0.3.3 simply --arch amd64 --out cmd\MultiRouteNetwork\rsrc --icon packaging\icons\hicolor\256x256\apps\MultiRouteNetwork.png --manifest cli --product-name MultiRouteNetwork --file-description $Description --original-filename $Exe --product-version $Version --file-version $Version --copyright 'MultiRouteNetwork contributors' }
}

# One source, two exes: a console CLI, and a windowless build that defaults to the tray.
New-Resources 'MultiRouteNetwork.exe' 'MultiRouteNetwork'
Invoke-Native 'build cli'  { go build -trimpath -ldflags "-s -w -X main.version=$Version" -o dist\MultiRouteNetwork.exe .\cmd\MultiRouteNetwork }
New-Resources 'MultiRouteNetwork-tray.exe' 'MultiRouteNetwork tray'
Invoke-Native 'build tray' { go build -trimpath -ldflags "-s -w -H=windowsgui -X main.defaultCmd=tray -X main.version=$Version" -o dist\MultiRouteNetwork-tray.exe .\cmd\MultiRouteNetwork }
Remove-Item cmd\MultiRouteNetwork\rsrc_windows_amd64.syso
Invoke-Native 'icon' { .\dist\MultiRouteNetwork.exe _icon dist\MultiRouteNetwork.ico }

& "$PSScriptRoot\fetch-wintun.ps1" -OutDir dist

$iscc = $env:ISCC
if (-not $iscc) { $iscc = (Get-Command ISCC.exe -ErrorAction SilentlyContinue).Source }
if (-not $iscc) {
    $iscc = Get-ChildItem 'C:\Program Files (x86)\Inno Setup 6\ISCC.exe', 'C:\Program Files\Inno Setup 6\ISCC.exe' -ErrorAction SilentlyContinue | Select-Object -First 1 -ExpandProperty FullName
}
if (-not $iscc) { throw 'ISCC.exe not found. Install Inno Setup 6 or set $env:ISCC.' }

$isccArgs = @("/DVersion=$Version")
if ($RepoUrl) { $isccArgs += "/DRepoUrl=$RepoUrl" }
Invoke-Native 'inno setup' { & $iscc @isccArgs installer\MultiRouteNetwork.iss }

$portable = "dist\MultiRouteNetwork-$Version-portable.zip"
Compress-Archive -Path dist\MultiRouteNetwork.exe, dist\MultiRouteNetwork-tray.exe, dist\wintun.dll, dist\wintun-LICENSE.txt, LICENSE, THIRD-PARTY-NOTICES.md, README.md -DestinationPath $portable
Get-ChildItem dist\MultiRouteNetwork-setup-$Version.exe, $portable | ForEach-Object { '{0}  {1:N1} MB' -f $_.Name, ($_.Length / 1MB) }
