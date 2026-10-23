#!/usr/bin/env node
/**
 * N14: pins the portal's Content-Security-Policy script-src to the exact inline
 * scripts of the static export instead of 'unsafe-inline'.
 *
 * Next's static export inlines its bootstrap and RSC payload as <script> tags.
 * This runs after `next build`, hashes every inline script in out/**\/*.html
 * (sha256, the CSP hash-source format) and rewrites the script-src directive
 * of the CSP header in firebase.json:
 *
 *   script-src 'self' 'sha256-…' 'sha256-…'
 *
 * Injected inline script (XSS) no longer runs: its hash isn't listed.
 *
 *   node scripts/csp-hashes.cjs          update firebase.json
 *   node scripts/csp-hashes.cjs --check  exit 1 if firebase.json is stale
 */
const fs = require('fs');
const path = require('path');
const crypto = require('crypto');

const root = path.join(__dirname, '..');
const outDir = path.join(root, 'out');
const firebaseFile = path.join(root, 'firebase.json');

function htmlFiles(dir) {
  return fs.readdirSync(dir, { withFileTypes: true }).flatMap((e) => {
    const p = path.join(dir, e.name);
    if (e.isDirectory()) return htmlFiles(p);
    return e.name.endsWith('.html') ? [p] : [];
  });
}

/** sha256 hash-sources for every inline (src-less) script, sorted and unique. */
function inlineScriptHashes(files) {
  const hashes = new Set();
  const re = /<script\b([^>]*)>([\s\S]*?)<\/script>/gi;
  for (const file of files) {
    const html = fs.readFileSync(file, 'utf8');
    let m;
    while ((m = re.exec(html))) {
      if (/\bsrc\s*=/i.test(m[1]) || m[2] === '') continue;
      hashes.add(`'sha256-${crypto.createHash('sha256').update(m[2], 'utf8').digest('base64')}'`);
    }
  }
  return [...hashes].sort();
}

function withScriptSrc(csp, hashes) {
  const directives = csp.split(';').map((d) => d.trim()).filter(Boolean);
  const scriptSrc = ["script-src 'self'", ...hashes].join(' ');
  const i = directives.findIndex((d) => d.startsWith('script-src'));
  if (i >= 0) directives[i] = scriptSrc;
  else directives.push(scriptSrc);
  return directives.join('; ');
}

function main() {
  if (!fs.existsSync(outDir)) {
    console.error('csp-hashes: out/ not found - run `next build` first');
    process.exit(1);
  }
  const files = htmlFiles(outDir);
  const hashes = inlineScriptHashes(files);
  if (hashes.length === 0) {
    console.error('csp-hashes: no inline scripts found - refusing to write an empty allowlist');
    process.exit(1);
  }

  const config = JSON.parse(fs.readFileSync(firebaseFile, 'utf8'));
  let found = 0;
  let stale = false;
  for (const rule of config.hosting.headers) {
    for (const header of rule.headers) {
      if (header.key !== 'Content-Security-Policy') continue;
      found++;
      const next = withScriptSrc(header.value, hashes);
      if (next !== header.value) stale = true;
      header.value = next;
    }
  }
  if (found === 0) {
    console.error('csp-hashes: no Content-Security-Policy header in firebase.json');
    process.exit(1);
  }

  if (process.argv.includes('--check')) {
    if (stale) {
      console.error('csp-hashes: firebase.json script-src is stale - run `npm run build` and commit firebase.json');
      process.exit(1);
    }
    console.log(`csp-hashes: firebase.json matches ${hashes.length} inline scripts in ${files.length} pages`);
    return;
  }
  fs.writeFileSync(firebaseFile, JSON.stringify(config, null, 2) + '\n');
  console.log(`csp-hashes: script-src pinned to ${hashes.length} inline scripts from ${files.length} pages`);
}

if (require.main === module) main();
module.exports = { inlineScriptHashes, withScriptSrc };
