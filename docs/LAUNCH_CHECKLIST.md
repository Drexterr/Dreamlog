# DreamLog Launch Checklist

Everything left to set up before (and right after) publishing to the App Store and Play Store.
Status as of 2026-06-10, following the full codebase audit (all tests passing, `go vet` clean,
TypeScript clean on mobile + therapist portal).

Legend: 🔴 blocker (store rejection or broken feature) · 🟡 should-do before launch · 🟢 fast follow

---

## 1. Payments 🔴

The single biggest launch risk. Decide the approach first — it affects everything below it.

- [ ] **Decide: In-App Purchase vs web-only purchases.** DreamLog+ / Pro subscriptions and
      therapy session packs are digital content consumed in-app. Apple (Guideline 3.1.1) and
      Google Play **require IAP / Play Billing** for digital goods — the current Stripe payment
      sheet will be rejected on iOS. Options:
  - Move subscriptions + session packs to IAP/Play Billing (RevenueCat is the usual way to
    manage both stores with one SDK and gives receipt validation + webhooks).
  - Or remove purchase UI from the apps entirely and sell only via web (Stripe stays; apps
    just reflect the plan from `GET /billing/plan`).
- [ ] **Implement therapy pay-per-use billing server-side.** `TherapyService.computeBilling`
      currently returns `402` in production with a "real payment would be processed here"
      comment — non-Pro users can never buy a session. Wire the charge (IAP receipt validation
      or Stripe PaymentIntent, per the decision above).
- [ ] **Replace the Stripe publishable key placeholder.** `mobile/.env` has
      `EXPO_PUBLIC_STRIPE_PUBLISHABLE_KEY=pk_live_...` (literal placeholder). Set the real key,
      or better: inject it via `eas.json` production `env` like the other public vars.
- [ ] Verify `STRIPE_SECRET_KEY` is set on the production backend (Railway) if Stripe stays.

### Billing integrity & subscription compliance (found 2026-06-10 audit)

- [x] **Fix `POST /billing/upgrade` trusting the client.** ✅ Done 2026-06-10: the endpoint now
      requires `payment_intent_id`, verifies it with Stripe server-side (status `succeeded`,
      plan metadata match, amount match), and each intent grants a plan exactly once
      (`payments` table, migration 000026). B2B is no longer self-serve in production.
- [x] **Enforce `plan_expires_at`.** ✅ Done: `User.EffectivePlan()` added; all plan-gating call
      sites (mood history, reviews, export, share, entries quota, therapy billing) and
      `GET /billing/plan` now treat an expired plan as `free`.
- [x] **Set plan expiry server-side.** ✅ Done: expiry is always `now + 30 days`, computed by the
      backend; the client field is no longer accepted.
- [x] **Fix the false auto-renewal disclosure.** ✅ Done: upgrade screen copy now states
      "One-time payment … Each purchase is a 30-day pass. It does not auto-renew"; Settings shows
      "Active until <date> · does not auto-renew"; the "renews monthly" banner is gone.
- [x] **Cancellation path.** ✅ Resolved by the 30-day-pass model: there is no recurring charge to
      cancel. NOTE: if/when payments move to IAP subscriptions, the OS manage-subscription link
      becomes mandatory again.
- [ ] **Add payment reconciliation webhook** (Stripe webhook or RevenueCat events) so
      "charged but plan not set" can't happen when the app dies mid-flow. Mitigated for now:
      the user can retry `POST /billing/upgrade` with the same `payment_intent_id` as long as
      the first attempt never recorded it; full webhook still recommended.

## 2. Store compliance 🔴

- [ ] **Host a privacy policy** and add the URL to App Store Connect + Play Console. Required —
      the app records voice, stores mental-health content, and writes to HealthKit. Add
      Privacy Policy / Terms links to the Settings screen too (reviewers look for them).
- [ ] **Fill the Play Console Data Safety form** (audio collected, transcripts stored,
      health data, account deletion available).
- [ ] **Fill Apple's App Privacy "nutrition label"** (data linked to user: audio → transcripts,
      email, health & fitness).
- [ ] **Prepare App Review notes** for the mental-health surface: explain the two-stage crisis
      detection, the "AI-assisted reflection, not therapy" positioning, in-app crisis resources
      (Settings → Get help now), and provide a demo account with seeded data.
- [ ] Store listing assets: screenshots (6.7" + 5.5" iOS, phone + tablet Android), feature
      graphic, app description, keywords, support URL/email (support@dreamlog.app is already
      referenced in-app — make sure the inbox exists).

## 2b. iOS launch readiness 🔴 (added 2026-06-11)

Same codebase ships to both stores — there is no separate iOS app. `eas build --platform ios`
produces the iOS binary from the same code. The items below are the iOS-only setup.

### Done in code (2026-06-11)

- [x] **Sign in with Apple** (Guideline 4.8 — mandatory because Google Sign-In is offered).
      `expo-apple-authentication` installed, native Apple button on the auth screen (iOS only),
      signs in via Supabase `signInWithIdToken(provider: 'apple')`. `ios.usesAppleSignIn` set.
- [x] **`ITSAppUsesNonExemptEncryption: false`** in `app.json` — skips the export-compliance
      question on every App Store submission.
- [x] **iOS static frameworks** via `expo-build-properties` (required by React Native Firebase).
- [x] **`eas.json` `development` profile** now points at the Railway backend instead of
      `localhost` (unreachable from a physical phone). For a local backend, set
      `EXPO_PUBLIC_API_URL` in `mobile/.env` to your PC's LAN IP — the dev client reads env
      from Metro, not from the build profile.

### Left to do (requires your accounts)

- [ ] **Enroll in the Apple Developer Program** ($99/year) — do this first, approval can take
      1–2 days. Everything below needs it.
- [ ] **Supabase: enable the Apple provider** (Dashboard → Authentication → Providers → Apple,
      bundle ID `com.dreamlog.app`). Without this, the new Apple button returns an error.
- [ ] **Firebase: add an iOS app** (bundle `com.dreamlog.app`) and download
      `GoogleService-Info.plist` → save as `mobile/GoogleService-Info.plist` and add
      `"googleServicesFile": "./GoogleService-Info.plist"` under `ios` in `app.json`.
      ⚠️ iOS EAS builds will FAIL until this file exists (the RN Firebase plugin requires it).
      Android builds are unaffected.
- [ ] **APNs key**: Apple Developer portal → Keys → create an APNs key → upload to Firebase
      project settings → Cloud Messaging. Without it, FCM cannot deliver to iOS devices.
- [ ] **Google Sign-In on iOS**: create an iOS OAuth client ID in Google Cloud Console, then
      add `{ "iosUrlScheme": "com.googleusercontent.apps.XXXX" }` to the
      `@react-native-google-signin/google-signin` plugin config in `app.json`. Without it,
      Google sign-in crashes on iOS.
- [ ] **Apple IAP** — see section 1. Applies with full force on iOS: Stripe payment sheets for
      digital goods are a guaranteed rejection.
- [ ] **Device testing without a Mac**: `eas device:create` (registers your iPhone's UDID), then
      `eas build --profile development --platform ios`, install via the EAS link, run
      `npx expo start` and connect over Wi-Fi. For release validation:
      `eas build --profile production --platform ios` + `eas submit --platform ios` → TestFlight.

## 2c. Push notifications — fixed in code, needs env + verification (added 2026-06-11)

Push was silently dead on **both** platforms: the mobile app never fetched/registered an FCM
token, and the backend's `fcm.go` OAuth exchange was a stub that always errored. Both fixed:

- [x] Backend: `getAccessToken` now uses `golang.org/x/oauth2/google` with a cached,
      auto-refreshing TokenSource (+ unit tests in `fcm_test.go`).
- [x] Mobile: `@react-native-firebase/app` + `messaging` installed; `src/services/push.ts`
      requests permission (incl. Android 13 `POST_NOTIFICATIONS`), fetches the FCM token,
      calls `POST /devices`, and re-registers on token refresh. Wired into `app/_layout.tsx`
      on auth — fail-silent, never blocks startup.

Remaining:

- [ ] Set `FCM_CREDENTIALS_JSON` (service-account JSON content) + `FCM_PROJECT_ID` on Railway
      (API **and** worker process — the nudge scheduler runs in the worker).
- [ ] Rebuild the Android app (`eas build --profile preview --platform android`) — push needs
      the new native modules; an OTA/JS update is not enough. Expo Go will not work for push.
- [ ] **Test (quick)**: Firebase console → Messaging → "Send test message" → paste the device
      token. To see the token, check the `user_devices` table after logging in on the new build
      (`make db-psql` → `SELECT * FROM user_devices;` — or query the production DB).
- [ ] **Test (end-to-end)**: record an entry → a row appears in `nudges` scheduled for tomorrow
      8 AM → `UPDATE nudges SET scheduled_at = NOW() WHERE id = '<id>';` → within 60s the worker
      sends it (watch worker logs; row flips to `status='sent'`).
- [ ] iOS push works only after the APNs key + `GoogleService-Info.plist` steps in section 2b.

## 2d. Demo / tester account (added 2026-06-11)

For App Review notes and human testers. Created in the production Supabase project:

```
Email:    bharatbanthia2207+tester@gmail.com
Password: DreamTest!2026
```

- [ ] **Activate it**: a confirmation email was sent to your Gmail inbox (plus-addressing
      delivers it to bharatbanthia2207@gmail.com) — click the link. Alternatively confirm the
      user in Supabase Dashboard → Authentication → Users → ⋮ → Confirm email.
- [ ] **Seed it with data** before App Review: log in on a device, record 3–5 entries so
      reviewers see reflections, mood charts, and the timeline populated.
- [ ] Housekeeping: an earlier orphan account `dreamlog.tester01@gmail.com` (unconfirmed) can
      be deleted from the Supabase users list.
- [ ] Note: these credentials are committed to the repo — fine for a seeded demo account, but
      never reuse this password elsewhere, and rotate it if the repo ever goes public.

## 3. Production environment verification 🟡

Backend on Railway (`https://dreamlog-production-f9e2.up.railway.app`):

- [ ] `STUB_AI_ANALYSIS=false` and real `ANTHROPIC_API_KEY` set (reflections are canned stubs otherwise).
- [ ] `OPENAI_API_KEY` + `WHISPER_API_URL` pointing at the real Whisper API.
- [ ] `AZURE_TTS_KEY` + `AZURE_TTS_REGION` set for therapy voice output (Azure Speech:
      empathetic SSML styles for English personas + Hindi voices for Hindi turns). Optional
      `AZURE_TTS_USE_HD=true` upgrades to per-persona DragonHD multilingual voices
      (EN+HI+Hinglish in one voice, emotion auto-detected; ~$22/1M chars vs ~$15 — verify the
      chosen region serves HD voices, e.g. Central India since 2026-03). Optional
      `AZURE_TTS_VOICE_OVERRIDE` forces one voice for all personas (e.g.
      `en-IN-Aarti:DragonHDLatestNeural`) and wins over `USE_HD`. When Azure is unset, TTS
      falls back to OpenAI.
- [ ] `FCM_CREDENTIALS_JSON` + `FCM_PROJECT_ID` set so morning nudges and weekly-review pushes
      actually send (token exchange + client registration fixed 2026-06-11, see section 2c).
- [ ] `CORS_ALLOWED_ORIGINS` includes the therapist-portal production origin (defaults to localhost otherwise).
- [ ] `SUPABASE_URL` set so ES256 Supabase tokens validate via JWKS.
- [ ] Storage env points at Cloudflare R2 (not MinIO), presign expiry sane.
- [ ] `JWT_SECRET` / `SUPABASE_JWT_SECRET` confirmed strong + matching Supabase project.
- [ ] Confirm DB migrations ran (18+ migrations, auto-run on API startup) and worker process is running alongside the API.

## 4. Monitoring & error tracking (the "maintenance portal" decision) 🟡

Decision: **do not build a custom portal** — use off-the-shelf monitoring. A homegrown portal
can't capture on-device crashes and would duplicate weeks of what these tools give free.

- [ ] **Create a Sentry account** (free Developer tier ~5k events/month) with three projects:
  - [ ] `dreamlog-mobile` — `@sentry/react-native` via the Expo config plugin; upload source
        maps in EAS build so JS stack traces are readable; tag releases with the app version.
  - [ ] `dreamlog-backend` — `sentry-go` + Gin middleware on `cmd/api`; panic capture in
        `cmd/worker` (wrap the job handler so failed pipeline jobs report with `entry_id`).
  - [ ] `dreamlog-portal` — `@sentry/nextjs` in `therapist-portal/`.
  - [ ] Set up alert rules: email/Slack on any new error type + on error-rate spike.
- [ ] **Uptime monitoring**: UptimeRobot (free) pinging `GET /health` every 5 min, alert on downtime.
- [ ] **Store dashboards**: check Android Vitals (Play Console) and Xcode Organizer crashes
      weekly after launch — these feed store ranking.
- [ ] 🟢 **DLQ admin endpoint** (fast follow): list + re-enqueue rows from the existing
      `dead_letter_jobs` table (failed transcription/analysis jobs). Auth-gated, admin-only.
      Optionally a small admin page in the therapist portal later.

## 5. CI & repo hygiene 🟡

- [ ] **CI must run `go test -race ./...`** on Linux — the Windows dev machine has no gcc, so
      race-detector runs can only happen in CI. TESTING.md gates merges on this.
- [ ] CI: `go vet ./...`, mobile + portal `tsc --noEmit`, crisis-detection tests blocking.
- [ ] Gitignore `therapist-portal/out/` and `therapist-portal/.firebase/` build artifacts
      (currently committed; deploy from local build instead of git).
- [ ] Rotate any secrets that were ever committed (`google-services.json` was tracked before
      being removed — verify the Firebase API key restrictions, or rotate).

## 6. Nice-to-have before scale 🟢

- [ ] Crash-free baseline: run the production EAS build through TestFlight / internal testing
      track for at least a week before public release.
- [ ] Add Sentry breadcrumbs around the upload flow (presign → PUT → register) — it's the most
      failure-prone user path (network, large files).
- [ ] Backend structured-log retention: Railway logs are ephemeral; consider shipping zap logs
      to a free tier of Better Stack / Axiom if debugging needs history.
- [ ] Load-test the worker with concurrent entries (`make scale-worker N=3` exists; verify on prod infra).
- [ ] Google Fit implementation is still a stub (`src/services/health.ts` — iOS HealthKit works,
      Android pending Google Fit credentials). Fails silently, so not a blocker.

---

## Already verified as done (no action needed)

- Backend: all tests pass, `go vet` clean, crisis fail-safe enforced everywhere (incl. worker
  screener-error path), audio deleted after transcription, server-side session expiry,
  3-turn conversation cap, JWT alg allowlist, CORS allowlist, config panics on missing required envs.
- Mobile: TypeScript clean, offline queue now auto-flushes on reconnect (with mode preserved),
  account deletion (`DELETE /me`) wired in Settings (Apple requirement), data export, in-app
  crisis resources, mic/HealthKit permission strings, bundle IDs, EAS production profile.
- Therapist portal: TypeScript clean.
- Legacy dead code removed (`mobile/src/screens/`).
