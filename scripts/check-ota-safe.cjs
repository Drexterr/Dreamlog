#!/usr/bin/env node
/**
 * N16: refuses an OTA update that could reach binaries lacking its native code.
 *
 * mobile/app.json uses a static runtimeVersion, so an OTA is delivered to every
 * installed binary with that runtime. That is only safe while nothing native
 * changed since those binaries were built. mobile/ota-baseline.json records the
 * runtime and the commit its binaries were built from; this script fails when:
 *   - app.json or android strings.xml disagree with the baseline runtime, or
 *   - any native-affecting file changed since the baseline commit
 *     (android/, package.json, package-lock.json, app.json, eas.json, plugins).
 *
 * A false positive (e.g. a JS-only dependency bump) is resolved by bumping the
 * runtime and shipping new binaries, or - after checking the change really is
 * JS-only - moving nativeBaselineCommit forward in the same reviewed commit.
 *
 *   node scripts/check-ota-safe.cjs
 */
const { execFileSync } = require('child_process');
const fs = require('fs');
const path = require('path');

const repo = path.join(__dirname, '..');
const mobile = path.join(repo, 'mobile');
const NATIVE_PATHS = [
  'mobile/android',
  'mobile/package.json',
  'mobile/package-lock.json',
  'mobile/app.json',
  'mobile/eas.json',
  'mobile/plugins',
];

function fail(msg) {
  console.error(`check-ota-safe: ${msg}`);
  process.exit(1);
}

const baseline = JSON.parse(fs.readFileSync(path.join(mobile, 'ota-baseline.json'), 'utf8'));
const appRuntime = JSON.parse(fs.readFileSync(path.join(mobile, 'app.json'), 'utf8')).expo.runtimeVersion;
if (typeof appRuntime !== 'string') fail(`app.json runtimeVersion must be a fixed string, got ${JSON.stringify(appRuntime)}`);
if (appRuntime !== baseline.runtimeVersion) {
  fail(`app.json runtimeVersion ${appRuntime} != baseline ${baseline.runtimeVersion}: build and release binaries for the new runtime, then update mobile/ota-baseline.json`);
}
const stringsXml = fs.readFileSync(path.join(mobile, 'android/app/src/main/res/values/strings.xml'), 'utf8');
const androidRuntime = (stringsXml.match(/name="expo_runtime_version">([^<]+)</) || [])[1];
if (androidRuntime !== appRuntime) fail(`android strings.xml runtime ${androidRuntime} != app.json ${appRuntime}`);

let changed;
try {
  changed = execFileSync('git', ['diff', '--name-only', baseline.nativeBaselineCommit, 'HEAD', '--', ...NATIVE_PATHS], { cwd: repo, encoding: 'utf8' }).trim();
} catch (e) {
  fail(`cannot diff against baseline ${baseline.nativeBaselineCommit} (fetch full history): ${e.message}`);
}
if (changed) {
  fail(`native-affecting files changed since the ${appRuntime} binaries were built - an OTA could crash them:\n${changed}`);
}
console.log(`check-ota-safe: no native changes since ${baseline.nativeBaselineCommit.slice(0, 7)} (runtime ${appRuntime}) - OTA is safe`);
