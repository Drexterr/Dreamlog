# Mobile Continuation: Settings, Lifecycle and Export

This batch uses the installed Mood-fix APK recorded in ODE_MOBILE_ADB_CHECKPOINT.md, dedicated phone ad36078a and Ode-owned UI metadata only. Audio upload and purchases remain excluded. Backend work is delegated separately and will be reported with fresh evidence, not merged into mobile source coverage.

## Step 1: Settings Baseline and Restoration

- Baseline phone-2026-10-14T12-44-11.303Z: original voice language Auto observed; two assertions/four taps passed. The screenshot command stores only app-owned hierarchy JSON, never a phone image.
- phone-2026-10-14T12-45-48.087Z: negative language-search checks passed, but Hindi matched the search field rather than the language row. No preference change occurred; selector corrected to the row's full accessibility label.
- phone-2026-10-14T12-46-51.838Z: Hindi was saved; the post-restart scroll moved past the row and timed out. Saved failure metadata contains Hindi in settings. Scroll direction/settle were corrected, not application source.
- phone-2026-10-14T12-49-43.488Z: eight assertions/eight taps passed, confirming Hindi after cold restart and restoring Auto, then confirming its selected state after another restart. No failed/blocked commands.
- Coverage after this step: cumulative passing ADB flows 14/20 (70%), unchanged because this is a scoped flow-05 completion only. Latest measured mobile source coverage remains 30.75% statements, 31.28% branches, 23.07% functions; UI runs do not collect source coverage.

## Step 2: Corrected Settings Persistence Scope

- phone-2026-10-14T12-50-45.789Z: all 16 assertions/16 taps passed, zero failed/blocked commands. Covers no-match search, selecting the Hindi row, persistence after cold restart, restoring Auto and verifying its selected state after another cold restart.
- Original QA preference is restored. This scope is complete, while full flow 05 remains partial.
- Coverage after this step: ADB 14/20 (70%); source coverage unchanged at 30.75%/31.28%/23.07% mobile statements/branches/functions. No fresh source instrumentation was run.

## Step 3: Real Background and Restore

- phone-2026-10-14T12-52-51.761Z: all 11 assertions/11 taps passed, zero failed/blocked commands. Uses Android Home, then launches Ode, verifying retained Settings, Home and the selected but unstarted recording mode. QA identity is checked before and after, followed by a cold-start Home check.
- No recording was started; no other app UI was inspected. Full flow 06 remains partial for permissions and network-fault controls.
- Coverage after this step: ADB 14/20 (70%); mobile source coverage unchanged at 30.75%/31.28%/23.07%.

## Step 4: Journey and Monthly Export Handoff

- Running journey/export scope has verified an unstarted journey screen through background/restore and reopened a matching in-progress Stress Relief row after cold start. Multiple matching sessions may exist; this does not establish exact session-ID identity or recorded progress.
- Monthly PDF generation reached the exact Android chooser component. Cancellation returned to export-screen with Generate & Download PDF available again. No recipient was selected and no chooser UI content was inspected. This verifies successful download-to-share handoff, not PDF structure/content.
- Coverage after this completed monthly substep remains ADB 14/20 (70%), mobile source 30.75%/31.28%/23.07%. The overall export scope is still running its yearly check.

## Step 5: Yearly Export and Completed Journey/Export Scope

- phone-2026-10-14T12-54-08.623Z: all 20 assertions/21 taps passed, zero failed/blocked commands. Both monthly and yearly exports reached the Android chooser, cancelled back to Ode and restored the export action. Journey background/restore, matching in-progress row reopen and final Home navigation passed.
- The chooser guard matches only exact Android chooser components. All 14 privacy/selector harness tests passed, including refusal to dismiss unrelated activities or a chooser listed only as a background app.
- Coverage after this step: ADB 14/20 (70%); source coverage unchanged at 30.75%/31.28%/23.07%. Full flow 10 remains partial for actual PDF receipt/content, populated relationship interactions and an open follow-up.
- Across the three final corrected scopes: 47 assertions and 48 taps passed. These are action counts, not 47 unique full-app scenarios. The earlier baseline/restoration checks are not added again to this total.
- Phone automation is finished for this batch, and the QA language is restored to Auto. No new APK was needed for these test-script changes.

## Step 6: Harness and Coverage Reporting Guards

- All 19 combined phone/privacy and coverage-checkpoint tests passed (14 phone guards, five ledger guards). A null/missing prior backend measurement now produces an unknown delta, never an artificial gain from zero.
- This is regression-tool coverage, not app coverage. ADB 14/20 (70%) and mobile source 30.75%/31.28%/23.07% remain unchanged. Fresh independent backend and export-screen unit measurements are pending in separate agent checkpoints.

## Step 7: Completed Parallel Work and Consolidation

- Backend: 843 named checks passed, no failures/skips, vet passed. Fresh whole-backend statement coverage 3422/6514 (52.53%), +1.03 percentage points against both historical and fresh comparable baseline. Focused ExportData 87.5%. See ODE_BACKEND_PARALLEL_CHECKPOINT.md for each measured step.
- Export-screen unit tests: 20/20 passed, focused 100% statements/branches/functions/lines. See ODE_EXPORT_UNIT_CHECKPOINT.md for retained setup/cold-render failures and corrected passing evidence.
- Parent combined mobile suite: 263/263 tests passed across 21 suites; TypeScript passed. Whole-mobile coverage is now 31.51% statements (1203/3818), 31.97% branches (831/2599), 23.61% functions (260/1101), gains +0.76/+0.69/+0.54 percentage points. This supersedes earlier mobile source figures in this document.
- ADB remains 14/20 (70%); full app/source milestones 70/80/90 remain unachieved. Both agents and parent verification are complete. No phone connection is required for these completed unit/integration checks.

## Remaining Scope

Settings failed-save rollback, controlled recording/upload, permission-dialog handling and app-scoped network failure injection remain unclosed. Native export chooser presence will not be presented as validation of PDF bytes or content. Journey reopen is not a completed recorded journey. Scoped passing checks do not certify the full feature.
