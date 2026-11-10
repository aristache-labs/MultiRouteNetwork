import assert from "node:assert/strict";
import test from "node:test";
import { smoothGraph } from "../src/graph-smoothing.ts";
test("short bursts are averaged without changing the source or time axis", () => {
 const data=Array(1200).fill(0);data[500]=1000;
 const result=smoothGraph(data);
 assert.equal(result.length,1200);
 assert.equal(Math.max(...result),70);
 assert.ok(Math.abs(result.reduce((a,b)=>a+b,0)-1000)<1e-8);
 assert.equal(data[500],1000);
 assert.equal(data[501],0);
});
test("sustained transfers keep their plateau and return to idle after the window", () => {
 const data=Array(1200).fill(0);data.fill(500,400,800);
 const result=smoothGraph(data);
 assert.equal(result[414],500);
 assert.equal(result[799],500);
 assert.equal(result[814],0);
 assert.ok(result[400]>0 && result[400]<500);
 assert.ok(result[800]>0 && result[800]<500);
 assert.deepEqual(smoothGraph(Array(1200).fill(0)),Array(1200).fill(0));
});
