# Export Screen Unit Checkpoint

Date: 2026-10-14. Workspace: `C:/Users/bharat.jain/Desktop/per/dream`.

## Scope

Added only `mobile/__tests__/ExportScreen.test.tsx` plus this checkpoint and
separate `test-results/export-unit-*` evidence. No production changes, backend
changes, phone/ADB interactions, live accounts, shared coverage ledger changes,
or edits to existing tests. Existing dirty worktree changes were preserved.

All export API, legacy file-system, and native-sharing operations are Jest mocks.
Storage is mocked by the existing Jest setup; the real API client is not loaded.
Fixture URLs use `.invalid`; no actual network, download, or share operation runs.
The filename month is deterministic through a restored Date method spy.

## Added Checks

- Monthly default, yearly selection, switching back, and back navigation.
- Exact selected-period API call, authorized download URL/headers/destination,
  and native share URI/MIME/dialog/UTI.
- Refusal to share statuses 201, 204, 301, 401, 403, and 500.
- Rejected export parameters, download, availability, and share operations;
  synchronous export-parameter exception; successful retries after errors.
- Sharing unavailable Saved alert, cache-directory null fallback, and retry.
- Deferred promises at each of the four async boundaries: loading indicator,
  disabled button, duplicate taps ignored, and finally-reenabled successful retry.

## Runs

1. Focused run with existing mobile collection patterns/exclusions failed before
   test collection: Windows sandbox `realpath` EPERM on its temporary directory.
   Evidence: `test-results/export-unit-focused-01.log`. No coverage measurement.
2. Retry using local evidence TEMP/TMP failed before test collection with the same
   `realpath` EPERM, now on the local directory.
   Evidence: `test-results/export-unit-focused-02.log`. No coverage measurement.
3. Approved unsandboxed focused run: exit 1, 19 passed / 1 failed / 20 total;
   154.998 seconds. First render test timed out at the default 5000 ms (8761 ms
   actual duration); subsequent tests took 21-128 ms. No assertion failure was
   reported. Retained log, JSON, and coverage under `export-unit-focused-03`.
   ExportScreen measured 100% statements (29/29), branches (18/18), functions
   (6/6), and lines (29/29). Whole-mobile collection aggregate for this focused
   run: statements 0.83% (32/3818), branches 0.69% (18/2599), functions 0.54%
   (6/1101), lines 0.95% (32/3334). This is not full-suite coverage.
4. Gave only the initial cold-render test a 15000 ms timeout; retained all
   assertions. Focused rerun: exit 0, 20/20 tests passed, 1/1 suite passed,
   83.42 seconds. Initial render took 5431 ms; subsequent tests took 23-132 ms.
   Evidence prefix `export-unit-focused-04`.

| Measured run | Statements | Branches | Functions | Lines |
| --- | --- | --- | --- | --- |
| 03 ExportScreen (timeout run) | 100% (29/29) | 100% (18/18) | 100% (6/6) | 100% (29/29) |
| 04 ExportScreen (passing run) | 100% (29/29) | 100% (18/18) | 100% (6/6) | 100% (29/29) |
| 04 whole-mobile collection, focused tests only | 0.83% (32/3818) | 0.69% (18/2599) | 0.54% (6/1101) | 0.95% (32/3334) |

No production defect was established by these checks. Duplicate-tap protection
was tested through the rendered disabled button at every pending stage, not by
calling a captured handler directly. This does not establish unmount cancellation,
atomic same-tick handler reentrancy protection, PDF contents, live authorization,
or native platform behavior; those are outside this isolated unit checkpoint.

## Command

Run from `mobile`, using installed dependencies only:

```powershell
npm test -- --runInBand --runTestsByPath __tests__/ExportScreen.test.tsx --coverage '--collectCoverageFrom=src/**/*.{ts,tsx}' '--collectCoverageFrom=app/**/*.{ts,tsx}' '--collectCoverageFrom=!**/__tests__/**' '--collectCoverageFrom=!src/types/**' '--collectCoverageFrom=!src/testIDs.ts' --coverageDirectory=../test-results/export-unit-focused-04/coverage --coverageReporters=json-summary --coverageReporters=json --coverageReporters=text --json --outputFile=../test-results/export-unit-focused-04/jest.json 2>&1 | Tee-Object -FilePath ../test-results/export-unit-focused-04.log; exit $LASTEXITCODE
```

## Consolidation

Full-mobile execution is deferred to the parent to avoid simultaneous heavy
coverage work with the backend agent. Parent-provided baseline (not independently
rerun here): 243 passing mobile tests; statements 30.74908328968046%, branches
31.28126202385533%, functions 23.06993642143506%.

A focused test run with whole-mobile collection is not a fresh whole-suite
coverage measurement. Its aggregate must not be compared as a regression against
the parent's full-suite baseline.

## Parent Full-Suite Verification

The parent combined mobile run passed 263/263 tests across 21 suites in 185.511 seconds; exit zero. Mobile TypeScript passed separately. Evidence: `test-results/export-mobile-consolidated.json`, `.log`, and `export-mobile-consolidated-coverage/coverage-summary.json`.

Fresh whole-mobile coverage: statements 1203/3818 (31.51%), branches 831/2599 (31.97%), functions 260/1101 (23.61%). Increases versus the matching previous full-suite baseline are 0.76/0.69/0.54 percentage points. These are whole-mobile source metrics, not whole-app scenario certification. Focused ExportScreen remains 100%; ADB flow coverage remains 14/20 (70%).
