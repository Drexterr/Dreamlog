import { spawn } from 'node:child_process';
import { mkdir, writeFile, readFile } from 'node:fs/promises';
import { fileURLToPath } from 'node:url';
import path from 'node:path';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../..');
const stamp = new Date().toISOString().replaceAll(':', '-');
const out = path.join(root, 'test-results', stamp);
await mkdir(out, { recursive: true });
const results = [];
const npm = process.platform === 'win32' ? 'npm.cmd' : 'npm';
const serial = process.env.ODE_ADB_SERIAL;
const adbArgs = serial ? ['-s', serial] : [];
const appId = process.env.ODE_APP_ID || 'com.talktoode.app';

async function command(name, program, args, cwd = root, timeout = 180000) {
  if (program === npm && process.platform === 'win32') {
    args = [path.join(path.dirname(process.execPath), 'node_modules/npm/bin/npm-cli.js'), ...args];
    program = process.execPath;
  }
  const started = new Date().toISOString();
  let stdout = '', stderr = '';
  const execution = await new Promise((resolve) => {
    const child = spawn(program, args, {
      cwd, windowsHide: true,
      shell: false,
      env: process.env,
    });
    const timer = setTimeout(() => child.kill(), timeout);
    child.stdout.on('data', data => { stdout += data; });
    child.stderr.on('data', data => { stderr += data; });
    child.on('error', error => { clearTimeout(timer); resolve({ code: null, error: error.message }); });
    child.on('close', (code, signal) => { clearTimeout(timer); resolve({ code, signal }); });
  });
  const log = `${stdout}\n${stderr}`;
  await writeFile(path.join(out, `${name}.log`), log);
  const result = {
    name, started, finished: new Date().toISOString(),
    status: execution.code === 0 ? 'passed' : execution.code === null ? 'blocked' : 'failed',
    reason: execution.code === 0 ? 'Command completed successfully' : execution.error || `Exit ${execution.code}; see ${name}.log`,
    command: [program, ...args], ...execution,
  };
  results.push(result);
  console.log(`${result.status.toUpperCase()} ${name}`);
  return { ...result, stdout, stderr };
}

const integrationEnabled = Boolean(process.env.ODE_TEST_DATABASE_URL && process.env.ODE_TEST_REDIS_ADDR);
await command('backend', 'go', ['test', ...(integrationEnabled ? ['-tags=integration'] : []), './...', '-json', '-coverprofile=' + path.join(out, 'backend.cover')], path.join(root, 'backend'), 600000);
if (!integrationEnabled) results.push({ name: 'database-queue-integration', status: 'blocked', reason: 'Set ODE_TEST_DATABASE_URL and ODE_TEST_REDIS_ADDR for isolated real-service tests' });
await command('backend-vet', 'go', ['vet', './...'], path.join(root, 'backend'), 300000);
await command('mobile', npm, ['test', '--', '--runInBand', '--coverage', '--collectCoverageFrom=src/**/*.{ts,tsx}', '--collectCoverageFrom=app/**/*.{ts,tsx}', '--collectCoverageFrom=!**/__tests__/**', '--collectCoverageFrom=!src/types/**', '--collectCoverageFrom=!src/testIDs.ts', '--coverageDirectory=' + path.join(out, 'mobile-coverage'), '--json', '--outputFile=' + path.join(out, 'mobile.json')], path.join(root, 'mobile'), 300000);
await command('mobile-types', npm, ['exec', '--', 'tsc', '--noEmit'], path.join(root, 'mobile'), 180000);
await command('portal-types', npm, ['exec', '--', 'tsc', '--noEmit'], path.join(root, 'therapist-portal'), 180000);

const devices = await command('adb-devices', 'adb', ['devices', '-l']);
const connected = devices.stdout.split(/\r?\n/).filter(line => /^\S+\s+device\b/.test(line));
const targetSerial = serial || connected[0]?.trim().split(/\s+/)[0];
if (connected.length !== 1 && !serial) {
  results.push({ name: 'android-ui', status: 'blocked', reason: connected.length ? 'Multiple devices: set ODE_ADB_SERIAL' : 'No authorized Android device connected' });
} else if (targetSerial && !/^emulator-\d+$/.test(targetSerial)) {
  await command('android-installed', 'adb', ['-s', targetSerial, 'shell', 'pm', 'path', appId]);
  results.push({ name: 'android-functional', status: 'blocked', reason: 'Physical-phone privacy policy: generic screenshots/device-wide logs are disabled; use app-scoped phone checks' });
} else {
  const installed = await command('android-installed', 'adb', [...adbArgs, 'shell', 'pm', 'path', appId]);
  if (!installed.stdout.includes('package:')) {
    results.push({ name: 'android-ui', status: 'blocked', reason: `App ${appId} not installed; install intended build or set ODE_APP_ID` });
  } else {
    await command('android-build', 'adb', [...adbArgs, 'shell', 'dumpsys', 'package', appId]);
    await command('android-launch', 'adb', [...adbArgs, 'shell', 'monkey', '-p', appId, '-c', 'android.intent.category.LAUNCHER', '1']);
    await command('android-ui-dump', 'adb', [...adbArgs, 'shell', 'uiautomator', 'dump', '/sdcard/ode-regression.xml']);
    await command('android-ui-pull', 'adb', [...adbArgs, 'pull', '/sdcard/ode-regression.xml', path.join(out, 'android-ui.xml')]);
    const ui = await readFile(path.join(out, 'android-ui.xml'), 'utf8').catch(() => '');
    results.push({ name: 'android-app-visible', status: ui.includes(`package="${appId}"`) ? 'passed' : 'failed', reason: ui.includes(`package="${appId}"`) ? 'App package is present in foreground UI' : 'App UI missing after launch; inspect crash log and screenshot' });
    await command('android-screenshot', 'adb', [...adbArgs, 'shell', 'screencap', '-p', '/sdcard/ode-regression.png']);
    await command('android-screenshot-pull', 'adb', [...adbArgs, 'pull', '/sdcard/ode-regression.png', path.join(out, 'android.png')]);
    await command('android-crash-log', 'adb', [...adbArgs, 'logcat', '-d', '-b', 'crash', '-t', '200']);
    // Launch and evidence capture are infrastructure checks, not functional passes.
    results.push({ name: 'android-functional', status: 'blocked', reason: 'Feature flows and assertions must run; launch/screenshot alone do not prove functionality' });
  }
}

const cases = [];
try {
  const log = await readFile(path.join(out, 'backend.log'), 'utf8');
  for (const line of log.split(/\r?\n/)) {
    let event;
    try { event = JSON.parse(line); } catch { continue; }
    if (event.Test && ['pass', 'fail', 'skip'].includes(event.Action)) {
      cases.push({ feature: event.Package, name: event.Test, status: { pass: 'passed', fail: 'failed', skip: 'skipped' }[event.Action], reason: 'See backend.log for test output' });
    }
  }
} catch { /* Suite status already records missing output. */ }
try {
  const jest = JSON.parse(await readFile(path.join(out, 'mobile.json'), 'utf8'));
  for (const suite of jest.testResults) for (const test of suite.assertionResults) {
    cases.push({ feature: path.relative(root, suite.name), name: test.fullName, status: test.status === 'pending' ? 'skipped' : test.status, reason: test.failureMessages.join('\n') || 'Assertion passed' });
  }
} catch { /* Suite status already records missing output. */ }

const summary = { target: 90, scenarioCoverage: null, scenarioCoverageReason: 'Full scenario-to-test mapping and device/live-provider runs remain required', results, cases };
try {
  const lines = (await readFile(path.join(out, 'backend.cover'), 'utf8')).split(/\r?\n/).slice(1);
  let total = 0, covered = 0;
  for (const line of lines) {
    const fields = line.trim().split(/\s+/);
    if (fields.length !== 3) continue;
    total += Number(fields[1]);
    if (Number(fields[2]) > 0) covered += Number(fields[1]);
  }
  summary.backendStatementCoverage = total ? 100 * covered / total : null;
} catch { summary.backendStatementCoverage = null; }
try {
  const coverage = JSON.parse(await readFile(path.join(out, 'mobile-coverage/coverage-final.json'), 'utf8'));
  summary.mobileCoverage = {};
  for (const [kind, name] of [['s', 'statements'], ['b', 'branches'], ['f', 'functions']]) {
    let total = 0, covered = 0;
    for (const file of Object.values(coverage)) for (const value of Object.values(file[kind])) {
      for (const hits of Array.isArray(value) ? value : [value]) { total++; if (hits > 0) covered++; }
    }
    summary.mobileCoverage[name] = total ? 100 * covered / total : null;
  }
} catch { summary.mobileCoverage = null; }
const counts = cases.reduce((acc, test) => { acc[test.status] = (acc[test.status] || 0) + 1; return acc; }, {});
summary.testCounts = counts;
const featureRules = [
  ['Safety', /crisis|helpline/i],
  ['Authentication', /auth|login|register|jwt|jwks|postAuthRoute|guestStorage/i],
  ['Therapy', /therapy|persona|tts|winddown/i],
  ['Recording And Processing', /record|transcription|whisper|upload|offlineQueue|pipeline/i],
  ['Replies And Reflection', /conversation|followup|analyze|analysis|claude|prompt|context/i],
  ['Billing', /billing|iap|subscription|plan|quota/i],
  ['Therapist Workspace', /therapist|ocr|note/i],
  ['History And Insights', /mood|journey|relationship|person|review|chapter|insight|analytics/i],
  ['Privacy And Sharing', /share|export|delete|crypto|envelope/i],
  ['Notifications And Retention', /nudge|fcm|streak|flashback|checkin/i],
  ['Settings And Launch', /profile|user|preference|version|region/i],
];
summary.features = {};
for (const test of cases) {
  const name = featureRules.find(([, pattern]) => pattern.test(`${test.feature} ${test.name}`))?.[0] || 'Infrastructure';
  test.featureGroup = name;
  const group = summary.features[name] ||= {};
  group[test.status] = (group[test.status] || 0) + 1;
}
const scenarioManifest = JSON.parse(await readFile(path.join(root, 'scripts/regression/scenarios.json'), 'utf8'));
summary.scopedScenarioCount = scenarioManifest.cases.length;
summary.scenarios = scenarioManifest.cases;
results.push({ name: 'scenario-coverage-target', status: 'blocked', reason: '120 scenario outlines require validated test mappings and device/provider evidence; named unit passes are not scenario passes' });
const coverageChecks = [
  ['backend-statements', summary.backendStatementCoverage],
  ['mobile-statements', summary.mobileCoverage?.statements],
  ['mobile-branches', summary.mobileCoverage?.branches],
  ['mobile-functions', summary.mobileCoverage?.functions],
];
for (const [name, percent] of coverageChecks) {
  results.push({ name: `${name}-90-percent-target`, status: percent == null ? 'blocked' : percent >= 90 ? 'passed' : 'failed', reason: percent == null ? 'Coverage unavailable' : `Measured ${percent.toFixed(2)}%; required 90% across full implementation scope` });
}
await writeFile(path.join(out, 'results.json'), JSON.stringify(summary, null, 2));
const rows = results.map(r => `| ${r.name} | ${r.status} | ${r.reason.replaceAll('|', '/')} |`).join('\n');
const featureRows = Object.entries(summary.features).map(([feature, totals]) => `| ${feature} | ${totals.passed || 0} | ${totals.failed || 0} | ${totals.skipped || 0} |`).join('\n');
await writeFile(path.join(out, 'SUMMARY.md'), `# Ode Regression Run\n\n${stamp}\n\nScenario coverage: not yet measured across ${summary.scopedScenarioCount} scoped scenarios. Target: 90%.\n\nBackend statement coverage: ${summary.backendStatementCoverage?.toFixed(2) ?? 'unavailable'}%.\n\nFull mobile source coverage: ${JSON.stringify(summary.mobileCoverage)}.\n\nTest outcomes: ${JSON.stringify(counts)}. Counts include named Go subtests.\n\n| Feature Group | Passed | Failed | Skipped |\n| --- | --- | --- | --- |\n${featureRows}\n\nFeature grouping organizes named checks; it does not imply end-to-end feature coverage.\n\n| Check | Status | Reason |\n| --- | --- | --- |\n${rows}\n\nPer-test results and unexecuted full scenarios: results.json. Provider mocks do not certify live speech or reply quality.\n`);
console.log(`Report: ${path.join(out, 'SUMMARY.md')}`);
process.exitCode = results.some(r => r.status === 'failed') ? 1 : results.some(r => r.status === 'blocked') ? 2 : 0;
