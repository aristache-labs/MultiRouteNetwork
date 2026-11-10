import assert from "node:assert/strict";
import test from "node:test";
import { runStateLabel } from "../src/run-state.ts";

test("routing states stay distinct from a missing service", () => {
  assert.equal(runStateLabel(null, true), "Starting…");
  assert.equal(runStateLabel({ running: true }, false), "Online");
  assert.equal(runStateLabel({ running: false }, false), "Offline");
  assert.equal(runStateLabel({ service: "uninstalled" }, false), "Service not installed");
  assert.equal(runStateLabel({ service: "authorization" }, false), "Authorization required");
  assert.equal(runStateLabel({ service: "stopped" }, false), "Engine stopped");
  assert.equal(runStateLabel({ service: "unreachable" }, false), "Communication error");
});
