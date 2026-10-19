import { readFile, writeFile, access } from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../..');
const output = path.join(root, 'scripts/regression/scenarios.json');
try { await access(output); throw new Error('Manifest exists; preserve its mappings and edit it explicitly.'); }
catch (error) { if (error.code !== 'ENOENT') throw error; }
const source = await readFile(path.join(root, 'docs/ODE_REGRESSION_ASSESSMENT.md'), 'utf8');
const cases = [];
for (const line of source.split(/\r?\n/)) {
  if (!/^\| \d{2} \|/.test(line)) continue;
  const [, id, feature, positive, negative, methods] = line.split('|').map(cell => cell.trim());
  for (const [suffix, scenario] of [['P', positive], ['N', negative]]) {
    cases.push({ id: `ODE-${id}${suffix}`, feature, polarity: suffix === 'P' ? 'positive' : 'negative', scenario, methods: methods.split('/'), evidence: [], status: 'not_executed' });
  }
}
if (cases.length !== 120 || new Set(cases.map(c => c.id)).size !== 120) throw new Error('Expected exactly 120 unique baseline scenarios');
await writeFile(output, JSON.stringify({ targetPercent: 90, coverageDefinition: 'Fully validated scoped scenarios / all scoped scenarios. Unit/provider/device evidence must not be substituted for one another.', cases }, null, 2) + '\n');
console.log(`Created ${cases.length} scenario records: ${output}`);
