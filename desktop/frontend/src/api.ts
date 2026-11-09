import type { Rule, Settings, Status, UsageRow } from "./types";

async function send<T>(path: string, body?: unknown): Promise<T> {
  const res = await fetch(path, {
    method: body === undefined ? "GET" : "POST",
    headers:
      body === undefined ? undefined : { "Content-Type": "application/json" },
    body: body === undefined ? undefined : JSON.stringify(body),
  });
  const data = (await res.json()) as { ok?: boolean; error?: string } & T;
  if (!res.ok || data.error) {
    throw new Error(data.error || res.statusText);
  }
  return data;
}

export const api = {
  windowPreferences: () =>
    send<{
      action: "ask" | "tray" | "exit";
      pending: boolean;
      closeError?: string;
    }>("/api/window/preferences"),
  saveClosePreference: (action: "ask" | "tray" | "exit") =>
    send<Record<string, never>>("/api/window/preferences", { action }),
  closeWindow: (action: "tray" | "exit" | "cancel", remember = false) =>
    send<Record<string, never>>("/api/window/close", { action, remember }),
  status: () => send<{ status: Status }>("/api/status"),
  start: () => send<Record<string, never>>("/api/start", {}),
  stop: () => send<Record<string, never>>("/api/stop", {}),
  saveRule: (rule: Rule) => send<Record<string, never>>("/api/rule", rule),
  deleteRule: (id: string) =>
    send<Record<string, never>>("/api/rule/delete", { id }),
  saveSettings: (settings: Settings) =>
    send<Record<string, never>>("/api/settings", settings),
  usage: (range: string) =>
    send<{ usage: UsageRow[] }>(`/api/usage?range=${range}`),
  csv: (range: string) => send<{ csv: string }>(`/api/csv?range=${range}`),
};
