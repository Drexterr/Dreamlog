// Focused contract mock: no real journal data, AI calls, billing or accounts.
const http = require('node:http');
const { randomUUID } = require('node:crypto');
const modes = ['processing', 'rant', 'gratitude', 'decision', 'dream'];
http.createServer(async (req, res) => {
  const pathname = new URL(req.url, 'http://localhost').pathname.replace(/^\/ode(?=\/)/, '');
  const answer = (code, body) => { res.writeHead(code, { 'Content-Type': 'application/json' }); res.end(JSON.stringify(body)); };
  if (!req.headers.authorization?.startsWith('Bearer ')) return answer(401, { error: 'authentication required' });
  const entries = await pm.state.get('ode.journals') || {};
  if (req.method === 'POST' && pathname === '/entries/text') {
    const body = req.body || {};
    const content = typeof body.content === 'string' ? body.content.trim() : '';
    const mode = body.mode || 'processing';
    if (!content || [...content].length > 20000 || !/^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i.test(body.request_id || '') || !modes.includes(mode)) return answer(400, { error: 'invalid typed journal request' });
    const existing = entries[body.request_id];
    if (existing) return existing.transcript === content && existing.mode === mode ? answer(201, existing) : answer(409, { error: 'request_id was already used for a different journal' });
    const id = randomUUID(), now = new Date().toISOString();
    const entry = { id, user_id: '00000000-0000-4000-8000-000000000001', audio_key: 'text/fixture/' + body.request_id, audio_size_bytes: 0, duration_sec: 0, status: 'pending', mode, transcript: content, retry_count: 0, created_at: now, updated_at: now };
    entries[body.request_id] = entry; await pm.state.set('ode.journals', entries);
    return answer(201, entry);
  }
  if (req.method === 'GET' && /^\/entries\/[0-9a-f-]+$/.test(pathname)) {
    const entry = Object.values(entries).find(item => item.id === pathname.split('/').pop());
    return entry ? answer(200, entry) : answer(404, { error: 'entry not found' });
  }
  return answer(501, { error: 'Endpoint not modeled by this focused contract mock' });
}).listen(process.env.PORT || 4918);
