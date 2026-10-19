# Ode Device ANR Investigation

Date: 2026-10-13. Status: unresolved; not a cleared app failure and not proof of invalid credentials.

## Observed Run

- Device: emulator-5554, Pixel_9, x86_64, API 37 system image with 16 KB pages.
- Emulator configuration: 2048 MB RAM, four CPU cores, GPU auto. Only the API 37 image is installed locally.
- App: com.talktoode.app, version 1.0.0, successfully rebuilt/installed October 13. Later recorder/reflection/sharing source fixes are not in this APK.
- Flow: 14-signed-in-account. Evidence: `test-results/ui-2026-10-13T12-20-32/`.
- Outcome: failed after 10m 34s at locating auth-password-input. Eleven mandatory assertions and 17 taps completed; password submission and signed-in assertions were not reached.
- Failure screenshot shows the Android launcher. The app process was no longer running after the flow.

## Evidence

Captured device log:

- 12:26:59: Ode focus-event input dispatch exceeded the ANR timeout.
- 12:27:58: ActivityManager reported ANR in com.talktoode.app, waiting 5001 ms for FocusEvent(hasFocus=false).
- 12:28:00: MainActivity was force-finished.
- 12:28:07: Ode process was killed as a user request after error.
- Subsequent launcher, taskbar and gesture-monitor ANRs were also logged. This indicates broader emulator instability, but does not prove the Ode ANR has no product cause.

The crash buffer did not contain an Ode Java exception stack. Absence of a crash stack does not exclude an ANR.

Local diagnostic archive: `test-results/qa-anr-2026-10-13.zip`. Ode trace entry: `FS/data/anr/anr_2026-10-13-12-26-59-974`. The archive contains device diagnostics and should not be shared publicly without a separate privacy review.

The captured main thread was Runnable, with frames in:

```text
com.android.internal.dev.perfetto.sdk.PerfettoTrace$Category.isEnabled
android.os.MessageQueue.decAndTraceMessageCount
android.os.MessageQueue.nextLegacy
android.os.MessageQueue.next
android.os.Looper.loopOnce
android.os.Looper.loop
android.app.ActivityThread.main
```

This snapshot does not identify an app lock, application callback or definitive root cause. Do not label it a fixed keyboard bug, invalid password, or confirmed OS-only defect.

## Account Isolation

The authorized tester account was separately checked using configured Supabase credentials held in memory. All seven checks passed:

1. Backend health: 200.
2. Unauthenticated profile access denied: 401.
3. QA provider sign-in: 200.
4. Backend identity matched the authorized QA account: 200.
5. Entry-list response contract: 200.
6. Missing entry inaccessible: 404.
7. Read-only billing-plan response contract: 200.

Evidence: `test-results/qa-account-2026-10-13T07-14-58.708Z.json`. Tokens and passwords are not written to this report. These API results do not substitute for device sign-in, session persistence, actual transcription or reply-quality tests.

## Required Follow-up

Reproduce the mandatory device flow on a stable Android image or physical device and compare resources/ANR traces. Retain this failed low-memory/API-37 run in the device matrix even if another device passes. Rebuild the latest source before certifying recorder, reflection or sharing fixes on a device. Root cause and supported-device behavior remain release risks until verified.
