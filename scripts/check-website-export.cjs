const fs=require('node:fs'),path=require('node:path'),assert=require('node:assert/strict');
const {JSDOM}=require('../mobile/node_modules/jsdom');
const root=path.resolve(__dirname,'../therapist-portal/out');
const routes=['/','/therapists','/about','/teams','/pricing','/privacy','/terms','/login','/dashboard','/dashboard/notes','/dashboard/clients/_'];
const errors=[];let links=0,assets=0;
for(const route of routes){const file=path.join(root,route,'index.html');const html=fs.readFileSync(file,'utf8');const dom=new JSDOM(html,{url:'https://dreamlog-48f94.web.app'+route});const d=dom.window.document;
 if(/OdePreview2026!|demo@ode.example|ODE WEBSITE PROPOSAL|review concept|not the live site/i.test(d.body.textContent))errors.push(route+': preview copy or credentials leaked');
 const ids=[...d.querySelectorAll('[id]')].map(e=>e.id);if(ids.length!==new Set(ids).size)errors.push(route+': duplicate IDs');
 for(const a of d.querySelectorAll('a[href]')){const raw=a.getAttribute('href');if(raw.startsWith('mailto:')||raw.startsWith('tel:')||/^https?:/.test(raw))continue;links++;const u=new URL(raw,'https://dreamlog-48f94.web.app'+route);let target=path.join(root,u.pathname,'index.html');if(!fs.existsSync(target)&&!fs.existsSync(path.join(root,u.pathname)))errors.push(route+': broken link '+raw);if(u.hash&&u.pathname.replace(/\/$/,'')===route.replace(/\/$/,'')){if(!d.getElementById(decodeURIComponent(u.hash.slice(1))))errors.push(route+': broken fragment '+raw);}}
 for(const e of d.querySelectorAll('img[src],script[src],link[rel=stylesheet][href],link[rel=preload][href]')){const raw=e.getAttribute('src')||e.getAttribute('href');if(!raw.startsWith('/'))continue;assets++;if(!fs.existsSync(path.join(root,decodeURIComponent(raw))))errors.push(route+': missing asset '+raw);}
 if(route==='/privacy')assert.equal(d.querySelectorAll('.legal-section').length,12);
 if(route==='/terms')assert.equal(d.querySelectorAll('.legal-section').length,14);
 dom.window.close();
}
assert.deepEqual(errors,[]);
console.log(JSON.stringify({routes:routes.length,links,assets,errors},null,2));
