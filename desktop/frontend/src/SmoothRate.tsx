import { useEffect, useRef, useState } from "react";

// Presentation only: publish numbers every 70 ms, never per animation frame.
// Keep collecting the latest telemetry without restarting the display timer.
export function useSmoothRate(bps = 0) {
  const target = Number.isFinite(bps) ? Math.max(0, bps) : 0;
  const latest = useRef(target);
  latest.current = target;
  const [displayed, setDisplayed] = useState(target);
  useEffect(() => {
    const timer = window.setInterval(() => setDisplayed(latest.current), 70);
    return () => window.clearInterval(timer);
  }, []);
  return displayed;
}

export function SmoothRate({ bps = 0 }: { bps?: number }) {
  const rate = useSmoothRate(bps) * 8 / 1000;
  return <>{rate >= 1000 ? `${(rate / 1000).toFixed(1)} Mbit/s` : `${Math.round(rate)} kbit/s`}</>;
}
