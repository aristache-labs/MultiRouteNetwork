import type { AppRow } from "./types";

const key = (app: AppRow) => `${app.exe.toLowerCase()}:${app.path.toLowerCase()}`;
const noiseFloor = 16000; // 128 kbit/s: background chatter should not shuffle rows.

// Presentation only. Keep measured rates and routing decisions untouched.
export class TrafficOrder {
  private scores = new Map<string, number>();
  private order: string[] = [];
  private sampledAt: number | undefined;
  private rankedAt = -Infinity;

  sort(apps: AppRow[], now: number): AppRow[] {
    const elapsed = this.sampledAt === undefined ? 0 : Math.max(0, now - this.sampledAt);
    this.sampledAt = now;
    const weight = 1 - Math.exp(-elapsed / 2000);
    const rows = new Map(apps.map(app => [key(app), app]));
    for (const app of apps) {
      const id = key(app);
      const rate = Math.max(0, app.downBps) + Math.max(0, app.upBps);
      const old = this.scores.get(id);
      this.scores.set(id, old === undefined ? rate : old + (rate - old) * weight);
    }
    this.order = this.order.filter(id => rows.has(id));
    for (const id of this.scores.keys()) if (!rows.has(id)) this.scores.delete(id);
    const known = new Set(this.order);
    this.order.push(...[...rows.keys()].filter(id => !known.has(id)).sort());
    if (now - this.rankedAt >= 1000) {
      this.rankedAt = now;
      // Stable insertion with hysteresis: only pass a neighbour on a clear lead.
      for (let i = 1; i < this.order.length; i++) {
        let j = i;
        while (j > 0) {
          const ahead = this.scores.get(this.order[j - 1]) ?? 0;
          const behind = this.scores.get(this.order[j]) ?? 0;
          if (behind <= ahead + Math.max(noiseFloor, ahead * 0.2)) break;
          [this.order[j - 1], this.order[j]] = [this.order[j], this.order[j - 1]];
          j--;
        }
      }
    }
    return this.order.map(id => rows.get(id)!);
  }
}
