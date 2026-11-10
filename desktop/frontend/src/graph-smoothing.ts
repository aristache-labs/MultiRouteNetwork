// Presentation only: a one-second trailing average, including a fractional
// oldest sample when the telemetry interval does not divide 1000 ms exactly.
export function smoothGraph(values: number[], sampleMs = 70): number[] {
  const window = 1000 / sampleMs;
  const whole = Math.floor(window);
  const fraction = window - whole;
  const result: number[] = [];
  let sum = 0;
  for (let i = 0; i < values.length; i++) {
    sum += values[i];
    if (i >= whole) sum -= values[i - whole];
    const extra = i >= whole ? values[i - whole] * fraction : 0;
    result.push((sum + extra) / Math.min(i + 1, window));
  }
  return result;
}
