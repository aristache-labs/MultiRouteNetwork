import { useEffect, useRef } from "react";

// Translate measured samples rather than morphing old peaks into new ones.
export function Spark({ down, up }: { down?: number[]; up?: number[] }) {
  const group = useRef<SVGGElement>(null);
  const lines = useRef<(SVGPolylineElement | SVGPolygonElement | null)[]>([]);
  const latest = useRef({ down, up });
  latest.current = { down, up };

  useEffect(() => {
    let frame = 0;
    let previousDown: number[] | undefined;
    let previousUp: number[] | undefined;
    let signature = "";
    let started = performance.now();
    let lastFrame = started;
    let displayMax = 1;
    let targetMax = 1;
    let rawMax = 1;
    let step = 100 / 239;
    let moving = false;
    const reduced = window.matchMedia("(prefers-reduced-motion: reduce)");

    const draw = (now: number) => {
      const current = latest.current;
      if (current.down !== previousDown || current.up !== previousUp) {
        previousDown = current.down;
        previousUp = current.up;
        const d = current.down?.length ? current.down : Array(240).fill(0);
        const u = current.up?.length ? current.up : Array(240).fill(0);
        const nextSignature = JSON.stringify([d, u]);
        if (nextSignature !== signature) {
          const initial = signature === "";
          signature = nextSignature;
          rawMax = Math.max(1, ...d, ...u);
          // A stable rounded ceiling avoids rescaling on every minor change.
          targetMax = 2 ** Math.ceil(Math.log2(rawMax));
          if (initial || displayMax < rawMax) displayMax = targetMax;
          step = 100 / Math.max(1, d.length - 1);
          const points = (values: number[]) =>
            values
              .map(
                (v, i) => `${i * step},${58 - (Math.max(0, v) / rawMax) * 54}`,
              )
              .join(" ");
          const pd = points(d),
            pu = points(u);
          lines.current[0]?.setAttribute("points", `0,58 ${pu} 100,58`);
          lines.current[1]?.setAttribute("points", `0,58 ${pd} 100,58`);
          lines.current[2]?.setAttribute("points", pu);
          lines.current[3]?.setAttribute("points", pd);
          started = now;
          moving = !initial;
        }
      }
      const elapsed = Math.min(100, now - lastFrame);
      lastFrame = now;
      displayMax = reduced.matches
        ? targetMax
        : displayMax +
          (targetMax - displayMax) * (1 - Math.exp(-elapsed / 1500));
      const offset =
        reduced.matches || !moving
          ? 0
          : step * Math.max(0, 1 - (now - started) / 250);
      group.current?.setAttribute(
        "transform",
        `translate(${offset} 58) scale(1 ${rawMax / displayMax}) translate(0 -58)`,
      );
      frame = requestAnimationFrame(draw);
    };
    frame = requestAnimationFrame(draw);
    return () => cancelAnimationFrame(frame);
  }, []);

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
