import fs from 'node:fs';
import path from 'node:path';
import { spawn } from 'node:child_process';
import { fileURLToPath } from 'node:url';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../..');
if (process.argv.includes('--help')) {
  console.log('Run with local PostgreSQL/Redis ready: node scripts/regression/run-backend-dual80.mjs');
  console.log('ODE_TEST_DATABASE_URL and ODE_TEST_REDIS_ADDR may override the isolated compose defaults.');
  process.exit(0);
}
const output = path.join(root, 'test-results', `backend-dual80-${new Date().toISOString().replaceAll(':', '-')}`);
fs.mkdirSync(output, { recursive: true });
const env = {
  ...process.env, GOMAXPROCS: '2',
  ODE_TEST_DATABASE_URL: process.env.ODE_TEST_DATABASE_URL || 'postgres://ode_test:regression-local-only@127.0.0.1:54331/ode_test?sslmode=disable',
  ODE_TEST_REDIS_ADDR: process.env.ODE_TEST_REDIS_ADDR || '127.0.0.1:63791'
};
async function run(name, command, args, cwd) {
  console.log(`Starting ${name}`);
  const out = fs.createWriteStream(path.join(output, `${name}.jsonl`));
  const errors = fs.createWriteStream(path.join(output, `${name}.stderr.log`));
  const code = await new Promise((resolve, reject) => {
    const child = spawn(command, args, { cwd, env, windowsHide: true, shell: false });
    child.stdout.pipe(out); child.stderr.pipe(errors);
    child.once('error', reject);
    child.once('close', async code => {
      await Promise.all([new Promise(r => out.end(r)), new Promise(r => errors.end(r))]);
      resolve(code);
    });
  });
  console.log(`${name}: ${code === 0 ? 'passed' : `exit ${code}`}`);
  return code;
}
const profile = path.join(output, 'backend.cover');
const testsExit = await run('tests', 'go', ['test', '-p', '1', '-tags=integration', './...', '-count=1', `-coverprofile=${profile}`, '-json'], path.join(root, 'backend'));
const vetExit = await run('vet', 'go', ['vet', '-p', '1', '-tags=integration', './...'], path.join(root, 'backend'));
const blackboxPath = path.join(output, 'blackbox-summary.json');
const reportExit = await run('report', process.execPath, [path.join(root, 'scripts/regression/backend-blackbox-report.mjs'), path.join(output, 'tests.jsonl'), blackboxPath], root);
let covered = 0, total = 0;
if (fs.existsSync(profile)) {
  const blocks = new Map();
  for (const line of fs.readFileSync(profile, 'utf8').split(/\r?\n/).slice(1)) {
    const m = line.match(/^(\S+)\s+(\d+)\s+(\d+)$/);
    if (!m) continue;
    const previous = blocks.get(m[1]);
    if (previous && previous.statements !== +m[2]) throw new Error(`Changed statement block ${m[1]}`);
    blocks.set(m[1], { statements: +m[2], hit: +m[3] > 0 || previous?.hit });
  }
  for (const block of blocks.values()) { total += block.statements; if (block.hit) covered += block.statements; }
}
const events = fs.readFileSync(path.join(output, 'tests.jsonl'), 'utf8').split(/\r?\n/).filter(Boolean).map(JSON.parse);
const blackbox = reportExit === 0 ? JSON.parse(fs.readFileSync(blackboxPath, 'utf8')) : null;
const summary = {
  testsExit, vetExit, reportExit,
  whitebox: { covered, total, percent: total ? 100 * covered / total : 0, validPassingCheckpoint: testsExit === 0 && vetExit === 0 },
  namedTests: { passed: events.filter(x => x.Test && x.Action === 'pass').length, failed: events.filter(x => x.Test && x.Action === 'fail').length, skipped: events.filter(x => x.Test && x.Action === 'skip').length },
  blackbox: blackbox && { passed: blackbox.passed, total: blackbox.total, percent: blackbox.percent, suitePassed: blackbox.suitePassed, endpointCoverage: blackbox.endpointCoverage },
  limitations: ['AI and storage use controlled boundaries in API tests; see planned external scenarios', 'Statement coverage does not measure branch coverage or clinical/AI quality', 'Race detection is a separate check']
};
summary.bothTargetsMet = testsExit === 0 && vetExit === 0 && summary.whitebox.percent >= 80 && Boolean(blackbox?.suitePassed && blackbox.percent >= 80);
fs.writeFileSync(path.join(output, 'summary.json'), JSON.stringify(summary, null, 2) + '\n');
console.log(JSON.stringify(summary, null, 2));
console.log(`Evidence: ${output}`);
process.exitCode = summary.bothTargetsMet ? 0 : 1;
