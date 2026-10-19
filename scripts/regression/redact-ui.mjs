import { readdir, readFile, writeFile } from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

export async function redactTextArtifacts(directory, secret) {
  if (!secret) throw new Error('A nonempty secret is required for artifact redaction');
  const xml = secret.replaceAll('&', '&amp;').replaceAll('<', '&lt;').replaceAll('>', '&gt;').replaceAll('"', '&quot;').replaceAll("'", '&apos;');
  const variants = [...new Set([secret, JSON.stringify(secret).slice(1, -1), xml])];
  let filesChanged = 0;
  async function walk(dir) {
    for (const entry of await readdir(dir, { withFileTypes: true })) {
      if (entry.isSymbolicLink()) continue;
      const target = path.join(dir, entry.name);
      if (entry.isDirectory()) { await walk(target); continue; }
      if (!entry.isFile() || !['.json', '.xml', '.log', '.txt', '.yaml', '.yml', '.html'].includes(path.extname(entry.name))) continue;
      const original = await readFile(target, 'utf8');
      let redacted = original;
      for (const value of variants) redacted = redacted.replaceAll(value, '[REDACTED]');
      if (redacted !== original) { await writeFile(target, redacted); filesChanged++; }
    }
  }
  await walk(directory);
  return { filesChanged };
}

if (process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  const reports = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../../test-results');
  const target = path.resolve(process.argv[2] || '.');
  const relative = path.relative(reports, target);
  if (!relative || relative.startsWith('..') || path.isAbsolute(relative)) throw new Error('Redaction requires a run directory inside test-results');
  console.log(JSON.stringify(await redactTextArtifacts(target, process.env.ODE_QA_PASSWORD)));
}
