# Manual download comparison

Ethernet and Wi-Fi share the home router and WAN. They are alternative paths to
the same internet service; adding their independent test results does not predict
combined throughput. Ethernet 4 is the phone hotspot and may provide a separate
WAN, subject to the phone connection. Link Speed is not internet capacity.

Open MultiRoute yourself before running the script. Use a normal PowerShell
terminal with Windows networking cmdlets. Close other downloads and curl
processes. The default test runs three parallel curl processes for 15 seconds on
each adapter directly, then repeats through the engine: six stages, potentially
multiple GB of data, including mobile data. Download output goes to NUL.

```powershell
Set-Location C:\Personal\MultiRouteNetwork
.\scripts\compare-downloads.ps1 -PlanOnly
.\scripts\compare-downloads.ps1
```

The script prompts before each stage. It never starts the GUI/engine, changes
routes, saves rules or toggles routing. For direct stages press Stop manually;
curl binds the adapter's IPv4 address. For engine stages press Start and manually
save a curl rule: One adapter, the requested adapter, fallback Wait, fastest OFF,
no speed limits or cap. Engine stages deliberately do not bind a physical IP.
The script reads status through the existing named pipe to validate the setup.
Restore your original curl rule yourself after the last stage; the original
status is saved for reference. On interruption the script removes its own worker
jobs but does not change the routing state or rule you selected.

Options: `-Seconds 20 -Streams 3`, `-Adapters 'Ethernet','Wi-Fi','Ethernet 4'`,
`-Url https://.../large-file`, and `-OutputDirectory C:\path\results`.
Keep the same URL, duration and stream count for direct/engine comparisons.

Results are under `diagnostics\download-comparison-<timestamp>` by default:

- `summary.csv`: aggregate Mbit/s based on received payload bytes divided by the
  common wall interval, not a sum of differently timed curl averages.
- Per-stream CSVs: bytes, curl rate, HTTP code, timing, connection count and exit code.
- Per-stage status, adapter details and physical receive/transmit counter deltas.
- Engine-stage curl diagnostic events and per-stream stderr files.

Exit 28 is an expected duration limit only if HTTP is successful and bytes were
received. Other transfer errors invalidate that stage. `Valid=False` also flags
wrong/missing diagnostic routing evidence, worker start skew over two seconds,
counter resets or substantial receive traffic on another adapter. Physical
counters include background traffic and are an egress sanity check, not exact
per-curl accounting. Curl diagnostics must identify the process; otherwise an
engine stage is inconclusive even if the saved rule looks correct.

Compare engine/direct on each adapter separately. Repeat the whole comparison
to check time-of-day/server variation before attributing a difference to the
engine. A low direct Wi-Fi rate points outside the engine; a consistently large
engine-only loss merits engine profiling. This test isolates paths and does not
evaluate the Use all scheduler or prove an ISP/server bottleneck by itself.

Offline verification (no traffic or engine access):

```powershell
.\scripts\compare-downloads.tests.ps1
```

If both old and renamed state directories exist, set `-StateDirectory` explicitly
to the directory used by the running engine, for example
`-StateDirectory "$env:LOCALAPPDATA\net-merge"` or
`-StateDirectory "$env:LOCALAPPDATA\MultiRouteNetwork"`. Automatic detection
prefers MultiRouteNetwork when its database exists. The named pipe remains
MultiRoute-<SID>. Direct checks recognize both TUN adapter names.

The OVH endpoint returned HTTP 429 with six parallel streams in a live test.
The default is three streams; any HTTP 429 saves the partial summary and aborts
the comparison, because a rate-limited server is not a valid engine baseline.
