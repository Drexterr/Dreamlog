import test from 'node:test';
import assert from 'node:assert/strict';
import { APP, ownedNodes, matches, parseBounds, shellQuote, normalizeCommand, adbFailure, odeHasFocus, exportChooserHasFocus, escapeSelectorText, redactEvidence } from './phone-ui.mjs';

test('permits export cancellation only for exact Android chooser components', () => {
  assert.equal(exportChooserHasFocus('mCurrentFocus=Window{ android/com.android.internal.app.ChooserActivity}'), true);
  assert.equal(exportChooserHasFocus('mCurrentFocus=Window{ com.android.intentresolver/.ChooserActivity}'), true);
  assert.equal(exportChooserHasFocus('mCurrentFocus=Window{ personal.app/.ChooserActivity}'), false);
  assert.equal(exportChooserHasFocus('mCurrentFocus=Window{ personal.app/.Main}\nmFocusedApp=Activity{ android/com.android.internal.app.ChooserActivity}'), false);
  assert.equal(exportChooserHasFocus('mCurrentFocus=Window{ fakeandroid/com.android.internal.app.ChooserActivity}'), false);
});

test('redacts all share-screen content while retaining selectors and bounds', () => {
  const nodes = [{ id: 'share-screen', text: '', description: '' }, { id: 'share-send-button', text: 'secret-passcode', description: 'https://example.invalid/secret-token', bounds: '[1,2][100,200]' }];
  const result = redactEvidence(nodes);
  assert.equal(result[1].id, 'share-send-button');
  assert.equal(result[1].bounds, '[1,2][100,200]');
  assert.equal(JSON.stringify(result).includes('secret'), false);
});

test('redacts credentials outside share screens without mutating source nodes', () => {
  const nodes = [{ id: 'auth-password-input', text: 'fixture-secret', description: 'fixture-secret', password: true }];
  const result = redactEvidence(nodes, 'fixture-secret');
  assert.equal(JSON.stringify(result).includes('fixture-secret'), false);
  assert.equal(nodes[0].text, 'fixture-secret');
});

test('escapes plus-addressed email values in text selectors', () => {
  const email = 'tester+qa@example.com';
  assert.equal(matches({ text: email, description: '' }, escapeSelectorText(email)), true);
  assert.equal(matches({ text: 'testerqa@exampleXcom', description: '' }, escapeSelectorText(email)), false);
});

test('matches fixture names containing literal parentheses', () => {
  const name = 'Ode QA Portal (20261014)';
  assert.equal(matches({ text: name, description: '' }, escapeSelectorText(name)), true);
  assert.equal(matches({ text: 'Ode QA Portal 20261014', description: '' }, escapeSelectorText(name)), false);
});

test('requires Ode current focus even when Ode remains behind another window', () => {
  assert.equal(odeHasFocus(`mCurrentFocus=Window{ ${APP}/.MainActivity}`), true);
  assert.equal(odeHasFocus(`mCurrentFocus=Window{ system/overlay}\nmFocusedApp=Activity{ ${APP}/.MainActivity}`), false);
  assert.equal(odeHasFocus('mCurrentFocus=null'), false);
});

test('retains only Ode UI nodes, never unrelated app or keyboard nodes', () => {
  const document = { hierarchy: { node: [{ $: { package: 'other.app', text: 'private' }, node: [{ $: { package: APP, text: 'Ode', 'resource-id': 'home-screen' } }, { $: { package: 'keyboard.app', text: 'suggestion' } }] }] } };
  const nodes = ownedNodes(document); assert.equal(nodes.length, 1); assert.equal(nodes[0].text, 'Ode');
  assert.equal(JSON.stringify(nodes).includes('private'), false); assert.equal(JSON.stringify(nodes).includes('suggestion'), false);
});

test('reads selection state only from app-owned nodes', () => {
  const nodes = ownedNodes({ hierarchy: { node: [{ $: { package: APP, 'resource-id': 'mood-range-90d', selected: 'true' } }, { $: { package: 'other.app', selected: 'true' } }] } });
  assert.equal(nodes.length, 1); assert.equal(nodes[0].selected, true);
});
test('matches exact IDs and full text expressions', () => {
  const node = { id: APP + ':id/home-screen', text: 'Ode', description: '' };
  assert.equal(matches(node, { id: 'home-screen' }), true);
  assert.equal(matches(node, 'Od'), false); assert.equal(matches(node, 'Ode'), true);
});
test('validates visible element bounds', () => {
  assert.deepEqual(parseBounds('[1,2][101,202]'), { left: 1, top: 2, right: 101, bottom: 202 });
  assert.throws(() => parseBounds('[0,0][0,0]')); assert.throws(() => parseBounds('invalid'));
});

test('does not match zero-area off-screen controls', () => {
  assert.equal(matches({ id: 'tab-mood', bounds: '[1,2][100,200]', visible: false }, { id: 'tab-mood' }), false);
  assert.equal(matches({ id: 'settings-help-row', bounds: '[0,0][0,0]' }, { id: 'settings-help-row' }), false);
  assert.equal(matches({ id: 'settings-help-row', bounds: '[1,2][100,200]' }, { id: 'settings-help-row' }), true);
});
test('quotes text as data rather than shell instructions', () => {
  assert.equal(shellQuote("QA's text"), "'QA'\\''s text'");
  assert.equal(shellQuote('test; command'), "'test; command'");
});
test('normalizes bare YAML commands and rejects ambiguous commands', () => {
  assert.deepEqual(normalizeCommand('clearState'), { clearState: null });
  assert.deepEqual(normalizeCommand({ tapOn: { id: 'home-screen' } }), { tapOn: { id: 'home-screen' } });
  assert.throws(() => normalizeCommand({ tapOn: {}, back: null }));
});
test('classifies phone security restrictions as blocked without exposing raw device output', () => {
  const error = adbFailure('SecurityException private-data CLEAR_APP_USER_DATA');
  assert.equal(error.blocked, true); assert.equal(error.message.includes('private-data'), false);
  assert.equal(adbFailure('device offline').blocked, true);
});
