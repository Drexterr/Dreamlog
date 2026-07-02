# DreamLog — Maestro E2E Test Suite

End-to-end UI automation for the DreamLog mobile app (React Native / Expo,
`com.dreamlog.app`) using [Maestro](https://maestro.mobile.dev).

```
maestro/
├── config.yaml                      # global config: appId, flow glob, hooks
├── data/
│   └── test-data.yaml               # env-var test data (accounts, inputs)
├── subflows/                        # reusable building blocks (runFlow only)
│   ├── complete-onboarding-guest.yaml
│   ├── open-auth-sheet.yaml
│   └── sign-in-email.yaml
└── flows/                           # top-level scenarios (run by `maestro test`)
    ├── 01-app-launch.yaml
    ├── 02-onboarding.yaml
    ├── 03-authentication.yaml
    ├── 04-navigation.yaml
    ├── 05-core-features.yaml
    └── 06-edge-cases.yaml
```

## Stable selectors (testIDs)

Every interactive element the suite touches now carries a stable `testID` from
the central registry `mobile/src/testIDs.ts` (and an `accessibilityLabel`).
Flows target these via Maestro's `id:` matcher, so wording changes never break a
test. Bottom tabs use `tabBarButtonTestID` (`tab-home`, `tab-explore`,
`tab-mood`, `tab-settings`). List rows are keyed by enum value, e.g.
`onboarding-goal-stress`, `record-mode-processing`, `persona-card-comforting`.

## E2E fast path (skip onboarding + sign-in)

Build (or start Metro) with the E2E flag to boot straight onto an authenticated
Home tab — turning the ~20-step onboarding+auth preamble into an instant launch:

```bash
EXPO_PUBLIC_E2E=1 \
EXPO_PUBLIC_E2E_TOKEN="<supabase access token for a seeded test user>" \
npx expo start
```

The flag is inert unless set (never true in production builds). See
`mobile/src/config/e2e.ts`. For deterministic reflection/therapy content, point
that build's `EXPO_PUBLIC_API_URL` at a backend running `STUB_AI_ANALYSIS=true`.

## Prerequisites

1. **Install Maestro** — `curl -fsSL https://get.maestro.mobile.dev | bash`
   (Windows: use WSL, or the PowerShell installer per the docs).
2. **A running device/emulator**: Android emulator (API 30+) or iOS simulator.
3. **A build of the app installed** on that device. Maestro drives an installed
   binary — it does not build. Produce one with EAS or a local dev client:
   - Android: `eas build --profile preview --platform android` then `adb install`.
   - The bundle id must be `com.dreamlog.app` (matches `config.yaml`).
4. **Backend reachable** for `05-core-features.yaml` (entry processing, therapy,
   mood). Point the installed app at your backend via `EXPO_PUBLIC_API_URL`
   baked into the build (see `mobile/CLAUDE.md`). UI-only flows (01–04, 06) do
   not require a backend.
5. **A seeded, verified account** for the sign-in and feature flows — see
   `data/test-data.yaml` (`EMAIL_EXISTING` / `PASSWORD`). The launch checklist
   §2d documents the canonical tester account.

## Running

```bash
# From the repo root. Runs every flow in flows/ (subflows are excluded).
maestro test maestro/

# A single scenario
maestro test maestro/flows/02-onboarding.yaml

# Inject / override test data (recommended: keep secrets out of the file)
maestro test --env-file maestro/data/test-data.yaml maestro/flows/03-authentication.yaml

# One-off overrides (unique email per CI run avoids 409 conflicts)
maestro test -e EMAIL_NEW="qa+$(date +%s)@dreamlog.dev" maestro/flows/03-authentication.yaml

# Run only smoke-tagged flows
maestro test --include-tags smoke maestro/

# Interactive selector explorer — invaluable for fixing a broken locator
maestro studio
```

Screenshots from every run land in `~/.maestro/tests/<timestamp>/`. Each flow
also calls `takeScreenshot` at key checkpoints and on completion (via the
`onFlowComplete` hook in `config.yaml`).

## CI

Maestro Cloud or a Linux runner with an Android emulator:

```bash
maestro test --format junit --output maestro-report.xml maestro/
```

`--format junit` emits a report most CI systems can ingest. Gate the pipeline on
the launch + onboarding + navigation flows (deterministic, no backend); treat
`05-core-features` as a nightly/backend-integration job.

## Tags

| Tag         | Flows                          | Needs backend | Needs account |
|-------------|--------------------------------|:-------------:|:-------------:|
| `smoke`     | 01                             | no            | no            |
| `launch`    | 01                             | no            | no            |
| `onboarding`| 02                             | no            | no            |
| `auth`      | 03                             | partial¹      | optional²     |
| `navigation`| 04                             | no            | no            |
| `features`  | 05                             | **yes**       | **yes**       |
| `edge`      | 06                             | no            | no            |

¹ Wrong-credentials assertion hits Supabase; happy-path sign-in is `optional`.
² Sign-in + persistence blocks in 03/05 are guarded and skip cleanly without a
  live account.

A per-scenario explanation, coverage gaps, and app-testability recommendations
were delivered with this suite (see the PR / handover notes).
