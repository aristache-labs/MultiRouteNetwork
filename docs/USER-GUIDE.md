# MultiRoute user and technical guide

Detailed behavior for the Windows desktop and routing engine. For downloads,
screenshots and quick setup, see the [project README](../README.md).

## What is included

The Windows desktop uses Wails v3, React/TypeScript and WebView2. Its window runs
without elevation and controls a separately elevated engine through a per-user
named pipe. No account, activation service or anonymous usage reporting is added.

| Area | Current behavior |
|---|---|
| Overview | Connected adapter cards, registered applications, running state, traffic rates and graphs |
| Application routing | All eligible adapters, one adapter, or a selected subset; configurable fallback |
| Matching | Executable name or exact path; exact-path rules take precedence |
| Limits | Separate download/upload limits for applications and adapters |
| Caps | Daily/monthly download-plus-upload caps, warning at 80%, blocking at the cap |
| Usage | Local SQLite history; Today, 7 days, 30 days, This month, All; CSV export |
| DNS | Per-app Windows default, Cloudflare, Google, Quad9 or custom plain DNS |
| Startup | Separate Start with Windows and Start routing when it opens controls |
| Window | Dark/light header toggle; styled Hide to tray / Exit prompt with remembered choice |
| Updates | Visual placeholder; automatic installation remains disabled |

Overview shows only applications you add with **+ Add app**. Other routed traffic
can still appear in Usage. Multiple processes matching one application rule are
aggregated; unidentified owners remain **Unknown** rather than being guessed.

The underlying engine retains command-line/tray code for Windows, macOS and Linux.
The Wails desktop and the recent buffer/lifecycle optimizations described here
are Windows work; they do not establish equivalent desktop features or performance
on other platforms. This repository does not contain an Android app.

## How routing works

Every new connection is assigned to one eligible adapter and stays pinned there.
An application opening several connections can use several adapters simultaneously.
A single TCP download, such as one ordinary `curl` transfer, is not split across links.

- **Use all** uses the automatic weighted scheduler across eligible adapters.
- **Several adapters** uses the same scheduler within the selected subset.
- **One adapter** pins new flows to the selected adapter, subject to its fallback rule.
- **Use whichever adapter answers fastest** intentionally chooses one permitted
  adapter for new connections, with an eight-second hold to reduce switching.
- Rule changes affect new connections. Existing ones retain their adapter.
- A preferred adapter in MultiRoute does not change the Windows default route.

Automatic balancing uses busy-flow load and smoothed capacity estimates, with
health checks, cooldowns and retry exclusions. Idle TCP connections are discounted.
See [DEVELOPING.md](DEVELOPING.md) for the exact policy and IPv6 trade-offs.

**Ethernet and Wi-Fi connected to the same router share its WAN capacity.** They
are not two independent internet connections. A phone hotspot may provide a
separate upstream, but shared bottlenecks and the server can still limit total speed.

Real VPNs such as Tailscale may appear as informational cards but are not routing
candidates. The engine TUN and WSL/Docker/Hyper-V infrastructure adapters are hidden.

## Run the Windows desktop

Build the package from source as described below, or use matching assets actually
available on the [repository's releases page](https://github.com/aristache-labs/MultiRouteNetwork/releases).
This README does not assume an installer or hosted package repository exists.

Keep these files together in `dist/multiroute`:

- `MultiRoute.exe` — desktop window and its system tray controls.
- `MultiRouteNetwork.exe` — routing engine.
- `wintun.dll` and `wintun-LICENSE.txt` — native driver and its license.
- `go-winio-LICENSE.txt` and `README.txt` — accompanying package notices/instructions.

WebView2 is required. Launch **MultiRoute.exe** yourself. Start requests elevation
for the engine and launches it as `run -hidden`; it does not launch engine tray mode.
Unsigned builds may receive Windows trust or antivirus prompts.

### Start, Stop and Exit

Adapters and saved applications are visible before routing starts. You can add or
edit application rules while stopped, including when the engine is not running.

- **Start** enables routing and statistics. The first Start performs full TUN setup.
- **Stop** closes routed TCP/UDP sockets and pending dials, removes capture routes
  and pauses routed accounting. The Windows adapter and stack remain prepared
  for faster subsequent starts. Applications may need to reconnect.
- **Hide to system tray** hides the window and leaves the engine's current state intact.
- **Exit** flushes state, stops routing and closes the engine, retained adapter and stack.

The window X shows a styled choice between Hide and Exit. Its checkbox remembers
the selected action. Change that preference in **Settings → When closing the window**.
A cancelled elevation prompt or a startup delay is distinct from an established
engine communication failure.

Start with Windows starts the window in the tray. Start routing when it opens is
a separate setting; enabling one does not imply enabling the other.

## Rates, graphs and limits

Overview lists running applications first and confirmed stopped applications last,
with unknown process states in between. Within each group, applications are ordered
by combined download/upload activity averaged
with a two-second response time. Ranking updates once per second; a row passes
another only with a lead above both 20% and 16 KB/s (128 kbit/s). Quiet traffic
and near-equal rates retain their existing order.

Traffic counters update every **70 ms**. Graphs display **60 seconds** (858 samples), matching the approximate scroll
rate measured in the reference recording within the existing card size,
with teal (`#00D2C1`) for download and yellow (`#FFCE00`) for upload, displayed at 85% opacity with subtler graph fills. Status polling and traffic measurements run every **70 ms**; numeric rates refresh every **70 ms**; graph scrolling and scale changes are smoothed;
raw byte accounting and traffic enforcement do not use interpolated display values.
Graph curves use a 1-second trailing average of the 70 ms samples to reduce short
burst spikes; measured numeric rates and byte totals remain unchanged.
On idle-to-traffic transitions, the vertical ceiling is seeded from available
measurements and held during the one-second ramp to prevent top-edge clipping.
Header and adapter graphs use a minimum visual ceiling of 100 Mbit/s so background traffic does not
fill the plot. This ceiling expands for higher traffic and imposes no traffic limit.
Adapter graphs use square-root vertical scaling to make low/sustained traffic more
visible below short peaks without enlarging cards. Download and upload share
this visual scale; numeric rates remain measured values. Header and application
graphs keep linear vertical scaling. Application graphs use their own automatic
scale with a minimum ceiling of 128 kbit/s to reveal small activity peaks; this
does not change the separate 100 Mbit/s scale of unlimited application rate bars.

Each application's `using` text lists adapters with open connections, ordered by
observed usage. Running state reflects process presence, not whether data is moving.

Application rate bars start at zero. With a directional speed limit, that limit
is the bar maximum. Without a limit, the visual maximum is **100 Mbit/s**. Values
above the maximum fill the bar; they remain visible numerically. This display scale
never imposes a 100 Mbit/s traffic cap and is not scaled to the fastest app or Link Speed.

Adapter cards omit negotiated Link Speed; they show current transfer rates and traffic totals.

Application-wide and applicable adapter limits combine. Empty fields mean unlimited.
Small UDP traffic counts toward application limits. Data caps count download plus
upload; reaching the cap stops that application's routed flows for the period.
Removing the cap clears its block and warning immediately; raising it rechecks
consumed usage without restarting the engine or clearing history. Interrupted
connections must reconnect.

## Usage, state and limitations

Rules, settings and usage are stored locally in
`%LOCALAPPDATA%\MultiRouteNetwork\multiroute.db`; window preferences are stored
alongside it. Usage covers traffic observed through the engine, not historical
Windows traffic from before installation. Adapter counters and per-application
payload totals measure different layers and need not match exactly.

If upgrading from a legacy `net-merge` state directory, back it up first. Local
state recovery performed during development is not a universal automatic migration.
If the application list appears empty after a rename, preserve both state directories
before attempting recovery.

Per-app DNS only covers ordinary TCP/UDP port 53 traffic routed through the engine.
Encrypted DNS, application-owned resolvers and Windows resolver traffic outside
that path are not covered. One connection remains on one adapter; existing flows
do not migrate automatically when you edit a rule.

Automatic updates are not implemented as an active installation channel. The
Settings controls are placeholders until a signed release channel is available.
Any future update installation must happen while routing is stopped.

The dashboard adapts to the space remaining beside Settings or an application
panel. Adapter cards move onto additional rows; narrow application rows stack
identity, labeled download/upload bars and usage totals. The dashboard and panel
scroll independently so controls remain reachable in smaller windows.

## Build and verify

Requires Go as specified in `go.mod`, Node/npm for the frontend, Windows for the
desktop package and WebView2 on the machine running it. From the repository root:

```powershell
# Build embedded assets before testing a fresh checkout.
npm ci --prefix desktop/frontend
npm run build --prefix desktop/frontend

# Run tests without starting the GUI.
go test ./...
go test -race ./internal/tun ./internal/control ./internal/rules ./internal/engine
go vet ./...

# Build frontend and package GUI/engine; does not launch them.
.\scripts\package-desktop.ps1
```

`VERSION` records the current source milestone. The desktop packaging script reads
it by default and accepts `-Version X.Y.Z` to override the engine build stamp.
`MultiRouteNetwork.exe version` reports the engine version. Generic builds also
have a matching source default; tagged builds can override it with `-X main.version=...`.

For manual builds:

```powershell
Push-Location desktop/frontend
npm ci
npm run build
Pop-Location

go build -ldflags "-H=windowsgui -X main.version=0.4.9" -o dist/multiroute/MultiRouteNetwork.exe ./cmd/MultiRouteNetwork
go build -o dist/multiroute/MultiRoute.exe ./desktop
```

The separate legacy installer/tray script, `scripts/build.ps1`, reads `VERSION`
by default too. It recreates `dist` and requires Inno Setup; use the desktop
packaging script for the Wails package. Building does not publish a release or
replace an installed application. Keep license files with redistributed binaries.

Linux/macOS engine build and packaging details remain in
[DEVELOPING.md](DEVELOPING.md). Package-host and Homebrew publishing must be
configured for this fork rather than inferred from the inherited changelog.

## Diagnostics and performance

Identified curl flows write bounded, rotating `curl-routing.jsonl` records in the
state directory. They include selection reasons, opened adapters, byte counts,
rates and close states. Selection attempts alone do not prove successful routing.
The log excludes URLs, payloads, addresses and executable paths.

`scripts/compare-downloads.ps1` compares direct and engine downloads with equal
stream counts and physical-counter checks. It asks you to control routing/rules
and does not start the GUI. Downloads consume real bandwidth; see the script's
parameters and [diagnostic documentation](DEVELOPING.md) before running it.

The engine supports a **loopback-only** Go profiler through `-pprof`. Windows
telemetry caches adapter identity metadata for three seconds while refreshing
traffic counters every 70 ms. Windows packet IO reuses buffers without changing
the scheduler or packet ownership guarantees.

Local validation after buffer reuse measured approximately **94% fewer sampled
allocated bytes**. Eight alternating Ethernet A/B runs measured **19% lower mean
CPU per transferred GiB**. Throughput averaged 733 Mbit/s with reuse versus
691 Mbit/s without; only four pairs against an external server were measured,
so this is not a guaranteed speedup or proof that smaller regressions are impossible.
CPU normalization uses interpolated connection counters. Repeated A/B comparison
was performed without CPU profiling active; allocation measurements came from
separate captures. Do not treat cumulative allocations as resident RAM.

## Engine commands

```text
MultiRouteNetwork run -hidden     headless elevated engine used by the Windows desktop
MultiRouteNetwork tray            legacy tray application
MultiRouteNetwork on | off        start / pause the running engine
MultiRouteNetwork quit            stop and exit the engine
MultiRouteNetwork status | list   status and adapter information
MultiRouteNetwork limit           inspect saved legacy adapter caps
MultiRouteNetwork socks           SOCKS5 proxy, default 127.0.0.1:1080
MultiRouteNetwork cleanup         remove leftover capture routes after a failure
MultiRouteNetwork version         engine build version
```

Use the desktop startup settings for the Wails application; legacy tray autostart
commands configure a different launch path. Advanced flags, platform setup and
profiling instructions are in [DEVELOPING.md](DEVELOPING.md).

## Project and support

- Repository and issues: [aristache-labs/MultiRouteNetwork](https://github.com/aristache-labs/MultiRouteNetwork).
- Changes: [CHANGELOG.md](../CHANGELOG.md); versions are published only when explicitly tagged/released.
- Contributing: [CONTRIBUTING.md](../CONTRIBUTING.md). Security reports: [SECURITY.md](../SECURITY.md).
- License: [MIT](../LICENSE). Dependencies and driver terms: [third-party notices](../THIRD-PARTY-NOTICES.md).
- Support development: [buymeacoffee.com/aristache](https://buymeacoffee.com/aristache).

### Desktop tray menu

The desktop tray remains available while the window is hidden. Open its menu to
show MultiRoute, see Online/Offline status and total/adapter transfer rates, start
or stop routing, or exit both the GUI and engine. Double-click the tray icon to
show the window. Rates refresh once per second and the tooltip shows the total.
Tray controls use the same engine commands as the dashboard; adapter-selection
and balancing policies are unchanged.
