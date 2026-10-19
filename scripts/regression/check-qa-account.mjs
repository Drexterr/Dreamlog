import { readFile, mkdir, writeFile } from 'node:fs/promises';
import { createRequire } from 'node:module';
import { fileURLToPath } from 'node:url';
import path from 'node:path';
import { randomUUID } from 'node:crypto';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../..');
const require = createRequire(import.meta.url);
const dotenv = require(path.join(root, 'mobile/node_modules/dotenv'));
const yaml = require(path.join(root, 'mobile/node_modules/yaml'));
const config = dotenv.parse(await readFile(path.join(root, 'mobile/.env')));
const fixture = yaml.parseAllDocuments(await readFile(path.join(root, 'maestro/flows/03-authentication.yaml'), 'utf8'))[0].toJSON().env;
const email = process.env.ODE_QA_EMAIL || fixture.EMAIL_EXISTING;
const password = process.env.ODE_QA_PASSWORD || fixture.PASSWORD;
const backend = new URL(config.EXPO_PUBLIC_API_URL);
const auth = new URL(config.EXPO_PUBLIC_SUPABASE_URL);
if (backend.protocol !== 'https:' || auth.protocol !== 'https:' || !auth.hostname.endsWith('.supabase.co')) throw new Error('QA check requires configured HTTPS backend and Supabase hosts');
if (!email || !password || !config.EXPO_PUBLIC_SUPABASE_ANON_KEY) throw new Error('QA credential/configuration is missing');
const checks = [];
let qaUserId;

async function get(name, endpoint, token, validate) {
  try {
    const response = await fetch(new URL(endpoint, backend), {
      headers: token ? { Authorization: `Bearer ${token}` } : {}, signal: AbortSignal.timeout(20000),
    });
    const payload = await response.json().catch(() => null);
    const passed = validate(response.status, payload);
    checks.push({ name, status: passed ? 'passed' : 'failed', httpStatus: response.status, reason: passed ? 'Expected status and response contract matched' : 'Unexpected status or response contract' });
  } catch {
    checks.push({ name, status: 'failed', reason: 'Network or timeout failure' });
  }
}

await get('backend-health', '/health', null, (status, data) => status === 200 && data?.status === 'ok');
await get('unauthenticated-profile-denied', '/me', null, status => status === 401);
let token;
try {
  const response = await fetch(new URL('/auth/v1/token?grant_type=password', auth), {
    method: 'POST', headers: { apikey: config.EXPO_PUBLIC_SUPABASE_ANON_KEY, 'Content-Type': 'application/json' },
    body: JSON.stringify({ email, password }), signal: AbortSignal.timeout(20000),
  });
  const payload = await response.json().catch(() => null);
  const passed = response.status === 200 && typeof payload?.access_token === 'string' && payload?.user?.email?.toLowerCase() === email.toLowerCase();
  if (passed) token = payload.access_token;
  checks.push({ name: 'qa-provider-sign-in', status: passed ? 'passed' : 'failed', httpStatus: response.status, reason: passed ? 'Configured QA account authenticated' : 'Configured QA credentials rejected or provider response invalid' });
} catch {
  checks.push({ name: 'qa-provider-sign-in', status: 'failed', reason: 'Network or timeout failure' });
}
if (token) {
  await get('qa-backend-identity', '/me', token, (status, data) => {
    const valid = status === 200 && data?.email?.toLowerCase() === email.toLowerCase() && typeof data.id === 'string';
    if (valid) qaUserId = data.id;
    return valid;
  });
  await get('qa-entry-list-contract', '/entries?page=1&page_size=1', token, (status, data) => status === 200 && Array.isArray(data?.entries) && typeof data?.total === 'number');
  await get('qa-missing-entry-denied', '/entries/00000000-0000-4000-8000-000000000000', token, status => status === 404);
  await get('qa-billing-read-contract', '/billing/plan', token, (status, data) => status === 200 && ['free', 'plus', 'pro', 'b2b'].includes(data?.plan));
  if (process.argv.includes('--history') && qaUserId) {
    const historyFixture = yaml.parseAllDocuments(await readFile(path.join(root, 'maestro/flows/20-qa-history-search.yaml'), 'utf8'))[0].toJSON().env;
    const qaEntryId = historyFixture.QA_ENTRY_ID;
    if (!/^[0-9a-f-]{36}$/.test(qaEntryId)) throw new Error('Invalid QA history fixture ID');
    const owned = data => Array.isArray(data?.entries) && data.entries.every(entry => entry.user_id === qaUserId);
    const query = value => '/entries/search?' + new URLSearchParams({ q: value, limit: '20' });
    await get('history-pagination-contract', '/timeline?page=1&page_size=1', token, (status, data) => status === 200 && Array.isArray(data?.entries) && data.entries.length <= 1 && data.entries.every(item => item.entry?.user_id === qaUserId) && typeof data.total === 'number');
    await get('search-known-synthetic-entry', query('synthetic test content'), token, (status, data) => status === 200 && owned(data) && data.entries.some(entry => entry.id === qaEntryId));
    await get('search-no-match', query('qa' + randomUUID().replaceAll('-', '')), token, (status, data) => status === 200 && owned(data) && data.entries.length === 0);
    await get('search-unicode-contract', query('\u0928\u092e\u0938\u094d\u0924\u0947'), token, (status, data) => status === 200 && owned(data));
    await get('search-empty-query-rejected', '/entries/search', token, status => status === 400);
    await get('search-whitespace-query-rejected', query('   '), token, status => status === 400);
    await get('history-invalid-id-rejected', '/entries/not-a-uuid', token, status => status === 400);
    await get('history-synthetic-entry-owned', '/entries/' + qaEntryId, token, (status, data) => status === 200 && data?.id === qaEntryId && data.user_id === qaUserId);
  }
} else {
  checks.push({ name: 'qa-authenticated-api-checks', status: 'blocked', reason: 'No verified QA provider session; no authenticated requests made' });
}
token = undefined;
const out = path.join(root, 'test-results', `qa-account-${new Date().toISOString().replaceAll(':', '-')}.json`);
await mkdir(path.dirname(out), { recursive: true });
await writeFile(out, JSON.stringify({ scope: 'Read-only QA API/account checks, not device sign-in or provider-quality certification', checks }, null, 2));
for (const check of checks) console.log(`${check.status.toUpperCase()} ${check.name}${check.httpStatus ? ` HTTP ${check.httpStatus}` : ''}`);
console.log(`Report: ${out}`);
process.exitCode = checks.some(check => check.status === 'failed') ? 1 : checks.some(check => check.status === 'blocked') ? 2 : 0;
