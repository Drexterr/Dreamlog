# Ode Regression Runner

Run from the repository root:

```powershell
node scripts/regression/run.mjs
```

Select a device/build explicitly when needed:

```powershell
$env:ODE_ADB_SERIAL = 'emulator-5554'
$env:ODE_APP_ID = 'com.talktoode.app'
node scripts/regression/run.mjs
```

Each run writes timestamped logs, backend coverage, Jest results/coverage, per-test outcomes and a summary under `test-results/`. The Android stage checks installation, launches the app and captures UI/screenshot/crash evidence. It does not reset app data, change permissions, buy products or delete accounts. Functional UI coverage remains blocked until real feature assertions run.

Exit codes: 0 = checks passed, 1 = failures, 2 = blocked prerequisites. A 0 does not imply 90% scenario coverage. Keep full-scope scenario coverage distinct from unit-test pass rate.

Install the newest local APK with an architecture check:

```powershell
./scripts/regression/install-apk.ps1 -Serial emulator-5554
```

Use `-NoStreaming` when streaming installation stalls. The ABI guard still applies; a physical ARM64 phone requires an ARM64 build. An unavailable/unauthorized target stops installation before transfer.

The installer refuses ARM-only APKs on x86_64 emulators. The August 17 APK is ARM64-only. Build a current-source emulator variant with the helper, then install it:

```powershell
./scripts/regression/build-emulator.ps1
./scripts/regression/install-apk.ps1 -Serial emulator-5554
./scripts/regression/run-ui.ps1 -Serial emulator-5554
```

The build helper uses a local Java socket workaround and disables Sentry source-map upload for this test build. It keeps native compile caches. The current source package is `com.talktoode.app`; older APKs use `com.ode.app`. `run-ui.ps1` defaults to launch, onboarding and navigation; these flows clear only the app under test on the explicitly selected target. Use a dedicated emulator. Results, screenshots and command evidence are saved per feature.

The authorized-account read-only flow requires runtime credentials:

```powershell
./scripts/regression/run-ui.ps1 -Flows @('14-signed-in-account') -UseSeededAccount
```

`ODE_QA_EMAIL` and `ODE_QA_PASSWORD` override the existing fixture values. The runner restores these environment values when it finishes and redacts the password from console output and generated text artifacts before summarization. Do not commit new credentials. Binary screenshots are not redacted; keep screenshots out of credential-entry steps. This flow signs in, checks session persistence/backend identity and opens history; it creates no journal, share link or purchase. Backend/database fixture tests do not replace signed-in device tests.

For separately scoped live QA account/API diagnostics:

```powershell
node scripts/regression/check-qa-account.mjs
```

This reads the configured HTTPS backend/Supabase endpoints, authenticates the authorized account, and performs only read-only backend requests. Tokens remain in memory; reports contain status/contract checks, not passwords, tokens or entry contents. It does not establish the native app's session or replace UI/provider-quality testing. The current app configuration points to a production-named backend; do not extend this diagnostic to journal creation, account deletion or purchases without checking the intended test scope.

Add `--history` to include bounded timeline/search checks for the synthetic QA entry identified in flow 20. The checks verify result ownership, known/no-match/Unicode queries and invalid/empty input responses. They remain read-only and retain no entry contents. A Unicode search contract pass is not transcription-quality evidence.

The user separately authorized labelled QA entries/conversations for the tester account. `create-audio-fixture.ps1` generates local synthetic English speech and AAC metadata. `node scripts/regression/check-qa-pipeline.mjs --allow-qa-mutations` creates at most one synthetic entry and sends at most three follow-up turns. It verifies QA identity, bounded fixture size/duration and entry ownership; tokens and response bodies are not saved. Reports retain stage outcomes and the created QA entry ID. `--resume-entry=UUID` reuses a tester-owned entry without another upload; use only to continue an unfinished QA run. These are live smoke tests, not full speech/AI quality certification. Do not run with other accounts, purchase, delete accounts or touch other users' data. QA records are retained for inspection.

Physical-phone privacy: the default Maestro runner is emulator-only because its evidence includes device-wide logs. `run.mjs` does not collect generic screenshots/crash buffers on physical devices; it checks only the selected app's install metadata and marks functional checks blocked. Use a verified app-scoped path before phone automation. Do not browse/copy unrelated phone files or collect full-phone bug reports.

For the dedicated phone, build `scripts/regression/build-phone-helper.ps1`, install the resulting `test-results/phone-helper-build/ode-qa-helper.apk`, and run `node scripts/regression/phone-ui.mjs SERIAL 01-app-launch --helper`. Add `--seeded` only for the authorized QA-account flow. The helper runs in its own process, bypasses idle-state waits, and emits only Ode-owned nodes with password fields masked. The runner checks Ode current focus before input, saves app-only hierarchy evidence instead of screenshots, and logs command stages without typed values. Extra USB debugging security and USB installation permission may be required. Zero-area nodes are not valid selectors. Visual inspection is not certified by hierarchy-only evidence.

Optional app logs: build a QA APK with `EXPO_PUBLIC_QA_DIAGNOSTICS=1`. `[ODE_QA]` records fixed feature categories and onboarding stage names only; dynamic route IDs, names, credentials and content are excluded. Disabled by default. These source logs require a rebuild; runner stage logs are available immediately. Do not collect unfiltered device logs to retrieve them.

Per-feature coverage: `node scripts/regression/coverage-checkpoint.mjs FEATURE test-results/RUN/results.json` appends measured percentages, deltas and separate 70/80/90 gates to `test-results/coverage-checkpoints.json`. The script records a code-coverage checkpoint, not feature completion or device/provider certification.

Enable real database/queue tests using isolated containers:

```powershell
docker compose -p ode-regression -f scripts/regression/compose.yml up -d --wait
$env:ODE_TEST_DATABASE_URL = 'postgres://ode_test:regression-local-only@127.0.0.1:54331/ode_test?sslmode=disable'
$env:ODE_TEST_REDIS_ADDR = '127.0.0.1:63791'
node scripts/regression/run.mjs
```

Repository tests apply all migrations into a unique schema per test and remove only that schema afterwards. They require a loopback database URL whose database name ends in `_test`. Redis tests use unique keys, never FLUSHDB. These services are bound to localhost and do not reuse the app's production configuration.

`scenarios.json` retains 120 full-scope positive/negative outlines. `create-manifest.mjs` seeds it once and refuses to overwrite an existing manifest. `inventory-controls.mjs` scans TypeScript JSX with the TypeScript parser and emits a source-control inventory under `test-results/control-inventory/`; source inspection never counts as a successful click.

The runner measures full backend/mobile source coverage and reports each 90% gate separately. A run can have all assertions passing while coverage targets remain red. Feature grouping organizes named tests, not end-to-end feature certification. Real-provider audio quality, hardware, store and iOS evidence remains separate from mocks.

## Backend White-Box and Black-Box 80% Gates

Run just the backend without a phone or mobile build:

```powershell
make test-backend-regression
```

Docker Desktop must be running. The target starts the isolated PostgreSQL/Redis
services, waits for healthy status, then invokes `run-backend-dual80.mjs`.

This runs the full integration-tag backend suite, statement coverage and vet,
then evaluates the versioned `backend-blackbox-matrix.json`. It exits successfully
only when the suite/vet pass and both independent percentages reach 80%.
Artifacts are retained in a fresh `test-results/backend-dual80-*` directory.

Black-box cases use a real loopback HTTP server with the production router,
authentication, services, PostgreSQL repositories and Redis queue. Synthetic
accounts/data use unique local schemas. AI/storage boundaries use controlled
responses; the matrix retains planned real-provider/store/outage scenarios.
The protected-route authentication sweep counts once. Endpoint reach beyond
authentication is reported separately. Failed feature parents prevent their
child checks from earning completion credit.

Regenerate a report for retained Go JSON output with:

```powershell
node scripts/regression/backend-blackbox-report.mjs test-results/backend-dual80-final.jsonl
```

The matrix refuses a second freeze. `--extend` adds newly declared scenarios
without removing earlier ones and records the preceding version/count.
