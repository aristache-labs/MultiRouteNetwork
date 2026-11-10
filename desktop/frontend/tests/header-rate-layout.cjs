// Headless CSS regression for header geometry while live rate labels change.
const {chromium}=require(process.env.PLAYWRIGHT_MODULE || 'playwright');
const fs=require('node:fs');
const path=require('node:path');
const assert=require('node:assert/strict');
(async()=>{
 const browser=await chromium.launch({headless:true});
 try {
  const page=await browser.newPage();
  const css=fs.readFileSync(path.resolve(__dirname,'../src/styles.css'),'utf8');
  await page.setContent(`<style>${css}</style><div class="app"><header class="top"><div class="identity"><div class="brand-mark"></div><div><h1>MultiRoute</h1><div class="run-state">Running</div></div></div><div class="header-spark"><svg class="spark"></svg></div><div class="header-speeds"><span class="down"></span><span class="up"></span></div><nav class="header-actions"><button>Usage</button><button>Dark</button><button>Settings</button><button>Stop</button></nav></header></div>`);
  for(const width of [1920,1400,960,768]) for(const theme of ['dark','light']) {
   await page.setViewportSize({width,height:900});
   await page.evaluate(theme=>document.documentElement.dataset.theme=theme,theme);
   let baseline;
   for(const text of ['↓ 0 kbit/s','↓ 956.3 Mbit/s','↓ 1901.6 Mbit/s','↓ 9999.9 Mbit/s','↓ 7 kbit/s']) {
    const rect=await page.evaluate(text=>{
      document.querySelector('.header-speeds .down').textContent=text;
      document.querySelector('.header-speeds .up').textContent=text.replace('↓','↑');
      const el=document.querySelector('.header-spark');const r=el.getBoundingClientRect();
      const speeds=document.querySelector('.header-speeds').getBoundingClientRect();
      return {x:r.x,y:r.y,width:r.width,height:r.height,gap:speeds.left-r.right};
    },text);
    if(width>=1400) assert.ok(rect.gap<=20.1 && rect.gap>=0,'top graph must sit beside rates: '+JSON.stringify({width,rect}));
    baseline??=rect;
    for(const key of ['x','y','width','height']) assert.ok(Math.abs(rect[key]-baseline[key])<0.1,JSON.stringify({width,theme,text,baseline,rect,key}));
   }
  }
  console.log('PASS: header graph position and dimensions remain stable across changing rate labels in both themes and four widths');
 }finally{await browser.close()}
})().catch(e=>{console.error(e);process.exitCode=1});
