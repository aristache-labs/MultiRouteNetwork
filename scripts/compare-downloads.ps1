# Run manually. Never starts/stops MultiRoute, writes rules, or changes OS routes.
[CmdletBinding()]
param(
    [string[]]$Adapters = @('Ethernet', 'Wi-Fi', 'Ethernet 4'),
    [string[]]$CounterAdapters = @(),
    [ValidateRange(3, 120)][int]$Seconds = 15,
    [ValidateRange(1, 24)][int]$Streams = 3,
    [string]$Url = 'https://sbg.proof.ovh.net/files/10Gb.dat',
    [string]$OutputDirectory,
    [string]$StateDirectory,
    [switch]$PlanOnly,
    [switch]$LibraryOnly
)
$ErrorActionPreference = 'Stop'

function Read-EngineStatus {
    $sid = [Security.Principal.WindowsIdentity]::GetCurrent().User.Value
    $pipe = [IO.Pipes.NamedPipeClientStream]::new('.', "MultiRoute-$sid", [IO.Pipes.PipeDirection]::InOut, [IO.Pipes.PipeOptions]::Asynchronous)
    try {
        $pipe.Connect(2000)
        $writer = [IO.StreamWriter]::new($pipe, [Text.UTF8Encoding]::new($false), 1024, $true)
        $reader = [IO.StreamReader]::new($pipe, [Text.Encoding]::UTF8, $false, 1024, $true)
        try {
            $writer.AutoFlush = $true
            $writer.WriteLine('{"op":"status"}')
            $read = $reader.ReadLineAsync()
            if (-not $read.Wait(7000)) { throw 'Engine status timed out.' }
            $reply = $read.Result | ConvertFrom-Json
            if (-not $reply.ok -or -not $reply.status) { throw 'Engine did not return status.' }
            return $reply.status
        } finally { $writer.Dispose(); $reader.Dispose() }
    } finally { $pipe.Dispose() }
}

function Test-NoPositiveLimits($Object) {
    if (-not $Object) { return $true }
    foreach ($property in $Object.PSObject.Properties) {
        if ([double]$property.Value -gt 0) { return $false }
    }
    return $true
}

function Assert-EngineRule($Status, [string]$Adapter, [string]$CurlPath) {
    if (-not $Status.running) { throw 'Routing is stopped.' }
    $exact = @($Status.rules | Where-Object { $_.exactPath -and $_.path -ieq $CurlPath })
    $named = @($Status.rules | Where-Object { -not $_.exactPath -and $_.exe -ieq 'curl.exe' })
    $rule = if ($exact.Count) { $exact[0] } elseif ($named.Count) { $named[0] } else { $null }
    if (-not $rule -or $rule.mode -ne 'one' -or @($rule.adapters).Count -ne 1 -or $rule.adapters[0] -ne $Adapter -or $rule.lowestLatency -or $rule.fallback -ne 'wait') {
        throw "Set curl.exe to One adapter: $Adapter; fallback: Wait; fastest adapter OFF."
    }
    if ($rule.downMbps -gt 0 -or $rule.upMbps -gt 0 -or $rule.capGB -gt 0 -or
        -not (Test-NoPositiveLimits $rule.adapterDown) -or -not (Test-NoPositiveLimits $rule.adapterUp) -or
        $Status.settings.downMbps -gt 0 -or $Status.settings.upMbps -gt 0 -or
        -not (Test-NoPositiveLimits $Status.settings.adapterDown) -or -not (Test-NoPositiveLimits $Status.settings.adapterUp)) {
        throw 'A configured speed limit/data cap would invalidate the comparison. Remove it manually first.'
    }
    $card = @($Status.adapters | Where-Object { $_.name -eq $Adapter -and $_.routable })
    if (-not $card.Count -or $card[0].state -ne 'online') { throw "$Adapter is not online and routable." }
}

function Convert-CurlMeasurement([string]$Metrics, [int]$ExitCode) {
    $parts = $Metrics.Trim().Split(',')
    if ($parts.Count -ne 5) { throw "Invalid curl metrics: $Metrics" }
    $culture = [Globalization.CultureInfo]::InvariantCulture
    $bytes = [double]::Parse($parts[0], $culture)
    $speed = [double]::Parse($parts[1], $culture)
    $elapsed = [double]::Parse($parts[2], $culture)
    $http = [int]$parts[3]
    [pscustomobject]@{
        Bytes = $bytes; CurlMbps = $speed * 8 / 1e6; ElapsedSeconds = $elapsed
        HttpCode = $http; Connections = [int]$parts[4]; ExitCode = $ExitCode
        Valid = ($ExitCode -in @(0, 28) -and $http -ge 200 -and $http -lt 300 -and $bytes -gt 0)
    }
}

function Get-ComparisonCounters([string[]]$Names) {
    # Apple's phone Ethernet driver is listed by Get-NetAdapter but can be
    # absent from Get-NetAdapterStatistics. .NET's IP counters cover it too.
    $interfaces = [Net.NetworkInformation.NetworkInterface]::GetAllNetworkInterfaces()
    foreach ($name in $Names) {
        $interface = @($interfaces | Where-Object { $_.Name -eq $name })
        if ($interface.Count -ne 1) { throw "Cannot identify counters for $name." }
        $value = $interface[0].GetIPStatistics()
        [pscustomobject]@{ Adapter = $name; Received = [double]$value.BytesReceived; Sent = [double]$value.BytesSent }
    }
}

if ($LibraryOnly) { return }
$plan = foreach ($phase in @('direct', 'engine')) {
    foreach ($adapter in $Adapters) {
        [pscustomobject]@{ Phase = $phase; Adapter = $adapter; Streams = $Streams; Seconds = $Seconds }
    }
}
if ($PlanOnly) { $plan | Format-Table; return }
if ($Url -notmatch '^https://') { throw 'Use an HTTPS download URL.' }
$curlPath = (Get-Command curl.exe -CommandType Application | Select-Object -First 1).Source
if (-not $OutputDirectory) { $OutputDirectory = Join-Path $PSScriptRoot ('../diagnostics/download-comparison-' + (Get-Date -Format 'yyyyMMdd-HHmmss')) }
$OutputDirectory = [IO.Path]::GetFullPath($OutputDirectory)
New-Item -ItemType Directory -Path $OutputDirectory -Force | Out-Null
$stateDirectory = Join-Path $env:LOCALAPPDATA 'MultiRouteNetwork'
$initial = Read-EngineStatus
if (-not $CounterAdapters.Count) { $CounterAdapters = $Adapters }
[pscustomobject]@{ Url = $Url; Adapters = $Adapters; CounterAdapters = $CounterAdapters; Streams = $Streams; Seconds = $Seconds; Started = (Get-Date).ToString('o') } | ConvertTo-Json -Depth 10 | Set-Content -LiteralPath (Join-Path $OutputDirectory 'test-config.json') -Encoding UTF8
$initial | ConvertTo-Json -Depth 30 | Set-Content -LiteralPath (Join-Path $OutputDirectory 'initial-status.json') -Encoding UTF8
Write-Host 'Ethernet and Wi-Fi share one router/WAN: do NOT add their internet rates together.'
Write-Host 'Ethernet 4 uses the phone hotspot. These tests can consume multiple GB, including mobile data.'
Write-Host 'Close other downloads and curl processes. No files are downloaded to disk (output is NUL).'
Write-Host 'The script only READS engine status. You control Start/Stop and rules manually.'
$allResults = @(); $summary = @()

foreach ($case in $plan) {
    Write-Host "`n$($case.Phase.ToUpper()) / $($case.Adapter): $Streams parallel transfers, $Seconds seconds each."
    if ($case.Phase -eq 'direct') {
        Read-Host 'Press Stop in MultiRoute, wait until stopped, then press Enter here (Ctrl+C cancels)' | Out-Null
        $status = Read-EngineStatus
        if ($status.running -or $status.starting) { throw 'Engine must be stopped for direct tests.' }
        $capture = @(Get-NetRoute -AddressFamily IPv4 | Where-Object { $_.InterfaceAlias -eq 'MultiRouteNetwork' -and $_.DestinationPrefix -in @('0.0.0.0/1', '128.0.0.0/1') })
        if ($capture.Count) { throw 'TUN capture routes remain installed. Direct test cancelled.' }
    } else {
        Read-Host "Press Start; set curl.exe to One adapter: $($case.Adapter), fallback Wait, no limits/cap, fastest OFF; Save, then Enter" | Out-Null
        $status = Read-EngineStatus
        Assert-EngineRule $status $case.Adapter $curlPath
        $legacy = Join-Path $stateDirectory 'status.json'
        if (Test-Path -LiteralPath $legacy) {
            $legacyStatus = Get-Content -LiteralPath $legacy -Raw | ConvertFrom-Json
            if (@($legacyStatus.NICs | Where-Object { $_.Name -eq $case.Adapter -and $_.LimitMbps -gt 0 }).Count) { throw 'Legacy engine adapter limit is enabled.' }
        }
    }
    $nic = Get-NetAdapter -Name $case.Adapter -ErrorAction Stop
    if ($nic.Status -ne 'Up') { throw "$($case.Adapter) is disconnected." }
    $addresses = @(Get-NetIPAddress -InterfaceIndex $nic.ifIndex -AddressFamily IPv4 | Where-Object { $_.AddressState -eq 'Preferred' -and $_.IPAddress -notlike '169.254.*' })
    if (-not $addresses.Count) { throw 'Adapter has no usable IPv4 address.' }
    $sourceIP = $addresses[0].IPAddress
    $label = $case.Phase + '-' + ($case.Adapter -replace '[^a-zA-Z0-9-]', '_')
    $status | ConvertTo-Json -Depth 30 | Set-Content -LiteralPath (Join-Path $OutputDirectory "$label-status.json") -Encoding UTF8
    $nic | Select-Object Name, InterfaceDescription, ifIndex, LinkSpeed | ConvertTo-Json | Set-Content -LiteralPath (Join-Path $OutputDirectory "$label-adapter.json") -Encoding UTF8
    $jobs = @(); $stageStart = Get-Date; $releaseAt = [DateTime]::UtcNow.AddSeconds(8)
    $beforeCounters = @(Get-ComparisonCounters $CounterAdapters)
    try {
        for ($i = 1; $i -le $Streams; $i++) {
            $errorFile = Join-Path $OutputDirectory "$label-$i-stderr.txt"
            $jobs += Start-Job -ArgumentList $curlPath, $Url, $Seconds, $case.Phase, $sourceIP, $releaseAt, $errorFile, $i -ScriptBlock {
                param($Curl, $Target, $Duration, $Phase, $Source, $Release, $ErrorFile, $Index)
                $ErrorActionPreference = 'Continue'
                while ([DateTime]::UtcNow -lt $Release) { Start-Sleep -Milliseconds 20 }
                $arguments = @('--ipv4', '--http1.1', '--noproxy', '*', '--location', '--fail', '--silent', '--show-error', '--max-time', "$Duration", '--output', 'NUL', '--write-out', '%{size_download},%{speed_download},%{time_total},%{http_code},%{num_connects}')
                # Binding a physical IP is ONLY for the bypass baseline. Engine tests
                # must use an ordinary socket so the TUN and curl rule can capture it.
                if ($Phase -eq 'direct') { $arguments += @('--interface', $Source) }
                $arguments += $Target
                $started = [DateTime]::UtcNow
                $metrics = (& $Curl @arguments 2> $ErrorFile) -join ''
                $code = $LASTEXITCODE
                [pscustomobject]@{ Index = $Index; Metrics = $metrics; ExitCode = $code; Started = $started.ToString('o'); Finished = [DateTime]::UtcNow.ToString('o') }
            }
        }
        $raw = @($jobs | Wait-Job | Receive-Job -ErrorAction Stop)
    } finally {
        foreach ($job in $jobs) { if ($job.State -eq 'Running') { Stop-Job $job }; Remove-Job $job }
    }
    $results = foreach ($row in $raw) {
        $measurement = Convert-CurlMeasurement $row.Metrics $row.ExitCode
        $measurement | Add-Member -NotePropertyName Phase -NotePropertyValue $case.Phase
        $measurement | Add-Member -NotePropertyName Adapter -NotePropertyValue $case.Adapter
        $measurement | Add-Member -NotePropertyName Stream -NotePropertyValue $row.Index
        $measurement | Add-Member -NotePropertyName Started -NotePropertyValue $row.Started
        $measurement | Add-Member -NotePropertyName Finished -NotePropertyValue $row.Finished
        $measurement
    }
    if (@($results).Count -ne $Streams) { throw "Missing worker results. Inspect $label stream logs." }
    $results | Export-Csv -LiteralPath (Join-Path $OutputDirectory "$label-streams.csv") -NoTypeInformation
    $allResults += $results
    $afterCounters = @(Get-ComparisonCounters $CounterAdapters)
    $counterDelta = foreach ($after in $afterCounters) {
        $before = $beforeCounters | Where-Object { $_.Adapter -eq $after.Adapter }
        [pscustomobject]@{ Adapter = $after.Adapter; ReceivedBytes = $after.Received - $before.Received; SentBytes = $after.Sent - $before.Sent }
    }
    $counterDelta | Export-Csv -LiteralPath (Join-Path $OutputDirectory "$label-physical-counters.csv") -NoTypeInformation
    $first = ($results.Started | Sort-Object | Select-Object -First 1)
    $last = ($results.Finished | Sort-Object | Select-Object -Last 1)
    $wallSeconds = ([DateTime]::Parse($last) - [DateTime]::Parse($first)).TotalSeconds
    $bytes = ($results | Measure-Object Bytes -Sum).Sum
    $skew = ((($results.Started | Sort-Object | Select-Object -Last 1) -as [DateTime]) - ([DateTime]::Parse($first))).TotalSeconds
    $verified = $case.Phase -eq 'direct'
    $targetReceived = ($counterDelta | Where-Object { $_.Adapter -eq $case.Adapter }).ReceivedBytes
    $otherReceived = ($counterDelta | Where-Object { $_.Adapter -ne $case.Adapter } | Measure-Object ReceivedBytes -Sum).Sum
    # Source-IP binding is checked against physical counters as well. Heavy
    # unrelated traffic or unexpected egress makes the result inconclusive.
    $physicalVerified = $targetReceived -gt 0 -and $otherReceived -le [Math]::Max(5e6, $targetReceived * 0.1)
    if (@($counterDelta | Where-Object { $_.ReceivedBytes -lt 0 -or $_.SentBytes -lt 0 }).Count) { $physicalVerified = $false }
    if ($case.Phase -eq 'engine') {
        $events = foreach ($logName in @('curl-routing.jsonl.1', 'curl-routing.jsonl')) {
            $logPath = Join-Path $stateDirectory $logName
            if (Test-Path -LiteralPath $logPath) {
                foreach ($line in Get-Content -LiteralPath $logPath) { try { $e = $line | ConvertFrom-Json; if ([DateTimeOffset]::Parse($e.time).UtcDateTime -ge $stageStart.ToUniversalTime()) { $e } } catch {} }
            }
        }
        $events | ConvertTo-Json -Depth 30 | Set-Content -LiteralPath (Join-Path $OutputDirectory "$label-curl-events.json") -Encoding UTF8
        $opened = @($events | Where-Object { $_.event -eq 'opened' })
        $verified = $opened.Count -ge $Streams -and @($opened | Where-Object { $_.adapter -ne $case.Adapter }).Count -eq 0
    }
    $valid = @($results | Where-Object { -not $_.Valid }).Count -eq 0 -and $results.Count -eq $Streams -and $skew -le 2 -and $verified -and $physicalVerified
    $summary += [pscustomobject]@{ Phase = $case.Phase; Adapter = $case.Adapter; Streams = $Streams; Bytes = $bytes; WallSeconds = [Math]::Round($wallSeconds, 3); AggregateMbps = [Math]::Round($bytes * 8 / 1e6 / $wallSeconds, 2); StartSkewSeconds = [Math]::Round($skew, 3); RoutingVerified = $verified; PhysicalCountersVerified = $physicalVerified; Valid = $valid }
    $summary | Export-Csv -LiteralPath (Join-Path $OutputDirectory 'summary.csv') -NoTypeInformation
    $summary[-1] | Format-List
    Write-Host 'curl exit 28 is expected at the configured time limit if HTTP succeeded and bytes were received.'
    if (@($results | Where-Object { $_.HttpCode -eq 429 }).Count) { throw 'Server returned HTTP 429. Results were saved; stop rather than compare a rate-limited endpoint.' }
}
$allResults | Export-Csv -LiteralPath (Join-Path $OutputDirectory 'all-streams.csv') -NoTypeInformation
Write-Host "`nResults: $OutputDirectory"
Write-Host 'Restore your original curl rule manually (see initial-status.json). The script leaves the final rule and routing state as YOU selected them.'
$summary | Format-Table
