// Deterministic offline motion regression: no engine, server, or live traffic.
const {chromium}=require(process.env.PLAYWRIGHT_MODULE || 'playwright');
const {buildSync}=require('../node_modules/esbuild');
const path=require('node:path');
const assert=require('node:assert/strict');
const bundle=buildSync({stdin:{contents:`
import React from 'react';
import {createRoot} from 'react-dom/client';
import {flushSync} from 'react-dom';
import {Spark} from './src/Spark';
const root=createRoot(document.getElementById('root'));
window.update=(down,up,verticalZoom=false,scale="traffic")=>flushSync(()=>root.render(<Spark down={down} up={up} verticalZoom={verticalZoom} scale={scale}/>));
`,resolveDir:path.resolve(__dirname,'..'),loader:'tsx'},bundle:true,write:false,format:'iife',define:{'process.env.NODE_ENV':'"production"'}}).outputFiles[0].text;
(async()=>{
 const browser=await chromium.launch({headless:true});
 try {
  const page=await browser.newPage();
  await page.setContent('<div id="root"></div>');
  await page.evaluate(()=>{
   window.clock=0;window.frames=new Map();window.frameID=0;
   performance.now=()=>window.clock;
   window.requestAnimationFrame=fn=>{window.frames.set(++window.frameID,fn);return window.frameID};
   window.cancelAnimationFrame=id=>window.frames.delete(id);
   window.tick=time=>{window.clock=time;const callbacks=[...window.frames.values()];window.frames.clear();callbacks.forEach(fn=>fn(time))};
  });
  await page.addScriptTag({content:bundle});
  const result=await page.evaluate(()=>{
   let d=Array(858).fill(10);d[800]=100;
   let u=Array(858).fill(1);
   const pointX=index=>{
    const g=document.querySelector('.spark g');
    const x=+g.getAttribute('transform').match(/translate\(([^ ]+)/)[1];
    const p=document.querySelector('.line-down').getAttribute('points').split(' ')[index + 1].split(',');
    return +p[0]+x;
   };
   window.update(d,u);window.tick(0);
   const initialX=pointX(800);
   // Polling faster than measurement must not restart movement on duplicates.
   window.update([...d],[...u]);window.tick(10);
   window.update([...d],[...u]);window.tick(20);
   window.tick(55);
   const before=pointX(800);
   d=[...d.slice(1),10];u=[...u.slice(1),1];window.update(d,u);window.tick(55);
   const after=pointX(799);
   window.tick(66);const next=pointX(799);
   window.tick(80);const beforeScale=document.querySelector('g').getAttribute('transform');
   d=[...d.slice(1),1000];window.update(d,u);window.tick(80);
   const afterScale=document.querySelector('g').getAttribute('transform');
   const scale=s=>+s.match(/scale\(1 ([^)]+)/)[1];
   // rawMax changes tenfold; rendered scale must preserve the old peak's height.
   const points=document.querySelector('.line-down').getAttribute('points').split(' ');
   const spacing=+points[2].split(',')[0] - +points[1].split(',')[0];
   return {initialX,pointCount:points.length,spacing,before,after,next,scaleRatio:scale(afterScale)/scale(beforeScale)};
  });
  assert.equal(result.pointCount,860,'show 858 seventy-millisecond samples plus two boundary points');
  assert.ok(Math.abs(result.spacing-100/857)<1e-8,'reference horizontal scale must use the latest 60 seconds');
  assert.ok(Math.abs(result.before-result.after)<1e-8,JSON.stringify(result));
  assert.ok(Math.abs((result.initialX-result.before)-(100/857)*55/70)<1e-8,'fast duplicate polls must preserve constant scroll speed');
  assert.ok(result.next<result.after,'graph must keep moving between polls');
  assert.ok(Math.abs(result.scaleRatio-79.3/16.3)<1e-8,'new maxima must not instantly shrink old peaks');
  const zoom = await page.evaluate(()=>{
    const d=Array(858).fill(1);d[0]=0;d.fill(100,790,810);
    window.update(d,Array(858).fill(1),true);window.tick(100);
    const values=selector=>document.querySelector(selector).getAttribute('points').split(' ').map(p=>+p.split(',')[1]);
    const down=values('.line-down'),up=values('.line-up');
    return {zero:down[1],downHeight:58-down[21],upHeight:58-up[21],peakHeight:58-down[801]};
  });
  assert.equal(zoom.zero,58,'zoom must keep idle samples at baseline');
  assert.ok(Math.abs(zoom.downHeight-5.4)<1e-8,'1% rate should occupy 10% height before common animated scale');
  assert.equal(zoom.downHeight,zoom.upHeight,'both directions must share vertical scaling');
  assert.ok(zoom.peakHeight<=54,'peaks must remain in the plot');
  const startup=await page.evaluate(()=>{
    const d=Array(858).fill(0),u=Array(858).fill(0);
    const x=index=>{
      const g=document.querySelector('.spark g');
      const offset=+g.getAttribute('transform').match(/translate\(([^ ]+)/)[1];
      const position=+document.querySelector('.line-down').getAttribute('points').split(' ')[index+1].split(',')[0];
      return position+offset;
    };
    window.update(d,u,false);window.tick(200);window.tick(1000);
    const idleOffset=+document.querySelector('g').getAttribute('transform').match(/translate\(([^ ]+)/)[1];
    d[857]=100;window.update([...d],[...u]);window.tick(1000);
    const start=x(857);window.tick(1100);const before=x(857);
    d.shift();d.push(100);d.shift();d.push(100);
    window.update([...d],[...u]);window.tick(1140);const after=x(855);
    window.tick(1200);const later=x(855);
    return {idleOffset,start,before,after,later};
  });
  assert.ok(startup.idleOffset>=0,'empty history must not extrapolate left of available samples');
  assert.ok(startup.before<=startup.start+1e-8 && startup.after<=startup.before+1e-8 && startup.later<=startup.after+1e-8,'rising edge must never recoil right after idle or a delayed poll: '+JSON.stringify(startup));
  const quiet=await page.evaluate(()=>{
    const d=Array(858).fill(100000),u=Array(858).fill(12500);
    const height=()=>{
      const scale=+document.querySelector('g').getAttribute('transform').match(/scale\(1 ([^)]+)/)[1];
      const y=+document.querySelector('.line-down').getAttribute('points').split(' ')[400].split(',')[1];
      return (58-y)*scale;
    };
    window.update(d,u,true);window.tick(2000);const adapter=height();
    window.update(d,u,false);window.tick(2010);const header=height();
    return {adapter,header};
  });
  assert.ok(quiet.adapter<6,'sub-Mbit background traffic should remain near adapter baseline: '+JSON.stringify(quiet));
  assert.ok(quiet.header<1,'sub-Mbit background traffic should remain near header baseline');
  const application=await page.evaluate(()=>{
    const d=Array(858).fill(8000),u=Array(858).fill(0);
    const height=()=>{
      const scale=+document.querySelector('g').getAttribute('transform').match(/scale\(1 ([^)]+)/)[1];
      const y=+document.querySelector('.line-down').getAttribute('points').split(' ')[400].split(',')[1];
      return (58-y)*scale;
    };
    window.update(d,u,false,'application');window.tick(2200);const app=height();
    window.update(d,u,false,'traffic');window.tick(2210);const header=height();
    return {app,header};
  });
  assert.ok(application.app>=20 && application.app<=30,'64 kbit/s should be visible on the application scale: '+JSON.stringify(application));
  assert.ok(application.header<1,'header must retain its quiet-traffic scale');
  console.log('PASS: delayed-poll position continuity, inter-poll motion, gradual vertical rescaling');
 }finally{await browser.close()}
})().catch(e=>{console.error(e);process.exitCode=1});
