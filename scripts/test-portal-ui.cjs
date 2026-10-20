// Run against postman/simulations/portal.sim.yaml, port 4920. No live data.
const fs = require('node:fs');
const path = require('node:path');
const Module = require('node:module');
const assert = require('node:assert/strict');
const { JSDOM } = require('../mobile/node_modules/jsdom');
const ts = require('../therapist-portal/node_modules/typescript');
const dom = new JSDOM('<!doctype html><div id="root"></div>', { url: 'http://localhost:4920/' });
for (const k of ['window','document','HTMLElement','HTMLInputElement','HTMLTextAreaElement','Event','MouseEvent','File','localStorage']) global[k] = dom.window[k];
Object.defineProperty(global, 'navigator', { value: dom.window.navigator, configurable: true });
global.IS_REACT_ACT_ENVIRONMENT = true;
window.confirm = global.confirm = () => true;
window.HTMLDialogElement.prototype.showModal = function () { this.open = true; };
window.HTMLDialogElement.prototype.close = function () { this.open = false; };
const React = require('../therapist-portal/node_modules/react');
const { createRoot } = require('../therapist-portal/node_modules/react-dom/client');
const { act, Simulate } = require('../therapist-portal/node_modules/react-dom/test-utils');
let pathname = '/dashboard', navigations = [];
const router = { replace: value => navigations.push(value), push: value => navigations.push(value) };
const originalLoad = Module._load;
Module._load = function (name, parent, isMain) {
  if (name === 'next/navigation') return { useRouter: () => router, usePathname: () => pathname };
  return originalLoad.call(this, name, parent, isMain);
};
for (const ext of ['.ts','.tsx']) Module._extensions[ext] = (module, filename) => {
  const source = fs.readFileSync(filename, 'utf8');
  module._compile(ts.transpileModule(source, { compilerOptions: { module: ts.ModuleKind.CommonJS, jsx: ts.JsxEmit.ReactJSX, esModuleInterop: true, target: ts.ScriptTarget.ES2020 } }).outputText, filename);
};
const { api, http, saveToken, notesApi } = require('../therapist-portal/src/lib/api.ts');
http.defaults.baseURL = process.env.ODE_PORTAL_TEST_URL || 'http://localhost:4920/ode';
assert(http.defaults.baseURL.startsWith('http://localhost:') || http.defaults.baseURL.startsWith('http://127.0.0.1:'));
http.defaults.adapter = 'http';
http.defaults.headers.common['x-mock-session'] = `portal-ui-${Date.now()}`;
let root;
const results = [];
function byText(tag, text) { return [...document.querySelectorAll(tag)].find(e => e.textContent.trim() === text); }
async function waitFor(fn, message) {
  for (let i=0;i<100;i++) { if (fn()) return; await act(async () => { await new Promise(r=>setTimeout(r,25)); }); }
  assert(fn(), message);
}
async function mount(file) {
  if (root) await act(async()=>root.unmount());
  const Component = require(path.resolve(__dirname, '../therapist-portal/src', file)).default;
  root = createRoot(document.getElementById('root'));
  await act(async()=>root.render(React.createElement(Component)));
}
async function click(element) { assert(element, 'Missing button'); await act(async()=>Simulate.click(element)); }
async function change(element,value) { assert(element); await act(async()=>{ const setter=Object.getOwnPropertyDescriptor(element.tagName==='TEXTAREA'?window.HTMLTextAreaElement.prototype:window.HTMLInputElement.prototype,'value').set;setter.call(element,value);Simulate.change(element,{target:{value}}); }); }
async function submit(form) { await act(async()=>Simulate.submit(form,{preventDefault(){}})); }
async function check(name,fn){await fn();results.push(name);}
(async()=>{
 await check('Actual Axios login and auth header reach local HTTP service',async()=>{const login=await api.login('portal-fixture@example.test','fixture-only');saveToken(login.token);assert.equal((await notesApi.me()).therapist.name,'Fixture Therapist');});
 await check('Real dashboard renders backend clients and search empty state',async()=>{await mount('app/dashboard/page.tsx');await waitFor(()=>document.querySelector('tbody tr'),'clients loaded');assert(document.body.textContent.includes('Fixture Client'));await change(document.querySelector('[aria-label="Search clients"]'),'no-match');assert(document.body.textContent.includes('No clients match'));await change(document.querySelector('[aria-label="Search clients"]'),'');});
 await check('Link request uses consent-gated API and reports pending approval',async()=>{await click(byText('button','Link client +'));await change(document.querySelector('#client-id'),'00000000-0000-4000-8000-000000000099');await submit(document.querySelector('dialog form'));await waitFor(()=>!document.querySelector('dialog').open,'link request completed');});
 await check('Failed consent request never unlocks note editing',async()=>{pathname='/dashboard/notes';await mount('app/dashboard/notes/page.tsx');await waitFor(()=>byText('button','I understand and accept'),'consent loaded');http.defaults.headers.common['x-fixture-failure']='consent';await click(byText('button','I understand and accept'));await waitFor(()=>document.querySelector('[role="alert"]'),'consent error');assert(document.querySelector('#note-text').disabled);delete http.defaults.headers.common['x-fixture-failure'];});
 await check('Successful consent enables saving a new real API session',async()=>{await click(byText('button','I understand and accept'));await waitFor(()=>document.querySelector('#note-text')&&!document.querySelector('#note-text').disabled,'consent enabled');await change(document.querySelector('#note-text'),'Fictional session notes.\nSecond fictional bullet.');await click(byText('button','Save notes'));await waitFor(()=>document.body.textContent.includes('Notes saved.'),'notes saved');assert((await notesApi.listSessions()).length>0);});
 await check('Failed save preserves draft and existing server notes',async()=>{await change(document.querySelector('#note-text'),'Unsaved fictional edit.');http.defaults.headers.common['x-fixture-failure']='save';await click(byText('button','Save notes'));await waitFor(()=>document.querySelector('[role="alert"]'),'save error');assert.equal(document.querySelector('#note-text').value,'Unsaved fictional edit.');assert.equal((await notesApi.listSessions())[0].bullets[0],'Fictional session notes.');delete http.defaults.headers.common['x-fixture-failure'];});
 await check('Saved edits and summary use existing endpoint contracts',async()=>{await click(byText('button','Save notes'));await waitFor(()=>document.body.textContent.includes('Notes saved.'),'saved edit');await click(byText('button','Generate summary'));await waitFor(()=>document.body.textContent.includes('Fictional summary from saved notes.'),'summary complete');});
 await check('Failed deletion preserves selected notes; successful deletion removes them',async()=>{http.defaults.headers.common['x-fixture-failure']='delete';await click(byText('button','Delete session'));await waitFor(()=>document.querySelector('[role="alert"]'),'delete failed');assert(byText('button','Delete session'));assert.equal((await notesApi.listSessions()).length,1);delete http.defaults.headers.common['x-fixture-failure'];await click(byText('button','Delete session'));await waitFor(()=>document.body.textContent.includes('Session notes deleted.'),'deleted');assert.equal((await notesApi.listSessions()).length,0);});
 await check('Direct client-brief path uses actual URL rather than static placeholder',async()=>{pathname='/dashboard/clients/00000000-0000-4000-8000-000000000011';dom.reconfigure({url:'http://localhost:4920'+pathname});await mount('app/dashboard/clients/[[...clientId]]/ClientBriefContent.tsx');await waitFor(()=>document.body.textContent.includes('Fictional context for a UI regression check.'),'brief loaded');assert(document.querySelector('svg.chart'));});
 await check('Login form uses real client and does not include demo credentials',async()=>{pathname='/login';await mount('app/login/page.tsx');assert(!document.body.textContent.includes('OdePreview'));await change(document.querySelector('#email'),'portal-fixture@example.test');await change(document.querySelector('#password'),'wrong');await submit(document.querySelector('.login-form'));await waitFor(()=>document.querySelector('[role="alert"]'),'login rejected');await change(document.querySelector('#password'),'fixture-only');await submit(document.querySelector('.login-form'));await waitFor(()=>navigations.includes('/dashboard'),'login navigated');});
 await act(async()=>root.unmount());
 console.log(JSON.stringify({passed:results.length,checks:results},null,2));
 fs.mkdirSync(path.resolve(__dirname,'../tmp'),{recursive:true});
 fs.writeFileSync(path.resolve(__dirname,'../tmp/portal-ui-results.json'),JSON.stringify({passed:results.length,checks:results},null,2));
 dom.window.close();
})().catch(e=>{console.error(e);process.exitCode=1;if(root)root.unmount();dom.window.close();});
