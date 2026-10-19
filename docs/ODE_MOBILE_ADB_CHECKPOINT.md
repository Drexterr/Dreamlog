# Ode Mobile ADB Checkpoint

Date: 2026-10-14. This is a mobile-only continuation on the installed history-fix QA APK. No backend regression suite, production deployment, APK rebuild, purchase, real recipient sharing, or account deletion was performed.

## Completion Rules

- A flow is complete only when all its declared mandatory assertions pass. A feature can still have additional scenarios outside that flow.
- Missing required fixture blocks, skipped optional assertions, and scoped continuations are marked partial, not passed.
- Flow execution coverage uses the existing 20-flow denominator. It is not source-code coverage or all-button/state coverage.
- Before this batch: 11/20 flows had a passing physical-phone run (55%). Authentication flow 03 increased this to 12/20 (60%); therapist portal flow 08 to 13/20 (65%); dashboard/archive flow 12 to 14/20 (70%). Runs still span APK builds.
- The narrow ODE-09P valid-login U/A outline is now signed off using existing live API identity evidence plus the successful phone authentication run. This audits 1/120 outlines (0.83%), not the full app. Other manifest outlines remain unclosed or only partially mapped.
- Source coverage is unchanged by these uninstrumented UI runs: mobile statements 28.19%, branches 28.94%, functions 20.91%; backend statements 51.50%.

## Current Mobile Results

| Flow | Status | Verified scope | Remaining scope / prerequisite |
| --- | --- | --- | --- |
| 03 Authentication | Complete scoped flow; passed again after interruption | Required/short-password validation, wrong-password rejection, dismissal, approved identity, successful sign-in and cold restart | OAuth, expired/revoked tokens, account switching and wider auth fault cases are outside this flow |
| 05 Core features | Partial; 19 available assertions passed | Goal/hour modal cancellation, language search without saving, profile identity, all five record modes and cancellation | User deferred controlled audio upload; preference save/rollback and live therapy messaging remain |
| 06 Edge cases | Partial; 21 available assertions passed | Guest gate/dismiss, keyboard handling, empty surfaces, rapid tabs, process restart | App-scoped offline recovery, permission changes and true OS background/restore remain |
| 07 Habit loop | Partial; 8 final assertions passed | Adaptive nudge toggle off/on and synthetic-entry check-in | Aged-entry flashback and push delivery fixtures remain |
| 08 Therapist portal | Complete scoped flow: 41 assertions and 50 taps | Verified disposable identity, role/workspace, blank-client rejection, client creation, typed notes, bullet edit, live AI summary, session/client deletion and journal return | OCR/camera, linked-client consent, broader failure/quality cases and stronger reload-based persistence checks remain outside this flow |
| 09 Mood/billing | Partial feature; range regression fixed and verified | Six focused code tests and 21 matching-APK phone assertions passed; range changes preserve the viewport and selected state | Streak-freeze fixtures, store setup and broader statistics/error cases remain |
| 10 Explore/share/export | Partial | QA journey start, relationship surface, synthetic-entry share creation/revocation; export period controls | Populated relationship actions, open follow-up, native PDF completion/recipient handling remain |
| 11 Therapy hub/consent | Partial; 13 final assertions passed | Hub, pricing, single-session CTA absence, Pro CTA, persona navigation, therapist requests/back | Active-session resume and explicitly linked disposable approve/decline fixtures remain |
| 12 Therapist dashboard/archive | Complete scoped flow: 43 assertions and 51 taps | Verified identity/role, metrics, dashboard-started typed notes, session/client deletion, archive cancel, confirmed archive | Restore, archived-session history preservation, linked clients and web portal authorization remain outside this flow |

Existing passing flows 01, 02, 04 and 13-20 remain historical evidence; this batch does not automatically recertify them on the newest APK.

## Retained Failures and Corrections

### Open App Issue

The Mood range viewport-reset issue is now fixed and verified on the matching APK. Originally reproduced in phone-2026-10-14T07-31-11.301Z, range selection changed the loadData callback dependency and retriggered the full-screen loader, alongside a duplicate history fetch. History loading is now independent, stale responses are ignored, and errors have a retry state. The new phone check verifies all three selected ranges and visible controls without post-range scrolling. Historical failure evidence remains retained; wider Mood/billing scenarios remain partial.

### Retained Test Failures

- Flow 07 first timed out at a 20-second sign-in wait while the form was still loading. The bounded sign-in wait was aligned with the established 60-second QA flow.
- Check-in succeeded but its label included a checkmark absent from the old exact selector; corrected and rerun successfully.
- Share creation succeeded; revoke confirmation failed because Android rendered the action uppercase. The new QA link was subsequently revoked with the app-owned native confirmation button. No other link existed before creation; no unrelated link was revoked.
- Therapy pricing used an uppercase heading; the test selector was corrected and the available checks passed on rerun.
- A new core-controls test initially used incorrect mode IDs. They were corrected to the existing decision/dream IDs, and all 19 available assertions passed.
- During session restoration, email verification opened Ode's callback screen. The user confirmed this was their verification action and requested a rerun. The runner now uses an explicit MAIN/LAUNCHER intent; the affected authentication/core-control batch passed afterward. This interruption is retained, not reported as an app defect.
- Therapist fixture names contained parentheses, which the old selectors interpreted as regex syntax. The clients were created successfully; selectors were escaped, and the original dummy clients were removed after explicit list loading.
- Two immediate post-add interactions did not navigate: opening a client and Android Back to the dashboard. A bounded 1.5-second modal settle and a parent-row selector allowed flow 08 to pass on a fresh run. A transition/targeting issue is suspected, not a proven production-code root cause. No app source fix or new APK was made for this batch.
- Dashboard-started session deletion correctly returned to the dashboard, not client details. Flow 12's route expectation was corrected; it now opens the specific new client for cleanup and passed in full. Archive cancellation and confirmation both passed.

## Evidence

| Saved folder | Outcome |
| --- | --- |
| phone-2026-10-14T06-37-19.407Z | Flow 03 passed: 29 assertions, 40 taps |
| phone-2026-10-14T06-40-42.627Z | Flow 07 sign-in timeout retained |
| phone-2026-10-14T06-43-31.454Z | Flow 07 success-label selector mismatch retained |
| phone-2026-10-14T06-45-57.680Z | Flow 07/09 partial; 16 combined assertions, 13 taps |
| phone-2026-10-14T06-48-56.625Z | Flow 10/11 selector failures retained |
| phone-2026-10-14T06-52-10.660Z | Share cleanup/export controls: 11 assertions, 11 taps; scoped partial |
| phone-2026-10-14T06-53-56.178Z | Flow 11 partial passed scope; flow 05 old-ID failure; flow 06 partial passed scope |
| phone-2026-10-14T06-57-35.273Z | Disposable signup submission: 4 assertions, 7 taps; preparation only |
| phone-2026-10-14T06-58-46.604Z | Verification interruption during restoration retained |
| phone-2026-10-14T07-00-21.231Z | Flow 03 passed and flow 05 partial: 48 combined assertions, 56 taps; zero failed/blocked commands |
| phone-2026-10-14T07-03-40.566Z | Therapist names-regex failures retained; account, role and registration setup passed |
| phone-2026-10-14T07-08-23.136Z | Cleanup cold-start route assumption failed before any assertion |
| phone-2026-10-14T07-10-03.882Z | Initial cleanup skipped names before async list loading; not credited as actual cleanup |
| phone-2026-10-14T07-11-27.380Z | Original two dummy clients confirmed deleted: 15 assertions, 11 taps |
| phone-2026-10-14T07-12-28.560Z | Two post-add navigation failures retained |
| phone-2026-10-14T07-18-46.085Z | B-run dummy clients confirmed deleted: 15 assertions, 11 taps |
| phone-2026-10-14T07-19-52.560Z | Flow 08 passed: 41 assertions, 50 taps; flow 12 failed on an incorrect post-delete route expectation |
| phone-2026-10-14T07-25-45.882Z | Flow 12 passed: 43 assertions, 51 taps; zero failed/blocked commands |
| phone-2026-10-14T07-29-26.473Z | Leftover zero-note C client confirmed deleted: 9 assertions, 8 taps |
| phone-2026-10-14T07-31-11.301Z | Main flow 14 passed; flow 09 failed when range selection reset chart visibility |
| phone-2026-10-14T07-37-34.429Z | Flow 09 partial: 16 assertions/13 taps; range controls required viewport recovery; no purchase |

All paths above are under test-results. Phone evidence is app-owned UI only, with password/share secrets redacted. No phone screenshots, device-wide logs, personal files or inbox access were collected. The signup account is bharatbanthia2207+odeqa-therapist-20261014@gmail.com; credentials are not copied into this report or its new fixture scripts.

## Resumed Mood Fix

- Reproduction: mood-reproduction.json retained one passing and five failing tests; range selection removed the scroll view, selected-range taps refetched needlessly, and history errors had no retry state.
- Scoped source fix: separate chart history from the mount-only screen loader, retain the scroll view, ignore superseded/unmounted responses, and expose loading/error/retry plus selected-range accessibility state.
- mood-fixed.json: all six focused tests passed. Mobile TypeScript passed.
- First full mobile run: 242 passed, one AuthSheet resolved-mock test hit five seconds. Its assertion now uses explicit async act rather than polling; the original report is retained. Retry passed all 243 tests: mood-mobile-regression-retry.json.
- New whole-mobile coverage: statements 30.75%, branches 31.28%, functions 23.07% (increases of 2.56/2.34/2.16 percentage points versus the prior valid mobile run). Focused Mood coverage is 52.79% statements, 46.81% branches, 44.64% functions. Backend tests were not rerun. See mood-mobile-summary.json.
- QA helper now exports only the selected boolean in addition to existing Ode-owned metadata; all 13 privacy/selector guard tests passed.
- mood-qa-build.log: ARM64 build succeeded in 16m 13s using the existing Gradle project, offline, without clean prebuild or backend changes. APK size 57,990,823 bytes; SHA-256 7CD00ACFAF1DE2D3C37542D9F482493C7383A7DFDA0C4C992AD9CD3B73214F2C. App and updated helper installed successfully.
- phone-2026-10-14T12-30-49.247Z: all 21 scoped assertions/eight taps passed, zero failed/blocked commands. No post-range scroll recovery was used. The runner labels this scoped subflow partial to avoid certifying the entire flow 09.
- Flow 09 stays partial because store/streak and other full-feature scenarios are still unclosed. Cumulative passing-flow coverage remains 14/20 (70%), not whole-app coverage.
- Matching-APK follow-up: phone-2026-10-14T12-33-15.858Z passed history/search (nine assertions/nine taps). Flow 19 timed out waiting for initial Home, before assertions or taps; its failure UI was empty. The read-only retry in phone-2026-10-14T12-37-41.495Z passed all 16 assertions/13 taps without changing the test or app. This is a retained, non-reproduced startup interruption, not a diagnosed production defect.
- This batch is finished; the phone is available for normal use. Audio upload remains deferred at the user's request, no purchase was made, and backend tests were not run.

## Settings, Lifecycle and Export Continuation

The subsequent authorized continuation passed three corrected scopes on the installed Mood-fix APK: settings language persistence/restoration (16 assertions/16 taps), real OS background/restore (11/11), and journey reopen plus monthly/yearly native export handoff/cancellation (20/21). Total 47 assertions/48 taps, zero failed/blocked commands in those final scopes. Earlier settings failures are retained: one selector matched the search field, and one post-restart scroll overshot the row. Hindi persisted; Auto was restored and reverified. No app production code or APK changed in this continuation.

ADB passing-flow coverage remains 14/20 (70%) because flows 05, 06 and 10 still have broader unclosed cases. Mobile source coverage remains 30.75% statements, 31.28% branches and 23.07% functions; these UI runs are not source-instrumented. Native chooser handoff does not certify PDF contents, and a matching in-progress journey row does not prove exact session-ID identity. Phone automation is finished. Per-step evidence and coverage updates are in ODE_MOBILE_CONTINUATION_2026-10-14.md. Backend tests are now delegated independently at the user's request and will have a separate checkpoint.
