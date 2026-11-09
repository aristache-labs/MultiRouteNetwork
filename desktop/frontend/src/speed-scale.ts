// Rates arrive in bytes/second; limits and the visual scale use Mbit/s.
export function speedScale(bps: number, limitMbps?: number) {
  const limited = Number.isFinite(limitMbps) && (limitMbps ?? 0) > 0;
  const maxMbps = limited ? limitMbps! : 100;
  const currentMbps = Number.isFinite(bps) ? (Math.max(0, bps) * 8) / 1e6 : 0;
  return {
    limited,
    maxMbps,
    currentMbps,
    percent: Math.min(100, (currentMbps / maxMbps) * 100),
  };
}
