import { SmoothRate, useSmoothRate } from "./SmoothRate";
import { Spark } from "./Spark";
import type { AppRow, Card, Rule, Settings, Status, UsageRow } from "./types";
import * as React from "react";
import { api } from "./api";
import { speedScale } from "./speed-scale";
import { CloseDialog, CloseBehaviorSettings } from "./CloseDialog";
const emptyRule: () => Rule = () => ({
    id: "",
    exe: "",
    exactPath: !1,
    mode: "all",
    fallback: "wait",
    lowestLatency: !1,
    capPeriod: "month",
    dns: "windows",
  }),
  defaults: Settings = {
    theme: "dark",
    startRouting: !0,
    startWindows: !1,
    autoUpdate: !1,
  },
  My: Record<string, string> = {
    today: "Today",
    "7d": "7 days",
    "30d": "30 days",
    month: "This month",
    all: "All",
  };
function bytes(s: number = 0) {
  return s
    ? s >= 1e9
      ? `${(s / 1e9).toFixed(2)} GB`
      : s >= 1e6
        ? `${(s / 1e6).toFixed(1)} MB`
        : s >= 1e3
          ? `${(s / 1e3).toFixed(0)} KB`
          : `${Math.round(s)} B`
    : "0 B";
}
function fmt(s: number = 0) {
  const _: any = (s * 8) / 1e3;
  return _ >= 1e3
    ? `${(_ / 1e3).toFixed(1)} Mbit/s`
    : `${Math.round(_)} kbit/s`;
}
function Icon({ name }: any) {
  const _: any = {
    usage: <path d="m3 17 6-7 5 4 7-9" />,
    overview: (
      <>
        <rect x="3" y="3" width="7" height="7" />
        <rect x="14" y="3" width="7" height="7" />
        <rect x="3" y="14" width="7" height="7" />
        <rect x="14" y="14" width="7" height="7" />
      </>
    ),
    settings: (
      <>
        <path d="m10 3-1 3-3 1-3 3 2 2-1 4 3 1 2 4 3-1 3 1 2-4 3-1-1-4 2-2-3-3-3-1-1-3Z" />
        <circle cx="12" cy="12" r="3" />
      </>
    ),
    moon: <path d="M9 3a9 9 0 1 1 0 18A10 10 0 0 0 9 3Z" />,
    sun: (
      <>
        <circle cx="12" cy="12" r="4" />
        <path d="M12 1v3m0 16v3M1 12h3m16 0h3M4 4l2 2m12 12 2 2M4 20l2-2M18 6l2-2" />
      </>
    ),
    play: <path d="m7 4 13 8-13 8Z" />,
    stop: <rect x="6" y="6" width="12" height="12" />,
    plus: <path d="M12 4v16M4 12h16" />,
    close: <path d="m5 5 14 14M5 19 19 5" />,
    trash: (
      <>
        <path d="M4 6h16M9 6V3h6v3M7 6l1 15h8l1-15" />
      </>
    ),
    save: (
      <>
        <path d="M4 3h13l3 3v15H4Z" />
        <path d="M8 3v6h8V3M8 21v-8h8v8" />
      </>
    ),
    download: <path d="M12 3v12m-5-5 5 5 5-5M4 20h16" />,
    app: (
      <>
        <rect x="3" y="4" width="18" height="16" rx="2" />
        <path d="M3 9h18M7 6h1m2 0h1" />
      </>
    ),
  };
  return (
    <svg
      className={`icon icon-${name}`}
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.8"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
    >
      {_[name] ?? _.app}
    </svg>
  );
}
function BrandMark() {
  return (
    <span className="brand-mark" aria-hidden="true">
      <svg viewBox="0 0 48 48" fill="none">
        <path
          d="M10 34V14l14 20 14-20v20"
          stroke="white"
          strokeWidth="4"
          strokeLinecap="round"
          strokeLinejoin="round"
        />
        <circle cx="10" cy="14" r="4" fill="#ffce00" />
        <circle cx="38" cy="34" r="4" fill="white" />
      </svg>
    </span>
  );
}
function AppIcon({ exe, path }: any) {
  const [j, r] = React.useState(false);
  return (
    React.useEffect(() => r(!1), [path]),
    (
      <span className="app-icon" aria-hidden="true">
        {path && !j ? (
          <img
            src={`/api/icon?path=${encodeURIComponent(path)}`}
            alt=""
            onError={() => r(!0)}
          />
        ) : (
          <span className="app-initial">
            {exe
              .replace(/\.exe$/i, "")
              .slice(0, 1)
              .toUpperCase() || <Icon name="app" />}
          </span>
        )}
      </span>
    )
  );
}
function SpeedBar({ bps, limit, kind }: any) {
  const displayedBps = useSmoothRate(bps);
  const r: any = speedScale(displayedBps, limit);
  return (
    <div
      className="speed"
      title={
        r.limited
          ? `0–${r.maxMbps} Mbit/s · app speed limit`
          : `No limit · 0–${r.maxMbps} Mbit/s · visual scale`
      }
    >
      <div className={bps > 0 ? kind : "muted"}>{fmt(displayedBps)}</div>
      {r.limited && (
        <div className="speed-limit-label">
          {"Limit: "}
          {r.maxMbps}
          {" Mbit/s"}
        </div>
      )}
      <div
        className="track"
        role="progressbar"
        aria-label={kind === "down" ? "Download speed" : "Upload speed"}
        aria-valuemin={0}
        aria-valuemax={r.maxMbps}
        aria-valuenow={Math.min(r.currentMbps, r.maxMbps)}
        aria-valuetext={`${fmt(bps)}; ${r.limited ? "speed limit" : "visual scale"} ${r.maxMbps} Mbit/s`}
      >
        <div
          className={`fill ${kind}`}
          style={{
            width: `${r.percent}%`,
          }}
        />
      </div>
    </div>
  );
}
function findRule(s: any, _: any) {
  return (
    _.find((j: any) => {
      var r: any;
      return (
        j.exactPath &&
        !!j.path &&
        j.path.toLowerCase() ===
          ((r = s.path) == null ? void 0 : r.toLowerCase())
      );
    }) ??
    _.find(
      (j: any) => !j.exactPath && j.exe.toLowerCase() === s.exe.toLowerCase(),
    )
  );
}
function routeLabel(s: any, _: any) {
  if (!s || s.mode === "all") return "any adapter";
  let j: any = (s.adapters ?? []).join(", ") || "choose adapter";
  return (
    s.fallback === "others" && (j += ", backup any adapter"),
    s.fallback === "default" && (j += `, backup default${_ ? ` (${_})` : ""}`),
    s.fallback === "choose" &&
      (j += `, backup ${(s.fallbackAdapters ?? []).join(", ")}`),
    j
  );
}
export function App() {
  const [status, setStatus] = React.useState<Status | null>(null),
    [error, setError] = React.useState(""),
    [view, setView] = React.useState<"overview" | "usage">("overview"),
    [panel, setPanel] = React.useState<"none" | "add" | "edit" | "settings">(
      "none",
    ),
    [draft, setDraft] = React.useState<Rule>(emptyRule()),
    [range, setRange] = React.useState("today"),
    [usage, setUsage] = React.useState<UsageRow[]>([]),
    [today, setToday] = React.useState<UsageRow[]>([]),
    [month, setMonth] = React.useState<UsageRow[]>([]),
    [busy, setBusy] = React.useState(false),
    [notice, setNotice] = React.useState(""),
    [startingRequest, setStartingRequest] = React.useState(false),
    starting: any = startingRequest || !!(status != null && status.starting),
    settings: any = (status == null ? void 0 : status.settings) ?? defaults,
    rules: any = (status == null ? void 0 : status.rules) ?? [],
    apps: any = (status == null ? void 0 : status.apps) ?? [],
    adapters: any = [
      ...((status == null ? void 0 : status.adapters) ?? []),
    ].sort(
      (N: any, z: any) =>
        +(z.state === "VPN") - +(N.state === "VPN") ||
        N.name.localeCompare(z.name),
    ),
    selectedRow: any = apps.find((N: any) => {
      var z: any;
      return ((z = findRule(N, rules)) == null ? void 0 : z.id) === draft.id;
    });
  async function refresh() {
    var N: any;
    try {
      const z: any = await api.status();
      (setStatus(z.status),
        setError(z.status.error ?? ""),
        (document.documentElement.dataset.theme =
          ((N = z.status.settings) == null ? void 0 : N.theme) === "light"
            ? "light"
            : "dark"));
    } catch (z) {
      setError(z instanceof Error ? z.message : "The engine is not available");
    }
  }
  (React.useEffect(() => {
    refresh();
    const N: any = window.setInterval(() => void refresh(), 250);
    return () => window.clearInterval(N);
  }, []),
    React.useEffect(() => {
      let N: any = !0;
      const z: any = async () => {
        try {
          const [Xl, se] = await Promise.all([
            api.usage("today"),
            api.usage("month"),
          ]);
          N && (setToday(Xl.usage ?? []), setMonth(se.usage ?? []));
        } catch {}
      };
      z();
      const jl: any = window.setInterval(() => void z(), 5e3);
      return () => {
        ((N = !1), window.clearInterval(jl));
      };
    }, []),
    React.useEffect(() => {
      if (view !== "usage") return;
      let N: any = !0;
      const z: any = () =>
        api
          .usage(range)
          .then((Xl: any) => {
            N && setUsage(Xl.usage ?? []);
          })
          .catch((Xl: any) => {
            N && setError(Xl.message);
          });
      z();
      const jl: any = window.setInterval(() => void z(), 5e3);
      return () => {
        ((N = !1), window.clearInterval(jl));
      };
    }, [view, range]),
    React.useEffect(() => {
      if (!notice) return;
      const N: any = window.setTimeout(() => setNotice(""), 4e3);
      return () => window.clearTimeout(N);
    }, [notice]));
  async function action(N: any) {
    if (!busy) {
      setBusy(!0);
      try {
        (await N(), await refresh());
      } catch (z) {
        setError(z instanceof Error ? z.message : String(z));
      } finally {
        setBusy(!1);
      }
    }
  }
  function edit(N: any) {
    const z: any = findRule(N, rules);
    (setDraft({
      ...emptyRule(),
      ...z,
      exe: (z == null ? void 0 : z.exe) || N.exe,
      path: (z == null ? void 0 : z.path) || N.path,
    }),
      setPanel("edit"));
  }
  const historyFor: any = (N: any, z: any) =>
    N.find((jl: any) => jl.exe.toLowerCase() === z.toLowerCase());
  async function exportCSV() {
    const N: any = await api.csv(range),
      z: any = URL.createObjectURL(
        new Blob([N.csv], {
          type: "text/csv",
        }),
      ),
      jl: any = document.createElement("a");
    ((jl.href = z),
      (jl.download = "multiroute-usage.csv"),
      jl.click(),
      window.setTimeout(() => URL.revokeObjectURL(z), 1e3));
  }
  return (
    <div className="app">
      <header className="top">
        <div className="identity">
          <BrandMark />
          <div>
            <h1>{"MultiRoute"}</h1>
            <div className="run-state">
              <i
                className={status != null && status.running ? "dot on" : "dot"}
              />
              {starting
                ? "Starting…"
                : status != null && status.running
                  ? "Running"
                  : "Stopped"}
            </div>
          </div>
        </div>
        <div className="header-spark">
          <Spark
            down={status == null ? void 0 : status.down}
            up={status == null ? void 0 : status.up}
          />
        </div>
        <div className="header-speeds">
          <span className="down">
            {"↓ "}
            {<SmoothRate bps={status == null ? void 0 : status.downBps} />}
          </span>
          <span className="up">
            {"↑ "}
            {<SmoothRate bps={status == null ? void 0 : status.upBps} />}
          </span>
        </div>
        <nav className="header-actions" aria-label="Main navigation">
          <button
            onClick={() => {
              (setView(view === "usage" ? "overview" : "usage"),
                setPanel("none"));
            }}
          >
            <Icon name={view === "usage" ? "overview" : "usage"} />
            {view === "usage" ? "Overview" : "Usage"}
          </button>
          <button
            onClick={() =>
              void action(() =>
                api.saveSettings({
                  ...settings,
                  theme: settings.theme === "light" ? "dark" : "light",
                }),
              )
            }
          >
            <Icon name={settings.theme === "light" ? "sun" : "moon"} />
            {settings.theme === "light" ? "Light" : "Dark"}
          </button>
          <button
            aria-pressed={panel === "settings"}
            onClick={() => {
              (setView("overview"),
                setPanel(panel === "settings" ? "none" : "settings"));
            }}
          >
            <Icon name="settings" />
            {"Settings"}
          </button>
          <button
            className={
              status != null && status.running ? "routing-stop" : "primary"
            }
            disabled={busy || starting}
            onClick={() => {
              if (status != null && status.running) {
                action(api.stop);
                return;
              }
              (setStartingRequest(!0),
                setError(""),
                action(api.start).finally(() => setStartingRequest(!1)));
            }}
          >
            <Icon name={status != null && status.running ? "stop" : "play"} />
            {starting
              ? "Starting…"
              : busy
                ? "Please wait…"
                : status != null && status.running
                  ? "Stop"
                  : "Start"}
          </button>
        </nav>
      </header>
      {error && !starting && (
        <div className="error" role="alert">
          <span>{error}</span>
          <button
            className="icon-button"
            aria-label="Dismiss error"
            onClick={() => setError("")}
          >
            <Icon name="close" />
          </button>
        </div>
      )}
      <div className={`workspace ${panel !== "none" ? "with-panel" : ""}`}>
        <main className="main">
          {view === "overview" ? (
            <>
              <section
                className="cards"
                aria-label="Connected network adapters"
              >
                {status ? (
                  adapters.length ? null : (
                    <div className="empty">
                      {
                        "No connected adapters. Connect Wi-Fi or Ethernet to get started."
                      }
                    </div>
                  )
                ) : (
                  <div className="empty">{"Loading connected adapters…"}</div>
                )}
                {adapters.map((N: any) => (
                  <AdapterCard
                    card={N}
                    running={!!(status != null && status.running)}
                    onDefault={() =>
                      void action(() =>
                        api.saveSettings({
                          ...settings,
                          defaultAdapter: N.name,
                        }),
                      )
                    }
                    key={N.name}
                  />
                ))}
              </section>
              <section
                className="applications surface"
                aria-labelledby="applications-title"
              >
                <header className="section-heading">
                  <div>
                    <h2 id="applications-title">{"Applications"}</h2>
                    <span className="muted">
                      {apps.length} {apps.length === 1 ? "app" : "apps"}
                      {" �"}{" "}
                      {
                        apps.filter((N: any) => N.runningKnown && N.running)
                          .length
                      }{" "}
                      {"running"}
                    </span>
                  </div>
                  <button
                    onClick={() => {
                      (setDraft(emptyRule()), setPanel("add"));
                    }}
                  >
                    <Icon name="plus" />
                    {"Add app"}
                  </button>
                </header>
                <div
                  className="app-table"
                  role="table"
                  aria-label="Application traffic"
                >
                  <div className="app-columns column-headings" role="row">
                    <span role="columnheader">{"App"}</span>
                    <span className="minute" role="columnheader">
                      {"Last minute"}
                    </span>
                    <span role="columnheader">{"Download"}</span>
                    <span role="columnheader">{"Upload"}</span>
                    <span role="columnheader">{"Data used"}</span>
                  </div>
                  <div className="app-table-body" role="rowgroup">
                    {apps.map((N: any) => {
                      const z: any = findRule(N, rules),
                        jl: any = historyFor(today, N.exe),
                        Xl: any = historyFor(month, N.exe);
                      return (
                        <div
                          className={`app-columns app-row ${panel === "edit" && draft.id === (z == null ? void 0 : z.id) ? "selected" : ""}`}
                          role="row"
                          key={
                            (z == null ? void 0 : z.id) || `${N.exe}:${N.path}`
                          }
                        >
                          <div className="app-cell" role="cell">
                            <AppIcon exe={N.exe} path={N.path} />
                            <div className="app-copy">
                              <span
                                className={`app-process-state ${N.running ? "is-running" : ""}`}
                              >
                                {N.runningKnown
                                  ? N.running
                                    ? "running"
                                    : "not running"
                                  : "status unavailable"}
                              </span>
                              <button
                                className="app-name"
                                onClick={() => edit(N)}
                              >
                                {N.exe}
                              </button>
                              <div className="route-meta">
                                <span
                                  className="route-badge"
                                  title={routeLabel(z, settings.defaultAdapter)}
                                >
                                  {"→ "}
                                  {routeLabel(z, settings.defaultAdapter)}
                                </span>
                                <span
                                  className="connection-state"
                                  title={
                                    N.via
                                      ? `Adapters ordered by current application traffic: ${N.via}`
                                      : void 0
                                  }
                                >
                                  {N.capped
                                    ? "data cap reached"
                                    : N.warn
                                      ? "80% of data cap used"
                                      : N.via &&
                                          status != null &&
                                          status.running
                                        ? `using ${N.via}`
                                        : "no traffic routed yet"}
                                </span>
                              </div>
                            </div>
                          </div>
                          <div className="minute" role="cell">
                            <Spark down={N.down} up={N.up} />
                          </div>
                          <div role="cell">
                            <SpeedBar
                              bps={N.downBps}
                              limit={z == null ? void 0 : z.downMbps}
                              kind="down"
                            />
                          </div>
                          <div role="cell">
                            <SpeedBar
                              bps={N.upBps}
                              limit={z == null ? void 0 : z.upMbps}
                              kind="up"
                            />
                          </div>
                          <div className="data-used" role="cell">
                            <span>
                              {bytes(
                                jl ? jl.down + jl.up : N.downBytes + N.upBytes,
                              )}{" "}
                              {"today"}
                            </span>
                            <small>
                              {"month: "}
                              {bytes(Xl ? Xl.down + Xl.up : 0)}
                            </small>
                          </div>
                        </div>
                      );
                    })}
                    {!apps.length && (
                      <p className="empty">
                        {
                          "Add an app to choose its adapters, speed limits and data cap."
                        }
                      </p>
                    )}
                  </div>
                </div>
              </section>
            </>
          ) : (
            <section className="usage surface">
              <header className="section-heading">
                <div>
                  <h2>{"Usage"}</h2>
                  <p className="muted">
                    {"Every app that used the network through MultiRoute."}
                  </p>
                </div>
                <button onClick={() => void action(exportCSV)}>
                  <Icon name="download" />
                  {"Export CSV"}
                </button>
              </header>
              <div className="range-tabs" aria-label="Usage period">
                {Object.entries(My).map(([N, z]: any) => (
                  <button
                    aria-pressed={range === N}
                    className={range === N ? "active" : ""}
                    onClick={() => setRange(N)}
                    key={N}
                  >
                    {z}
                  </button>
                ))}
              </div>
              <div className="usage-summary">
                <span className="down">
                  {"↓ "}
                  {bytes(usage.reduce((N: any, z: any) => N + z.down, 0))}
                </span>
                <span className="up">
                  {"↑ "}
                  {bytes(usage.reduce((N: any, z: any) => N + z.up, 0))}
                </span>
                <span className="muted">
                  {usage.length}
                  {" apps"}
                </span>
              </div>
              <div className="usage-scroll">
                <table>
                  <thead>
                    <tr>
                      <th>{"App"}</th>
                      <th>{"Download"}</th>
                      <th>{"Upload"}</th>
                      <th>{"Total"}</th>
                      <th aria-label="Relative usage" />
                    </tr>
                  </thead>
                  <tbody>
                    {[...usage]
                      .sort((N: any, z: any) => z.down + z.up - N.down - N.up)
                      .map((N: any) => (
                        <tr key={N.exe}>
                          <td>
                            <div className="app-cell">
                              <AppIcon exe={N.exe} path={N.path} />
                              <div>
                                {N.exe}
                                <div className="muted">
                                  {Object.keys(N.byNic ?? {}).join(" · ")}
                                </div>
                              </div>
                            </div>
                          </td>
                          <td className="down">{bytes(N.down)}</td>
                          <td className="up">{bytes(N.up)}</td>
                          <td>{bytes(N.down + N.up)}</td>
                          <td>
                            <div className="track">
                              <div
                                className="fill down"
                                style={{
                                  width: `${((N.down + N.up) / Math.max(1, ...usage.map((z: any) => z.down + z.up))) * 100}%`,
                                }}
                              />
                            </div>
                          </td>
                        </tr>
                      ))}
                  </tbody>
                </table>
                {!usage.length && (
                  <p className="empty">
                    {"No recorded traffic for this period."}
                  </p>
                )}
              </div>
              <p className="history-note muted">
                {
                  "History counts traffic routed by MultiRoute. Statistics pause when routing is stopped."
                }
              </p>
            </section>
          )}
        </main>
        {panel !== "none" && (
          <aside
            className="side surface"
            aria-label={
              panel === "settings"
                ? "Settings"
                : panel === "add"
                  ? "Add an app"
                  : `Edit ${draft.exe}`
            }
          >
            {panel === "settings" ? (
              <SettingsPanel
                adapters={adapters}
                settings={settings}
                busy={busy}
                onSave={(N: any) => void action(() => api.saveSettings(N))}
                onClose={() => setPanel("none")}
              />
            ) : (
              <RuleForm
                draft={draft}
                row={selectedRow}
                today={historyFor(today, draft.exe)}
                month={historyFor(month, draft.exe)}
                adapters={adapters}
                settings={settings}
                busy={busy}
                onChange={setDraft}
                onClose={() => setPanel("none")}
                onSave={() =>
                  void action(async () => {
                    (await api.saveRule(draft),
                      setPanel("none"),
                      setNotice(`${draft.exe} was saved.`));
                  })
                }
                onDelete={
                  draft.id
                    ? () =>
                        void action(async () => {
                          (await api.deleteRule(draft.id),
                            setPanel("none"),
                            setNotice(
                              `${draft.exe} was removed from the list.`,
                            ));
                        })
                    : void 0
                }
                key={panel === "add" ? "new" : draft.id}
              />
            )}
          </aside>
        )}
      </div>
      {notice && (
        <div className="toast" role="status">
          {notice}
        </div>
      )}
      <CloseDialog />
    </div>
  );
}
function AdapterCard({ card, running, onDefault }: any) {
  const r: any = (D: any) =>
    D != null && Number.isFinite(D) && D > 0
      ? `${D.toLocaleString("en-US", {
          maximumFractionDigits: 2,
        })} Mbit/s`
      : "—";
  return (
    <article className={`card surface ${card.default ? "is-default" : ""}`}>
      <header className="card-top">
        <div className="adapter-identity">
          <h2>{card.name}</h2>
          <p className="muted">
            {card.ip || card.ip6 || card.reason}
            {running && card.latencyMs ? ` · ${card.latencyMs} ms` : ""}
          </p>
        </div>
        <div className="card-badges">
          <span
            className={`badge ${card.state === "VPN" ? "vpn" : running ? "online" : "available"}`}
          >
            {card.state === "VPN" ? "VPN" : running ? card.state : "available"}
          </span>
          <button
            className={`badge ${card.default ? "default" : "make-default"}`}
            disabled={!card.routable || card.default}
            onClick={onDefault}
          >
            {card.default ? "DEFAULT" : "Make default"}
          </button>
        </div>
      </header>
      <div className="rates">
        <span className="down">
          {"↓ "}
          {<SmoothRate bps={card.downBps} />}
        </span>
        <span className="up">
          {"↑ "}
          {<SmoothRate bps={card.upBps} />}
        </span>
      </div>
      <div className="link-speeds">
        <span className="link-speed-label">{"Link Speed"}</span>
        <span
          aria-label="Download Link Speed"
          title="Negotiated receive link speed reported by Windows"
        >
          {"↓ "}
          {r(card.linkDownMbps)}
        </span>
        <span
          aria-label="Upload Link Speed"
          title="Negotiated transmit link speed reported by Windows"
        >
          {"↑ "}
          {r(card.linkUpMbps)}
        </span>
      </div>
      <div className="adapter-spark">
        <Spark down={card.down} up={card.up} />
      </div>
      <div className="adapter-totals">
        <p>
          {"Today "}
          <span>
            {"↓ "}
            {bytes(card.todayDown)}
          </span>
          <span>
            {"↑ "}
            {bytes(card.todayUp)}
          </span>
        </p>
        <p>
          {"Month "}
          <span>
            {"↓ "}
            {bytes(card.monthDown)}
          </span>
          <span>
            {"↑ "}
            {bytes(card.monthUp)}
          </span>
        </p>
      </div>
    </article>
  );
}
function Choice({
  checked,
  onChange,
  children,
  detail,
  name,
  checkbox: Q = !1,
  disabled: V = !1,
}: any) {
  return (
    <div className={`choice ${V ? "disabled" : ""}`}>
      <label>
        <input
          type={Q ? "checkbox" : "radio"}
          checked={checked}
          name={name}
          disabled={V}
          onChange={onChange}
        />
        <span>{children}</span>
      </label>
      {detail && <p className="muted">{detail}</p>}
    </div>
  );
}
function Section({ title, children }: any) {
  return (
    <section className="form-section">
      <h3>{title}</h3>
      {children}
    </section>
  );
}
function Amount({ label, value, unit, placeholder, onChange }: any) {
  return (
    <label className="amount">
      <input
        aria-label={label}
        type="number"
        min="0"
        step="any"
        value={value ?? ""}
        placeholder={placeholder}
        onChange={(Q: any) =>
          onChange(Q.target.value === "" ? void 0 : Number(Q.target.value))
        }
      />
      <span>{unit}</span>
    </label>
  );
}
function RuleForm({
  draft,
  row,
  today,
  month,
  adapters,
  settings,
  busy,
  onChange,
  onSave,
  onDelete,
  onClose,
}: any) {
  var ll: any, Fl: any;
  const g: any = (O: any) =>
      onChange({
        ...draft,
        ...O,
      }),
    M: any = adapters.filter((O: any) => O.routable),
    P: any = draft.adapters ?? [],
    zl: any = (O: any, rl: any) =>
      O.includes(rl) ? O.filter((Ul: any) => Ul !== rl) : [...O, rl],
    Cl: any =
      !!draft.exe.trim() &&
      (!draft.exactPath || !!((ll = draft.path) != null && ll.trim())) &&
      (draft.mode === "all" ||
        (P.length > 0 && (draft.mode !== "one" || P.length === 1))) &&
      (draft.fallback !== "choose" ||
        !!((Fl = draft.fallbackAdapters) != null && Fl.length));
  return (
    <form
      className="panel-form"
      onSubmit={(O: any) => {
        (O.preventDefault(), Cl && onSave());
      }}
    >
      <header className="panel-heading">
        {draft.id && <AppIcon exe={draft.exe} path={draft.path} />}
        <div>
          <h2>{draft.id ? draft.exe : "Add an app"}</h2>
          <p className="muted truncate">
            {draft.id ? draft.path : "Pick a program and an adapter"}
          </p>
        </div>
        <button
          type="button"
          className="icon-button"
          aria-label="Close app panel"
          onClick={onClose}
        >
          <Icon name="close" />
        </button>
      </header>
      <div className="panel-scroll">
        {draft.id ? (
          <>
            <Section title="Data used">
              <dl className="data-breakdown">
                <dt>{"Today"}</dt>
                <dd>
                  {"↓ "}
                  {bytes(today == null ? void 0 : today.down)}
                  {"　↑ "}
                  {bytes(today == null ? void 0 : today.up)}
                </dd>
                <dt>{"This month"}</dt>
                <dd>
                  {"↓ "}
                  {bytes(month == null ? void 0 : month.down)}
                  {"　↑ "}
                  {bytes(month == null ? void 0 : month.up)}
                </dd>
                <dt>{"This session"}</dt>
                <dd>
                  {"↓ "}
                  {bytes(row == null ? void 0 : row.downBytes)}
                  {"　↑ "}
                  {bytes(row == null ? void 0 : row.upBytes)}
                </dd>
              </dl>
            </Section>
            {!!today && Object.keys(today.byNic ?? {}).length > 0 && (
              <Section title="Today, by adapter">
                <dl className="data-breakdown">
                  {Object.entries(today.byNic).map(([O, rl]: any) => (
                    <div className="breakdown-line" key={O}>
                      <dt>{O}</dt>
                      <dd>
                        {today.byNicDown && today.byNicUp ? (
                          <>
                            {"↓ "}
                            {bytes(today.byNicDown[O])}
                            {"　↑"} {bytes(today.byNicUp[O])}
                          </>
                        ) : (
                          bytes(rl)
                        )}
                      </dd>
                    </div>
                  ))}
                </dl>
              </Section>
            )}
          </>
        ) : (
          <Section title="Program">
            <label className="field-label">
              {"Executable name"}
              <input
                required={!0}
                value={draft.exe}
                onChange={(O: any) =>
                  g({
                    exe: O.target.value,
                  })
                }
                placeholder="brave.exe"
              />
            </label>
            <label className="field-label">
              {"Program path"}
              <input
                value={draft.path ?? ""}
                onChange={(O: any) =>
                  g({
                    path: O.target.value,
                  })
                }
                placeholder="C:\\Program Files\\App\\app.exe"
              />
            </label>
            <p className="muted">
              {
                "Enter an executable name, or its full path for an exact-file rule."
              }
            </p>
          </Section>
        )}
        <Section title="Connect through">
          <Choice
            name="mode"
            checked={draft.mode === "all"}
            onChange={() =>
              g({
                mode: "all",
              })
            }
            detail="Spreads connections over every internet adapter."
          >
            {"Use all"}
          </Choice>
          <Choice
            name="mode"
            checked={draft.mode === "one"}
            onChange={() =>
              g({
                mode: "one",
                adapters: P.slice(0, 1),
              })
            }
            detail="Always uses the adapter you pick."
          >
            {"One adapter"}
          </Choice>
          <Choice
            name="mode"
            checked={draft.mode === "several"}
            onChange={() =>
              g({
                mode: "several",
              })
            }
            detail="Uses the adapters you pick together."
          >
            {"Several adapters"}
          </Choice>
        </Section>
        {draft.mode !== "all" && (
          <Section title={draft.mode === "one" ? "Adapter" : "Adapters"}>
            {adapters.map((O: any) => (
              <Choice
                name="adapters"
                checkbox={draft.mode === "several"}
                checked={P.includes(O.name)}
                disabled={!O.routable}
                onChange={() =>
                  g({
                    adapters: draft.mode === "one" ? [O.name] : zl(P, O.name),
                  })
                }
                detail={
                  O.routable ? void 0 : "VPN adapter · not eligible for routing"
                }
                key={O.name}
              >
                {O.name}
                {O.state === "VPN" ? " (VPN)" : ""}
              </Choice>
            ))}
            {!M.length && (
              <p className="muted">
                {"Connect an eligible adapter to select it."}
              </p>
            )}
            {P.filter(
              (O: any) => !adapters.some((rl: any) => rl.name === O),
            ).map((O: any) => (
              <Choice
                name="adapters"
                checked={!0}
                checkbox={draft.mode === "several"}
                onChange={() =>
                  g({
                    adapters: P.filter((rl: any) => rl !== O),
                  })
                }
                key={O}
              >
                {O}
                {" (disconnected)"}
              </Choice>
            ))}
          </Section>
        )}
        {draft.mode !== "all" && (
          <Section title="Fallback">
            <p className="muted">{"If its selected adapters go down."}</p>
            <Choice
              name="fallback"
              checked={draft.fallback === "wait"}
              onChange={() =>
                g({
                  fallback: "wait",
                })
              }
              detail="Its connections stop until a selected adapter returns."
            >
              {"Nothing"}
            </Choice>
            <Choice
              name="fallback"
              checked={draft.fallback === "default"}
              onChange={() =>
                g({
                  fallback: "default",
                })
              }
              detail="Used until its own adapter is back."
            >
              {"Default adapter"}
              {settings.defaultAdapter
                ? ` (${settings.defaultAdapter})`
                : " (not set)"}
            </Choice>
            <Choice
              name="fallback"
              checked={draft.fallback === "choose"}
              onChange={() =>
                g({
                  fallback: "choose",
                })
              }
              detail="Tried in the order selected."
            >
              {"Choose adapters"}
            </Choice>
            {draft.fallback === "choose" && (
              <div className="indented">
                {[
                  ...(draft.fallbackAdapters ?? []),
                  ...M.map((O: any) => O.name).filter((O: any) => {
                    var rl: any;
                    return !(
                      (rl = draft.fallbackAdapters) != null && rl.includes(O)
                    );
                  }),
                ].map((O: any) => {
                  var rl: any;
                  return (
                    <Choice
                      checkbox={!0}
                      checked={
                        !!(
                          (rl = draft.fallbackAdapters) != null &&
                          rl.includes(O)
                        )
                      }
                      onChange={() =>
                        g({
                          fallbackAdapters: zl(draft.fallbackAdapters ?? [], O),
                        })
                      }
                      key={O}
                    >
                      {O}
                    </Choice>
                  );
                })}
              </div>
            )}
            <Choice
              name="fallback"
              checked={draft.fallback === "others"}
              onChange={() =>
                g({
                  fallback: "others",
                })
              }
              detail="Uses another eligible adapter when the selected ones are unavailable."
            >
              {"Use any other adapter"}
            </Choice>
          </Section>
        )}
        {draft.fallback === "choose" && (
          <Section title="Last resort">
            <Choice
              checkbox={!0}
              checked={!!draft.lastResort}
              onChange={() =>
                g({
                  lastResort: !draft.lastResort,
                })
              }
              detail="Off: its connections stop instead."
            >
              {"Use any other adapter"}
            </Choice>
          </Section>
        )}
        <Section title="Choosing among adapters">
          <Choice
            checkbox={!0}
            checked={draft.lowestLatency}
            onChange={() =>
              g({
                lowestLatency: !draft.lowestLatency,
              })
            }
            detail="Each new connection uses the permitted adapter with the lowest measured delay."
          >
            {"Use whichever adapter answers fastest"}
          </Choice>
        </Section>
        <Section title="Speed limit">
          <Amount
            label="Download limit"
            value={draft.downMbps}
            unit="Mbit/s download"
            placeholder="no limit"
            onChange={(O: any) =>
              g({
                downMbps: O,
              })
            }
          />
          <Amount
            label="Upload limit"
            value={draft.upMbps}
            unit="Mbit/s upload"
            placeholder="no limit"
            onChange={(O: any) =>
              g({
                upMbps: O,
              })
            }
          />
        </Section>
        <Section title="Speed limit per adapter">
          {M.map((O: any) => {
            var rl: any, Ul: any;
            return (
              <div className="adapter-limit" key={O.name}>
                <p className="muted">{O.name}</p>
                <div className="limit-pair">
                  <Amount
                    label={`${O.name} download limit`}
                    value={
                      (rl = draft.adapterDown) == null ? void 0 : rl[O.name]
                    }
                    unit="↓"
                    placeholder="no limit"
                    onChange={(Ll: any) =>
                      g({
                        adapterDown: {
                          ...draft.adapterDown,
                          [O.name]: Ll ?? 0,
                        },
                      })
                    }
                  />
                  <Amount
                    label={`${O.name} upload limit`}
                    value={(Ul = draft.adapterUp) == null ? void 0 : Ul[O.name]}
                    unit="↑"
                    placeholder="no limit"
                    onChange={(Ll: any) =>
                      g({
                        adapterUp: {
                          ...draft.adapterUp,
                          [O.name]: Ll ?? 0,
                        },
                      })
                    }
                  />
                  <span className="muted">{"Mbit/s"}</span>
                </div>
              </div>
            );
          })}
          <p className="muted">
            {
              "This app’s own limit on each adapter, in addition to the limits above. Leave a box empty for no limit."
            }
          </p>
        </Section>
        <Section title="Data cap">
          <Amount
            label="Data cap"
            value={draft.capGB}
            unit="GB"
            placeholder="no cap"
            onChange={(O: any) =>
              g({
                capGB: O,
              })
            }
          />
          <Choice
            name="capPeriod"
            checked={draft.capPeriod !== "day"}
            onChange={() =>
              g({
                capPeriod: "month",
              })
            }
          >
            {"per month"}
          </Choice>
          <Choice
            name="capPeriod"
            checked={draft.capPeriod === "day"}
            onChange={() =>
              g({
                capPeriod: "day",
              })
            }
          >
            {"per day"}
          </Choice>
          <p className="muted">
            {
              "At the cap the app’s connections stop until the period starts again. You are warned at 80%."
            }
          </p>
        </Section>
        <Section title="DNS server">
          {[
            ["windows", "Windows default"],
            ["cloudflare", "Cloudflare (1.1.1.1)"],
            ["google", "Google (8.8.8.8)"],
            ["quad9", "Quad9 (9.9.9.9)"],
            ["custom", "Another server"],
          ].map(([O, rl]: any) => (
            <Choice
              name="dns"
              checked={(draft.dns || "windows") === O}
              onChange={() =>
                g({
                  dns: O,
                })
              }
              key={O}
            >
              {rl}
            </Choice>
          ))}
          {draft.dns === "custom" && (
            <label className="field-label">
              {"DNS server"}
              <input
                required={!0}
                value={draft.customDNS ?? ""}
                placeholder="1.1.1.1"
                onChange={(O: any) =>
                  g({
                    customDNS: O.target.value,
                  })
                }
              />
            </label>
          )}
          <p className="muted">
            {
              "Applies to plain TCP/UDP DNS through the engine. Apps using Windows’ shared resolver or encrypted DNS are not covered."
            }
          </p>
        </Section>
        <Section title="Which program">
          <Choice
            checkbox={!0}
            checked={draft.exactPath}
            onChange={() =>
              g({
                exactPath: !draft.exactPath,
              })
            }
            detail={`Off: any file named ${draft.exe || "the executable"} follows this rule.`}
          >
            {"Only this exact file"}
          </Choice>
          {draft.exactPath && (
            <label className="field-label">
              {"Exact file path"}
              <input
                required={!0}
                value={draft.path ?? ""}
                onChange={(O: any) =>
                  g({
                    path: O.target.value,
                  })
                }
                placeholder="C:\\Program Files\\App\\app.exe"
              />
            </label>
          )}
          <p className="muted">{"Applies to new connections."}</p>
        </Section>
      </div>
      <footer className="panel-footer">
        {onDelete && (
          <button type="button" disabled={busy} onClick={onDelete}>
            <Icon name="trash" />
            {"Remove app"}
          </button>
        )}
        <button className="primary" type="submit" disabled={busy || !Cl}>
          <Icon name="save" />
          {"Save"}
        </button>
      </footer>
    </form>
  );
}
function SettingsPanel({
  settings,
  adapters,
  busy,
  onSave,
  onClose,
}: {
  settings: Settings;
  adapters: Card[];
  busy: boolean;
  onSave: (s: Settings) => void;
  onClose: () => void;
}) {
  const [adapterDown, setAdapterDown] = React.useState(
    settings.adapterDown ?? {},
  );
  const [adapterUp, setAdapterUp] = React.useState(settings.adapterUp ?? {});
  return (
    <div className="panel-form">
      <header className="panel-heading">
        <h2>Settings</h2>
        <button
          className="icon-button"
          aria-label="Close settings"
          onClick={onClose}
        >
          <Icon name="close" />
        </button>
      </header>
      <div className="panel-scroll">
        <Section title="Startup">
          <Choice
            checkbox
            checked={settings.startWindows}
            disabled={busy}
            onChange={() =>
              onSave({ ...settings, startWindows: !settings.startWindows })
            }
          >
            Start with Windows
          </Choice>
          <Choice
            checkbox
            checked={settings.startRouting}
            disabled={busy}
            onChange={() =>
              onSave({ ...settings, startRouting: !settings.startRouting })
            }
          >
            Start routing when it opens
          </Choice>
          <p className="muted">
            Starts in the tray with Windows. Routing starts only when the second
            option is enabled.
          </p>
        </Section>
        <Section title="Window">
          <CloseBehaviorSettings />
        </Section>
        <Section title="Speed limit per adapter">
          {adapters.map((adapter) => (
            <div className="adapter-limit" key={adapter.name}>
              <p className="muted">
                {adapter.name}
                {!adapter.routable ? " (not routed)" : ""}
              </p>
              <fieldset
                disabled={!adapter.routable || busy}
                className="adapter-limit-fields"
              >
                <div className="limit-pair">
                  <Amount
                    label={`${adapter.name} total download limit`}
                    value={adapterDown[adapter.name] || undefined}
                    unit="↓"
                    placeholder="no limit"
                    onChange={(value: number | undefined) =>
                      setAdapterDown({
                        ...adapterDown,
                        [adapter.name]: value ?? 0,
                      })
                    }
                  />
                  <Amount
                    label={`${adapter.name} total upload limit`}
                    value={adapterUp[adapter.name] || undefined}
                    unit="↑"
                    placeholder="no limit"
                    onChange={(value: number | undefined) =>
                      setAdapterUp({ ...adapterUp, [adapter.name]: value ?? 0 })
                    }
                  />
                  <span className="muted">Mbit/s</span>
                </div>
              </fieldset>
            </div>
          ))}
          <button
            className="primary"
            disabled={busy}
            onClick={() => onSave({ ...settings, adapterDown, adapterUp })}
          >
            Apply limits
          </button>
          <p className="muted">
            Limits traffic routed by MultiRoute through each adapter, for all
            apps together. ↓ download, ↑ upload. Leave a box empty for no limit.
            Per-app limits also apply.
          </p>
        </Section>
        <Section title="Updates">
          <Choice checkbox checked={false} disabled onChange={() => {}}>
            Install updates automatically
          </Choice>
          <p className="muted">
            Coming later. Update checks and installation are not available yet.
          </p>
          <button disabled>Check now</button>
        </Section>
        <Section title="About">
          <p>MultiRoute</p>
          <p className="muted">
            Built on the MIT net-merge engine. Usage history stays on this PC.
          </p>
        </Section>
      </div>
    </div>
  );
}
