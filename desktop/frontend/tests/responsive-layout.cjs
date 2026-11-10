// Headless, offline layout regression check. Build the frontend first.
// Requires Playwright (or PLAYWRIGHT_MODULE pointing to an installed module).
const { chromium } = require(process.env.PLAYWRIGHT_MODULE || 'playwright');
const fs = require('node:fs');
const path = require('node:path');
const assert = require('node:assert/strict');
const dist = path.resolve(__dirname, '../dist');
const output = process.env.LAYOUT_OUTPUT || path.resolve(__dirname, '../../..', 'diagnostics/resize-layout');
const origin = 'https://multiroute.test';
const sizes = [[960,640],[1050,700],[1250,800],[1400,900],[1734,1117],[1920,1080],[2560,1440],[768,640]];
const names = ['Tailscale corporate VPN with a long adapter name','Ethernet','Ethernet 4 phone hotspot','Wi-Fi wireless connection'];
const samples = Array.from({length:858},(_,i)=>1e6+(i%17)*1e5);
const adapters = names.map((name,i)=>({name,ip:i===0?'2001:db8:1234:5678:abcd:ef01:2345:6789':`192.168.100.${100+i}`,ip6:'',state:i===0?'VPN':'online',routable:i>0,excluded:i===0,default:i===3,downBps:1250000000,upBps:125000000,linkDownMbps:1000000,linkUpMbps:1000000,todayDown:987650000000,todayUp:876540000000,monthDown:9876500000000,monthUp:8765400000000,latencyMs:123,down:samples,up:samples}));
const apps = Array.from({length:8},(_,i)=>({exe:i===0?'VeryLongApplicationExecutableNameForLayoutTesting.exe':`Application${i}.exe`,path:`C:\Programs\App${i}\application.exe`,running:true,runningKnown:true,downBps:1250000000,upBps:125000000,downBytes:987650000000,upBytes:876540000000,via:names.slice(1).join(', '),down:samples,up:samples,warn:false,capped:false}));
const rules = apps.map((app,i)=>({id:`rule-${i}`,exe:app.exe,path:app.path,exactPath:false,mode:'several',adapters:names.slice(1),fallback:'others',lowestLatency:false,downMbps:1000000,upMbps:1000000,dns:'windows',capPeriod:'month'}));
const usage = apps.map(app=>({...app,down:987650000000,up:876540000000,byNic:Object.fromEntries(names.map(name=>[name,123456789]))}));

(async()=>{
  fs.mkdirSync(output,{recursive:true});
  const browser = await chromium.launch({headless:true});
  const failures=[], results=[];
  try {
    for(const theme of ['dark','light']) {
      const page=await browser.newPage();
      const errors=[];
      page.on('pageerror',e=>errors.push(e.message));
      await page.route('**/*',async route=>{
        const url=new URL(route.request().url());
        if(url.origin!==origin)return route.abort();
        if(url.pathname.startsWith('/api/')) {
          let response={};
          if(url.pathname==='/api/status')response={status:{running:true,theme,settings:{theme,startWindows:false,startRouting:false,autoUpdate:false,defaultAdapter:names[3]},defaultAdapter:names[3],adapters,apps,rules,downBps:1250000000,upBps:125000000,down:samples,up:samples}};
          else if(url.pathname==='/api/usage')response={usage};
          else if(url.pathname==='/api/window/preferences')response={action:'ask',pending:false};
          else if(url.pathname==='/api/icon')return route.fulfill({status:404,body:''});
          return route.fulfill({json:response});
        }
        const file=path.join(dist,url.pathname==='/'?'index.html':url.pathname.slice(1));
        if(!fs.existsSync(file))return route.fulfill({status:404,body:''});
        const ext=path.extname(file);
        return route.fulfill({body:fs.readFileSync(file),contentType:({'.js':'application/javascript','.css':'text/css','.html':'text/html','.svg':'image/svg+xml'})[ext]||'application/octet-stream'});
      });
      for(const [width,height] of sizes) {
        await page.setViewportSize({width,height});
        await page.goto(origin);
        await page.locator('.card').first().waitFor();
        for(const panel of ['none','settings','add','edit','usage']) {
          if(panel==='settings')await page.getByRole('button',{name:'Settings',exact:true}).click();
          if(panel==='add'){
            await page.getByRole('button',{name:'Close settings'}).click();
            await page.getByRole('button',{name:'Add app',exact:true}).click();
          }
          if(panel==='edit')await page.locator('.app-name').first().click();
          if(panel==='usage')await page.getByRole('button',{name:'Usage',exact:true}).click();
          await page.evaluate(()=>new Promise(resolve=>requestAnimationFrame(()=>requestAnimationFrame(resolve))));
          const inspect=()=>{
            const bad=[];
            const visible=el=>el.getClientRects().length>0;
            for(const el of document.querySelectorAll('.app,.top,.header-actions,.workspace,.main,.cards,.card,.rates,.link-speeds,.adapter-totals,.applications,.app-row,.side,.panel-scroll,.limit-pair,.data-breakdown')){
              if(visible(el)&&el.scrollWidth>el.clientWidth+2)bad.push(`${el.className}: horizontal overflow ${el.scrollWidth}/${el.clientWidth}`);
            }
            for(const el of document.querySelectorAll('.card .rates > span,.card .link-speeds > span,.card .adapter-totals span,.app-row .speed,.app-row .data-used')){
              if(!visible(el))continue;
              const box=el.getBoundingClientRect(),parent=el.closest('.card,.app-row').getBoundingClientRect();
              if(box.left<parent.left-1||box.right>parent.right+1)bad.push(`${el.className}: escapes card/row`);
            }
            for(const card of document.querySelectorAll('.card')){
              const text=card.querySelector('.adapter-identity').getBoundingClientRect();
              const badges=card.querySelector('.card-badges').getBoundingClientRect();
              if(Math.min(text.right,badges.right)>Math.max(text.left,badges.left)+1 && Math.min(text.bottom,badges.bottom)>Math.max(text.top,badges.top)+1)bad.push('adapter identity overlaps badges');
            }
            const body=document.querySelector('.app-table-body');
            if(body){
              body.scrollTop=body.scrollHeight;
              const last=body.querySelector('.app-row:last-child');
              if(last&&last.getBoundingClientRect().bottom>body.getBoundingClientRect().bottom+2)bad.push('last application cannot be reached by scrolling');
              body.scrollTop=0;
            }
            const panel=document.querySelector('.panel-scroll');
            if(panel){
              panel.scrollTop=panel.scrollHeight;
              const last=panel.lastElementChild;
              if(last&&last.getBoundingClientRect().bottom>panel.getBoundingClientRect().bottom+2)bad.push('last panel section cannot be reached by scrolling');
              panel.scrollTop=0;
            }
            return bad;
          };
          const problems=await page.evaluate(inspect);
          const entry={theme,width,height,panel,problems};results.push(entry);
          if(problems.length){failures.push(entry);await page.screenshot({path:path.join(output,`failure-${theme}-${width}-${panel}.png`)});}
          if(theme==='dark'&&panel==='settings'&&[960,1400,1734].includes(width))await page.screenshot({path:path.join(output,`settings-${width}.png`)});
          // Resize an already open panel in both directions, without reloading.
          const changedWidth=width>960?960:1920;
          await page.setViewportSize({width:changedWidth,height});
          const resizeProblems=await page.evaluate(inspect);
          const resized={theme,width:changedWidth,height,panel,resizedFrom:width,problems:resizeProblems};
          results.push(resized);
          if(resizeProblems.length)failures.push(resized);
          await page.setViewportSize({width,height});
        }
      }
      assert.deepEqual(errors,[],`Browser errors in ${theme}`);
      await page.close();
    }
  } finally {await browser.close();}
  fs.writeFileSync(path.join(output,'results.json'),JSON.stringify(results,null,2));
  console.log(JSON.stringify({cases:results.length,failures},null,2));
  assert.equal(failures.length,0,'Responsive layout failures');
})().catch(error=>{console.error(error);process.exitCode=1;});
