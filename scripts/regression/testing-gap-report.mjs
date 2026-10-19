import { readFile, readdir, writeFile, mkdir } from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../..');
const reportDir = path.resolve(root, process.argv[2] || 'test-results/2026-10-14T05-31-26.727Z');
const relative = path.relative(path.join(root, 'test-results'), reportDir);
if (relative.startsWith('..') || path.isAbsolute(relative)) throw new Error('Use a saved report inside test-results');
const manifest = JSON.parse(await readFile(path.join(root, 'scripts/regression/scenarios.json'), 'utf8'));
const codeReport = JSON.parse(await readFile(path.join(reportDir, 'results.json'), 'utf8'));
const flowFiles = (await readdir(path.join(root, 'maestro/flows'))).filter(name => /^\d{2}-.*\.yaml$/.test(name));
const passed = new Map();
for (const name of (await readdir(path.join(root, 'test-results'))).filter(name => /^phone-\d{4}-/.test(name)).sort()) {
  let results, actions;
  try {
    results = JSON.parse(await readFile(path.join(root, 'test-results', name, 'results.json'), 'utf8'));
    actions = JSON.parse(await readFile(path.join(root, 'test-results', name, 'command-results.json'), 'utf8'));
  } catch (error) { if (error.code === 'ENOENT') continue; throw error; }
  for (const result of results.filter(result => result.status === 'passed' && flowFiles.includes(result.feature + '.yaml'))) {
    const commands = actions.filter(action => action.feature === result.feature);
    passed.set(result.feature, {
      flow: result.feature, evidence: name,
      assertions: commands.filter(action => action.assertion && action.status === 'passed').length,
      taps: commands.filter(action => action.kind === 'tapOn' && action.status === 'passed').length,
    });
  }
}
const groups = [
  { name: 'Launch / onboarding / terms / navigation', ids: [1,2,3,4,5,6,7,13], needs: 'Idle Android phone; matching APK; fresh/returning guest fixtures; QA version/terms/API-outage controls.' },
  { name: 'Authentication / session / account switching', ids: [8,9,10,11,12], needs: 'Disposable second QA user and therapist accounts; QA email/OAuth accounts; token expiry/revocation and rate-limit fixtures; idle phone.' },
  { name: 'Recording / permission / interruption', ids: [14,15,16,17,18,19], needs: 'Idle phone; controlled speech/speaker setup; permission-dialog cases; 30-minute boundary window; headset/Bluetooth and interruption setup.' },
  { name: 'Upload / offline queue / workers', ids: [20,21,22,23,24,25,26], needs: 'Healthy isolated DB/Redis/storage; QA fault injection; offline/reconnect/device fixtures; approved deployment of the local completion-order fix.' },
  { name: 'Transcription / audio / provider failures', ids: [27,28,29,30,31], needs: 'Reference English/Hindi/mixed/accent/noise/silence/corrupt audio corpus; agreed word/entity error thresholds; live-provider budget and fault fixtures.' },
  { name: 'Reflection / context / follow-up', ids: [32,33,34,35,36], needs: 'Five-mode QA entries; second QA account for isolation; grounding/injection rubric; expert review where needed; ordering, retry and partial-persistence faults.' },
  { name: 'Therapy / voice / TTS / audio focus', ids: [37,38,39,40,41,42,43], needs: 'QA session entitlements; four-persona text/voice fixtures; one-hour expiry controls; TTS language fixtures; phone/speaker/headset/Bluetooth.' },
  { name: 'Crisis / safety / resources', ids: [44,45], needs: 'Approved safety corpus and reviewer; QA screener fault controls; test link handlers. Never dial real emergency resources.' },
  { name: 'History / search', ids: [46], needs: 'Larger multi-page QA history; second authorized QA user; device offline/error controls. Search-to-reflection/no-match phone checks already pass.' },
  { name: 'Mood / habits / notifications', ids: [47,48,49], needs: 'Seeded statistics/date/timezone fixtures; QA push project/device tokens; notification permission and stale/deleted-target controls.' },
  { name: 'Journeys / relationships / reviews', ids: [50,51], needs: 'Seeded journeys/progress, synthetic relationship data and reference reviews; scheduler clocks; provider and state-transition fault fixtures.' },
  { name: 'Sharing / consent / export', ids: [52,53], needs: 'Explicit authorization for QA share/export mutations; disposable therapist consent account; synthetic data; isolated expiry/lockout/ownership/native-export handlers.' },
  { name: 'Settings persistence / rollback', ids: [54], needs: 'Disposable QA preferences; idle phone; restart/account-switch cases; failed-save and rollback controls.' },
  { name: 'Billing / entitlement / restore', ids: [55,56], needs: 'Play internal-test build and license-tester account; sandbox products/receipts; QA entitlement fixtures and verification backend. No real purchases.' },
  { name: 'Therapist / clients / OCR / portal', ids: [57,58,59], needs: 'Dedicated therapist and client fixtures; synthetic OCR images and QA encryption keys; consent/role fixtures; Android plus browser automation.' },
  { name: 'Account deletion', ids: [60], needs: 'Explicit destructive-test approval; new disposable account, synthetic records and deletion policy; backend partial-failure controls. Never delete the primary tester account.' },
];
const groupFor = item => groups.find(group => group.ids.includes(Number(/^ODE-(\d+)/.exec(item.id)?.[1])));
if (manifest.cases.some(item => !groupFor(item))) throw new Error('Unmapped scenario group');
const incomplete = manifest.cases.filter(item => !['passed', 'complete'].includes(item.status));
const certifiedCount = manifest.cases.length - incomplete.length;
const certifiedPercent = certifiedCount / manifest.cases.length * 100;
const sum = field => [...passed.values()].reduce((total, item) => total + item[field], 0);
const pct = value => value.toFixed(2);
const cell = value => String(value).replaceAll('|', '\\|').replaceAll('\n', ' ');
const lines = [
  '# Ode Testing Coverage and Remaining Cases', '',
  'Snapshot: 2026-10-14. Generated from saved reports, not a new test run.', '',
  '## What the Percentages Mean', '',
  `- ADB flow-script execution: ${passed.size}/${flowFiles.length} = ${pct(passed.size / flowFiles.length * 100)}% have a passing phone run. ${sum('assertions')} mandatory assertion actions and ${sum('taps')} taps across those successful runs. Counts include repeated actions, not distinct scenarios/buttons.`,
  '- Runs span different APK builds. The mobile-only continuation adds matching-APK evidence, but this is not a complete latest-build release regression. Consult ODE_MOBILE_ADB_CHECKPOINT.md for the latest batch and partial/blocked flows.',
  '- Flow scripts have unequal breadth; adding or splitting scripts changes this denominator. Do not call this whole-app ADB coverage or all-button/state coverage.',
  `- Code coverage: backend statements ${pct(codeReport.backendStatementCoverage)}%; mobile statements ${pct(codeReport.mobileCoverage.statements)}%, branches ${pct(codeReport.mobileCoverage.branches)}%, functions ${pct(codeReport.mobileCoverage.functions)}%.`,
  '- These code percentages come from Jest/Go instrumentation. The current ADB APK does not collect source-code coverage, so an ADB pass alone does not raise them.',
  `- Full-scenario checklist: ${manifest.cases.length} outlines. ${certifiedCount} are signed off against their declared scope (${pct(certifiedPercent)}%); ${manifest.cases.filter(item => item.status === 'partial').length} have partial evidence mappings; ${manifest.cases.filter(item => item.status === 'not_executed').length} remain untracked/not audited in the manifest. This does NOT mean no unit/UI test has touched those areas. Overall app scenario certification remains incomplete.`,
  `- Each fully completed outline contributes ${pct(100 / manifest.cases.length)} percentage points to this fixed checklist. Completing 84/120 gives 70%; 96/120 gives 80%. These are scenario milestones, not code-coverage milestones.`,
  '- The impact below is checklist weight, not a forecast of statement/branch/function gains. Partial work earns no full-scenario completion credit until all required subcases/layers are validated. Counts must be audited before claiming a scenario percentage.', '',
  'Methods: U = Android UI/ADB; A = API/integration; J = Jest; E = audio/AI evaluation; H = hardware/provider setup or human review; browser = web UI automation.', '',
  '## Feature Completion Gaps', '',
  '| Feature | Unclosed outlines | Maximum checklist points | What is needed |',
  '| --- | ---: | ---: | --- |',
];
for (const group of groups) {
  const cases = incomplete.filter(item => groupFor(item) === group);
  lines.push(`| ${cell(group.name)} | ${cases.length} | ${pct(cases.length / manifest.cases.length * 100)} pp | ${cell(group.needs)} |`);
}
lines.push('', 'All rows are unclosed scope, not lists of confirmed app failures. Much of the mechanics already has passing unit/API/phone evidence.', '', '## Passing ADB Flows', '', '| Flow | Assertions | Taps | Saved evidence folder |', '| --- | ---: | ---: | --- |');
for (const item of [...passed.values()].sort((a, b) => a.flow.localeCompare(b.flow))) lines.push(`| ${item.flow} | ${item.assertions} | ${item.taps} | ${item.evidence} |`);
lines.push('', '## Every Unclosed Scenario', '', '| ID | Feature / polarity | Required scenario | Current evidence status | Remaining work / setup | Checklist points |', '| --- | --- | --- | --- | --- | ---: |');
for (const item of incomplete) {
  const group = groupFor(item);
  lines.push(`| ${item.id} | ${cell(item.feature)} / ${item.polarity} | ${cell(item.scenario)} | ${item.status === 'not_executed' ? 'Not fully mapped/audited' : cell(item.status)} | ${cell(item.remaining || 'Complete and map required ' + item.methods.join('/') + ' evidence. ' + group.needs)} | ${pct(100 / manifest.cases.length)} pp |`);
}

const mobile = JSON.parse(await readFile(path.join(reportDir, 'mobile-coverage/coverage-final.json'), 'utf8'));
let mobileTotal = 0;
const mobileGaps = Object.entries(mobile).map(([file, coverage]) => {
  const values = Object.values(coverage.s);
  mobileTotal += values.length;
  return { file: path.relative(root, file).replaceAll('\\', '/'), missing: values.filter(value => value === 0).length };
});
const backend = new Map(); let backendTotal = 0;
for (const line of (await readFile(path.join(reportDir, 'backend.cover'), 'utf8')).split(/\r?\n/)) {
  const match = /^(.+):\d+\.\d+,\d+\.\d+\s+(\d+)\s+(\d+)$/.exec(line);
  if (!match) continue;
  const statements = Number(match[2]); backendTotal += statements;
  if (Number(match[3]) === 0) backend.set(match[1], (backend.get(match[1]) || 0) + statements);
}
lines.push('', '## Code-Coverage Impact Bounds', '',
  'Exact gains from a proposed test cannot be known before running it. The following are arithmetic upper bounds: the gain if ALL currently unhit statement counters in that file become covered, with source/denominator unchanged. They are not promises, do not add to scenario percentages, and do not certify branches, hardware or AI quality.', '',
  '### Mobile: Largest Statement Gaps', '', '| File | Unhit statements | Maximum whole-mobile statement gain |', '| --- | ---: | ---: |');
for (const item of mobileGaps.filter(item => item.missing).sort((a,b) => b.missing-a.missing).slice(0, 15)) lines.push(`| ${cell(item.file)} | ${item.missing} | ${pct(item.missing / mobileTotal * 100)} pp |`);
lines.push('', '### Backend: Largest Statement Gaps', '', '| File | Unhit statements | Maximum whole-backend statement gain |', '| --- | ---: | ---: |');
for (const [file, missing] of [...backend].sort((a,b) => b[1]-a[1]).slice(0, 15)) lines.push(`| ${cell(file)} | ${missing} | ${pct(missing / backendTotal * 100)} pp |`);
lines.push('', '## Additional Boundaries', '',
  '- ADB covers Android, not iOS/Apple sign-in/Apple Health. iOS requires its own authorized device/runner and store sandbox.',
  '- Therapist web workflows require browser automation and disposable clients; no real client notes or personal phone files may be used.',
  '- ADB cannot reliably inject a reference WAV into a physical microphone. Downstream fixture uploads, real microphone capture and speaker/headset playback are separate evidence.',
  '- Keep the phone idle only during announced UI/hardware batches. Code/API/database work does not require exclusive phone use.',
  '- Current user authorization covers labelled tester entries/conversations. Share/export, store transactions and destructive deletion require explicit scoped authorization and suitable fixtures.',
  '- The backend completion-order fix is locally verified but not deployed. Do not deploy the production-named service without approval.', '');
const destination = path.join(root, 'docs/ODE_TESTING_COVERAGE_GAPS.md');
await mkdir(path.dirname(destination), { recursive: true });
await writeFile(destination, lines.join('\n'));
await writeFile(path.join(root, 'test-results/testing-gap-snapshot.json'), JSON.stringify({
  sourceReport: path.relative(root, reportDir), adb: { passingFlows: passed.size, totalFlows: flowFiles.length, percent: passed.size / flowFiles.length * 100, assertions: sum('assertions'), taps: sum('taps') },
  fullScenarioCoverage: certifiedPercent, certifiedOutlines: certifiedCount, incompleteOutlines: incomplete.length,
  featureGaps: groups.map(group => ({ name: group.name, unclosed: incomplete.filter(item => groupFor(item) === group).length, checklistPoints: incomplete.filter(item => groupFor(item) === group).length / manifest.cases.length * 100, needs: group.needs })),
  scenarios: incomplete,
}, null, 2));
console.log('Report: ' + destination);
console.log(JSON.stringify({ adbFlowPercent: passed.size / flowFiles.length * 100, assertions: sum('assertions'), taps: sum('taps'), unclosedOutlines: incomplete.length }));
