# Changelog

Changes in MultiRouteNetwork, whose Windows desktop application is named MultiRoute.
Current source version: **0.4.6**. See [README.md](README.md) for current behavior and limitations.

Versions **0.4.0–0.4.5** were documented retrospectively on **2026-11-10** to group
work that accumulated after 0.3.7. Their dates record this documentation milestone,
not six separate public releases. No corresponding tags or published assets are
implied. Entries **0.3.7 and earlier** preserve the inherited engine history;
references there to package hosts, upstream products or release availability are
historical, not claims about this fork's infrastructure.

## [Unreleased]

## [0.4.6] - 2026-11-10

### Changed
- Sampled traffic and refreshed displayed rates every 70 ms, independently of the unchanged balancing cycle.
- Retained approximately 60 seconds of graph history using 858 samples and one-second visual smoothing.
- Applied separate application graph scaling to reveal small activity peaks; preserved header and adapter scale floors.
- Lifted adapter graph detail with square-root vertical scaling without enlarging cards.
- Stabilized application ordering using averaged combined traffic, one-second ranking and a minimum lead before reordering.
- Applied translucent teal download and yellow upload accents to traffic graphs, values and bars.
- Highlighted monthly application usage with the download accent.
- Increased application history width and strengthened graph lines.

### Fixed
- Prevented header graph movement when changing rate labels altered their digit count or width.
- Positioned the header graph beside its traffic values.
- Prevented startup curves from clipping above the plot when traffic begins after an empty history.
- Preserved graph scroll continuity across telemetry polls and prevented extrapolation beyond available samples.
- Kept low background traffic near the baseline in header and adapter graphs.
- Reflowed adapter cards, badges, rates and totals when resizing or opening Settings.
- Added compact application rows and preserved scrolling in narrow windows and side panels.
- Wrapped long application titles and corrected the application/process count separator.
- Extracted higher-resolution executable icons and preserved native transparency for clearer high-DPI display.

### Removed
- Removed negotiated Link Speed labels from adapter cards.

### Added
- Added offline browser checks for responsive layouts, graph startup, scrolling, scaling and header position stability.
- Added regression coverage for graph smoothing, stable application ordering and 70 ms traffic accounting.

## [0.4.5] - 2026-11-10

Retrospective milestone: Windows packet-buffer efficiency and repeated performance validation. Not tagged or published.

### Changed
- Reused Windows TUN read buffers while retaining independent, reference-counted storage for packets held by the network stack.
- Replaced per-packet `ToBuffer().Flatten()` allocation with reusable output storage and borrowed-view copying.
- Preserved packet framing, MTU, queue size, native write semantics and the existing balancing algorithm.
- Kept the upstream packet endpoint on other platforms; the buffer optimization applies to Windows.

### Added
- Added packet-integrity, buffer-growth, retained-packet, IPv4/IPv6 and blocked-IO shutdown tests, plus allocation microbenchmarks.
- Added repeated Ethernet A/B measurements using identical builds apart from the buffer wrapper, without launching the GUI.

### Fixed
- Closed the native device if preparing the optimized packet endpoint fails.

Performance evidence: sampled allocations fell approximately 94% in the profiled comparison.
Eight subsequent A/B runs measured about 19% lower mean CPU/GiB with buffer reuse.
Mean throughput was 733 versus 691 Mbit/s, but four pairs against an external server
do not establish a universal speedup or rule out a smaller regression. These are
local measurements, not hardware-independent performance guarantees.

## [0.4.4] - 2026-11-10

Retrospective milestone: project identity, state continuity and adapter metadata caching.

### Changed
- Renamed the module, command, engine executable and tunnel identity to `MultiRouteNetwork`; retained the desktop name `MultiRoute`.
- Updated repository references to `aristache-labs/MultiRouteNetwork` and support links to `buymeacoffee.com/aristache`.
- Cached adapter names, indices and visible VPN metadata for three seconds while sampling traffic counters every 250 ms.
- Kept scheduler discovery, health checks, weights and adapter selection independent of the telemetry cache.
- Made package-repository hosting explicit rather than assuming this fork owns inherited upstream infrastructure.

### Fixed
- Updated tunnel-filtering checks, including `TestOwnTunnelIsHidden`, for the renamed engine adapter.
- Reset counter baselines after adapter identity changes, counter resets and failed reads to prevent false speed spikes.
- Added shorter retries after adapter-enumeration failures instead of retaining stale lookup results.

Existing local rules and usage were recovered from the legacy state directory during
development, with backups. This recovery was a local maintenance operation, not an
automatic migration promise for every installation. Back up legacy state before upgrading.

## [0.4.3] - 2026-11-10

Retrospective milestone: connection diagnostics and reproducible network comparisons.

### Added
- Added bounded, rotating `curl-routing.jsonl` diagnostics for identified curl connections, including scheduler eligibility and selection reasons.
- Added connection IDs, process IDs, selected adapters, retry observations and confirmed socket-open events.
- Added per-connection byte totals, interval and average rates, read/write terminal states and final close reasons.
- Added parallel direct-versus-engine comparison scripts with routing checks, physical counters and CSV results.
- Added loopback-only CPU, allocation, heap and goroutine capture workflows with saved configuration and binary fingerprints.

### Changed
- Distinguished attempted adapter selection from successfully opened connections in diagnostic analysis.
- Kept diagnostic writes asynchronous and bounded; reported dropped records without recording URLs, payloads or executable paths.
- Documented shared-router capacity: Ethernet and Wi-Fi on one WAN do not provide two independent internet connections.

## [0.4.2] - 2026-11-10

Retrospective milestone: live dashboard, application status and clearer speed displays.

### Added
- Added per-application running/not-running state and counts of added and running applications beside Applications.
- Added `using` lists covering adapters with open connections, ordered by observed usage without primary/secondary labels.
- Added separate download/upload negotiated Link Speed values on adapter cards; unavailable values remain unspecified.
- Displayed configured application download and upload limits beside their corresponding rate bars.
- Added adapter-wide download/upload limit controls and retained startup settings in the Settings panel.

### Changed
- Updated telemetry every 250 ms and retained 240 graph samples representing the latest 60 seconds.
- Smoothed graphs and displayed rate transitions while preserving raw traffic accounting and limit enforcement.
- Scaled each application bar from zero to its configured directional limit, or 100 Mbit/s when unlimited.
- Clamped visual bars at 100%; the unlimited display scale does not impose a traffic limit.
- Kept theme selection in the header and removed duplicate Dark/Light controls from Settings.
- Displayed connected physical adapters and real VPNs while hiding the engine TUN and infrastructure virtual adapters.

### Fixed
- Kept `Use all` on the automatic scheduler and `Several adapters` on the same scheduler restricted to the selected subset.
- Preserved simultaneous adapter use across multiple connections when lowest-latency selection is disabled.
- Clarified that fastest-adapter selection intentionally chooses one adapter for new connections; existing connections stay pinned.

## [0.4.1] - 2026-11-10

Retrospective milestone: reliable control, startup and window lifecycle.

### Added
- Added a styled close dialog offering Hide to system tray or Exit, with a remembered-choice checkbox.
- Added a Settings control to restore asking on close or change the remembered window-close behavior.
- Added close-request recovery when the initial JavaScript event is missed.

### Changed
- Kept the configured Windows TUN and network stack ready across Stop/Start for faster subsequent resumes.
- Closed routed TCP/UDP connections and pending dials at Stop, removed capture routes and restored egress settings.
- Released the retained device and stack at Exit after flushing state and stopping routing.
- Kept Start on the elevated `run -hidden` engine path rather than launching a separate tray-mode engine.

### Fixed
- Allowed adding and editing application rules while stopped, including when no engine pipe is available.
- Fixed transient busy-pipe failures and reduced control/telemetry contention.
- Replaced premature startup communication errors with startup-aware waiting and retry behavior.
- Made adapters and configured applications available before Start; stopped routing no longer accumulates routed usage.
- Restored the window X action when the close-event handoff failed to display its dialog.

## [0.4.0] - 2026-11-10

Retrospective milestone: MultiRoute Windows desktop and per-application controls.

### Added
- Added an unelevated Wails v3, React/TypeScript and WebView2 desktop communicating with the elevated engine over a per-user named pipe.
- Added Overview, Usage and Settings views with adapter cards, application icons, combined traffic graphs and routing controls.
- Added explicit application registration; Overview lists added applications rather than every operating-system process.
- Added executable-name and exact-path rules; exact-path matches take precedence and unidentified processes remain Unknown.
- Added all/one/several adapter policies, wait/other-adapter fallbacks, and optional lowest-latency selection with an eight-second hold.
- Added Windows TCP/UDP process ownership lookup and per-flow routing policy applied when connections open.
- Added per-application and per-adapter directional speed limits; applicable limits combine and empty fields mean unlimited.
- Added daily/monthly data caps counting download plus upload, 80% warnings and flow blocking at the cap.
- Added local SQLite usage history, adapter day/month totals, date-range views and CSV export.
- Added per-app plain TCP/UDP port-53 DNS choices: Windows default, Cloudflare, Google, Quad9 and custom servers.
- Added separate Start with Windows and Start routing settings, plus dark/light themes.
- Added a desktop packaging script producing the GUI, engine, driver and accompanying notices.

### Changed
- Shared control logic between the desktop, tray and engine while retaining legacy status and control markers.
- Applied rule edits to new connections while keeping existing connections pinned to their original adapter.
- Kept preferred-adapter selection separate from the Windows default route.
- Kept usage history local without adding accounts, activation, commercial licensing or anonymous usage reporting.
- Kept update controls as an unavailable visual placeholder; automatic installation remains disabled pending a signed release channel.

## [0.3.7] - 2026-11-09

### Fixed
- **`MultiRouteNetwork limit` on Windows no longer fails now and then with "Access is denied".** The running
  instance could not save the new cap while the command was reading the saved caps to see it applied.

## [0.3.6] - 2026-10-28

### Added
- **Speed limits per connection** from the tray menu: click a connection and pick 1, 2, 5, 10, 20, 50 or
  100 Mbit/s (or *Unlimited*). The cap covers that connection's downloads and uploads across all of its
  flows and takes effect immediately; the choice is saved and applied again after a restart (on macOS
  it is kept under `/Library/Application Support`, so it survives a reboot too). From a
  terminal, `MultiRouteNetwork limit <adapter> <Mbit/s>` sets any other speed, and `MultiRouteNetwork limit` lists the
  saved caps. Only traffic MultiRouteNetwork relays is capped, so an app bound directly to an adapter is not
  affected.

### Changed
- **MultiRouteNetwork has a new home: [github.com/aristache-labs/MultiRouteNetwork](https://github.com/aristache-labs/MultiRouteNetwork).**
  Old links, the install commands and Homebrew keep working. New Homebrew installs use
  `brew install MultiRouteNetwork/tap/MultiRouteNetwork`. If your Linux update fails with "404 Not Found" for
  `rohit267.github.io`, see *Troubleshooting* in the README.

### Fixed
- **A fast link no longer gets skipped because it holds many idle connections** (chat apps and browsers
  keep some open). New connections now go by how busy each link really is. Speed estimates are smoothed,
  so one burst doesn't flip which link is preferred, and they start over when an adapter gets a new
  address. Thanks to @vwidmer (#48).
- **The built-in SOCKS5 proxy checks a request before it reads it.** A request with a non-zero reserved
  byte is now dropped, and a command other than CONNECT gets its "not supported" answer straight away.
  Thanks to @b0r1ssh (#61).

## [0.3.5] - 2026-10-24

### Changed
- **The Linux repositories have a new address: `pkg.MultiRouteNetwork.org`.** This update moves your apt, dnf,
  zypper, apk or pacman setup to it by itself, so there is nothing to do. New installs use it from the
  start. MultiRouteNetwork also has a website now: [MultiRouteNetwork.org](https://MultiRouteNetwork.org).

## [0.3.4] - 2026-10-23

### Added
- **The tray menu shows the version** you are running (like `v0.3.3`), on the line above *Quit*, so a
  bug report has it without opening a terminal. It also asks GitHub for the newest release, at start and
  then every 6 hours, and turns into *Update to v0.3.4* when there is one; click it to open the release
  page.

### Fixed
- **MultiRouteNetwork no longer fails to turn on when another app already uses its address** (#43). The
  adapter used to always take `198.18.0.1`, which VPN and proxy apps such as Clash, sing-box and v2rayN
  often use as well. It now picks a free range at start.
- **Turning on is more reliable on Windows.** MultiRouteNetwork now waits for its adapter to appear before giving
  it an address, and tries again if the first attempt fails.
- **Errors in the tray now show the reason.** The menu has room for only the start of an error, and that
  used to be taken up by the command that failed, so the reason never showed (#43).

## [0.3.3] - 2026-10-20

### Added
- **Start at sign-in from the tray menu** on Windows, macOS and Linux (#28): tick *Start at sign-in*. It
  does what `MultiRouteNetwork autostart on` does, without a terminal. On a Mac it needs `sudo MultiRouteNetwork install` once,
  as before.

### Fixed
- **Windows: fewer antivirus false positives.** `MultiRouteNetwork.exe` and `MultiRouteNetwork-tray.exe` now carry a
  name, version, icon and manifest. Bare unsigned Go programs are what antivirus machine-learning models
  flag (Microsoft Defender's `Trojan:Win32/Wacatac.B!ml` on VirusTotal).

## [0.3.2] - 2026-10-19

### Fixed
- **Windows: no more PowerShell window flashing when a new adapter connects** (#10). The adapter capability
  check now hides its PowerShell window, the way the other helper commands in `internal/tun` already did.

## [0.3.1] - 2026-10-19

### Added
- **One-line install.** `curl -fsSL https://raw.githubusercontent.com/aristache-labs/MultiRouteNetwork/main/install.sh | sh` on
  macOS and Linux (it uses Homebrew, apt, dnf, zypper, apk or pacman when it can, the release tarball
  otherwise, and starts the tray), and `irm https://raw.githubusercontent.com/aristache-labs/MultiRouteNetwork/main/install.ps1 | iex`
  in PowerShell on Windows.
- Releases also carry the downloads under names without the version (`MultiRouteNetwork-setup.exe`,
  `MultiRouteNetwork-macos-universal.tar.gz`, `MultiRouteNetwork-linux-$(uname -m).tar.gz`), so
  `releases/latest/download/...` links always give the newest one.
- **One-line uninstall** on macOS and Linux: `curl -fsSL https://raw.githubusercontent.com/aristache-labs/MultiRouteNetwork/main/install.sh | sh -s -- uninstall`
  stops MultiRouteNetwork and removes it the way it was installed, with its repository and key. The README has the
  Windows line and every manual way.

### Fixed
- **macOS: the wired Ethernet disappeared when an Android phone was connected through TetherKit** (#20).
  When the phone became the primary connection, macOS could not add its own route for the Ethernet next
  to MultiRouteNetwork's, and MultiRouteNetwork then removed its route, leaving the Ethernet with none until the cable was
  plugged in again. MultiRouteNetwork now leaves that route to macOS.
- **Windows: turning on merging could stop all internet, with every adapter showing "no internet"**
  (#10). On an adapter with *weak host send* or *forwarding* turned on, which Mobile Hotspot, Internet
  Connection Sharing, Hyper-V and some VPNs do, Windows ignores the adapter a connection is pinned to, so
  MultiRouteNetwork's own connections went back into its virtual adapter. While merging, MultiRouteNetwork now turns those
  two settings off on the adapters it uses and puts them back afterwards (sharing that connection pauses
  meanwhile). It also re-applies its virtual adapter's address when Windows drops it right after start.
- `sudo MultiRouteNetwork install` refused on Macs where `/usr/local/bin` is writable by your user (Intel Macs
  with Homebrew). It now puts its root-owned copy in `/opt/MultiRouteNetwork/bin` there, and links `MultiRouteNetwork` into
  `/usr/local/bin`.

## [0.3.0] - 2026-10-17

### Added
- **MultiRouteNetwork as a normal package on every Linux distribution.** Releases now include a `.deb`, `.rpm`,
  `.apk` (Alpine) and Arch package for every CPU: x86-64, ARM64, ARMv7, ARMv6 (Raspberry Pi), x86 32-bit,
  RISC-V, PowerPC (ppc64le), s390x and LoongArch, plus a tarball for each. They are also published as signed
  repositories on GitHub Pages, so `sudo apt install MultiRouteNetwork`, `sudo dnf install MultiRouteNetwork`,
  `sudo zypper install MultiRouteNetwork`, `sudo apk add MultiRouteNetwork` and `sudo pacman -S MultiRouteNetwork` work after a
  one-time setup. The packages give MultiRouteNetwork its network rights themselves, and add an app-menu entry,
  icons and AppStream information.
- **Pauses by itself when only one connection works, and resumes when two do.** With fewer than two
  connections that reach the internet there is nothing to merge, so the tray pauses merging ("Paused: one
  connection") and starts it again when a second connection works. Pausing or starting by hand always wins,
  and it waits for things to settle first: 10 seconds with nothing moving (2 minutes if a download or
  upload is still going) before pausing, 3 seconds before resuming, and at least 30 seconds between
  automatic switches.
  Untick *Pause when there is only one connection* in the tray menu, or run `MultiRouteNetwork tray -auto-pause=false`.

### Changed
- On macOS, pausing and resuming no longer removes and recreates the virtual adapter, so it no longer
  drops a TetherKit (Android) link. Starting and quitting still do.
- New connections no longer wait while the once-a-second adapter speed reading runs (thanks @b0r1ssh).

## [0.2.0] - 2026-10-16

### Added
- **Downloads for macOS and Linux.** Each release now also has `MultiRouteNetwork-X.Y.Z-macos-universal.tar.gz`
  (Apple Silicon and Intel) and `MultiRouteNetwork-X.Y.Z-linux-x86_64.tar.gz` / `-linux-aarch64.tar.gz`. The steps
  to run them are in the README.
- **macOS support** (Apple Silicon and Intel). System-wide mode, the menu-bar tray, the SOCKS5 proxy, live
  per-adapter speeds and start at login (a LaunchAgent) all work. The TUN is a utun, flows are pinned with
  `IP_BOUND_IF` plus a scoped default route per adapter, and adapters come from the routing table. iPhone
  hotspot over USB works as is; Android USB tethering needs the TetherKit app. `sudo MultiRouteNetwork install`
  sets up a passwordless rule for `MultiRouteNetwork tray`.
- **Linux support**, including NixOS. System-wide mode, the tray icon (any desktop with
  StatusNotifierItem support: KDE, GNOME with the AppIndicator extension, XFCE, Cinnamon, ...), the
  SOCKS5 proxy, live per-adapter speeds and autostart (XDG autostart entry) all work. The TUN and
  its routes are set up over netlink, flows are pinned with `SO_BINDTOIFINDEX`, and adapters are
  detected from sysfs and the routing tables. Needs root or `CAP_NET_ADMIN` (`setcap`).
- **Nix flake** with a package and a NixOS module: `programs.MultiRouteNetwork.enable` installs a capability
  wrapper so the tray runs without root, and `programs.MultiRouteNetwork.autostart` starts it in every
  graphical session.

### Changed
- Clicking the tray icon no longer turns merging on or off (it was easy to hit by accident). Open the
  menu (right-click) and choose *Pause merging* / *Start merging*.
- The source is split into packages: `cmd/MultiRouteNetwork` (the command) and `internal/...` (engine, nics,
  tun, tray, platform, state, icon, logx). Build with `go build ./cmd/MultiRouteNetwork`.

## [0.1.1] - 2026-10-14

### Added
- **Speeds on hover.** Hovering the tray icon shows each adapter's download/upload speed and a Total
  line. The numbers come from Windows' own adapter counters, so every app counts, including apps that
  bypass MultiRouteNetwork (qBittorrent binds to adapters directly), and they show while paused too.
- **Real speeds in the menu.** The right-click menu shows each adapter's speed, how many connections
  MultiRouteNetwork put on it, and whether it has IPv6.
- `MultiRouteNetwork status` shows each adapter's total traffic ("link") next to the part MultiRouteNetwork relayed.
- `-mtu` flag to change the virtual adapter's MTU (default 9000).
- `-pprof` flag to serve Go's profiler on a localhost address, for diagnosing CPU use.
- README: screenshots, features and how to use.

### Changed
- **About 3x less CPU.** The virtual adapter now runs at MTU 9000 instead of 1500, so about 5x fewer
  packets go through the userland network stack. Measured on the same machine and links: 0.54 → 0.18
  CPU cores per 100 Mbit/s.
- Tray wording: **Merging** / **Paused**, and *Pause merging* / *Start merging* in the menu (was ON/OFF
  and *Turn on* / *Turn off*).
- Idle UDP sessions end after 30 seconds instead of 60, which means fewer sockets and less memory with
  torrent clients.
- Quieter log: torrent traffic no longer writes a warning for every dropped packet.

### Fixed
- **Tray icon missing after sign-in.** With "start at sign-in" on, the tray could start before the
  taskbar existed and stay invisible and inactive. It now waits for the taskbar, and the sign-in task
  retries if the tray fails to start. If you enabled autostart in 0.1.0, install 0.1.1 with the option
  ticked to update it.
- UDP datagrams larger than 1500 bytes going through the virtual adapter were silently dropped.
- Tooltip cut off after "Ethernet 4 ↓": Windows 11 shows only 64 characters of a tray tooltip, so the
  text now fits that limit.
- A garbled "…" appeared when long tooltip text was shortened.
- Errors from the tray library were not written to the log in the windowless build.

## [0.1.0] - 2026-10-13

First release.

### Added
- System-wide per-connection balancing: every new TCP and UDP connection, IPv4 and IPv6, is pinned to
  one adapter, so parallel connections use Wi-Fi, Ethernet and phone/USB tethers at the same time.
- Tray app: left-click turns it on or off, right-click shows per-adapter speeds, optional start at
  sign-in without a UAC prompt.
- Automatic adapter detection: real hardware with a default gateway is used, while VPN and virtual adapters
  (Tailscale, WireGuard, Hyper-V, WSL) are skipped; `MultiRouteNetwork list` shows why.
- Per-adapter internet checks: a link without internet gets no connections until it recovers.
- Connections are spread by each adapter's measured speed, separately for IPv4 and IPv6.
- IPv6 policy (`-ipv6 auto|on|off`): when only some adapters have IPv6, apps are steered to IPv4 so
  every adapter stays in use.
- Crash-safe: the virtual adapter and its routes disappear with the process.
- Command line: `on`, `off`, `quit`, `status`, `list`, `autostart`, `cleanup`, plus a SOCKS5 mode
  that needs no admin.
- Installer and portable zip.
