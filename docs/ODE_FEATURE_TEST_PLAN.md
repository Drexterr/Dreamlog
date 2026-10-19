# Ode Feature-by-Feature Test Plan

Created: 2026-10-13. Milestones: 70%, then 80%, with the existing 90% final objective retained. Coverage is measured separately for backend and mobile; passing tests are not coverage percentages or a guarantee of no bugs.

## Baseline

Latest consolidated evidence: `test-results/2026-10-13T06-34-55.565Z/`.

| Metric | Current |
| --- | --- |
| Backend statements | 51.48% |
| Mobile statements | 23.68% |
| Mobile branches | 24.10% |
| Mobile functions | 17.00% |
| Named backend/mobile checks | 1045 passed; includes Go parent/subtests |
| Separate live QA API checks | 7 passed |
| Separate redaction helper checks | 2 passed |
| Full scenario coverage | Not certified; 120 positive/negative outlines need complete evidence mappings |

Open release risks: API-37 emulator ANR and one prior authentication-test timeout. Recorder/reflection/sharing fixes are in the installed ARM64 APK; focused functional verification is still required. Tester credentials and backend access have been verified independently of the device flows.

## Execution Order

| Step | Feature / Windows | Scope and Negative Cases | Initial Status |
| --- | --- | --- | --- |
| 0 | Android harness / device matrix | Stable Android image, app build provenance, ANR reproduction, launch/foreground, keyboard, evidence and credential hygiene | In progress |
| 1 | Launch / onboarding / terms / navigation | Journal and therapy onboarding, validation, back/skip/restart, version gate, failed API, every tab and modal/back path | Partial prior evidence |
| 2 | Authentication / profile / sessions | QA sign-in, persistence, logout, refresh/expiry, wrong credentials, cancellation, role/account isolation, profile load failures | API passed; device failure unresolved |
| 3 | Journal recording / mode gate | All five modes, microphone permission, duration boundary, instant/repeated presses, discard/unmount, background/lock/audio interruption | Partial unit evidence |
| 4 | Upload / offline queue / processing workers | Presign/binary upload, retries, missing/corrupt files, reconnect, concurrent mutations, lost responses, worker redelivery/DLQ and polling recovery | Partial unit/integration evidence |
| 5 | Transcription | Reference English/Hindi/mixed-language audio, entities, accents/noise, silence/music, corrupt/unsupported audio, provider status/timeout/JSON failures | Contract fixtures only; live accuracy pending |
| 6 | Reflection / follow-up / safety | Five-mode structures, dream lenses, grounding/context isolation, missing content, route races, three-turn limit, send recovery, localized resource intents | Strong focused unit evidence; device/live quality pending |
| 7 | Therapy hub / personas / text / voice / TTS | Four personas, entitlement, text/voice switching, ordering/history, upload/transcription/TTS failures, one-hour expiry, end/cancel, audio focus and safety routing | Pending broader execution |
| 8 | Entries / history / search | Correct reflection/session routes, search/no-match/Unicode, pagination, loading/retry, stale responses, deleted/foreign IDs | Pending screen tests and device assertions |
| 9 | Mood / habits / notifications | Seeded chart/range statistics, empty/error data, timezone boundaries, streak/flashback/check-in, push permission and stale/deleted targets | Partial API/handler evidence |
| 10 | Guided journeys / relationships / reviews | Browse/start/advance, progress/retry, extraction, rename/merge, duplicates, weekly/yearly results and provider failures | Pending broader execution |
| 11 | Sharing / therapist consent / export | Create/send/revoke, passcode/lockout/expiry, raw-data/ownership boundaries, consent withdrawal, export filters/empty/offline/unauthorized cases | Sharing unit/DB passed; device/consent/export pending |
| 12 | Settings / account lifecycle | Theme/goal/language/nudge persistence, save rollback, account switching, deletion cancel/partial failure with disposable fixtures | Partial settings/resource evidence |
| 13 | Billing / entitlements | Store sandbox, purchase/cancel/restore, expiry/limits, forged/replayed receipts, concurrent callbacks and outages | Handler fixtures passed; store execution pending |
| 14 | Therapist mobile / web workspace | Registration/roles, clients/notes/OCR, archive/restore, consent withdrawal, encryption failures, browser direct-route authorization | Pending broader execution |
| 15 | Cross-feature regression / release gates | Latest APK, full suite/race CI, API/device/provider matrix, privacy/security, accessibility, layout and performance repetitions | Pending |

Within each feature: inspect current code/contracts; enumerate positive, negative, boundary and concurrency cases; reproduce failures; apply scoped fixes; run focused tests; run affected device/API checks; run consolidated regression; save a coverage checkpoint and send a short feature summary.

## Completion and Notifications

Latest mobile-only checkpoint: [ODE_MOBILE_ADB_CHECKPOINT.md](ODE_MOBILE_ADB_CHECKPOINT.md). The original table above is the initial planning baseline, not the current execution status. Authentication flow 03 and therapist flows 08/12 now pass their declared UI scope; cumulative phone-flow execution is 14/20 (70%). Features still have unclosed scenarios, including audio/upload, offline recovery, push/data fixtures, native exports, linked consent and Play sandbox checks. Do not treat this as 70% source or whole-app coverage, and do not start the backend phase merely because this flow-script milestone was reached.

A feature is not complete just because its unit suite passes. Completion requires its declared scenario matrix to pass on the intended layers and build, with no unresolved critical failure. Hardware/provider/store/iOS gaps remain explicitly outstanding rather than silently excluded. A code-test batch may be reported complete while the feature itself remains partial.

After each feature checkpoint, report:

- Feature name and status: complete, partial, or blocked for a named prerequisite.
- Passed/failed/blocked counts and confirmed fixes, with retained original failures.
- Whole backend statements and mobile statements/branches/functions, plus percentage-point change.
- Focused feature coverage separately, and scenario evidence still missing.
- APK/source provenance and next feature.

The 70/80 milestones require backend statements and mobile statements/branches/functions to reach their respective threshold independently. Full scenario certification is a separate gate. Do not inflate results by excluding production modules, counting optional/skipped assertions as passes, or replacing real-provider/device requirements with mocks.

## Test Environment Rules

Use isolated PostgreSQL/Redis and disposable fixtures for mutations. The configured deployed backend is production-named; the authorized tester account currently has read-only live checks. Confirm mutation scope before seeding journals/shares/therapy sessions there. Use store sandbox only, never real purchases. Do not dial live emergency resources. Keep tokens in memory and redact credentials from artifacts. Preserve unrelated user changes.

Use `scripts/regression/run.mjs` for full-suite measurements and `run-ui.ps1` for mandatory device flows. Coverage checkpoints are generated from real consolidated results, not predicted percentage improvements. The original API-37 ANR stays in the device matrix even if another device passes.

Generate each checkpoint with `node scripts/regression/coverage-checkpoint.mjs FEATURE test-results/RUN/results.json`. This appends measured values and deltas to `test-results/coverage-checkpoints.json`, checks each milestone independently, and leaves unaudited scenario coverage unknown. Do not run heavy builds/test suites concurrently with device flows during baseline ANR reproduction; hardware/resource-stress scenarios are separate deliberate cases.

## Current Execution Checkpoint

2026-10-13: baseline coverage ledger created; six coverage/redaction helper tests pass. Latest-source x86_64 and ARM64 release builds succeeded. The isolated API-35 emulator was created, but the software-rendered run failed before feature assertions because a System-UI ANR dialog covered the rendered welcome screen (`test-results/ui-2026-10-13T16-01-37/`). Host graphics initialization and boot later succeeded, but its UI retest was superseded by the user's dedicated phone.

The dedicated phone was authorized and identified as ARM64/API 35. Ode's test package was absent. It disconnected during the ARM64 build, so installation did not start; no phone files were copied. The physical-device test scope is restricted to Ode's APK, app-owned UI/evidence and required device metadata. No unrelated phone files, device-wide logs or full-phone bug reports may be collected. The default Maestro runner is therefore emulator-only until an app-scoped phone path is verified. The consolidated runner also disables broad diagnostics for physical devices.

Step 0 remains partial, not complete. Current last-measured coverage remains backend 51.48%, mobile statements 23.68%, branches 24.10%, functions 17.00%; no fresh feature checkpoint or coverage increase is claimed for environment work. Next: reconnect the dedicated phone, finish app-scoped phone automation, install the matching latest APK, execute required launch/navigation/auth flows, then rerun consolidated code coverage and send the feature-batch summary.

### Physical Phone Functional Batch

The dedicated phone now has the matching ARM64 Ode APK plus an independent 13 KB QA helper. The helper reads only Ode-owned UI nodes without requiring animation idle; the runner keeps current-focus guards and no phone screenshots/device-wide logs. Stage diagnostics contain command names, never typed credential values.

| Flow | Mandatory Assertions | Completed Taps | Outcome |
| --- | --- | --- | --- |
| Fresh/warm/returning launch | 15 | 15 | Passed |
| Detailed journal onboarding | 18 | 19 | Passed |
| Navigation / Help / back / auth dismissal | 23 | 27 | Passed |
| Guest negative auth validation | 20 | 29 | Passed |
| QA sign-in / restart / profile / history | 20 | 26 | Passed |
| Five recording-mode controls / cancel / reset | 12 | 10 | Passed |

Total: 108 mandatory assertions and 126 completed taps across successful runs. Original harness failures are retained: idle-state dump failure, transient accessibility root readiness, off-screen zero-area Settings selector, and unescaped plus-address email regex. Successful reruns are not a claim that every feature scenario or visual state is certified. No audio was captured in this batch.

New opt-in app diagnostics cover fixed feature categories and onboarding step names. Four privacy tests and TypeScript checks pass; these source diagnostics are not yet rebuilt into the installed APK. The immediate runner stage logs are already active. Nine phone-harness tests pass, alongside the existing redaction/checkpoint helper checks. A fresh consolidated coverage run follows this batch.

The user authorized labelled QA entries and conversations on the configured backend for the tester account, including normal AI/transcription usage. Purchases, account deletion and other users' data remain excluded. Next: controlled audio capture/transcription fixtures, recording boundaries and failure recovery, then reflection/reply quality. Steps 1-3 remain partial because terms/version failure cases, broader auth/session negative cases and real microphone/audio scenarios are not yet fully covered.

Fresh post-batch coverage: `test-results/2026-10-13T12-09-49.546Z/`, 1049 named checks passed, vet/types passed. Backend statements 51.48% (unchanged); mobile statements 23.84% (+0.16 points), branches 24.32% (+0.22), functions 17.28% (+0.27). All 70/80/90 milestones remain unmet. App-only device evidence is separate from the consolidated runner's intentionally disabled generic phone diagnostics. Full scenario coverage is not certified.

### Recording / Provider Checkpoint

Recording screen/hook: 28 focused tests pass after reproducing/fixing microphone-denial false listening, native-start failure false listening and duplicate rapid starts. Matching QA APK rebuilt and installed with diagnostics enabled. Phone flows 16/17/18 pass (19 assertions, 17 taps): microphone denied, native capture/discard/restart, and synthetic reflection/closed follow-up/back navigation.

One labelled synthetic English entry completed real transcription. An early-completion race caused analysis 404 and reply 500; the same entry later passed twelve live smoke checks including three replies, empty rejection and fourth-turn rejection. Local worker fix now preserves processing status until analysis is saved and retries failed crisis persistence; worker suite passes. Deployment and PostgreSQL verification remain pending, so steps 4-6 are partial, not complete.

Latest measurement: `test-results/2026-10-13T12-40-10.586Z/`, all 226 mobile tests and types pass. Mobile statements 26.09% (+2.25 points), branches 26.61% (+2.29), functions 19.29% (+2.01). Eight isolated repository tests failed because Docker/PostgreSQL were unavailable. Incomplete backend coverage 48.17% must not replace the last successful 51.48% baseline as a comparable result. No 70/80/90 gate passes. See the progress report for original failures, APK hash and precise evidence paths.

### History / Search Code Checkpoint (2026-10-14)

Seven history/search defects reproduced and fixed; 11 focused cases and all 22 sharing cases pass. Entries coverage: 93.61% statements, 86.74% branches, 100% functions. Read-only live QA history checks passed (eight new checks, 15 total with account baseline). Matching QA APK rebuilt and installed; flow 20 passed with nine assertions/nine taps (`test-results/phone-2026-10-14T06-00-35.971Z/`). It checks tester identity, synthetic search-result navigation, no-match and Back. Earlier ADB failures remain retained. The phone batch is finished and phone use is available again. Multi-page UI, device offline/recovery and live cross-account isolation remain pending.

Fresh full code measurement `test-results/2026-10-14T05-31-26.727Z/`: 1073 named checks passed, vet/types passed. Backend statements 51.50%; mobile statements 28.19% (+2.10 points), branches 28.94% (+2.34), functions 20.91% (+1.63). The earlier sharing timeout is retained; no test timeout was simply raised. The runner's physical install check failed on a transient disconnect, separately from passing code checks. Step 8 remains partial; 70/80/90 gates and full scenario certification remain unmet.
