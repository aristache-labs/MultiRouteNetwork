# Offline tests only: no engine, GUI, route changes or downloads.
$ErrorActionPreference = 'Stop'
. (Join-Path $PSScriptRoot 'compare-downloads.ps1') -LibraryOnly
function Assert($Value, $Message) { if (-not $Value) { throw $Message } }
$measurement = Convert-CurlMeasurement '10000000,5000000.25,2.0,200,2' 28
Assert $measurement.Valid 'Expected timed download should be valid.'
Assert ($measurement.CurlMbps -eq 40.000002) 'Invariant rate conversion failed.'
Assert (-not (Convert-CurlMeasurement '0,0,2.0,000,0' 28).Valid) 'Timeout without data should fail.'
Assert (-not (Convert-CurlMeasurement '100,50,2.0,404,1' 22).Valid) 'HTTP failure should fail.'
Assert (-not (Convert-CurlMeasurement '100,50,2.0,200,1' 56).Valid) 'Transport failure should fail.'
$culture = [Threading.Thread]::CurrentThread.CurrentCulture
try {
    [Threading.Thread]::CurrentThread.CurrentCulture = [Globalization.CultureInfo]::GetCultureInfo('ro-RO')
    Assert ((Convert-CurlMeasurement '1000,1000.5,1.0,200,1' 0).CurlMbps -eq 0.008004) 'Romanian locale changed metric parsing.'
} finally { [Threading.Thread]::CurrentThread.CurrentCulture = $culture }
$status = '{"running":true,"settings":{},"rules":[{"exe":"curl.exe","mode":"one","adapters":["Wi-Fi"],"fallback":"wait"}],"adapters":[{"name":"Wi-Fi","routable":true,"state":"online"}]}' | ConvertFrom-Json
Assert-EngineRule $status 'Wi-Fi' 'C:\Windows\System32\curl.exe'
$status.rules[0].mode = 'all'
$rejected = $false
try { Assert-EngineRule $status 'Wi-Fi' 'C:\Windows\System32\curl.exe' } catch { $rejected = $true }
Assert $rejected 'Use all must not pass the fixed-adapter comparison.'
$status.rules[0].mode = 'one'
$status.settings | Add-Member -NotePropertyName adapterDown -NotePropertyValue ([pscustomobject]@{ 'Wi-Fi' = 10 })
$rejected = $false
try { Assert-EngineRule $status 'Wi-Fi' 'C:\Windows\System32\curl.exe' } catch { $rejected = $true }
Assert $rejected 'Adapter limit must invalidate the comparison.'
$tokens = $null; $errors = $null
[Management.Automation.Language.Parser]::ParseFile((Join-Path $PSScriptRoot 'compare-downloads.ps1'), [ref]$tokens, [ref]$errors) | Out-Null
Assert ($errors.Count -eq 0) 'Script parsing failed.'
Write-Host 'PASS: parsing, metric conversion, Romanian locale, timeout/errors, fixed-adapter rule and limits. No downloads performed.'
