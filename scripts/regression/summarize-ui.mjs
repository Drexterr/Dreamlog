import { readdir, readFile, writeFile } from 'node:fs/promises';
import path from 'node:path';
const directory = path.resolve(process.argv[2] || '');
if (!process.argv[2]) throw new Error('Supply a completed UI run directory');
const commands = [];
function selector(value) {
  if (!value || typeof value !== 'object') return null;
  if (value.idRegex || value.textRegex) return { id: value.idRegex || null, text: value.textRegex || null };
  for (const [key, child] of Object.entries(value)) {
    if (key === 'env') continue;
    const found = selector(child);
    if (found) return found;
  }
  return null;
}
async function scan(dir) {
  for (const entry of await readdir(dir, { withFileTypes: true })) {
    const file = path.join(dir, entry.name);
    if (entry.isDirectory()) { await scan(file); continue; }
    if (entry.name !== 'commands.json') continue;
    const records = JSON.parse(await readFile(file, 'utf8'));
    for (const record of records) {
      const kind = Object.keys(record.command || {})[0];
      if (!kind || ['defineVariablesCommand', 'applyConfigurationCommand'].includes(kind)) continue;
      const evaluated = record.metadata?.evaluatedCommand || record.command;
      const value = evaluated[kind] || record.command[kind];
      commands.push({ feature: path.basename(path.dirname(file)), kind, target: selector(value), status: record.metadata?.status || 'UNKNOWN', optional: Boolean(value?.optional || value?.selector?.optional), durationMs: record.metadata?.duration, evidence: path.relative(directory, file) });
    }
  }
}
await scan(directory);
const stats = {};
for (const command of commands) {
  const feature = stats[command.feature] ||= { mandatoryAssertionsPassed: 0, tapsCompleted: 0, failedCommands: 0, skippedCommands: 0 };
  if (command.kind === 'assertConditionCommand' && !command.optional && command.status === 'COMPLETED') feature.mandatoryAssertionsPassed++;
  if (command.kind === 'tapOnElement' && command.status === 'COMPLETED') feature.tapsCompleted++;
  if (['FAILED', 'ERROR'].includes(command.status)) feature.failedCommands++;
  if (command.status === 'SKIPPED') feature.skippedCommands++;
}
await writeFile(path.join(directory, 'command-results.json'), JSON.stringify({ stats, commands, note: 'A completed tap does not prove its outcome; mandatory assertions and per-flow reports supply outcome evidence. Optional and skipped checks are not functional passes.' }, null, 2));
await writeFile(path.join(directory, 'COMMAND_SUMMARY.md'), `# Device Command Results\n\n| Feature | Mandatory Assertions Passed | Taps Completed | Failed Commands | Skipped Commands |\n| --- | --- | --- | --- | --- |\n${Object.entries(stats).map(([name, s]) => `| ${name} | ${s.mandatoryAssertionsPassed} | ${s.tapsCompleted} | ${s.failedCommands} | ${s.skippedCommands} |`).join('\n')}\n\nComplete targets/statuses: command-results.json. Taps alone are not functional passes.\n`);
console.log(JSON.stringify(stats));
