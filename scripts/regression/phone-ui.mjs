import { execFileSync } from 'node:child_process';
import { readFile, writeFile, mkdir, realpath } from 'node:fs/promises';
import { createRequire } from 'node:module';
import { randomUUID } from 'node:crypto';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../..');
const require = createRequire(import.meta.url);
const yaml = require(path.join(root, 'mobile/node_modules/yaml'));
const xml = require(path.join(root, 'mobile/node_modules/xml2js'));
export const APP = 'com.talktoode.app';
export const escapeSelectorText = value => value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
export function odeHasFocus(output) {
  return output.split(/\r?\n/).some(line => line.includes('mCurrentFocus=') && line.includes(APP + '/'));
}
export function exportChooserHasFocus(output) {
  return output.split(/\r?\n/).some(line => line.includes('mCurrentFocus=') &&
    /\s(?:android\/com\.android\.internal\.app\.ChooserActivity|com\.android\.intentresolver\/(?:\.ChooserActivity|com\.android\.intentresolver\.ChooserActivity))(?:\s|})/.test(line));
}
export function adbFailure(stderr) {
  const error = new Error(/INJECT_EVENTS|CLEAR_APP_USER_DATA|GRANT_RUNTIME_PERMISSIONS/.test(stderr)
    ? 'Phone denied the required Ode test operation; additional USB debugging security permission is needed'
    : 'ADB operation failed or timed out; no command data or unrelated device output retained');
  error.blocked = /INJECT_EVENTS|CLEAR_APP_USER_DATA|GRANT_RUNTIME_PERMISSIONS|device.*not found|unauthorized|device offline/.test(stderr);
  return error;
}
export function normalizeCommand(raw) {
  if (typeof raw === 'string') return { [raw]: null };
  if (!raw || Array.isArray(raw) || typeof raw !== 'object' || Object.keys(raw).length !== 1) throw new Error('Invalid test command');
  return raw;
}
export const shellQuote = value => "'" + value.replaceAll("'", "'\\''") + "'";
export function ownedNodes(document) {
  const result = [];
  function walk(nodes) {
    for (const node of nodes || []) {
      const a = node.$ || {};
      if (a.package === APP) result.push({ id: a['resource-id'] || '', text: a.text || '', description: a['content-desc'] || '', bounds: a.bounds || '', enabled: a.enabled !== 'false', visible: a['visible-to-user'] !== 'false', scrollable: a.scrollable === 'true', password: a.password === 'true', selected: a.selected === 'true' });
      walk(node.node);
    }
  }
  walk(document.hierarchy?.node);
  return result;
}
export function isVisibleNode(node) {
  if (node.visible === false) return false;
  if (node.bounds !== undefined) {
    try { parseBounds(node.bounds); } catch { return false; }
  }
  return true;
}
export function matches(node, selector) {
  if (!isVisibleNode(node)) return false;
  if (typeof selector === 'string') selector = { text: selector };
  if (selector.id) return node.id === selector.id || node.id.endsWith('/' + selector.id);
  if (selector.text) {
    const pattern = new RegExp('^(?:' + selector.text + ')$');
    return pattern.test(node.text) || pattern.test(node.description);
  }
  throw new Error('Unsupported selector');
}
export function parseBounds(value) {
  const match = /^\[(\d+),(\d+)\]\[(\d+),(\d+)\]$/.exec(value);
  if (!match) throw new Error('Invalid app element bounds');
  const [left, top, right, bottom] = match.slice(1).map(Number);
  if (right <= left || bottom <= top) throw new Error('App element has no visible area');
  return { left, top, right, bottom };
}

export function redactEvidence(list, password = '\u0000') {
  const shareScreen = list.some(node => node.id === 'share-screen');
  return list.map(node => ({ ...node, text: node.password || shareScreen ? '[REDACTED]' : node.text.replaceAll(password, '[REDACTED]'), description: node.password || shareScreen ? '[REDACTED]' : node.description.replaceAll(password, '[REDACTED]') }));
}

async function run() {
  const serial = process.argv[2];
  const flows = (process.argv[3] || '01-app-launch').split(',');
  if (!/^[a-zA-Z0-9-]+$/.test(serial || '')) throw new Error('An explicit dedicated-device serial is required');
  if (flows.some(flow => !/^\d{2}-[a-z-]+$/.test(flow))) throw new Error('Invalid flow identifier');
  const seeded = process.argv.includes('--seeded');
  const helper = process.argv.includes('--helper');
  const subflow = process.argv.find(arg => arg.startsWith('--subflow='))?.slice('--subflow='.length);
  if (subflow && (!/^[a-z0-9-]+\.yaml$/.test(subflow) || flows.length !== 1)) throw new Error('A scoped subflow requires one flow and a test-tree YAML filename');
  const variables = {};
  if (seeded) {
    const config = yaml.parseAllDocuments(await readFile(path.join(root, 'maestro/flows/03-authentication.yaml'), 'utf8'))[0].toJSON().env;
    variables.EMAIL_EXISTING = process.env.ODE_QA_EMAIL || config.EMAIL_EXISTING;
    variables.EMAIL_EXISTING_REGEX = escapeSelectorText(variables.EMAIL_EXISTING);
    variables.PASSWORD = process.env.ODE_QA_PASSWORD || config.PASSWORD;
  }
  if (process.argv.includes('--therapist-qa')) {
    if (!seeded || !helper || flows.some(flow => !/^(08-therapist-portal|12-therapist-dashboard-and-archive)$/.test(flow))) throw new Error('Disposable therapist fixtures require the scoped therapist flows');
    variables.EMAIL_THERAPIST = 'bharatbanthia2207+odeqa-therapist-20261014@gmail.com';
    variables.PASSWORD_THERAPIST = variables.PASSWORD;
    variables.THERAPIST_NAME = 'Dr Ode QA';
    variables.THERAPIST_CREDENTIALS = 'QA TEST ONLY - NOT A CLINICIAN';
  }
  const out = path.join(root, 'test-results', 'phone-' + new Date().toISOString().replaceAll(':', '-'));
  await mkdir(out, { recursive: true });
  const actions = [], results = [];
  let activeFlow;
  let incomplete = false;
  const sleep = ms => new Promise(resolve => setTimeout(resolve, ms));
  function adb(args, timeout = 30000) {
    try { return execFileSync('adb', ['-s', serial, ...args], { encoding: 'utf8', timeout, windowsHide: true, stdio: ['ignore', 'pipe', 'pipe'] }); }
    catch (error) {
      throw adbFailure(String(error.stderr || ''));
    }
  }
  function foreground() {
    const focus = adb(['shell', "dumpsys window | grep -E 'mCurrentFocus|mFocusedApp'"]);
    return odeHasFocus(focus);
  }
  function guard() { if (!foreground()) throw new Error('Ode is not foreground; no input sent to another app'); }
  async function nodes() {
    if (!foreground()) return [];
    if (helper) {
      const output = adb(['shell', 'am', 'instrument', '-w', '-r', 'com.talktoode.qa/com.talktoode.qa.Hierarchy']);
      const hierarchy = output.split(/\r?\n/).find(line => line.startsWith('INSTRUMENTATION_RESULT: ode_hierarchy='));
      if (!hierarchy && output.includes('INSTRUMENTATION_RESULT: ode_error=Ode does not own the active window')) return [];
      if (!hierarchy) throw new Error('Ode-only helper could not read the active app hierarchy');
      return ownedNodes(await xml.parseStringPromise(hierarchy.slice('INSTRUMENTATION_RESULT: ode_hierarchy='.length)));
    }
    const temporary = '/data/local/tmp/ode-qa-' + randomUUID() + '.xml';
    try {
      const status = adb(['shell', `uiautomator dump --compressed ${temporary}`]);
      if (/could not get idle state/i.test(status)) {
        const error = new Error('UI Automator could not obtain an idle app hierarchy; functional assertion not executed');
        error.blocked = true;
        throw error;
      }
      const raw = adb(['shell', `cat ${temporary}`]);
      return ownedNodes(await xml.parseStringPromise(raw));
    } finally { adb(['shell', `rm -f ${temporary}`]); }
  }
  function scrub(list) {
    return redactEvidence(list, variables.PASSWORD);
  }
  async function select(selector, timeout = 10000, absent = false) {
    const deadline = Date.now() + timeout;
    do {
      const current = await nodes();
      const found = current.filter(node => matches(node, selector));
      if (absent ? found.length === 0 && current.length > 0 : found.length > 0) return absent ? true : found[(typeof selector === 'object' && selector.index) || 0];
      await sleep(300);
    } while (Date.now() < deadline);
    const error = new Error('Required app selector condition did not match: ' + JSON.stringify(selector));
    error.selectorUnavailable = true;
    throw error;
  }
  async function tap(spec) {
    guard();
    let x, y;
    if (spec.point) {
      const match = /^(\d+(?:\.\d+)?)%,(\d+(?:\.\d+)?)%$/.exec(spec.point);
      if (!match) throw new Error('Invalid percentage point');
      const list = await nodes();
      const areas = list.map(node => { try { return parseBounds(node.bounds); } catch { return null; } }).filter(Boolean);
      const right = Math.max(...areas.map(area => area.right)), bottom = Math.max(...areas.map(area => area.bottom));
      x = Math.round(right * Number(match[1]) / 100); y = Math.round(bottom * Number(match[2]) / 100);
    } else {
      const node = await select(spec, spec.timeout || 10000);
      if (!node.enabled) throw new Error('Requested app control is disabled');
      const bounds = parseBounds(node.bounds);
      x = Math.round((bounds.left + bounds.right) / 2); y = Math.round((bounds.top + bounds.bottom) / 2);
    }
    guard(); adb(['shell', `input tap ${x} ${y}`]); await sleep(300);
  }
  async function scroll(spec) {
    const deadline = Date.now() + (spec.timeout || 30000);
    do {
      if ((await nodes()).some(node => matches(node, spec.element))) return;
      guard();
      const current = await nodes();
      const candidates = current.filter(node => node.scrollable && isVisibleNode(node));
      if (!candidates.length) { await sleep(300); continue; }
      const b = parseBounds(candidates.at(-1).bounds);
      const x = Math.round((b.left + b.right) / 2);
      const low = Math.round(b.top + (b.bottom - b.top) * 0.8), high = Math.round(b.top + (b.bottom - b.top) * 0.25);
      adb(['shell', `input swipe ${x} ${spec.direction === 'UP' ? high : low} ${x} ${spec.direction === 'UP' ? low : high} 450`]);
      await sleep(300);
    } while (Date.now() < deadline);
    throw new Error('App scroll target not visible');
  }
  function expand(value, env) {
    if (typeof value === 'string') return value.replace(/\$\{([A-Z0-9_]+)\}/g, (_, key) => { if (env[key] === undefined) throw new Error('Missing test variable ' + key); return env[key]; });
    if (Array.isArray(value)) return value.map(item => expand(item, env));
    if (value && typeof value === 'object') return Object.fromEntries(Object.entries(value).map(([key, item]) => [key, expand(item, env)]));
    return value;
  }
  async function load(file, env, ancestry = []) {
    const actual = await realpath(file), base = await realpath(path.join(root, 'maestro'));
    const relative = path.relative(base, actual);
    if (relative.startsWith('..') || path.isAbsolute(relative) || ancestry.includes(actual)) throw new Error('Flow path outside test tree or recursive flow');
    const docs = yaml.parseAllDocuments(await readFile(actual, 'utf8'));
    if (docs.some(doc => doc.errors.length)) throw new Error('Invalid flow YAML');
    const config = docs[0].toJSON();
    if (config.appId !== APP) throw new Error('Only Ode flows are permitted');
    const scope = { ...env, ...config.env, ...variables };
    if (scope.CLIENT_NAME) {
      scope.CLIENT_NAME_REGEX = escapeSelectorText(scope.CLIENT_NAME);
      scope.CLIENT_ROW_REGEX = '[^,]+, ' + scope.CLIENT_NAME_REGEX + ', .*';
    }
    if (config.incompleteReason) incomplete = true;
    if (helper && config.phoneFlow) {
      await load(path.resolve(path.dirname(actual), config.phoneFlow), scope, [...ancestry, actual]);
      return;
    }
    await execute(docs[1].toJSON(), scope, actual, [...ancestry, actual]);
  }
  async function execute(commands, env, file, ancestry) {
    for (const raw of commands) {
      const command = expand(normalizeCommand(raw), env);
      const [kind, value] = Object.entries(command)[0];
      const record = { feature: activeFlow, kind, status: 'passed', startedAt: new Date().toISOString() };
      console.log(`[${activeFlow}] START ${kind}`);
      if (['tapOn', 'assertVisible', 'assertNotVisible'].includes(kind)) record.target = value;
      try {
        switch (kind) {
          case 'clearState': if (adb(['shell', 'pm', 'clear', APP]).trim() !== 'Success') throw new Error('Phone did not confirm Ode-only app data reset'); break;
          case 'launchApp': adb(['shell', 'am', 'start', '-W', '-a', 'android.intent.action.MAIN', '-c', 'android.intent.category.LAUNCHER', '-n', APP + '/.MainActivity']); break;
          case 'openLink': {
            if (!/^ode:\/\/(upgrade|entries|reflection\/[0-9a-f-]{36})$/.test(value)) throw new Error('Only approved Ode QA deep links are permitted');
            adb(['shell', 'am', 'start', '-W', '-a', 'android.intent.action.VIEW', '-d', value, '-n', APP + '/.MainActivity']); break;
          }
          case 'stopApp': adb(['shell', 'am', 'force-stop', APP]); break;
          case 'backgroundApp': guard(); adb(['shell', 'input', 'keyevent', '3']); await sleep(1000); break;
          case 'assertExportChooser': {
            const deadline = Date.now() + Math.min(value?.timeout || 60000, 90000);
            let ready = false;
            do {
              ready = exportChooserHasFocus(adb(['shell', "dumpsys window | grep 'mCurrentFocus'"]));
              if (ready) break;
              await sleep(500);
            } while (Date.now() < deadline);
            if (!ready) throw new Error('Expected Android export chooser did not appear; no external UI read or input sent');
            record.assertion = true;
            record.reason = 'Chooser component only; PDF bytes and recipient UI not inspected';
            break;
          }
          case 'dismissExportChooser': {
            if (!exportChooserHasFocus(adb(['shell', "dumpsys window | grep 'mCurrentFocus'"]))) throw new Error('Expected export chooser is not foreground; no Back input sent');
            adb(['shell', 'input', 'keyevent', '4']); await sleep(1000);
            guard();
            break;
          }
          case 'waitForAnimationToEnd': guard(); await sleep(Math.min(value?.timeout || 1500, 5000)); guard(); record.reason = 'Bounded transition settle, not an animation-idle measurement'; break;
          case 'tapOn': await tap(typeof value === 'string' ? { text: value } : value); break;
          case 'inputText': {
            guard();
            if (!/^[\x20-\x7e]*$/.test(value) || value.includes('%s')) throw new Error('This restricted runner requires supported ASCII text; no unsupported input sent');
            adb(['shell', 'input text ' + shellQuote(value.replaceAll(' ', '%s'))]); break;
          }
          case 'hideKeyboard': {
            guard();
            const state = adb(['shell', "dumpsys input_method | grep -E 'mInputShown|mIsInputViewShown'"]);
            if (/mInputShown=true|mIsInputViewShown=true/.test(state)) adb(['shell', 'input', 'keyevent', '4']);
            break;
          }
          case 'back': guard(); adb(['shell', 'input', 'keyevent', '4']); await sleep(300); break;
          case 'assertVisible': await select(value, value.timeout || 10000); record.assertion = true; break;
          case 'assertSelected': {
            const node = await select(value, value.timeout || 10000);
            if (node.selected !== (value.selected !== false)) throw new Error('App control selection state did not match');
            record.assertion = true;
            break;
          }
          case 'assertQaIdentity': {
            try {
              const expected = value?.email || env.EMAIL_EXISTING;
              if (!['bharatbanthia2207+tester@gmail.com', 'bharatbanthia2207+odeqa-therapist-20261014@gmail.com'].includes(expected)) throw new Error('Unapproved account');
              await select({ text: escapeSelectorText(expected) }, value?.timeout || 20000);
              record.assertion = true;
            } catch {
              const error = new Error('Approved tester identity not verified; no history data read or failure hierarchy retained');
              error.blocked = true;
              throw error;
            }
            break;
          }
          case 'assertNotVisible': await select(value, value.timeout || 10000, true); record.assertion = true; break;
          case 'extendedWaitUntil': await select(value.visible || value.notVisible, value.timeout || 30000, Boolean(value.notVisible)); record.assertion = true; break;
          case 'extendedWaitUntilAny': {
            if (!Array.isArray(value.visible) || !value.visible.length || value.visible.length > 6) throw new Error('A bounded list of app selectors is required');
            const deadline = Date.now() + (value.timeout || 30000);
            let found = false;
            do {
              const current = await nodes();
              found = value.visible.some(selector => current.some(node => matches(node, selector)));
              if (!found) await sleep(300);
            } while (!found && Date.now() < deadline);
            if (!found) throw new Error('None of the required app route selectors appeared');
            record.assertion = true;
            break;
          }
          case 'longPressOn': {
            guard();
            const node = await select(value);
            const b = parseBounds(node.bounds), x = Math.round((b.left + b.right) / 2), y = Math.round((b.top + b.bottom) / 2);
            guard(); adb(['shell', `input swipe ${x} ${y} ${x} ${y} 1200`]); await sleep(300);
            break;
          }
          case 'scrollUntilVisible': await scroll(value); break;
          case 'takeScreenshot': record.status = 'not_captured_privacy'; record.reason = 'No phone screenshots; app-owned hierarchy only'; await writeFile(path.join(out, `${activeFlow}-ui-${actions.length}.json`), JSON.stringify(scrub(await nodes()), null, 2)); break;
          case 'runFlow': {
            if (typeof value === 'string') await load(path.resolve(path.dirname(file), value), env, ancestry);
            else {
              const current = await nodes();
              const condition = !value.when || (value.when.visible ? current.some(node => matches(node, value.when.visible)) : !current.some(node => matches(node, value.when.notVisible)));
              if (condition) await execute(value.commands, env, file, ancestry);
              else {
                record.status = 'not_executed_condition';
                if (value.required === true) incomplete = true;
              }
            }
            break;
          }
          default: throw new Error('Unsupported flow command; not silently skipped: ' + kind);
        }
      } catch (error) {
        if (value?.optional === true && error.selectorUnavailable) { incomplete = true; record.status = 'optional_not_executed'; record.reason = 'Optional app selector unavailable'; }
        else { record.status = error.blocked ? 'blocked' : 'failed'; record.reason = String(error.message).replaceAll(variables.PASSWORD || '\u0000', '[REDACTED]'); actions.push(record); const failure = new Error(record.reason); failure.blocked = error.blocked; throw failure; }
      }
      actions.push(record);
      console.log(`[${activeFlow}] ${record.status.toUpperCase()} ${kind}`);
      await writeFile(path.join(out, 'command-results.json'), JSON.stringify(actions, null, 2));
    }
  }
  if (!adb(['shell', 'pm', 'path', APP]).includes('package:')) throw new Error('Ode must be installed first');
  for (const flow of flows) {
    activeFlow = flow;
    incomplete = Boolean(subflow);
    try { await load(subflow ? path.join(root, 'maestro/subflows', subflow) : path.join(root, 'maestro/flows', flow + '.yaml'), variables); results.push({ feature: flow, status: incomplete ? 'partial' : 'passed', scope: subflow || 'full-flow', reason: incomplete ? 'Some feature assertions or required fixture blocks were not executed; flow is not complete' : 'Mandatory app-scoped functional assertions passed; visual screenshots not executed' }); }
    catch (error) { results.push({ feature: flow, status: error.blocked ? 'blocked' : 'failed', reason: error.message }); if (!error.blocked) await writeFile(path.join(out, flow + '-failure-ui.json'), JSON.stringify(scrub(await nodes().catch(() => [])), null, 2)); }
    await writeFile(path.join(out, 'results.json'), JSON.stringify(results, null, 2));
    await writeFile(path.join(out, 'command-results.json'), JSON.stringify(actions, null, 2));
    console.log(`${results.at(-1).status.toUpperCase()} ${flow}`);
  }
  console.log('Report: ' + out);
  console.log(JSON.stringify({ assertionsPassed: actions.filter(action => action.assertion && action.status === 'passed').length, tapsCompleted: actions.filter(action => action.kind === 'tapOn' && action.status === 'passed').length, failedCommands: actions.filter(action => action.status === 'failed').length, blockedCommands: actions.filter(action => action.status === 'blocked').length }));
  process.exitCode = results.some(result => result.status === 'failed') ? 1 : results.some(result => result.status === 'blocked') ? 2 : 0;
}
if (process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)) await run();
