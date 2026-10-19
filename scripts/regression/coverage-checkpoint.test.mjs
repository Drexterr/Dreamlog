import test from 'node:test';
import assert from 'node:assert/strict';
import { buildCheckpoint } from './coverage-checkpoint.mjs';

const report = (values = {}) => ({
  backendStatementCoverage: 51.48,
  mobileCoverage: { statements: 23.68, branches: 24.1, functions: 17 },
  results: ['backend', 'backend-vet', 'mobile', 'mobile-types', 'portal-types'].map(name => ({ name, status: 'passed' })),
  testCounts: { passed: 1045 }, scenarioCoverage: null, ...values,
});
test('records baseline percentages without inventing scenario coverage', () => {
  const result = buildCheckpoint('baseline', report());
  assert.equal(result.coverage.backendStatements, 51.48);
  assert.equal(result.deltaPercentagePoints, null);
  assert.equal(result.scenarioCoverage, null);
  assert.equal(result.milestones[70], false);
});
test('calculates percentage-point changes and independent metric gates', () => {
  const previous = buildCheckpoint('baseline', report());
  const result = buildCheckpoint('auth', report({ backendStatementCoverage: 80, mobileCoverage: { statements: 80, branches: 75, functions: 70 } }), previous);
  assert.equal(result.deltaPercentagePoints.backendStatements, 28.52);
  assert.equal(result.deltaPercentagePoints.mobileStatements, 56.32);
  assert.equal(result.milestones[70], true);
  assert.equal(result.milestones[80], false);
});
test('does not round coverage up or certify a failing code check', () => {
  const low = buildCheckpoint('auth', report({ backendStatementCoverage: 80, mobileCoverage: { statements: 80, branches: 69.999, functions: 80 } }));
  assert.equal(low.milestones[70], false);
  const failed = buildCheckpoint('auth', report({ backendStatementCoverage: 90, mobileCoverage: { statements: 90, branches: 90, functions: 90 }, results: [] }));
  assert.equal(failed.milestones[70], false);
});
test('rejects missing percentages and unsafe feature identifiers', () => {
  assert.throws(() => buildCheckpoint('auth', report({ mobileCoverage: {} })), /invalid measured coverage/);
  assert.throws(() => buildCheckpoint('../auth', report()), /lowercase identifier/);
});

test('does not interpret an unmeasured backend baseline as zero', () => {
  const previous = { coverage: { backendStatements: null, mobileStatements: 23, mobileBranches: 24, mobileFunctions: 17 } };
  const result = buildCheckpoint('after-mobile-only', report(), previous);
  assert.equal(result.deltaPercentagePoints.backendStatements, null);
  assert.equal(result.deltaPercentagePoints.mobileStatements, 0.68);
  assert.equal(buildCheckpoint('missing-baseline', report(), { coverage: {} }).deltaPercentagePoints.backendStatements, null);
});
