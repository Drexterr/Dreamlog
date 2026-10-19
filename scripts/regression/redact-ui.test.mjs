import test from 'node:test';
import assert from 'node:assert/strict';
import { mkdtemp, readFile, writeFile, rm, realpath } from 'node:fs/promises';
import path from 'node:path';
import os from 'node:os';
import { redactTextArtifacts } from './redact-ui.mjs';

test('redacts plain, JSON-escaped and XML-escaped secrets without changing binary artifacts', async () => {
  const tempRoot = await realpath(os.tmpdir());
  const dir = await mkdtemp(path.join(tempRoot, 'ode-redaction-'));
  try {
    const secret = 'Fake&"QA!';
    await writeFile(path.join(dir, 'run.log'), `input: ${secret}`);
    await writeFile(path.join(dir, 'commands.json'), JSON.stringify({ password: secret }));
    await writeFile(path.join(dir, 'report.xml'), '<value>Fake&amp;&quot;QA!</value>');
    await writeFile(path.join(dir, 'image.png'), Buffer.from(secret));
    const result = await redactTextArtifacts(dir, secret);
    assert.equal(result.filesChanged, 3);
    assert.equal(await readFile(path.join(dir, 'run.log'), 'utf8'), 'input: [REDACTED]');
    assert.equal(JSON.parse(await readFile(path.join(dir, 'commands.json'), 'utf8')).password, '[REDACTED]');
    assert.equal(await readFile(path.join(dir, 'report.xml'), 'utf8'), '<value>[REDACTED]</value>');
    assert.equal(await readFile(path.join(dir, 'image.png'), 'utf8'), secret);
    assert.equal((await redactTextArtifacts(dir, secret)).filesChanged, 0);
  } finally {
    const resolved = await realpath(dir);
    assert.equal(path.dirname(resolved), tempRoot);
    assert.ok(path.basename(resolved).startsWith('ode-redaction-'));
    await rm(resolved, { recursive: true, force: true });
  }
});

test('rejects empty secrets', async () => {
  await assert.rejects(redactTextArtifacts('.', ''), /nonempty secret/);
});
