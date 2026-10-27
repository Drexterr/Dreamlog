// Contract fixtures from docs/API_CONTRACT.md. Fictional data only.
const http = require('node:http');
const { randomUUID } = require('node:crypto');
const user = '00000000-0000-4000-8000-000000000010';
const client = '00000000-0000-4000-8000-000000000011';
const external = '00000000-0000-4000-8000-000000000012';
const now = '2026-10-20T09:00:00Z';
const therapist = {id:user,user_id:user,name:'Fixture Therapist',email:'portal-fixture@example.test',credentials:'Fixture only',plan:'trial',created_at:now};
http.createServer(async (req,res) => {
 res.setHeader('Access-Control-Allow-Origin','http://127.0.0.1:3092');
 res.setHeader('Access-Control-Allow-Headers','Authorization, Content-Type');
 res.setHeader('Access-Control-Allow-Methods','GET, POST, PATCH, DELETE, PUT, OPTIONS');
 if(req.method==='OPTIONS'){res.writeHead(204);res.end();return;}
 const url = new URL(req.url,'http://localhost');
 const p = url.pathname.replace(/^\/ode(?=\/)/,'');
 const answer = (code,body) => {res.writeHead(code,{'Content-Type':'application/json'});res.end(body===undefined?'':JSON.stringify(body));};
 if (p==='/waitlist' && req.method==='POST') {
   if (req.body?.email==='failure@example.test') return answer(503,{message:'Unavailable'});
   if (req.body?.email==='limited@example.test') return answer(429,{message:'Too many requests'});
   if (!req.body?.email?.includes('@')) return answer(400,{message:'Enter a valid email address.'});
   const emails = await pm.state.get('ode.waitlist') || [];
   if (!req.body.website && !emails.includes(req.body.email)) { emails.push(req.body.email); await pm.state.set('ode.waitlist',emails); }
   return answer(200,{message:"You're on the list. We'll email you when Ode launches."});
 }
 if (p==='/auth/login' && req.method==='POST') return req.body?.password==='fixture-only' ? answer(200,{token:'fictional-token',user:{id:user,name:therapist.name,email:therapist.email}}):answer(401,{error:'invalid credentials'});
 if (p==='/auth/register' && req.method==='POST') return answer(201,{token:'fictional-token',user:{id:user,name:req.body.name,email:req.body.email}});
 if(!req.headers.authorization?.startsWith('Bearer '))return answer(401,{error:'authentication required'});
 let state=await pm.state.get('ode.portal.fixture');
 if(!state)state={consent:false,clients:[{client_id:client,name:'Fixture Client',linked_at:now,last_entry_at:now,avg_mood_30d:61,entry_count:4}],external:[{id:external,therapist_id:user,name:'Fixture Notes Client',role:'client',archived:false,session_count:0,created_at:now,updated_at:now}],sessions:[]};
 const save=()=>pm.state.set('ode.portal.fixture',state);
 if(req.headers['x-fixture-failure']==='consent'&&p==='/therapists/consent')return answer(503,{error:'fixture unavailable'});
 if(req.headers['x-fixture-failure']==='save'&&req.method==='PATCH')return answer(503,{error:'fixture unavailable'});
 if(req.headers['x-fixture-failure']==='delete'&&req.method==='DELETE')return answer(503,{error:'fixture unavailable'});
 if(p==='/therapists/register'&&req.method==='POST')return answer(201,therapist);
 if(p==='/therapists/me')return answer(200,{therapist,client_consent_accepted:state.consent});
 if(p==='/therapists/consent'&&req.method==='POST'){state.consent=true;await save();return answer(200,{client_consent_accepted:true});}
 if(p==='/therapists/clients'&&req.method==='GET')return answer(200,{clients:state.clients});
 if(p==='/therapists/clients/link'&&req.method==='POST')return answer(200,{therapist_id:user,client_id:req.body.client_id,status:'pending'});
 if(p.endsWith('/brief'))return answer(200,{client_id:client,client_name:'Fixture Client',generated_at:now,brief:'Fictional context for a UI regression check.',top_emotions:['reflective'],mood_trend:'stable',avg_mood_7d:61,entry_count:4,recent_entries:[{date:now,summary:'A fictional journal summary.',mood_score:61,topics:['routine'],key_quote:''}]});
 if(p.startsWith('/therapists/clients/')&&req.method==='DELETE'){state.clients=[];await save();return answer(204);}
 if(p==='/therapists/external-clients'){
   if(req.method==='GET')return answer(200,{clients:state.external});
   if(!state.consent)return answer(403,{error:'consent required'});
   const c={id:randomUUID(),therapist_id:user,name:req.body.name,role:'client',archived:false,session_count:0,created_at:now,updated_at:now};state.external.push(c);await save();return answer(201,c);
 }
 if(p==='/therapists/sessions/presign')return answer(200,{upload_url:`http://${req.headers.host}/ode/__uploads/fixture`,image_key:'notes/fixture.jpg'});
 if(p==='/__uploads/fixture'&&req.method==='PUT')return answer(200,{});
 if(p==='/therapists/sessions'){
   if(req.method==='GET'){const id=url.searchParams.get('external_client_id');return answer(200,{sessions:state.sessions.filter(s=>!id||s.external_client_id===id)});}
   if(!state.consent)return answer(403,{error:'consent required'});
   const s={id:randomUUID(),therapist_id:user,external_client_id:req.body.external_client_id,session_date:now,status:'completed',bullets:req.body.bullets||['Fictional extracted note'],created_at:now,updated_at:now};state.sessions.unshift(s);await save();return answer(201,s);
 }
 if(p.startsWith('/therapists/sessions/')){
   const id=p.split('/')[3];const s=state.sessions.find(s=>s.id===id);if(!s)return answer(404,{error:'not found'});
   if(req.method==='DELETE'){state.sessions=state.sessions.filter(s=>s.id!==id);await save();return answer(204);}
   if(req.method==='PATCH'){s.bullets=req.body.bullets;await save();}
   if(p.endsWith('/summarize')){s.summary='Fictional summary from saved notes.';await save();}
   return answer(200,s);
 }
 return answer(501,{error:'Endpoint not modelled by this contract fixture'});
}).listen(process.env.PORT||4921);
