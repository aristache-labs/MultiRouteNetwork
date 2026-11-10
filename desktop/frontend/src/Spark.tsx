import { useEffect, useRef } from "react";
import { smoothGraph } from "./graph-smoothing";

// Translate measured samples rather than morphing old peaks into new ones.
// Reference recording scrolls about 1/60 of the plot width per second.
// Keep 858 seventy-millisecond samples (about 60 seconds) and interpolate movement between polls.
const visibleSamples = 858;
const sampleIntervalMs = 70;
export function Spark({ down, up, verticalZoom = false, scale = "traffic" }: {
  down?: number[]; up?: number[]; verticalZoom?: boolean; scale?: "traffic" | "application";
}) {
  const group = useRef<SVGGElement>(null);
  const lines = useRef<(SVGPolylineElement | SVGPolygonElement | null)[]>([]);
  const latest = useRef({ down, up });
  latest.current = { down, up };

  useEffect(() => {
    // Per-application curves reveal small activity; the aggregate/adapter floor
    // remains 100 Mbit/s. These ceilings never impose traffic limits.
    const minimumScaleBps = (scale === "application" ? 128000 : 100e6) / 8;
    // Square-root vertical scaling lifts sustained/low traffic below brief peaks.
    // Both directions share the same scale; zero stays at the baseline.
    const exponent = verticalZoom ? 0.5 : 1;
    let frame = 0;
    let previousDown: number[] | undefined;
    let previousUp: number[] | undefined;
    let signature = "";
    let lastFrame = performance.now();
    let offset = 0;
    let displayMax = 1;
    let targetMax = 1;
    let startupUntil = -Infinity;
    let rawMax = 1;
    let step = 100 / (visibleSamples - 1);
    let moving = false;
    const reduced = window.matchMedia("(prefers-reduced-motion: reduce)");

    const draw = (now: number) => {
      const elapsed = Math.max(0, now - lastFrame);
      lastFrame = now;
      // Carry the fractional position across polls. Network jitter must not
      // restart a whole sample's animation or force it to stop at each poll.
      if (moving) offset = Math.max(0, offset - step * elapsed / sampleIntervalMs);
      const current = latest.current;
      if (current.down !== previousDown || current.up !== previousUp) {
        const oldDown = previousDown?.slice(-visibleSamples);
        const oldUp = previousUp?.slice(-visibleSamples);
        previousDown = current.down;
        previousUp = current.up;
        const d = current.down?.length ? current.down.slice(-visibleSamples) : Array(visibleSamples).fill(0);
        const u = current.up?.length ? current.up.slice(-visibleSamples) : Array(visibleSamples).fill(0);
        const nextSignature = JSON.stringify([d, u]);
        if (nextSignature !== signature) {
          const initial = signature === "";
          signature = nextSignature;
          const drawnDown = smoothGraph(d, sampleIntervalMs);
          const drawnUp = smoothGraph(u, sampleIntervalMs);
          rawMax = Math.max(1, ...drawnDown, ...drawnUp);
          // Leave modest headroom; ease changes instead of halving peak height.
          targetMax = Math.max(minimumScaleBps, rawMax * 1.04);
          if (initial) displayMax = targetMax;
          const startsTraffic = !initial &&
            !(oldDown?.some(v => v > 0) || oldUp?.some(v => v > 0)) &&
            (d.some(v => v > 0) || u.some(v => v > 0));
          if (startsTraffic) {
            // The one-second average ramps up after idle. Seed its ceiling
            // from available measurements so that ramp cannot outrun the scale.
            displayMax = Math.max(targetMax, Math.max(1, ...d, ...u) * 1.04);
            startupUntil = now + 1000;
          }
          step = 100 / Math.max(1, d.length - 1);
          const points = (values: number[]) => {
            const y = (v: number) => 58 - Math.pow(Math.max(0, v) / rawMax, exponent) * 54;
            // Hold the boundary values outside the clip while samples slide.
            // This avoids empty edge strips during the presentation delay.
            return [
              `${-16 * step},${y(values[0] ?? 0)}`,
              ...values.map((v, i) => `${i * step},${y(v)}`),
              `${100 + 2 * step},${y(values[values.length - 1] ?? 0)}`,
            ].join(" ");
          };
          const pd = points(drawnDown),
            pu = points(drawnUp);
          lines.current[0]?.setAttribute("points", `0,58 ${pu} 100,58`);
          lines.current[1]?.setAttribute("points", `0,58 ${pd} 100,58`);
          lines.current[2]?.setAttribute("points", pu);
          lines.current[3]?.setAttribute("points", pd);
          // Detect skipped samples, including unchanged plateaus in one series.
          // Match both directions so a quiet upload cannot hide download motion.
          let shift = 0;
          if (!initial && oldDown?.length === d.length && oldUp?.length === u.length) {
            for (let n = 1; n <= Math.min(16, d.length - 1); n++) {
              if (d.slice(0, -n).every((v, i) => v === oldDown[i + n]) &&
                  u.slice(0, -n).every((v, i) => v === oldUp[i + n])) {
                shift = n;
                break;
              }
            }
          }
          // One sample of presentation delay leaves room for late polls.
          // Never extrapolate past the available sample. A late/empty poll
          // must not drag the curve left and then reset it to the right.
          offset = initial ? step : offset + Math.max(1, shift) * step;
          moving = true;
        }
      }
      displayMax = now <= startupUntil
        ? Math.max(displayMax, targetMax)
        : reduced.matches
        ? targetMax
        : displayMax +
          (targetMax - displayMax) * (1 - Math.exp(-elapsed / (targetMax > displayMax ? 300 : 1500)));
      const translation = reduced.matches ? 0 : offset;
      group.current?.setAttribute(
        "transform",
        `translate(${translation} 58) scale(1 ${Math.pow(rawMax / displayMax, exponent)}) translate(0 -58)`,
      );
      frame = requestAnimationFrame(draw);
    };
    frame = requestAnimationFrame(draw);
    return () => cancelAnimationFrame(frame);
  }, [verticalZoom, scale]);

  return (
    <svg
      className="spark"
      viewBox="0 0 100 60"
      preserveAspectRatio="none"
      aria-hidden="true"
    >
      <g ref={group}>
        <polygon
          ref={(el) => {
            lines.current[0] = el;
          }}
          className="area-up"
        />
        <polygon
          ref={(el) => {
            lines.current[1] = el;
          }}
          className="area-down"
        />
        <polyline
          ref={(el) => {
            lines.current[2] = el;
          }}
          className="line-up"
          vectorEffect="non-scaling-stroke"
        />
        <polyline
          ref={(el) => {
            lines.current[3] = el;
          }}
          className="line-down"
          vectorEffect="non-scaling-stroke"
        />
      </g>
    </svg>
  );
}
