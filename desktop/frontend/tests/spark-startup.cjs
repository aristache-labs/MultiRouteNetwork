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
window.reset=()=>flushSync(()=>root.render(null));
window.update=(down,up,verticalZoom=false)=>flushSync(()=>root.render(<Spark down={down} up={up} verticalZoom={verticalZoom}/>));
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


 for(const verticalZoom of [false,true]) for(const direction of ['download','upload','both']) {
 const result=await page.evaluate(({verticalZoom,direction})=>{
  window.reset();window.clock=0;
  let d=Array(858).fill(0),u=Array(858).fill(0);
  window.update(d,u,verticalZoom);window.tick(0);window.tick(1000);
  let clipped=0, peakHeight=0;
  for(let ms=0;ms<=2200;ms+=10){
   if(ms%70===0){
    d=[...d.slice(1),direction==='upload'?0:12500000];
    u=[...u.slice(1),direction==='download'?0:2500000];
    window.update([...d],[...u],verticalZoom);
   }
   window.tick(1000+ms);
   const transform=document.querySelector('.spark g').getAttribute('transform');
   const scale=+transform.match(/scale\(1 ([^)]+)/)[1];
   const offset=+transform.match(/translate\(([^ ]+)/)[1];
   for(const selector of ['.line-down','.line-up']){
    const points=document.querySelector(selector).getAttribute('points').split(' ').map(p=>p.split(',').map(Number));
    for(const [x,y] of points) if(x+offset>=0 && x+offset<=100){
     const height=(58-y)*scale;
     if(height>58+1e-8 || height< -1e-8)clipped++;
     peakHeight=Math.max(peakHeight,height);
    }
   }
  }
  return {clipped,peakHeight};
 },{verticalZoom,direction});
 assert.equal(result.clipped,0,JSON.stringify({verticalZoom,direction,...result}));
 assert.ok(result.peakHeight>(direction==='upload'?8:40),'startup must show traffic on the minimum visual scale');
 }
 console.log('PASS: zero-to-traffic remains inside the plot for download/upload/both, linear and zoomed');
 }finally{await browser.close()}
})().catch(e=>{console.error(e);process.exitCode=1});
