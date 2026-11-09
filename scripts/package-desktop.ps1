# Packages MultiRoute next to MultiRouteNetwork.exe. Automatic updates stay off.
# Requires Go, Node, and a WebView2 runtime on the machine that runs the window.
param(
    [string] $OutDir = "dist\multiroute"
)
$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
Set-Location $root

Push-Location desktop\frontend
try {
if (-not (Test-Path node_modules)) { npm install; if ($LASTEXITCODE -ne 0) { throw "npm install failed" } }
npm run build
if ($LASTEXITCODE -ne 0) { throw "Frontend build failed" }
} finally { Pop-Location }

New-Item -ItemType Directory -Force -Path $OutDir | Out-Null
go build -ldflags "-H=windowsgui" -o (Join-Path $OutDir "MultiRouteNetwork.exe") .\cmd\MultiRouteNetwork
if ($LASTEXITCODE -ne 0) { throw "MultiRouteNetwork engine build failed" }
go build -o (Join-Path $OutDir "MultiRoute.exe") .\desktop
if ($LASTEXITCODE -ne 0) { throw "MultiRoute desktop build failed" }

$winioModule = go list -m -f '{{.Dir}}' github.com/Microsoft/go-winio
if ($LASTEXITCODE -ne 0) { throw "Could not locate go-winio license" }
Copy-Item -LiteralPath (Join-Path $winioModule "LICENSE") -Destination (Join-Path $OutDir "go-winio-LICENSE.txt")

$wintun = Join-Path $OutDir "wintun.dll"
if (-not (Test-Path $wintun)) {
    & (Join-Path $root "scripts\fetch-wintun.ps1") -OutDir (Join-Path $root $OutDir)
}

@(
    "MultiRoute",
    "Built from the MIT MultiRouteNetwork engine. Third-party notices stay in the repository.",
    "WebView2 must be installed (current Windows 11 includes it).",
    "Automatic updates are disabled until a signed release channel exists.",
    "Install an update only while routing is stopped."
) | Set-Content -Encoding utf8 (Join-Path $OutDir "README.txt")

Write-Host "Packaged $OutDir"
