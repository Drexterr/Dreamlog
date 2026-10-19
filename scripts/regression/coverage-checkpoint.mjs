import { readFile, writeFile, mkdir, rename } from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const keys = ['backendStatements', 'mobileStatements', 'mobileBranches', 'mobileFunctions'];
export function buildCheckpoint(feature, report, previous) {
  if (!/^[a-z0-9-]+$/.test(feature)) throw new Error('Feature must be a lowercase identifier');
  const coverage = {
    backendStatements: report.backendStatementCoverage,
    mobileStatements: report.mobileCoverage?.statements,
    mobileBranches: report.mobileCoverage?.branches,
    mobileFunctions: report.mobileCoverage?.functions,
  };
  for (const key of keys) {
    if (typeof coverage[key] !== 'number' || !Number.isFinite(coverage[key]) || coverage[key] < 0 || coverage[key] > 100) throw new Error(`Missing/invalid measured coverage: ${key}`);
  }
  const required = ['backend', 'backend-vet', 'mobile', 'mobile-types', 'portal-types'];
  const codeChecksPassed = required.every(name => report.results?.some(result => result.name === name && result.status === 'passed'));
  const deltaPercentagePoints = previous ? Object.fromEntries(keys.map(key => {
    const baseline = previous.coverage?.[key];
    return [key, typeof baseline === 'number' && Number.isFinite(baseline)
      ? Number((coverage[key] - baseline).toFixed(2)) : null];
  })) : null;
  return {
    feature, recordedAt: new Date().toISOString(), coverage, deltaPercentagePoints,
    codeChecksPassed, namedTestCounts: report.testCounts,
    milestones: Object.fromEntries([70, 80, 90].map(target => [target, codeChecksPassed && keys.every(key => coverage[key] >= target)])),
    scenarioCoverage: report.scenarioCoverage ?? null,
    scope: 'Whole-source code coverage checkpoint only; not feature/device/provider certification',
  };
}

if (process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../..');
  const reports = path.join(root, 'test-results');
  const source = path.resolve(process.argv[3] || '.');
  const relative = path.relative(reports, source);
  if (relative.startsWith('..') || path.isAbsolute(relative) || path.basename(source) !== 'results.json') throw new Error('Use a consolidated results.json inside test-results');
  const report = JSON.parse(await readFile(source, 'utf8'));
  const ledgerPath = path.join(reports, 'coverage-checkpoints.json');
  let ledger;
  try { ledger = JSON.parse(await readFile(ledgerPath, 'utf8')); } catch (error) {
    if (error.code !== 'ENOENT') throw error;
    ledger = { version: 1, checkpoints: [] };
  }
  const id = `${process.argv[2]}:${relative}`;
  if (ledger.checkpoints.some(checkpoint => checkpoint.id === id)) throw new Error('This feature/report checkpoint already exists');
  const checkpoint = { id, sourceReport: path.relative(root, source), ...buildCheckpoint(process.argv[2], report, ledger.checkpoints.at(-1)) };
  ledger.checkpoints.push(checkpoint);
  await mkdir(reports, { recursive: true });
  const temporary = ledgerPath + '.tmp';
  await writeFile(temporary, JSON.stringify(ledger, null, 2));
  await rename(temporary, ledgerPath);
  console.log(JSON.stringify(checkpoint, null, 2));
}
