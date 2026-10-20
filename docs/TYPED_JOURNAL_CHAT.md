# Typed journaling and therapy chat — 20 October 2026

## Behavior

- Recording keeps its existing voice flow and offers **Write journal** in the header.
- The writing screen accepts the whole journal in one message, with the same five
  modes and a 20,000-character limit. It explains that the resulting reflection
  supports three follow-up exchanges. The initial journal is not one of those turns.
- Typed journals enter the same processing/reflection flow as voice journals. The
  existing three-turn and crisis closures are unchanged. No new AI prompt or
  unrestricted conversation endpoint was introduced.
- Failed/offline text stays in memory for retry; it is not silently queued or stored
  unencrypted on the device. Cancel/switch-to-voice and Android Back protect an
  unsent draft. Signing out clears it and invalidates pending callbacks.
- Therapy stays in text chat after a reply, preserves a failed message, and returns
  to voice only when chosen. Typed replies do not auto-play TTS. Chat switching is
  disabled while recording/uploading/sending, so it cannot hide a live microphone.

## Backend

`POST /entries/text` accepts content, mode and a user-scoped UUID retry token.
The entry's UUID is deterministic within that user, and `INSERT ... ON CONFLICT`
makes concurrent identical retries converge on one row. A conflicting payload is
rejected. Existing retries resolve before quota checks and do not enqueue again.
Text is stored as the transcript; the reserved `text/` audio-key namespace does
not represent an uploaded object. The existing worker skips transcription and
still performs crisis screening, context construction, AI analysis and persistence.
Lost enqueues use the existing reconciler. No database migration is needed.

See `docs/API_CONTRACT.md` and `postman/journaling.openapi.yaml`.

## Verification

- Mobile: full **29 suites / 304 tests passed**. Final auth-generation/draft-cleanup
  refinement: all six writing-composer tests passed again.
- Backend: `go test ./...` passed, including models, handlers, services and workers;
  `go vet ./...` passed. The last quota-boundary test is included in the focused
  backend rerun.
- TypeScript: `npx tsc --noEmit` passed.
- Postman: the real mobile Axios client passed create/read, identical retry,
  conflicting retry, invalid input and missing-entry checks over local HTTP. The
  simulation request log confirmed these requests reached the modeled service.
- Added a real-PostgreSQL integration test for eight simultaneous retries,
  cross-user isolation and conflicting input. Its integration target compiles.
  Docker is not running locally; execution belongs to the existing PostgreSQL
  integration/race CI gate. No fake database substitutes for that check.
- Added a read-only Maestro writing-screen/draft-discard flow. No Android device
  is connected, so native interaction and visual/device checks remain unverified.

## APK

Internal Android release: **1.0.1 / versionCode 2**, ARM64, using the project's
existing debug signing configuration. This is an installable test APK, not a Play
Store signing artifact. E2E auth bypass is disabled. Runtime version 1.0.1 isolates
it from older preview OTA bundles. API host remains the existing production host.
Android release lint exhausted the prior 512 MB JVM metaspace limit; the build
setting is increased to 1536 MB. No lint or compiler checks are disabled.

Typed journals require the updated backend endpoint to be deployed; the APK alone
does not update a running server. The code push uses the repository's existing CI
and deployment gates; no separate production deployment is performed here.
