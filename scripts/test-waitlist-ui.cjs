// Actual React component + fetch against the local Postman simulation.
const fs=require('node:fs'),Module=require('node:module'),assert=require('node:assert/strict');
const {JSDOM}=require('../mobile/node_modules/jsdom');
const ts=require('../therapist-portal/node_modules/typescript');
const dom=new JSDOM('<div id="root"></div><a href="#waitlist" data-open-waitlist>Join</a>',{url:'http://127.0.0.1:3092/'});
for(const k of ['window','document','HTMLElement','HTMLInputElement','Event','MouseEvent','FormData'])global[k]=dom.window[k];
Object.defineProperty(global,'navigator',{value:dom.window.navigator,configurable:true});
global.IS_REACT_ACT_ENVIRONMENT=true;
window.HTMLDialogElement.prototype.showModal=function(){this.open=true};
window.HTMLDialogElement.prototype.close=function(){this.open=false;this.dispatchEvent(new Event('close'))};
for(const ext of ['.tsx','.ts'])Module._extensions[ext]=(m,f)=>m._compile(ts.transpileModule(fs.readFileSync(f,'utf8'),{compilerOptions:{module:ts.ModuleKind.CommonJS,jsx:ts.JsxEmit.ReactJSX,esModuleInterop:true}}).outputText,f);
const React=require('../therapist-portal/node_modules/react');
const {createRoot}=require('../therapist-portal/node_modules/react-dom/client');
const {Simulate}=require('../therapist-portal/node_modules/react-dom/test-utils');
const {act}=React;
process.env.NEXT_PUBLIC_API_URL='http://127.0.0.1:4920/ode';
const Component=require('../therapist-portal/src/components/Waitlist.tsx').default;
async function until(fn){for(let i=0;i<100&&!fn();i++)await act(async()=>new Promise(r=>setTimeout(r,25)));assert(fn());}
(async()=>{
 let root;
 for(const [email,success,message]of [['failure@example.test',false,"couldn't save"],['limited@example.test',false,'Too many'],['hello@example.test',true,''],['hello@example.test',true,'']]){
  if(root)await act(async()=>root.unmount());
  root=createRoot(document.getElementById('root'));await act(async()=>root.render(React.createElement(Component)));
  await act(async()=>document.querySelector('a[data-open-waitlist]').click());assert(document.querySelector('dialog').open);
  document.querySelector('input[name=email]').value=email;
  await act(async()=>Simulate.submit(document.querySelector('form')));
  await until(()=>document.querySelector('[role=status]')||document.querySelector('[role=alert]')?.textContent);
  if(success)assert(document.querySelector('[role=status]').textContent.includes('email you'));
  else{assert(document.querySelector('[role=alert]').textContent.includes(message));assert.equal(document.querySelector('input[name=email]').value,email);assert(!document.querySelector('button[type=submit]').disabled);}
 }
 await act(async()=>root.unmount());dom.window.close();console.log('PASS: form opening, submission, retryable failures, rate limit, success and duplicate success over HTTP');
})().catch(e=>{console.error(e);process.exitCode=1});
