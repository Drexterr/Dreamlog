import fs from 'node:fs';
import path from 'node:path';
import crypto from 'node:crypto';

const root = path.resolve(import.meta.dirname, '../..');
const matrixPath = path.join(root, 'scripts/regression/backend-blackbox-matrix.json');
const sources = fs.readdirSync(path.join(root, 'backend/tests/blackbox')).filter(f => f.endsWith('_test.go')).sort();
function plannedCases() {
  const cases = [];
  for (const file of sources) {
    const source = fs.readFileSync(path.join(root, 'backend/tests/blackbox', file), 'utf8');
    let parent;
    for (const line of source.split(/\r?\n/)) {
      const fn = line.match(/^func (Test\w+)\(/);
      if (fn) parent = fn[1];
      const request = line.match(/h\.request\(t,\s*"([^"]+)"/);
      const custom = line.match(/t\.Run\("(security\.[^"]+)"/);
      const name = request?.[1] ?? custom?.[1];
      if (name) cases.push({ id: name, test: `${parent}/${name}`, source: `backend/tests/blackbox/${file}`, boundary: 'HTTP with real application, PostgreSQL and Redis; controlled AI/storage' });
    }
  }
  if (new Set(cases.map(c => c.id)).size !== cases.length) throw new Error('Duplicate scenario ID');
  return cases;
}

if (process.argv[2] === '--include-endpoint-gaps') {
  const matrix = JSON.parse(fs.readFileSync(matrixPath, 'utf8'));
  const summary = JSON.parse(fs.readFileSync(path.resolve(root, process.argv[3]), 'utf8'));
  const known = new Set(matrix.cases.map(c => c.id));
  const added = summary.endpointCoverage.remaining.flatMap(route => ['success', 'negative'].map(kind => ({
    id: `endpoint.${route.method}:${route.path}.${kind}`,
    description: `${route.method} ${route.path}: ${kind === 'success' ? 'authorized successful feature behavior and returned/persisted result' : 'validation, entitlement, ownership or dependency failure as applicable'}`,
    test: null, status: 'planned', boundary: 'HTTP endpoint scenario awaiting implementation'
  }))).filter(c => !known.has(c.id));
  if (!added.length) throw new Error('No new endpoint gaps to include');
  matrix.history ??= [];
  matrix.history.push({ version: matrix.version, total: matrix.cases.length });
  matrix.version++;
  matrix.cases.push(...added);
  matrix.changeReason = 'Endpoint inventory audit added positive and negative placeholders for all six untouched enabled endpoints. No earlier case removed.';
  fs.writeFileSync(matrixPath, JSON.stringify(matrix, null, 2) + '\n');
  console.log(JSON.stringify({ version: matrix.version, added: added.length, total: matrix.cases.length }));
} else if (process.argv[2] === '--extend') {
  const matrix = JSON.parse(fs.readFileSync(matrixPath, 'utf8'));
  const known = new Set(matrix.cases.map(c => c.id));
  const added = plannedCases().filter(c => !known.has(c.id));
  matrix.history ??= [];
  matrix.history.push({ version: matrix.version, total: matrix.cases.length });
  matrix.version++;
  matrix.cases.push(...added);
  matrix.changeReason = 'Added client-consent lifecycle and conversation ownership scenarios discovered during endpoint gap review; no cases removed.';
  fs.writeFileSync(matrixPath, JSON.stringify(matrix, null, 2) + '\n');
  console.log(JSON.stringify({ version: matrix.version, added: added.length, total: matrix.cases.length }));
} else if (process.argv[2] === '--freeze') {
  if (fs.existsSync(matrixPath)) throw new Error('Matrix already frozen; preserve version and explicitly add new scenarios');
  const pending = [
    ['external.supabase-es256-login', 'Real Supabase-issued session against the isolated backend'],
    ['external.supabase-key-rotation', 'JWKS key rotation and cache refresh through HTTP'],
    ['external.supabase-jwks-outage', 'JWKS endpoint outage with and without cached keys'],
    ['external.reference-audio-english', 'Reference speech upload and completed transcription through API'],
    ['external.reference-audio-hindi', 'Hindi reference transcription and language handling'],
    ['external.reference-audio-noise', 'Noisy/silent audio rejection and safe processing outcome'],
    ['external.transcription-provider-outage', 'Transcription provider timeout/retry visible through entry status'],
    ['external.ai-provider-timeout', 'AI timeout and recovery through conversation API'],
    ['external.ai-malformed-result', 'Malformed provider output and safe API failure'],
    ['external.note-ocr-end-to-end', 'Queued synthetic image reaches decrypted completed session through API'],
    ['external.note-ocr-failure', 'Invalid image and provider failure reach safe session state'],
    ['external.google-valid-receipt', 'Google sandbox purchase verification grants correct entitlement'],
    ['external.google-replayed-receipt', 'Google receipt replay cannot grant another account entitlement'],
    ['external.apple-valid-receipt', 'Apple sandbox purchase verification grants correct entitlement'],
    ['external.apple-invalid-receipt', 'Invalid/revoked Apple receipt fails without entitlement change'],
    ['external.storage-service-outage', 'Object storage outage is surfaced safely through API'],
    ['external.queue-service-outage', 'Redis outage during entry creation has documented recoverable outcome'],
    ['external.database-service-outage', 'Database outage returns safe errors and recovers'],
    ['external.concurrent-api-workload', 'Bounded concurrent API workload preserves ownership and consistency'],
    ['external.export-native-pdf-validation', 'Generated PDF has expected content and excludes foreign/crisis data'],
    ['external.therapy-voice-roundtrip', 'Reference voice turn reaches transcript, reply and playable voice output'],
    ['external.notification-delivery', 'Registered sandbox device receives only expected synthetic notification']
  ].map(([id, description]) => ({ id, description, test: null, status: 'planned', boundary: 'requires additional fixture or service configuration' }));
  const cases = [...plannedCases(), ...pending];
  const router = fs.readFileSync(path.join(root, 'backend/internal/handlers/router.go'), 'utf8');
  const prefixes = { r: '', auth: '', entries: '/entries', therapy: '/therapy' };
  const routes = [...router.matchAll(/\b(r|auth|entries|therapy)\.(GET|POST|PUT|DELETE|PATCH)\("([^"]*)"/g)].map(m => ({ method: m[2], path: prefixes[m[1]] + m[3], optional: m[3] === '/upload' }));
  const matrix = { version: 1, date: '2026-10-15', targetPercent: 80, frozenBeforeFirstRun: true, definition: 'Passed named API contract scenarios divided by every declared scenario, including planned external cases. Parent feature must pass. Authentication route sweep counts once.', sourceHash: crypto.createHash('sha256').update(sources.map(f => fs.readFileSync(path.join(root, 'backend/tests/blackbox', f))).join('')).digest('hex'), cases, routes };
  fs.writeFileSync(matrixPath, JSON.stringify(matrix, null, 2) + '\n');
  console.log(JSON.stringify({ frozenScenarios: cases.length, executable: cases.length - pending.length, planned: pending.length, routes: routes.length }));
} else {
  const input = process.argv[2];
  if (!input) throw new Error('Usage: node backend-blackbox-report.mjs --freeze | results.jsonl [summary.json]');
  const matrix = JSON.parse(fs.readFileSync(matrixPath, 'utf8'));
  const declared = new Set(matrix.cases.map(c => c.id));
  for (const c of plannedCases()) if (!declared.has(c.id)) throw new Error(`Unregistered scenario ${c.id}`);
  const events = fs.readFileSync(path.resolve(root, input), 'utf8').split(/\r?\n/).filter(Boolean).map(JSON.parse);
  const outcomes = new Map(events.filter(x => x.Test && ['pass', 'fail', 'skip'].includes(x.Action)).map(x => [x.Test, x.Action]));
  const cases = matrix.cases.map(c => {
    if (!c.test) return { ...c, result: 'planned' };
    const parent = c.test.split('/')[0];
    const own = outcomes.get(c.test);
    return { ...c, result: own === 'pass' && outcomes.get(parent) === 'pass' ? 'passed' : own === 'fail' ? 'failed' : own === 'pass' ? 'blocked-by-feature-failure' : own === 'skip' ? 'skipped' : 'not-run' };
  });
  const routeHits = new Set();
  for (const event of events) {
    const hit = event.Output?.match(/BLACKBOX_REQUEST (\S+) (\S+) (\S+) (\d+)/);
    if (hit && cases.find(c => c.id === hit[1])?.result === 'passed' && hit[3] !== 'unregistered') routeHits.add(`${hit[2]} ${hit[3]}`);
  }
  const passed = cases.filter(c => c.result === 'passed').length;
  const summary = { date: '2026-10-15', matrixVersion: matrix.version, passed, total: cases.length, percent: 100 * passed / cases.length, failed: cases.filter(c => c.result === 'failed').length, planned: cases.filter(c => c.result === 'planned').length, suitePassed: events.some(x => !x.Test && x.Action === 'pass') && !events.some(x => x.Action === 'fail'), endpointCoverage: { exercisedBeyondAuth: routeHits.size, inventoryTotal: matrix.routes.filter(r => !r.optional).length, remaining: matrix.routes.filter(r => !r.optional && !routeHits.has(`${r.method} ${r.path}`)) }, cases };
  const output = path.resolve(root, process.argv[3] ?? 'test-results/backend-blackbox-summary.json');
  fs.writeFileSync(output, JSON.stringify(summary, null, 2) + '\n');
  console.log(JSON.stringify({ passed: summary.passed, total: summary.total, percent: summary.percent, failed: summary.failed, planned: summary.planned, suitePassed: summary.suitePassed, endpoints: summary.endpointCoverage }, null, 2));
}
