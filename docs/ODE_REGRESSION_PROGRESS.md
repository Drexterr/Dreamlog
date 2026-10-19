# Ode Regression Progress

## Batch 1: Infrastructure, Recording, Offline Upload and Transcription

Date: 2026-10-12. The 90% full regression target is not achieved. The goal remains in progress.

Latest full run: `test-results/2026-10-12T10-15-21.030Z/`. Its `results.json` lists each named test outcome. Logs and screenshots are retained in the same directory.

| Group | Result | Evidence / reason |
| --- | --- | --- |
| Backend handlers | 326 named tests/subtests passed | backend.log |
| Backend services | 320 named tests/subtests passed | backend.log |
| Backend workers | 104 passed | backend.log |
| Backend middleware | 50 passed | backend.log |
| Backend models / errors / crypto | 20 passed | backend.log |
| Mobile recorder | 12 passed | mobile.json |
| Mobile offline queue | 13 passed | mobile.json |
| Mobile upload | 6 passed | mobile.json |
| Mobile RecordButton / mood API | 11 passed | mobile.json |
| Backend vet / mobile types / portal types | Passed | Individual logs |
| APK installation | Passed | ADB installed com.ode.app 1.0.0 on emulator-5554 |
| Android foreground app | Failed | ARM64-only APK on x86_64 emulator: SoLoaderDSONotFoundError, missing libreactnative.so |
| Current-source emulator APK rebuild | Blocked | Java/Gradle cannot establish loopback connection; IPv4 and selector retries did not resolve it |
| Functional screen flows | Blocked | App must launch before button workflows can run |
| Live audio accuracy / AI reply quality / store flows / iOS | Not executed | Separate provider fixtures, accounts and device prerequisites remain |

Total: 820 backend named tests/subtests plus 42 mobile tests passed (862 total). This is a pass count, not full scenario coverage.

Measured coverage:

- Whole backend statement coverage: 47.89%.
- Whole mobile source statement coverage: 4.92%; branches 2.70%; functions 2.89% (all app/src implementation files, not only imported files).
- Focused recorder/upload/offline run: statements 94.08%, branches 69.86%. This narrow result does not establish 90% app coverage.
- Full scenario coverage: not yet measured; 120 proposed scenarios still require explicit mappings and execution evidence.

Confirmed defects fixed and retested:

1. Concurrent offline enqueue operations overwrote one another, losing a recording. Read-modify-write mutations are now serialized; removal during an in-flight failed upload also no longer resurrects the removed item.
2. A recording stopped automatically at 30 minutes was unloaded again on submission, returning no audio. Auto-stop now preserves URI/duration for one subsequent submission and handles stop errors.

New scripts:

- `scripts/regression/run.mjs`: captures suite logs, per-test JSON, coverage, type checks and ADB installation/foreground evidence, with passed/failed/blocked results.
- `scripts/regression/install-apk.ps1`: selects the newest local APK and rejects incompatible native architectures before installation.

Next feature batches: recorder remaining negative branches; authentication/onboarding/navigation; journal processing/reflection/follow-up; therapy voice/text/TTS/safety; history/mood/journeys/reviews; sharing/privacy/billing; therapist mobile/web; live-provider and hardware/device matrix. Real PostgreSQL/Redis/storage integration coverage remains a major backend gap. Do not turn optional checks or screenshots into functional passes.

## Batch 2: Integration, Authentication and Android Navigation

Status updated 2026-10-13. Historical failures above are retained; the build and emulator blockers have since been resolved. The 90% target remains unachieved.

Full suite evidence: `test-results/2026-10-12T11-03-33.243Z/` contains 830 backend named tests/subtests and 137 mobile tests passed (967 checks). Backend vet, mobile types and portal types passed. This run predates the final Settings resource tests and the latest APK installation.

Whole-source coverage from that run: backend statements 49.78%; mobile statements 16.13%, branches 14.12%, functions 10.34%. Named check counts are not scenario coverage. The 120-scenario manifest still needs explicit execution mappings.

Resume verification on 2026-10-13: all 140 mobile tests across 14 suites passed (`test-results/mobile-resume.json`), including the three new Settings country-resource cases. Mobile TypeScript checking also passed. This test-only run did not recalculate coverage or rerun the backend.

| Feature / check | Verified result | Evidence / reason |
| --- | --- | --- |
| PostgreSQL repository lifecycle, ownership, constraints and concurrent claiming | Passed | Isolated integration database; all actual migrations applied |
| Redis FIFO, timeout, serialization and dead-letter queue | Passed | Dedicated local Redis, unique keys; no global flush |
| Transcription provider contract | Passed | Local HTTP fixtures for multipart requests, provider errors, invalid responses and cancellation; not speech accuracy |
| Authentication validation and concurrency | Passed | Unit tests cover password whitespace, duplicate submits and sheet reopen/reset |
| Processing and follow-up limits | Passed | Unit tests cover non-overlapping polling, unmount handling and three-turn send limit |
| Launch and guest onboarding on Android | Passed twice | Latest evidence `test-results/ui-2026-10-12T17-18-11/` |
| Guest auth validation with Android keyboard | Passed after fix | Flow 13: 20 mandatory assertions and 29 taps; focused inputs remain visible |
| Back navigation from Entries, Journeys and Relationships | Passed after root navigator fix | Mandatory return-to-Explore assertions in flow 04; full flow later failed another step |
| Settings help row discovery | Test corrected | Row was offscreen; added explicit scrolling rather than weakening assertions |
| Guest Settings country resources | Failed on device, source fixed | India onboarding showed international resources; `ui-2026-10-12T17-44-30/`. Three new unit tests pass; rebuilt-device retest pending |
| Current-source x86_64 APK | Built and installed | Package `com.talktoode.app`; Java socket fallback and generated namespace cache fixes resolved build blockers |
| Every button / all screens | Not complete | AST inventory: 230 source controls, 84 without explicit test IDs; not runtime coverage |
| Live speech accuracy, AI quality, paid store and native OAuth flows | Not executed | Need controlled audio corpus, provider/account fixtures and store test environment |
| iOS, physical microphone, device matrix and local race checks | Not executed | Android emulator only; local Windows race toolchain absent |

Additional confirmed fixes: meaningful password whitespace is preserved; duplicate auth requests are guarded; reopening the auth sheet clears stale loading state; processing polls no longer overlap or navigate after unmount; follow-up sends enforce the turn limit; root Stack navigation preserves the originating tab; Android auth sheet adjusts for the keyboard. The country-resource fix is not yet device-certified.

Reusable scripts now include `build-emulator.ps1`, `run-ui.ps1`, `summarize-ui.mjs`, `inventory-controls.mjs` and the positive/negative `scenarios.json` manifest. UI summaries record mandatory assertions separately from taps and optional steps. CI configuration was updated for isolated integration services and coverage artifacts, but remote CI has not been executed here.

## APK Rebuild: 2026-10-13

The first packaging attempt failed; a diagnostic retry was stopped at the user's request. A subsequent user-requested offline Gradle rebuild succeeded in 13m 59s (`:app:assembleRelease`, x86_64, 1168 tasks). Log: `test-results/rebuild-2026-10-13.log`.

APK: `mobile/android/app/build/outputs/apk/release/app-release.apk`, 59,562,597 bytes. SHA-256: `64E9CEB5C07BA8F48DAD924AA75F0ED1BBC08E0B69839B898C5DBD09EE31BCED`.

Installation initially stalled in an Android package-manager session; a non-streaming retry reported a duplicate pending package. After abandoning the verified session and cold-restarting the dedicated Pixel_9 emulator without wiping data, non-streaming installation returned `Success`. Launch returned `Status: ok` for `com.talktoode.app/.MainActivity`, and the app process remained running. This APK includes the country-resource source fix, but the full navigation flow and that modal's device assertions have not yet been rerun on it. Build/install/launch success is not full functional regression certification.

## Batch 3: Navigation Retest and Recorder Cleanup

Date: 2026-10-13. The previous batch changed authoritative state through a successful build/install/launch. This batch adds device assertion evidence and reproduces/fixes recorder cleanup failures. The complete regression objective remains unfinished.

| Check | Result | Evidence / reason |
| --- | --- | --- |
| Rebuilt APK identity | Verified | Installed package lastUpdateTime 2026-10-13 11:24:01 |
| Android navigation flow 04 | Passed | `test-results/ui-2026-10-13T11-26-51/`: 23 mandatory assertions, 26 taps, zero failed/skipped commands |
| Guest India help resources | Passed in navigation flow | Previously failing iCall assertion now passes; no call buttons were triggered |
| Recorder cleanup negative cases before fix | 5 failed, 12 existing checks passed | Discard, failed native discard, creation failure, missing URI and active unmount all left allowsRecordingIOS=true |
| Recorder cleanup after source fix | 17 passed | `test-results/recorder-cleanup.json`; playback mode restored on those paths |
| Mobile TypeScript | Passed | tsc --noEmit after recorder change |
| Full mobile coverage run | 144 passed, 1 failed | `test-results/mobile-regression-2026-10-13.json`: email credential test exceeded the unchanged 5-second timeout |
| Unchanged full mobile rerun | 145 passed, 0 failed, 14 suites | `test-results/mobile-regression-2026-10-13-retest.json`; original timeout retained as test-reliability evidence |

Current whole-mobile coverage: statements 18.60% (684/3678), branches 17.63% (436/2473), functions 12.01% (130/1082). Backend was not rerun in this batch; its last measured statement coverage remains 49.78%. A transient test timeout cannot be silently converted into an initially clean run.

Scenario manifest now links navigation and resource-rendering evidence as `partial`, with explicit remaining requirements. The broader scenarios are not certified by these checks. Resource-link intents, every remaining back/close control, live audio/reply evaluations, billing, privacy, therapist workflows and the device/platform matrix remain outstanding.

The new recorder cleanup change is source/unit-verified only and is NOT included in the installed APK (which was built before this batch). Physical iOS audio routing and remaining recorder concurrency/interruption paths still need separate validation. The next signed-in device batch requires a dedicated QA backend and disposable account fixtures; isolated tests can continue independently.

## Batch 4: Reflection, Follow-up Seeds and Check-in Controls

Date: 2026-10-13. Previous batch classified as progress: device assertions, failure reproductions and source fixes changed the evidence and worktree. This batch adds 44 reflection-screen fixture tests, including positive and negative paths. The 90% whole-app target is still not achieved.

| Feature / scenario group | Result | Evidence / reason |
| --- | --- | --- |
| Loading, independent fetches, pending/processing/failed states and retry | Passed | ReflectionScreen.test.tsx; both-fetch failure recovery and metadata-only failure exercised |
| Missing entry route | Failed before fix, passed after | Previously stayed in loading forever; now shows a visible unavailable/recovery state |
| Empty/whitespace reflection content | Failed before fix, passed after | Empty paragraph list rendered a blank reflection; now displays fallback content |
| Slow previous-route response | Failed before fix, passed after | Late response overwrote the current entry; effect lifetime guard now ignores it |
| Late response after unmount | Passed | Does not trigger reflection animation after unmount |
| Paragraphs, topic/emotion limits, grounding threshold, quotes, insight and dream lenses | Passed | Empty/missing metadata and partial dream analyses included |
| Follow-up seed and share/done destinations | Passed | Summary/emotion/topic/quote seed limits and general fallback tested; entry-specific routes asserted |
| Check-in scheduling | Passed | Success, pending duplicate press, 409 existing/disabled/missing detail, 500 and network retry |
| Crisis UI and resource controls | Passed with mocks | Normal reflection/sharing hidden; country fallback and each local resource callback tested using stub URL handlers only |
| Mobile types | Passed | tsc --noEmit after source and test changes |
| Complete mobile suite | 189 passed, 0 failed, 15 suites | `test-results/mobile-reflection-batch.json` |

Failure evidence is retained in `test-results/reflection-initial.json` (2 failed/27 passed) and `test-results/reflection-race-reproduction.json` (3 failed/27 passed). Fixed focused evidence: `test-results/reflection-fixed.json` (31 passed); the final full run includes all 44 expanded reflection cases.

Whole-mobile coverage: statements 21.67% (800/3692), branches 22.55% (559/2479), functions 15.04% (163/1084). Reflection screen alone: statements 98.31% (116/118), branches 97.62% (123/126), functions 100% (33/33). Focused screen coverage must not be presented as whole-app coverage. Coverage artifacts: `test-results/mobile-reflection-batch-coverage/`.

These tests validate screen mechanics against mocked API/provider/resource fixtures, not transcription accuracy, clinical appropriateness, real AI grounding, actual sharing authorization, delivered notifications or physical/native resource handlers. Remaining link-handler rejection behavior and device visual/layout checks need further work. The reflection screen does not display the full transcript; transcript presentation and audio accuracy remain separate feature batches.

Reflection fixes, like the previous recorder cleanup change, are not yet included in the installed October 13 APK. A new build and device regression pass are required before release certification. The scenario manifest records the exercised reflection-negative coverage as partial rather than certifying all modes or the live pipeline.

## Batch 5: Sharing Mechanics and Real Database Privacy

Date: 2026-10-13. Previous batch classified as progress: 44 fixture tests, three reproduced/fixed defects and new whole-source coverage evidence. This batch adds 22 sharing-screen tests and four PostgreSQL integration tests. The user subsequently authorized the existing tester email for signed-in checks.

| Check | Result | Evidence / reason |
| --- | --- | --- |
| First sharing harness attempt | 17 harness failures, not app failures | `share-initial.json`: nested async act/render setup failed before assertions; test setup corrected |
| Corrected pre-fix sharing run | 13 passed, 4 failed | `share-reproduction.json`: hidden list error, stale-list overwrite, token/UUID mismatch, revoked passcode/send controls retained |
| Fixed sharing screen | 22 passed | `share-expanded.json`: creation/list recovery, once-returned passcode, mocked share success/failure/dismissal, confirmation, UUID revocation, duplicate-request guard and unmount cases |
| PostgreSQL ownership/expiry/revocation | Passed | Dedicated schema; foreign revocation rejected, expired/revoked/missing tokens unavailable, revoked/expired links still count toward quota |
| PostgreSQL concurrent passcode lockout/reset | Passed | Five concurrent updates produce one threshold crossing; reset clears lock/counter |
| PostgreSQL shared-data filtering | Passed | Foreign, 31-day-old, crisis and pending entries excluded; raw transcript, reflection, quote and audio markers absent from serialized result |
| PostgreSQL empty-account view | Passed | No fabricated mood, emotion or summary data |
| Full backend/mobile run | 1045 passed | 834 backend named tests/subtests + 211 mobile tests; `test-results/2026-10-13T06-34-55.565Z/` |
| Vet and mobile/portal types | Passed | Consolidated runner logs |
| ADB install/launch/foreground capture | Passed | Infrastructure evidence only; latest source changes are not in installed APK |
| 90% gates | Failed | Backend statements 51.48%; mobile statements 23.68%, branches 24.10%, functions 17.00% |
| Scenario certification / full new-source Android functional pass | Incomplete | Manifest mappings remain partial; needs rebuilt APK and device/provider checks |

Confirmed fixes: newly created links resolve their real revoke UUID through the existing list endpoint rather than substituting the public token; latest-request guards stop stale list overwrites; list errors expose a retry action; successful revocation clears the new-link passcode/send card. Creation and revocation have in-flight guards and late unmount results are ignored. No backend response contract was changed.

The disclosure now accurately lists the display name that the backend actually shares and no longer promises exclusion of all personal details or an entry-count field absent from the payload. Integration checks verify field/filter boundaries, not that AI summaries can never contain personal information. Source changes still need APK/UI verification.

Focused sharing-screen coverage: statements 96.63%, branches 87.27%, functions 100%. This is not whole-app coverage. Real PostgreSQL evidence is in `test-results/share-postgres.log`; all suite names/outcomes and coverage gates are retained in the consolidated results JSON.

Signed-in preparation: configured backend health endpoint returned 200. A mandatory read-only account/session/profile/history flow (`14-signed-in-account.yaml`) is running against the authorized tester account, with runtime credentials from the existing fixture rather than a new committed password. Text artifacts and console lines are redacted; binary screenshots are not rewritten, so the flow does not capture screenshots while credentials are entered. Two redaction helper tests pass; the first sandboxed Node test invocation failed to spawn its child process, then the approved rerun passed. Full device result is pending at this checkpoint.

### Signed-in Device Outcome and API Isolation

The device flow subsequently failed before password entry: no visible auth-password-input after 10m 34s. `test-results/ui-2026-10-13T12-20-32/` records 11 mandatory assertions, 17 taps and one failed command. It did not establish a session or reach the backend-profile/history checks. The failure screenshot shows the Android launcher.

Captured logs prove an Ode focus-event ANR, forced activity finish and process kill, followed by launcher/gesture-service ANRs. A local bug report was captured; the Ode main-thread snapshot is in Android Perfetto/MessageQueue code. Root cause is unresolved, not confirmed as invalid credentials, a keyboard-layout defect or OS-only failure. See `docs/ODE_DEVICE_ANR_INVESTIGATION.md` for timestamps, trace location and reproduction requirements.

Separate live read-only QA API checks passed all seven cases: health 200, unauthenticated profile 401, Supabase sign-in 200, expected backend identity 200, entry-list contract 200, missing entry 404 and billing-plan contract 200. Evidence: `test-results/qa-account-2026-10-13T07-14-58.708Z.json`. `scripts/regression/check-qa-account.mjs` keeps tokens in memory and emits only status/contract outcomes, not credentials or user records. These passes do not certify device sign-in or transcription/reply quality.

Redaction helper tests were added to CI and passed locally after adding an explicit temporary-cleanup boundary. CI YAML and the 120-case manifest validate, but remote CI has not run here. The consolidated 1045 passing checks, seven live QA API checks and two redaction tests are separate evidence groups; do not silently merge them into a full scenario percentage.

## Feature Plan / Device Preparation Checkpoint

Date: 2026-10-13. Feature-by-feature execution and notification rules are saved in `docs/ODE_FEATURE_TEST_PLAN.md`; milestones are 70%, 80%, then the original 90% objective. `coverage-checkpoint.mjs` records full-source metrics and percentage-point deltas, without substituting named test counts for scenario coverage. Four checkpoint tests plus two redaction tests pass; CI configuration includes both.

Latest-source x86_64 rebuild succeeded in 14m 2s (`feature-plan-latest-apk-build.log`), APK 59,565,065 bytes, SHA-256 `2D78EE93E50BF086D5E0B66825BEB7E7A6A55A7EF3143B672661E469461EF5B7`. Recorder/reflection/sharing source fixes are now in that build. Verified Google command-line tools and an API-35 Google APIs x86_64 image were installed; a workspace-isolated Pixel-7 AVD was created after correcting the tool's SDK-layout discovery. Runtime settings: 3 GB RAM, two cores, initially software graphics.

Device flows 04 and 13 failed at onboarding-welcome before any feature assertions/taps passed (`test-results/ui-2026-10-13T16-01-37/`). The screenshot shows System UI isn't responding over Ode's rendered welcome screen; Ode remained alive and the Java crash buffer was empty. These are failed environment/device runs, not successful navigation/auth certifications. A host-GPU comparison initialized successfully on Intel UHD Graphics 620 and booted, but was stopped when the dedicated physical phone became available. Both emulator failures remain in the matrix.

The user authorized a dedicated test phone and restricted file access to testing requirements. Metadata identified it as ARM64/API 35; com.talktoode.app was absent. Latest-source ARM64 rebuild succeeded in 15m 28s (`test-results/physical-phone-arm64-build.log`). The phone subsequently disappeared from ADB, so installation failed its availability check before transferring files. Installer handling now reports unavailable/unauthorized targets clearly rather than dereferencing null output.

Privacy controls: generic Maestro evidence capture is now restricted to dedicated emulators because it captures device-wide logs. The consolidated runner limits physical-device access to the selected app's install metadata and reports functional checks blocked until an app-scoped phone runner is verified. No physical-phone private files, screenshots or logs were collected, and no phone files were copied. No full-phone bug report is permitted by the current scope. Installer and runner syntax checks pass.

Code coverage was not recalculated during this environment-only step; last measured values remain backend statements 51.48%, mobile statements 23.68%, branches 24.10%, functions 17.00%. Step 0 is partial. Next required actions: reconnect the phone, verify the restricted app-only test path, install ARM64 APK and execute feature assertions before a fresh consolidated coverage checkpoint.

## Physical Phone Launch Checkpoint

Latest-source ARM64 APK was installed successfully on the dedicated phone. Initial Ode cold launch returned Status OK in 1873 ms. USB debugging security initially denied app-data reset; after the user enabled it, reset and launch passed.

The restricted runner retains only Ode-owned hierarchy nodes, with password masking; it captures no phone screenshots or device-wide logs. Foreground input requires Ode's current window focus, not merely a background activity. Seven focused harness tests pass. The phone-specific focus query was corrected from `dumpsys window windows` to `dumpsys window`.

Evidence: `test-results/phone-2026-10-13T11-48-53.287Z/`. Three mandatory checks passed (welcome, branding, Begin control), and one Begin tap completed. The next hierarchy retrieval failed. A separate command diagnostic reported `ERROR: could not get idle state`; this is an automation limitation, not a confirmed app defect or completed launch flow. Historical results remain unchanged; future idle-state failures are explicitly classified as blocked. The temporary diagnostic file was removed. No signed-in credentials, recording, journal or sharing mutations were attempted during this phone run.

Coverage remains the last measured baseline: backend statements 51.48%; mobile statements 23.68%, branches 24.10%, functions 17.00% (no measured delta). Launch/onboarding is partial, not certified. Next work is a reliable app-scoped hierarchy strategy for animated screens, followed by navigation and authentication. The user also authorized targeted screen/feature diagnostic logs after testing; those must exclude passwords, tokens, transcripts and personal content.

## Resumed Phone Batch / Animation Blocker Resolved

An independent QA instrumentation helper was built with the installed Android SDK and installed after the user approved USB installation. It reads the active Ode hierarchy without waiting for animation idle, emits only Ode-owned nodes, masks password fields and runs in its own process. It does not collect screenshots, device-wide logs or unrelated files. The runner now prints action-stage diagnostics without typed values.

Successful physical-phone evidence:

| Flow | Assertions | Taps | Evidence Folder |
| --- | --- | --- | --- |
| 01 launch | 15 | 15 | phone-2026-10-13T11-57-39.587Z |
| 02 onboarding | 18 | 19 | phone-2026-10-13T11-58-43.549Z |
| 04 navigation | 23 | 27 | phone-2026-10-13T12-03-08.027Z |
| 13 guest auth negatives | 20 | 29 | phone-2026-10-13T11-58-43.549Z |
| 14 signed-in QA identity / session / history | 20 | 26 | phone-2026-10-13T12-06-45.663Z |
| 15 five recording-mode controls / cancel / reset | 12 | 10 | phone-2026-10-13T12-08-58.808Z |

Total across these successful flows: 108 mandatory assertions, 126 taps. Repeated onboarding assertions are counted as actions, not distinct scenarios. Visual screenshots were not executed. No audio/entry upload occurred in this batch. The QA account signed in, opened/cancelled the recording screen, retained its session after process restart, displayed the expected profile email and traversed history/back navigation.

Retained failures: helper connection startup/read races; a zero-area off-screen Settings Help node; plus-address QA email interpreted as a regex. Fixes: bounded helper startup retry, transient non-owned active-root retries, visible-area selector filtering and escaped expected-email regex. Nine phone-harness tests pass. Original failing reports remain intact, followed by the successful reruns above.

Opt-in app-level screen/onboarding diagnostics were added behind `EXPO_PUBLIC_QA_DIAGNOSTICS=1`, disabled by default. They emit only fixed categories/stages, never IDs, names, credentials or content. Four diagnostic privacy tests and mobile TypeScript checks pass. These source diagnostics are not yet built into the installed APK; test-runner stage logs are already active. CI now includes phone-harness tests, but remote CI has not run here.

The user explicitly authorized labelled QA entries/conversations and normal AI/transcription usage under the tester account. No purchases, account deletion or access to other users' data is authorized. Next controlled-audio/transcription/reply tests remain pending. A fresh consolidated code-coverage run is in progress; do not equate these six phone-flow passes with 70-80% whole-app coverage.

Fresh consolidated run completed: `test-results/2026-10-13T12-09-49.546Z/`. All 1049 named code checks passed (834 backend plus 215 mobile), as did backend vet and mobile/portal TypeScript. Backend statement coverage remains 51.48%; mobile statements 23.84% (+0.16 percentage points), branches 24.32% (+0.22), functions 17.28% (+0.27). Small increases come from the diagnostic-helper tests, not broad new feature coverage. The 70/80/90 milestones remain unmet. The runner exits 1 for coverage gates, not failing code assertions. Its generic phone-functional check remains intentionally blocked by privacy policy; the six separate app-scoped phone runs above provide the functional evidence. Scenario coverage remains unaudited. The new checkpoint is recorded as `phone-launch-auth-record-controls` in the coverage ledger.

## Recording / Live Transcription / Follow-up Batch

Three recording regressions reproduced in focused tests: the screen claimed listening after microphone denial, claimed listening after native capture failure, and issued duplicate start requests for rapid taps. The installed phone also reproduced denial plus listening (`phone-2026-10-13T12-17-32.066Z`, two assertions/four taps before failure). `startRecording` now returns explicit success/failure; the screen enters recording only after success and guards in-flight orb operations. Upload-error restart also checks start success. All 28 recording screen/hook checks pass, including all five upload modes, offline enqueue and cancellation. Focused coverage: record screen 78.64% statements/74.35% branches/80% functions; combined screen/hook 84.07% statements. This is not full recording certification.

QA ARM64 rebuild succeeded in 8m 1s via existing Gradle project, without `make apk`'s clean prebuild/native-file restore. APK 57,988,555 bytes; SHA-256 `9DA665D240BEBDA23E43A64BF107F67A16A9D646AAAFBB4EF2F8E7700F680D12`. Build log: `test-results/recording-fix-qa-build.log`. Installed successfully. Opt-in diagnostics are now in this APK; 13 fixed-category events were verified from Ode's process only and retained in `test-results/ode-qa-screen-events.log`. No device-wide logs or phone screenshots were collected.

| Phone Flow | Assertions | Taps | Outcome / Evidence |
| --- | --- | --- | --- |
| 16 microphone denied | 5 | 5 | Passed; phone-2026-10-13T12-32-53.704Z |
| 17 native capture / cancel / second-mode capture | 8 | 9 | Passed; phone-2026-10-13T12-34-35.187Z |
| 18 synthetic reflection / closed follow-up / return Home | 6 | 3 | Passed; phone-2026-10-13T12-38-25.700Z |

The captured microphone audio was not uploaded. Only Ode's microphone permission/test flags were changed. A first flow-18 attempt failed because scrolling started before the loading transition exposed a scroll container; bounded waiting resolved the harness issue. Original failures remain intact. Earlier six phone flows were on the preceding APK; this is not a fresh nine-flow certification on one build.

Live pipeline: generated labelled English speech locally, converted to AAC mono 44.1 kHz, uploaded 133,354 bytes and created one tester-owned processing entry (`4904e2b1-1ef9-4b11-af98-0dabe886f156`). Initial evidence `qa-pipeline-2026-10-13T12-27-09.243Z.json`: ten checks passed, analysis returned 404 after entry completion, first reply returned 500, remaining stages blocked. Resuming that same entry (no second upload) produced twelve passing checks in `qa-pipeline-2026-10-13T12-30-49.352Z.json`: transcription keyword smoke, reflection schema/safety smoke, conversation creation, empty reply rejection (400), three live AI replies and fourth-turn rejection (409). This is one controlled English smoke fixture, not word-error-rate, multilingual, noise, hallucination or clinical-safety certification. Original failure evidence is retained.

The live failure exposed a backend lifecycle race: worker `SetCompleted` ran immediately after transcription, before analysis persistence. Two new worker tests reproduced early completion and ignored crisis-analysis persistence failure. Local fix adds `SetTranscript` (preserving processing status), marks completion only after successful normal/crisis analysis persistence and propagates crisis-save errors for retry. Full worker suite passes. A real PostgreSQL regression assertion was added, but verification is pending because Docker Desktop's engine was unavailable. No backend deployment was performed; the deployed service still needs the lifecycle fix and a fresh no-race smoke run.

Fresh consolidated evidence `test-results/2026-10-13T12-40-10.586Z/`: 226 mobile tests, backend vet and mobile/portal types passed. Eight repository integration tests failed connecting to isolated localhost PostgreSQL; Docker engine pipe was absent. Installed Docker Desktop was launched hidden, but remained unavailable; user assistance was requested for startup prompts. The run reports 1049 passing named checks and eight environment-caused failures. Mobile whole-source coverage is 26.09% statements (+2.25 points), 26.61% branches (+2.29), 19.29% functions (+2.01). The incomplete backend run measured 48.17%; this is not comparable to the last successful 51.48% backend baseline because real database tests could not execute. The ledger records this as a failed-code-check checkpoint, not a certified coverage gain or 70/80/90 milestone.

Scenario mappings 15P/15N/24P/27P/35P/35N now retain partial evidence and explicit remaining work. All 120 IDs validate; no scenario percentage is invented. Remaining prerequisites: Docker startup for local integration verification, backend deployment approval for the race fix, broader controlled audio corpus, store sandbox for purchase/restore and a therapist QA account. Recording/processing/transcription/follow-up features remain partial.

## History / Search Checkpoint (2026-10-14)

Resumed from the completed `2026-10-13T12-59-29.116Z` run: backend/database checks passed after Docker was restored; one sharing test hit a five-second timeout (1061 named passes, one failure). The isolated sharing suite later passed all 22 tests. Its resolved-mock creation helper now uses explicit async `act` synchronization rather than polling for a state already produced by resolved promises; no blanket timeout increase was applied. Historical failure evidence is retained.

Seven entries-screen regressions were reproduced: completed search results without joined analysis did not navigate; pending entries did not navigate; late searches overwrote newer results; old searches overwrote a cleared-query timeline; scheduled searches ran after unmount; timeline errors implied an empty account; and search errors had no visible retry. Scoped fixes add request-generation/unmount guards, debounce cleanup, visible error/retry states preserving prior data, and status-based routing to reflection/processing. Stable entry-card IDs support the upcoming device assertions. Pull-to-refresh preserves an active search query.

Eleven history tests plus 22 sharing tests pass. Focused entries coverage is 93.61% statements, 86.74% branches, 100% functions. Fifteen regression-harness helper tests pass separately. These are not end-to-end feature certification or distinct full-scope scenarios.

Fresh consolidated run `test-results/2026-10-14T05-31-26.727Z/`: 1073 named code checks passed (836 backend plus 237 mobile), with backend vet and mobile/portal TypeScript passing. Backend statements 51.50%; mobile statements 28.19%, branches 28.94%, functions 20.91%. Versus the prior valid mobile measurement, increases are +2.10, +2.34 and +1.63 percentage points respectively. The ledger's +3.33 backend delta compares against the Docker-failed partial run, not a genuine like-for-like gain; the prior successful backend value was already 51.50%. All 70/80/90 milestones remain unmet. Physical install-metadata collection failed because the phone disconnected during the run; no UI assertions were attempted then. The user reconnected it and restarted Docker; both were subsequently verified.

Live read-only QA evidence `test-results/qa-account-2026-10-14T05-38-17.871Z.json`: 15 checks passed, including eight history/search checks (bounded pagination, known synthetic entry, no-match, Unicode contract, empty/whitespace rejection, invalid ID, owned-entry fetch). No journal, conversation, entitlement or account mutation was made by these checks. Tokens and response bodies were not persisted. This does not certify multilingual transcription accuracy.

A QA ARM64 APK rebuild with history fixes and diagnostics was started (`test-results/history-search-qa-build.log`). Phone use was permitted during code/build work; the user was notified before installation/UI automation. Backend lifecycle deployment, broader audio quality corpus, store sandbox and therapist fixtures remain outstanding.

### Matching-APK Phone Outcome

History QA build completed in 13m 7s and installed successfully. APK 57,989,887 bytes; SHA-256 `595AC4A23E066EDFCF5EFD04D01F24ED154F7FE67CDFF46571F8D7E06A55E9BE`. The user paused UI work, then explicitly resumed it. The earlier attempt (`phone-2026-10-14T05-51-18.815Z`) had seven passing assertions/seven taps before an ADB operation failed; this was not a reproduced app assertion failure. A resumed preflight also briefly failed before any assertions, followed by a successful direct package check. These failures are retained, not erased or counted as passes.

Flow 20 passed on the matching APK: `test-results/phone-2026-10-14T06-00-35.971Z/`, nine mandatory assertions, nine completed taps, zero failed/blocked commands. It verified the approved tester identity before reading history, searched the known synthetic entry, opened its reflection from a search result lacking joined analysis, returned Home, verified a no-match query and navigated Back. No audio, journal/conversation creation, sharing, purchase or account mutation was initiated by the test. Phone interaction stopped after the batch; the user was told the phone was available again.

Code coverage was not recalculated for this UI-only completion: backend statements 51.50%; mobile statements 28.19%, branches 28.94%, functions 20.91%, no new measured delta. This completes the scoped flow-20 checks, not all history scenarios: multi-page UI, device offline/recovery and live cross-account isolation still require broader evidence. The overall 70/80/90 milestones remain unmet.

## Mood Range Fix and Mobile-Only Resume

The Mood range-change viewport reset was reproduced, fixed by separating chart history loading from whole-screen loading, and tested for duplicate requests, stale responses, selected state and error/retry handling. Six focused tests pass. The full mobile retry passes 243 tests across 20 suites, and mobile TypeScript passes. The first full run's AuthSheet timeout is retained; only its resolved-mock synchronization was corrected. Whole-mobile source coverage is now 30.75% statements, 31.28% branches and 23.07% functions, gains of 2.56/2.34/2.16 percentage points. Backend tests were not rerun; historical backend coverage must not be presented as fresh evidence.

The existing Gradle ARM64 release build succeeded in 16m 13s; the matching APK and app-scoped QA helper installed successfully. Thirteen helper privacy/selector tests pass. Phone evidence: `phone-2026-10-14T12-30-49.247Z` passed 21 focused Mood assertions/eight taps without range-scroll recovery; `phone-2026-10-14T12-33-15.858Z` passed nine history assertions/nine taps but flow 19 timed out before initial Home appeared; the unmodified read-only retry `phone-2026-10-14T12-37-41.495Z` passed all 16 flow-19 assertions/13 taps. The startup timeout is retained, not diagnosed as an app defect.

Cumulative cross-build ADB passing-flow evidence remains 14/20 (70%). Flows 05, 06, 07, 09, 10 and 11 remain partial; this is not whole-app scenario or source coverage. Controlled audio upload remains deferred, Play sandbox is unavailable, and linked-session/date/network/native-export fixtures remain necessary for broader mobile certification. No purchases, backend deployment, personal-device file access or device-wide logs were performed. Phone automation is finished for this batch. Detailed evidence and APK hash are in `ODE_MOBILE_ADB_CHECKPOINT.md`; the mobile-only ledger entry has no backend result.

## Parallel Export Regression Work

The user authorized independent parallel testing. Mobile phone scopes completed settings language persistence/restoration, real background/restore, and journey/monthly/yearly export chooser handoff/cancellation: 47 passing assertions/48 taps in the final corrected scopes. ADB remains 14/20 (70%) because the broader flows are still partial. Evidence and each step's coverage update are in ODE_MOBILE_CONTINUATION_2026-10-14.md.

Backend export repository integration added three top-level tests with seven named checks, covering date boundaries, ownership/crisis filtering, weighted aggregates, ordered summaries/quotes, empty/prior-only/current-only periods, malformed emotion shape and cancellation. All 843 whole-backend named checks passed, zero failures/skips; backend vet passed. Fresh baseline and after profiles have the same 6514-statement denominator: 3355 covered (51.50445%) to 3422 (52.53301%), +1.02855 percentage points, 67 new covered statements and none lost. Focused ExportData is 56/64 (87.5%). No backend production change, deployment or live-user mutation was made. Retained diagnostic/setup failures and full evidence are in ODE_BACKEND_PARALLEL_CHECKPOINT.md.

The second agent added 20 mocked ExportScreen tests, all passing with focused 100% statements/branches/functions/lines. They cover success, unavailable sharing, non-200 download statuses, rejection at each async stage, retry and disabled/loading duplicate-tap guards. Two sandbox startup failures and an initial cold-render timeout are retained. Only that initial test uses a bounded 15-second timeout. Focused aggregate mobile coverage is not a whole-suite measurement; parent consolidation is in progress. See ODE_EXPORT_UNIT_CHECKPOINT.md.

Parent consolidation completed: 263/263 mobile tests passed across 21 suites in 185.511 seconds, exit zero; mobile TypeScript passed. Fresh mobile statements 1203/3818 (31.51%), branches 831/2599 (31.97%), functions 260/1101 (23.61%), deltas +0.76/+0.69/+0.54 percentage points versus the preceding full suite. Evidence: export-mobile-consolidated.json, its log and coverage directory. Both agent tasks and parent verification are finished. Backend and mobile were measured in separate fresh runs; no fresh portal type check or whole-product certification is claimed. The phone is not needed for the completed code checks.

## Backend 70-80% Target

The isolated whole-backend integration suite now passes 1070 named tests with zero failures/skips. Statement coverage is 4976/6533 (76.16715%), up 23.63415 percentage points from the valid 52.53301% baseline, and whole-backend `go vet` passes. Six reproduced defects were fixed across relationship retries, chapter date scanning, nullable review narratives, nudge interval SQL, therapist mood-stat error handling, and the recent-user scheduler query. The 70-80% objective is achieved. Exactly 80% would require 251 additional covered statements at the current denominator. Race mode remains unavailable because the environment lacks a CGO C compiler. Full evidence and limitations are in `ODE_BACKEND_70_80_CHECKPOINT.md`.

## Backend Dual 80% Checkpoint (2026-10-15)

Fresh full integration-tag run: 1452 named tests passed, zero failures/skips. Backend statement coverage is 5424/6547 (82.84711%), +6.67995 percentage points from the preceding passing profile; whole-backend vet passed. The independently measured API matrix v3 is 221/255 (86.66667%) with 34 planned cases retained. Requests exercise 80/86 enabled endpoints beyond authentication. Four agents completed notes, worker, storage and independent API work; parent integrated, reproduced/fixed journey and conversation ownership defects plus timezone validation, and retained all earlier failure evidence. The phone was not used. Full feature outcomes and remaining cases are in `ODE_DUAL_80_CHECKPOINT.md`; repeat both gates with `node scripts/regression/run-backend-dual80.mjs` after starting isolated PostgreSQL/Redis.
