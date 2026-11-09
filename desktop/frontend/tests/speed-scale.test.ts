import assert from "node:assert/strict";
import test from "node:test";
import { speedScale } from "../src/speed-scale.ts";
const bps = (mbps: number) => (mbps * 1e6) / 8;
test("unlimited bars use a fixed 100 Mbit/s ceiling", () => {
  for (const rate of [0, 25, 50, 100, 150]) {
    assert.equal(speedScale(bps(rate)).maxMbps, 100);
    assert.equal(speedScale(bps(rate)).percent, Math.min(100, rate));
  }
});
test("app limits override the visual ceiling independently per direction", () => {
  assert.equal(speedScale(bps(0.5), 1).percent, 50);
  assert.equal(speedScale(bps(1), 1).percent, 100);
  assert.equal(speedScale(bps(25), 50).percent, 50);
  assert.equal(speedScale(bps(100), 200).percent, 50);
});
test("empty or invalid limits use 100; invalid rates remain empty", () => {
  for (const limit of [undefined, 0, -1, NaN, Infinity])
    assert.equal(speedScale(bps(25), limit).percent, 25);
  for (const rate of [-1, NaN, Infinity])
    assert.equal(speedScale(rate).percent, 0);
});
