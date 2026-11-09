# Writes the GitHub release notes for one version: an install line plus that version's section of
# CHANGELOG.md. Fails if the section is missing, so a release can't go out without a changelog.
#   scripts\release-notes.ps1 -Version 0.1.1 -Out notes.md
param(
    [Parameter(Mandatory)][string]$Version,
    [string]$Out = 'release-notes.md'
)
$ErrorActionPreference = 'Stop'
$changelog = Get-Content (Join-Path (Split-Path $PSScriptRoot -Parent) 'CHANGELOG.md') -Raw
$m = [regex]::Match($changelog, "(?ms)^## \[$([regex]::Escape($Version))\][^\r\n]*\r?\n(.*?)(?=^## \[|\z)")
if (-not $m.Success -or -not $m.Groups[1].Value.Trim()) {
    throw "CHANGELOG.md has no entry for $Version. Add a '## [$Version] - <date>' section first."
}
$install = "**MultiRouteNetwork ${Version}:** see the [README](https://github.com/aristache-labs/MultiRouteNetwork#readme) " +
    "for Windows desktop builds and engine setup. Use only matching assets actually attached to this release. " +
    "The Windows desktop needs WebView2; routing requests administrator privileges. " +
    "Build scripts do not publish an installer, configure a package repository or install updates automatically."
# UTF-8 without a BOM (Windows PowerShell's -Encoding utf8 adds one, which GitHub would show)
$path = $ExecutionContext.SessionState.Path.GetUnresolvedProviderPathFromPSPath($Out)
$footer = "---`n[Report an issue](https://github.com/aristache-labs/MultiRouteNetwork/issues) · " +
    "[Support development](https://buymeacoffee.com/aristache)"
[IO.File]::WriteAllText($path, $install + "`n`n" + $m.Groups[1].Value.Trim() + "`n`n" + $footer + "`n", (New-Object Text.UTF8Encoding $false))
Write-Host "wrote $Out"
