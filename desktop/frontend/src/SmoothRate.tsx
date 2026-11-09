import { useEffect, useRef, useState } from "react";

// Presentation only: telemetry, accounting, and routing keep the measured rate.
export function useSmoothRate(bps = 0) {
  const target = Number.isFinite(bps) ? Math.max(0, bps) : 0;
  const current = useRef(target);
  const [displayed, setDisplayed] = useState(target);
  useEffect(() => {
    if (target === 0 || window.matchMedia("(prefers-reduced-motion: reduce)").matches) {
      current.current = target;
      setDisplayed(target);
      return;
    }
    const from = current.current;
    const start = performance.now();
    let frame = 0;
    const animate = (now: number) => {
      const progress = Math.min(1, (now - start) / 250);
      current.current = from + (target - from) * progress;
      setDisplayed(current.current);
      if (progress < 1) frame = requestAnimationFrame(animate);
    };
    frame = requestAnimationFrame(animate);
    return () => cancelAnimationFrame(frame);
  }, [target]);
  return target === 0 ? 0 : displayed;
}

export function SmoothRate({ bps = 0 }: { bps?: number }) {
  const rate = useSmoothRate(bps) * 8 / 1000;
  return <>{rate >= 1000 ? `${(rate / 1000).toFixed(1)} Mbit/s` : `${Math.round(rate)} kbit/s`}</>;
}
