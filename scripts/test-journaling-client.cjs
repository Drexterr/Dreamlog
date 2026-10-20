// Runs the real typed Axios client over HTTP against the local Postman simulation.
const assert = require('node:assert/strict'), path = require('node:path'), fs = require('node:fs');
const root = path.resolve(__dirname, '..');
const ts = require(path.join(root, 'mobile/node_modules/typescript'));
const out = path.join(root, 'tmp/journaling-http-client.cjs');
fs.mkdirSync(path.dirname(out), { recursive: true });
process.env.EXPO_PUBLIC_API_URL = process.env.ODE_SIM_URL || 'http://127.0.0.1:4917/ode';
const compiled = ts.transpileModule(fs.readFileSync(path.join(root, 'mobile/src/api/client.ts'), 'utf8'), { compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2020, esModuleInterop: true } }).outputText;
fs.writeFileSync(out, compiled.replace('require("expo-secure-store")', 'require(' + JSON.stringify(path.join(root, 'postman/test-secure-store.cjs')) + ')'));
// Axios remains external so this check uses the app's installed Axios version.
process.env.NODE_PATH = [path.join(root, 'mobile/node_modules'), process.env.NODE_PATH].filter(Boolean).join(path.delimiter);
require('node:module').Module._initPaths();
require(path.join(root, 'mobile/node_modules/axios/dist/node/axios.cjs')).default.defaults.headers.common['x-mock-session'] = 'ode-journaling-http-check';
const { api } = require(out);
(async () => {
  const request = { content: 'Fictional journal for HTTP verification.', request_id: require('node:crypto').randomUUID(), mode: 'gratitude' };
  const first = await api.createTextEntry(request);
  const retry = await api.createTextEntry(request);
  assert.equal(first.id, retry.id);
  assert.equal(first.status, 'pending'); assert.equal(first.mode, 'gratitude'); assert.equal(first.duration_sec, 0);
  const loaded = await api.getEntry(first.id); assert.equal(loaded.transcript, request.content);
  await assert.rejects(api.createTextEntry({ ...request, content: 'Different journal' }), error => error.response?.status === 409);
  await assert.rejects(api.createTextEntry({ ...request, request_id: require('node:crypto').randomUUID(), content: '   ' }), error => error.response?.status === 400);
  await assert.rejects(api.getEntry(require('node:crypto').randomUUID()), error => error.response?.status === 404);
  console.log('Real API client over HTTP: create, read, idempotent retry, conflict, invalid input and missing entry passed.');
})().catch(error => { console.error(error.message); process.exitCode = 1; });
