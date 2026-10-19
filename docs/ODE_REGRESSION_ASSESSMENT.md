# Ode Regression Automation Assessment

Assessment date: 2026-10-12. This is a source-based feasibility assessment, not an executed regression report. No application tests were run for this assessment and no application behavior was certified.

## Coverage Estimate

Planning estimate: 85-90% of a finite Android functional regression checklist can be automated after the test environment and missing tests are built. ADB-only black-box checks would cover approximately 60-70%; combining Maestro/UIAutomator, backend integration tests, Jest, and audio/AI evaluations reaches substantially further. These are engineering estimates, not measured coverage statistics, and the ranges are not additive.

| Surface | Estimated automatable share | Conditions |
| --- | --- | --- |
| In-app buttons, navigation, forms, settings | 90-95% | Inventory by screen, account role and state; add missing selectors; assert outcomes |
| API rules, validation, ownership and failure handling | 90-95% | Isolated DB/storage/queue, seeded accounts and fault injection |
| Recording/upload/offline/processing lifecycle | 85-95% | Controlled audio and storage/network failures; microphone hardware remains separate |
| Transcription on a declared audio corpus | 85-90% | Human reference transcripts and agreed error thresholds; not all possible speech |
| Reflection, follow-up and therapy mechanics | 90-95% | Stub responses for deterministic mechanics; real-provider contract checks separately |
| Subjective AI quality and crisis safety evaluation | 60-75% | Automated rubric/corpus scoring with expert review for ambiguous and clinical cases |
| Store billing, OAuth, notifications and hardware | 60-80% | Sandbox/store accounts, provider access and real devices |

No defensible percentage exists for "all possible positive and negative scenarios": inputs, devices, networks and AI responses are unbounded. Freeze a scenario inventory first, then report automated cases / total scoped cases. Report execution coverage (executed / total), pass rate (passed / executed), button-state coverage, and code coverage separately. Never count a screenshot, optional assertion, skipped branch or stubbed provider as validation of the live provider.

## Repository Evidence

- React Native/Expo mobile app, Go API/workers and Next.js therapist portal.
- 12 top-level Maestro flow files, reusable subflows and `mobile/src/testIDs.ts` selector registry.
- 794 top-level Go `func Test...` declarations found; this is a source count, not passing-test or code-coverage evidence.
- Two discovered mobile Jest test files: RecordButton and mood-history API client. Recorder, offline queue and therapy UI need broader direct tests.
- `.github/workflows/test.yml` runs Go vet/race tests and mobile/portal TypeScript checks. It does not run mobile Jest or Maestro in that workflow.
- `maestro/flows/05-core-features.yaml` makes processing/reflection-dependent checks optional and conditional. It does not compare captured speech to a reference transcript or check reply content.
- `maestro/flows/06-edge-cases.yaml` denies microphone permission while using a guest; the action opens auth instead of exercising denied-permission handling inside the recorder.
- `mobile/src/config/e2e.ts` supports a seeded-token fast path; the backend supports stub AI analysis. Auth/onboarding must also be tested with the fast path disabled.
- ADB is installed. `adb devices -l` returned no connected devices. Maestro was not found on the current PowerShell PATH; WSL or other installations were not checked.
- Five journal modes: processing, rant, gratitude, decision and dream. Recording limit: 30 minutes. Follow-up limit: three turns. Therapy sessions have a server-enforced one-hour limit, four personas and voice/text input.

## Proposed Regression Cases

Each row defines two initial cases, identified by `P` and `N` suffixes: 60 rows / 120 case outlines. Parameterization creates additional cases. Each outline needs fixture data, precise expected results and assertions before execution. Negative cases below are desired checks, not claims that the current app handles them correctly.

Methods: U = Android UI automation through Maestro/UIAutomator and ADB; A = API/integration; J = Jest; E = live audio/AI evaluation; H = hardware/provider setup or human review. U/A/J are repeatable after setup; E has probabilistic thresholds; H is not fully unattended.

| ID | Area | Positive case (P) | Negative/boundary case (N) | Method |
| --- | --- | --- | --- | --- |
| 01 | Launch | Fresh install reaches onboarding | API unavailable at startup yields recoverable UI | U/A |
| 02 | Relaunch | Existing session returns to intended screen | Force-stop while loading does not corrupt local state | U/J |
| 03 | Version gate | Supported version proceeds | Unsupported version blocks and opens the correct store destination | U/A/H |
| 04 | Onboarding | Complete journal and therapy paths | Back/skip/restart preserves only intended answers | U/J |
| 05 | Profile inputs | Save valid name, country, age and goal | Blank/oversized/Unicode values validate without broken layout | U/A |
| 06 | Terms | Acceptance persists after relaunch | Unchecked consent cannot continue; failed save is handled | U/A |
| 07 | Guest gating | Guest can browse permitted screens | Direct recording/paid/deep-link access prompts authentication | U/A |
| 08 | Registration | Unique valid account registers | Duplicate email, missing name and weak password fail clearly | U/A |
| 09 | Login | Valid account authenticates | Wrong password, malformed email and rate limits are handled | U/A |
| 10 | Social auth | Google sign-in returns to original intent | Cancelled/provider-failed sign-in leaves a usable app | U/H |
| 11 | Session tokens | Valid refresh retains user session | Expired/revoked token does not expose another user's data | U/A/J |
| 12 | Logout/role | Logout and user/therapist switch work | Account switching does not leak cached journal or role state | U/A/J |
| 13 | Navigation | Every tab and back/close control reaches its destination | Rapid switching and nested back do not strand the user | U |
| 14 | Mode gate | Select each of five modes and start recording | No mode selected prevents recording | U/J |
| 15 | Microphone | Authenticated user grants permission and records | Denial/permanent denial/revocation produces usable recovery | U/J/H |
| 16 | Recording | Start/stop produces playable nonzero-duration audio | Instant stop or repeated taps do not submit invalid/duplicate audio | U/J/A |
| 17 | Discard | Cancel discards recording and releases microphone | Back/unmount during recording does not leave capture active | U/J/H |
| 18 | Duration | Recording below 30 minutes submits | At/above 30-minute boundary auto-stop and server validation agree | U/J/A |
| 19 | Interruptions | Background and return follows declared recording policy | Screen lock/call/audio-focus loss does not silently lose audio | U/J/H |
| 20 | Presign/upload | Native binary upload creates a processable entry | Expired URL, missing file, wrong media type and oversized audio fail | U/A/J |
| 21 | Upload retries | Temporary failure recovers within retry policy | Permanent 4xx is not retried; lost response does not duplicate entry | A/J/U |
| 22 | Offline capture | Queued recording survives restart and reconnects | Missing local file or exhausted retry budget is visible to user | U/J/A |
| 23 | Queue concurrency | Multiple recordings flush once each | Concurrent enqueue/flush/reconnect loses no items and creates no duplicates | J/A/U |
| 24 | Processing | Queued entry becomes complete with transcript and analysis | Failed worker/poll timeout offers clear recovery | U/A |
| 25 | Worker retry | Transient provider failure retries successfully | Retry exhaustion reaches failed state/DLQ without hanging | A |
| 26 | Worker idempotency | Redelivered completed job causes no duplicate writes | Analysis failure after transcript persistence can recover correctly | A |
| 27 | English transcription | Clear reference clips preserve words and key entities | Noise/quiet speech scores against declared error thresholds | E/A |
| 28 | Multilingual transcription | Hindi and mixed-language reference clips transcribe | Accent/code-switching/misdetected language is measured, not silently accepted | E/A |
| 29 | No speech | Speech after initial silence remains in transcript | Silence/music/ambient noise does not produce unchecked invented speech | E/A |
| 30 | Audio boundaries | Short/long supported clips return usable transcripts | Empty/truncated/corrupt/unsupported clips produce clear failure | A/E |
| 31 | Transcription provider | Correct multipart payload and response persist language/text | Provider 401/429/500, timeout and malformed JSON are handled | A |
| 32 | Reflection modes | Each mode receives its expected response structure | Wrong-mode fields, invalid JSON and missing required content fail safely | U/A/J |
| 33 | Reflection quality | Reply stays grounded in transcript and mode | Fabricated people/facts, diagnoses and prompt injection are evaluated | E/H |
| 34 | Journal context | Relevant prior entries influence reflection | Another account's history or irrelevant private notes never enter context | A/E |
| 35 | Follow-up | Three ordered user/reply turns persist after reopening | Fourth turn, empty message and closed conversation are rejected | U/A/J |
| 36 | Follow-up failure | Temporary reply failure supports retry | Double send or partial persistence does not duplicate/reorder turns | U/A/J |
| 37 | Therapy persona | Start sessions for comforting/rational/CBT/mindful | Invalid persona and unavailable entitlement do not create usable sessions | U/A |
| 38 | Therapy text | Text reply appears in order and history reloads | Blank/oversized message, 409/410 and send timeout recover clearly | U/A/J |
| 39 | Therapy voice | Record-upload-transcribe-reply completes | Missing/foreign audio key and failed upload/transcription fail safely | U/A/E |
| 40 | Voice/text switch | Switch input modes during an active session | Switch while sending/recording does not lose draft or overlap capture | U/J/H |
| 41 | Therapy expiry | End session and load summary/history | One-hour expiry rejects new messages and repeated end is handled | U/A/J |
| 42 | TTS | Selected language/persona yields playable speech | TTS/playback failure preserves readable text and usable controls | U/A/E/H |
| 43 | Audio focus | Reply ends cleanly before next capture | Bluetooth/headset/speaker changes and interrupted playback avoid overlap | U/H |
| 44 | Crisis mechanics | Benign input follows normal path | Seeded crisis inputs follow warning/hard-stop and screener fail-safe policy | U/A/E/H |
| 45 | Safety UI | Relevant resources render and link intent is correct | Missing country, unavailable handler and indirect crisis language are checked | U/A/E/H |
| 46 | History/search | Entries paginate/search and open correct reflection | Empty/no-match/Unicode queries and another user's IDs are handled | U/A |
| 47 | Mood analytics | Seeded records yield expected chart/range statistics | Empty data, date boundaries and malformed data do not mislead or crash | U/A/J |
| 48 | Habit loop | Streak, flashback and scheduled check-in render correctly | Timezone/DST/missed-day/duplicate scheduler run preserve rules | U/A/J |
| 49 | Notifications | Test push opens intended entry/session | Denied permission, stale token and deleted target recover | U/A/H |
| 50 | Guided journeys | Start, record and advance a seeded journey | Invalid session, incomplete previous step and retry preserve progress | U/A |
| 51 | Relationships/reviews | Seeded extraction/map and weekly/yearly reviews agree with fixtures | Empty data, duplicate people and provider failure are handled | U/A/E |
| 52 | Sharing/consent | Create permitted share and accept/reject therapist request | Wrong passcode, expired/revoked link and revoked consent prevent access | U/A |
| 53 | Export | Date-filtered journal export has correct content | No records, offline failure and unauthorized data access are handled | U/A/H |
| 54 | Settings | Goal/nudge/time/language/theme persist after restart | Save failure, invalid values and account switch preserve correct settings | U/A/J |
| 55 | Billing | Sandbox purchase grants correct verified entitlement | Cancelled/forged/replayed receipt or store outage grants no false plan | U/A/H |
| 56 | Entitlements | Plan limits/expiry and implemented purchase recovery behave correctly | Duplicate callback/concurrent use/expired plan cannot bypass limits | U/A/H |
| 57 | Therapist registration | Required professional fields and consent create workspace | Missing fields/consent and non-therapist access are rejected | U/A |
| 58 | Clients/notes/OCR | Add client, save/edit note and process seeded image | Invalid image, OCR failure, unauthorized client and bad encryption key fail safely | U/A/E |
| 59 | Therapist archive/portal | Archive/restore clients and browse permitted web records | Consent withdrawal, direct URL access and role mismatch block private data | U/A/browser |
| 60 | Account deletion | Disposable account and associated data follow deletion policy | Cancel or backend partial failure does not falsely report completion | U/A |

## Audio and Reply Evaluation

Use owned/synthetic fixtures with reference transcripts for clear English, Hindi, mixed language, accents, quiet speech, background noise, silence, repeated phrases, names/numbers, short and long clips. Measure word error rate and key-entity accuracy; agree thresholds per language/noise class before calling a run successful. Don't require exact punctuation or exact live AI reply strings.

For replies, validate required schema, nonempty text, language, mode/persona, grounding, turn order, history/context, no cross-user data, and crisis-policy behavior. Use fixed responses for mechanics, and a smaller real-provider corpus for quality. Repeat probabilistic evaluations and retain model/prompt versions and results. Expert review remains necessary for clinical appropriateness and ambiguous safety judgments.

ADB can tap, type, scroll, press Back/Home, change supported device state, dump UI, capture screenshots/logcat and inspect app state. It cannot universally inject a WAV into a physical phone's microphone through a standard command. API fixture uploads test downstream audio processing; a test-only audio source or validated emulator audio routing tests the app path; real microphone/speaker/headset tests need hardware setup. Keep these coverage claims distinct.

Do not dial live emergency resources in automated runs. Validate link intents with a test handler. Use store sandbox purchases and disposable accounts for destructive cases. Android ADB does not cover iOS/Apple sign-in/Apple Health; those need a separate iOS runner and device matrix. The web therapist portal needs browser automation, not ADB.

## Efficient Execution Plan

1. Capture baseline test results and Go coverage; enumerate screens and every interactive control by role/state, including dialogs and dynamic rows.
2. Connect a dedicated Android emulator/device with the intended build. Check Maestro/WSL availability, backend reachability and seeded journal/therapist accounts.
3. Build isolated seeded DB/queue/storage and deterministic provider fixtures. Add controlled timeout/429/500/corrupt-response failures.
4. Convert critical optional checks into mandatory assertions. Assert UI result plus API/DB persistence; attach screenshot, logcat and request correlation IDs on failure.
5. Add focused Jest tests for recorder, upload, offline queue and session UI; integration tests for actual repositories/migrations/queue/storage, beyond mocks.
6. Run short smoke checks per change, deterministic regression in CI/nightly, and live-provider/audio corpus plus hardware/store checks on a separate cadence.
7. Publish passed/failed/blocked/skipped cases with exact denominator and environment. Record button-state coverage separately from business-logic and AI-quality coverage.

Most routine regression execution should consume no Codex tokens: run scripts/CI directly. Ask Codex to create/update tests and investigate failures; supply only relevant logs, changed files and failed screenshots. Avoid agent-driven coordinate clicking for every repeated run; stable selectors and checked-in scripts are more repeatable and economical.

## Codex Model Recommendation

GPT-6 Luna is the efficient choice for scoped test additions, existing flow maintenance and failure triage. Start with the client's recommended/default reasoning setting; reduce effort for mechanical tasks and increase only when needed. GPT-6.1 Sol is the better single-model choice for designing and debugging this multi-layer suite, especially voice/queue/session interactions. Use Sol for initial architecture and difficult failures, Luna for bounded maintenance, and ordinary test runners for execution.

Model cost and token count are different: a cheaper model does not guarantee fewer tokens, and higher reasoning/repeated retries can raise usage. API per-token pricing is separate from Codex subscription credits; account/client availability must be checked in the model picker.

Official sources checked on 2026-10-12:

- https://learn.chatgpt.com/docs/models
- https://developers.openai.com/api/docs/guides/model-selection
- https://developers.openai.com/api/docs/models/gpt-6-luna
