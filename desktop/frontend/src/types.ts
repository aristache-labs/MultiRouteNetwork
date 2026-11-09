export type Rule = {
  id: string;
  exe: string;
  path?: string;
  exactPath: boolean;
  mode: "all" | "one" | "several";
  adapters?: string[];
  preferred?: string;
  fallback: "wait" | "others" | "default" | "choose";
  fallbackAdapters?: string[];
  lastResort?: boolean;
  lowestLatency: boolean;
  downMbps?: number;
  upMbps?: number;
  adapterDown?: Record<string, number>;
  adapterUp?: Record<string, number>;
  capGB?: number;
  capPeriod?: "day" | "month" | "";
  dns?: string;
  customDNS?: string;
};

export type Settings = {
  adapterDown?: Record<string, number>;
  adapterUp?: Record<string, number>;
  theme: "dark" | "light";
  startWindows: boolean;
  startRouting: boolean;
  autoUpdate: boolean;
  defaultAdapter?: string;
  downMbps?: number;
  upMbps?: number;
};

export type Card = {
  name: string;
  ip: string;
  ip6: string;
  state: string;
  excluded: boolean;
  reason?: string;
  downBps: number;
  upBps: number;
  todayDown: number;
  todayUp: number;
  monthDown: number;
  monthUp: number;
  latencyMs: number;
  down: number[];
  up: number[];
  default: boolean;
  routable: boolean;
  linkDownMbps?: number;
  linkUpMbps?: number;
};

export type AppRow = {
  running?: boolean;
  runningKnown?: boolean;
  exe: string;
  path: string;
  downBps: number;
  upBps: number;
  downBytes: number;
  upBytes: number;
  via: string;
  warn: boolean;
  capped: boolean;
  down: number[];
  up?: number[];
};

export type Status = {
  running: boolean;
  starting?: boolean;
  error?: string;
  theme: string;
  defaultAdapter: string;
  downBps: number;
  upBps: number;
  down: number[];
  up: number[];
  adapters: Card[] | null;
  apps: AppRow[] | null;
  rules: Rule[] | null;
  settings: Settings;
};

export type UsageRow = {
  exe: string;
  path: string;
  down: number;
  up: number;
  byNic: Record<string, number>;
  byNicDown?: Record<string, number>;
  byNicUp?: Record<string, number>;
};
