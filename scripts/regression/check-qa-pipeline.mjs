import { readFile, writeFile, mkdir } from 'node:fs/promises';
import { createRequire } from 'node:module';
import { fileURLToPath } from 'node:url';
import path from 'node:path';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../..');
if (!process.argv.includes('--allow-qa-mutations')) throw new Error('Explicit QA mutation flag required');
const require = createRequire(import.meta.url);
const config = require(path.join(root, 'mobile/node_modules/dotenv')).parse(await readFile(path.join(root, 'mobile/.env')));
const fixture = require(path.join(root, 'mobile/node_modules/yaml')).parseAllDocuments(await readFile(path.join(root, 'maestro/flows/03-authentication.yaml'), 'utf8'))[0].toJSON().env;
const email = process.env.ODE_QA_EMAIL || fixture.EMAIL_EXISTING;
if (email !== 'bharatbanthia2207+tester@gmail.com') throw new Error('Only the authorized tester account is permitted');
const backend = new URL(config.EXPO_PUBLIC_API_URL), auth = new URL(config.EXPO_PUBLIC_SUPABASE_URL);
if (backend.protocol !== 'https:' || auth.protocol !== 'https:' || !auth.hostname.endsWith('.supabase.co')) throw new Error('Configured QA HTTPS endpoints required');
const audio = await readFile(path.join(root, 'test-results/audio-fixtures/qa-english-processing.m4a'));
const metadata = JSON.parse((await readFile(path.join(root, 'test-results/audio-fixtures/qa-english-processing.json'), 'utf8')).replace(/^\uFEFF/, ''));
if (!metadata.synthetic || audio.length > 2_000_000 || !(metadata.durationSec > 0 && metadata.durationSec < 120)) throw new Error('Bounded synthetic fixture required');
const report = { scope: 'One authorized synthetic QA entry and at most three follow-up replies; not phone microphone or comprehensive AI quality certification', checks: [], entryId: null };
const resume = process.argv.find(arg => arg.startsWith('--resume-entry='))?.slice('--resume-entry='.length);
if (resume && !/^[0-9a-f-]{36}$/.test(resume)) throw new Error('Invalid QA entry ID');
const output = path.join(root, 'test-results', 'qa-pipeline-' + new Date().toISOString().replaceAll(':', '-') + '.json');
let token;
async function save() { await mkdir(path.dirname(output), { recursive: true }); await writeFile(output, JSON.stringify(report, null, 2)); }
async function check(name, passed, fields = {}) {
  report.checks.push({ name, status: passed ? 'passed' : 'failed', ...fields }); await save();
  console.log(`${passed ? 'PASSED' : 'FAILED'} ${name}`);
  return passed;
}
async function request(endpoint, method = 'GET', body) {
  const response = await fetch(new URL(endpoint, backend), { method,
    headers: { Authorization: `Bearer ${token}`, ...(body ? { 'Content-Type': 'application/json' } : {}) },
    ...(body ? { body: JSON.stringify(body) } : {}), signal: AbortSignal.timeout(60000) });
  return { status: response.status, data: await response.json().catch(() => null) };
}
try {
  const login = await fetch(new URL('/auth/v1/token?grant_type=password', auth), { method: 'POST',
    headers: { apikey: config.EXPO_PUBLIC_SUPABASE_ANON_KEY, 'Content-Type': 'application/json' },
    body: JSON.stringify({ email, password: process.env.ODE_QA_PASSWORD || fixture.PASSWORD }), signal: AbortSignal.timeout(20000) });
  const session = await login.json();
  if (!await check('qa-sign-in', login.status === 200 && session.user?.email === email && typeof session.access_token === 'string')) throw new Error('QA login prerequisite failed');
  token = session.access_token;
  const identity = await request('/me');
  if (!await check('qa-backend-identity', identity.status === 200 && identity.data?.email === email)) throw new Error('QA identity prerequisite failed');
  const invalid = await request('/entries', 'POST', {});
  await check('empty-entry-rejected', invalid.status === 400, { httpStatus: invalid.status });
  if (!resume) {
  const presign = await request('/entries/presign', 'POST');
  if (!await check('audio-presign', presign.status === 200 && typeof presign.data?.audio_key === 'string' && typeof presign.data?.upload_url === 'string')) throw new Error('Presign prerequisite failed');
  const storage = new URL(presign.data.upload_url);
  if (storage.protocol !== 'https:' || storage.username || storage.password) throw new Error('Unsafe upload destination');
  const uploaded = await fetch(storage, { method: 'PUT', headers: { 'Content-Type': 'audio/mp4' }, body: audio, signal: AbortSignal.timeout(60000) });
  if (!await check('synthetic-audio-upload', uploaded.ok, { httpStatus: uploaded.status, bytes: audio.length })) throw new Error('Storage upload prerequisite failed');
  const created = await request('/entries', 'POST', { audio_key: presign.data.audio_key, audio_size_bytes: audio.length, duration_sec: metadata.durationSec, mode: 'processing' });
  if (!await check('qa-entry-created', created.status === 201 && /^[0-9a-f-]{36}$/.test(created.data?.id || ''), { httpStatus: created.status })) throw new Error('Entry creation prerequisite failed');
  report.entryId = created.data.id; await save();
  } else { report.entryId = resume; await save(); }
  let entry;
  const deadline = Date.now() + 240000;
  do {
    const current = await request('/entries/' + report.entryId);
    if (current.status !== 200) throw new Error('Entry polling request failed');
    entry = current.data;
    if (entry.user_id !== identity.data.id) throw new Error('QA entry ownership prerequisite failed');
    if (['completed', 'failed'].includes(entry.status)) break;
    await new Promise(resolve => setTimeout(resolve, 5000));
  } while (Date.now() < deadline);
  if (!await check('live-processing-completed', entry?.status === 'completed', { terminalStatus: entry?.status, retryCount: entry?.retry_count, providerErrorPresent: Boolean(entry?.error_msg) })) throw new Error('Live worker/provider processing did not complete');
  const transcript = (entry.transcript || '').toLowerCase();
  const anchors = ['work', 'deadline', 'walk', 'help'];
  await check('transcription-content-smoke', anchors.every(word => transcript.includes(word)), { matchedAnchorCount: anchors.filter(word => transcript.includes(word)).length, totalAnchorCount: anchors.length, transcriptLength: transcript.length });
  let analysis = await request(`/entries/${report.entryId}/analysis`);
  const analysisDeadline = Date.now() + 120000;
  while (analysis.status === 404 && Date.now() < analysisDeadline) {
    await new Promise(resolve => setTimeout(resolve, 5000));
    analysis = await request(`/entries/${report.entryId}/analysis`);
  }
  if (!await check('reflection-schema-and-safety-smoke', analysis.status === 200 && Number.isInteger(analysis.data?.mood_score) && analysis.data.mood_score >= 1 && analysis.data.mood_score <= 100 && Boolean(analysis.data?.summary?.trim()) && Boolean(analysis.data?.reflection?.trim()) && analysis.data.is_crisis === false, { httpStatus: analysis.status })) throw new Error('Analysis prerequisite failed');
  const conversation = await request(`/entries/${report.entryId}/conversation`, 'POST');
  if (!await check('qa-conversation-created', conversation.status === 200 && typeof conversation.data?.id === 'string' && conversation.data.entry_id === report.entryId)) throw new Error('Conversation prerequisite failed');
  const route = `/conversations/${conversation.data.id}/messages`;
  const empty = await request(route, 'POST', { content: '' });
  await check('empty-reply-rejected', empty.status === 400, { httpStatus: empty.status });
  const messages = [
    'Automated QA test: what is one practical step for managing the deadline in this synthetic journal?',
    'Automated QA test: I can divide the task into smaller steps. How can I ask a colleague for help?',
    'Automated QA test: summarize a small action plan using only this synthetic conversation.',
  ];
  for (let index = 0; index < messages.length; index++) {
    const reply = await request(route, 'POST', { content: messages[index] });
    const list = reply.data?.messages || [];
    if (!await check(`reply-turn-${index + 1}`, reply.status === 200 && reply.data?.turn_count === index + 1 && list.some(item => item.role === 'user' && item.content === messages[index]) && list.at(-1)?.role === 'assistant' && Boolean(list.at(-1)?.content?.trim()), { httpStatus: reply.status })) throw new Error('Reply turn failed');
  }
  const fourth = await request(route, 'POST', { content: 'Automated QA test: this fourth turn must be rejected.' });
  await check('fourth-reply-turn-rejected', fourth.status === 409, { httpStatus: fourth.status });
} catch (error) {
  report.checks.push({ name: 'remaining-pipeline', status: 'blocked', reason: error.name === 'TimeoutError' ? 'Network/provider timeout; no secrets or response bodies retained' : 'A preceding prerequisite or live pipeline stage did not complete; see stage results' });
} finally {
  token = undefined; await save(); console.log('Report: ' + output);
  process.exitCode = report.checks.some(item => item.status === 'failed') ? 1 : report.checks.some(item => item.status === 'blocked') ? 2 : 0;
}
