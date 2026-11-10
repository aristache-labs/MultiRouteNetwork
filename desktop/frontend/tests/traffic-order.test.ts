import assert from "node:assert/strict";
import test from "node:test";
import { TrafficOrder } from "../src/traffic-order.ts";
const app = (exe: string, downBps: number, upBps = 0) => ({exe, path: exe, downBps, upBps}) as any;
test("quiet background traffic retains order across 50ms samples", () => {
 const order = new TrafficOrder();
 for (let t=0;t<10000;t+=50) {
  const rows=order.sort([app('B', t%100 ? 12000 : 0),app('A',t%100 ? 0 : 12000)],t);
  assert.deepEqual(rows.map(a=>a.exe),['A','B']);
 }
});
test("clear sustained combined traffic rises, brief spikes do not shuffle rows", () => {
 const order=new TrafficOrder();
 assert.deepEqual(order.sort([app('A',1000000),app('B',0)],0).map(a=>a.exe),['A','B']);
 assert.deepEqual(order.sort([app('A',1000000),app('B',5000000)],50).map(a=>a.exe),['A','B']);
 for(let t=100;t<=6000;t+=50) order.sort([app('A',1000000),app('B',500000,1500000)],t);
 assert.deepEqual(order.sort([app('A',1000000),app('B',500000,1500000)],6050).map(a=>a.exe),['B','A']);
 assert.deepEqual(order.sort([app('A',0)],7000).map(a=>a.exe),['A']);
 assert.deepEqual(order.sort([],8000),[]);
});
