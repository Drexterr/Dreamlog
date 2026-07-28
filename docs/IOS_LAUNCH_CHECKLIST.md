# iOS Launch Checklist (App Store)

Status as of **2026-07-20**, verified against the actual codebase (not just docs).
Companion file: `docs/PLAYSTORE_LAUNCH_CHECKLIST.md`. Cross-platform payments/compliance
detail lives in `docs/LAUNCH_CHECKLIST.md` — this file is the iOS-specific execution list.

Legend: ✅ done · 🔴 blocker (build fails or store rejects) · 🟡 required before submission · 🟢 fast follow

---

## Division of work (updated 2026-07-20)

**Done by Claude in the codebase (2026-07-20):**
- ✅ Therapy pricing screen is Pro-only for v1 — the unbuyable pay-per-use "Single Session" card is hidden
  (`app/therapy/pricing.tsx`; Maestro flow 11 updated). Re-add when a session SKU is wired through IAP
- ✅ Privacy Policy + Terms links added to Settings → Privacy (point at https://dreamlog.app/privacy|terms)
- ✅ `app.json`: `ios.googleServicesFile` reference added + Google Sign-In `iosUrlScheme` **placeholder** added —
  you must replace `REPLACE_WITH_IOS_OAUTH_CLIENT_ID` with the real iOS OAuth client and drop in the real plist
- ✅ CI test gate added (`.github/workflows/test.yml`): `go vet` + `go test -race` + mobile/portal `tsc --noEmit`
- ✅ `therapist-portal/out/` + `.firebase/` untracked and gitignored; `mobile/GoogleService-Info.plist` pre-gitignored
- ✅ Store submission content drafted: `docs/STORE_SUBMISSION.md` (descriptions, App Review notes,
  privacy nutrition label, Data Safety answers, screenshot shot-list)
- ✅ Deploy discrepancy resolved: **Railway is production** (health returns 200); the Fly.io app
  (`dreamlog-api.fly.dev`) does not resolve — the `deploy-api`/`deploy-worker` jobs in
  `.github/workflows/deploy.yml` deploy to a dead target and should be removed or repointed (your call)

**Only you can do (accounts/dashboards) — in order:**
1. Apple Developer enrollment ($99/yr) — start first, 1–2 day approval blocks everything
2. Firebase console: add iOS app → put `GoogleService-Info.plist` in `mobile/` · create + upload APNs key
3. Google Cloud console: iOS OAuth client → replace the `iosUrlScheme` placeholder in `app.json`
4. Supabase dashboard: enable the Apple auth provider
5. App Store Connect: app record, 4 IAP products, grab `APPLE_SHARED_SECRET`
6. Railway env vars (API + worker): `APPLE_SHARED_SECRET`, `GOOGLE_PLAY_CREDENTIALS_JSON`,
   `FCM_CREDENTIALS_JSON`/`FCM_PROJECT_ID`, `SENTRY_DSN`, real `ANTHROPIC_API_KEY`, etc. (§7)
7. Point https://dreamlog.app at the Firebase Hosting portal site (project `dreamlog-48f94`) so the
   /privacy and /terms URLs used in-app actually resolve
8. Confirm demo-account email + seed 3–5 entries; create talktoode.dev@gmail.com inbox
9. `SENTRY_AUTH_TOKEN` in EAS env; device testing; sandbox IAP testing; TestFlight; submit

---

## 1. Code readiness — ✅ done (verified in repo)

- [x] Single codebase ships to iOS via EAS (`make mobile-build-prod-ios`, cloud macOS worker — no Mac needed)
- [x] Bundle ID `com.ode.app` set in `app.json` → `ios.bundleIdentifier`
- [x] Sign in with Apple implemented (`expo-apple-authentication`, native button iOS-only, Supabase `signInWithIdToken`); `ios.usesAppleSignIn: true` — required by Guideline 4.8 since Google Sign-In is offered
- [x] `ITSAppUsesNonExemptEncryption: false` in `app.json` (skips export-compliance question)
- [x] iOS static frameworks via `expo-build-properties` (required by RN Firebase)
- [x] Permission strings present: `NSMicrophoneUsageDescription`, `NSHealthUpdateUsageDescription`; HealthKit entitlement set
- [x] Apple Health MindfulSession write implemented (`src/services/health.ts`, fail-silent)
- [x] IAP client code done: `expo-iap` + `mobile/src/services/iap.ts`; server-side Apple `verifyReceipt` in `backend/internal/services/iap.go`; transaction finished only after backend grants (interrupted flows re-delivered)
- [x] Account deletion in Settings (Apple requirement)
- [x] Push client code done: `src/services/push.ts` registers FCM token on auth, fail-silent
- [x] Sentry mobile SDK wired (`@sentry/react-native` + Expo plugin, DSN in eas.json preview/production)
- [x] Force-update gate: app checks `GET /version` on cold start, fail-open
- [x] Demo/tester account created in production Supabase (see LAUNCH_CHECKLIST §2d)

## 2. Apple accounts & signing 🔴 (nothing below works without these)

- [ ] 🔴 **Enroll in the Apple Developer Program** ($99/year) — do FIRST, approval takes 1–2 days
- [ ] 🔴 **Create the App Store Connect app record** (bundle `com.ode.app`, name "Ode")
- [ ] 🔴 Let EAS manage certificates/profiles on first `eas build --platform ios` (interactive login)

## 3. Firebase / push (iOS build FAILS without the plist) 🔴

- [ ] 🔴 **Firebase: add an iOS app** (bundle `com.ode.app`) → download `GoogleService-Info.plist` → save as `mobile/GoogleService-Info.plist`
      — verified missing from the repo today
- [ ] 🔴 **Add `"googleServicesFile": "./GoogleService-Info.plist"` under `ios` in `app.json`**
      — verified absent today; the RN Firebase plugin makes iOS EAS builds fail until both exist
- [ ] 🔴 **APNs key**: Apple Developer portal → Keys → create APNs key → upload to Firebase → Cloud Messaging. Without it FCM cannot deliver to any iOS device
- [ ] 🟡 Verify iOS push end-to-end on a TestFlight/dev build (token row in `user_devices`, Firebase console test message)

## 4. Auth providers 🔴

- [ ] 🔴 **Supabase: enable the Apple provider** (Dashboard → Authentication → Providers → Apple, bundle `com.ode.app`) — the Apple button errors until this is on
- [ ] 🔴 **Google Sign-In on iOS**: create an iOS OAuth client in Google Cloud Console, then replace the
      `iosUrlScheme` **placeholder** (`REPLACE_WITH_IOS_OAUTH_CLIENT_ID`) in `app.json` with the real value —
      plugin config scaffolded 2026-07-20; Google sign-in **crashes on iOS** until the real client ID is in

## 5. Payments (IAP) 🔴

- [ ] 🔴 **Create the 4 consumable products in App Store Connect** with these exact IDs
      (must match `backend/internal/services/iap.go` + `mobile/src/services/iap.ts`):
      `com.ode.app.plus.monthly`, `com.ode.app.plus.annual`,
      `com.ode.app.pro.monthly`, `com.ode.app.pro.annual` — prices per docs/PRICING.md
- [ ] 🔴 **Set `APPLE_SHARED_SECRET` on the backend** (App Store Connect → App Information → App-Specific Shared Secret).
      While unset, `/billing/upgrade` grants plans WITHOUT verification — do not launch without it
- [x] ✅ **Therapy pay-per-use gap defused for v1 (2026-07-20):** the unbuyable "Single Session" card is
      hidden on `app/therapy/pricing.tsx` — therapy ships as free-first-session + Pro-only.
      🟢 Fast follow: wire a therapy-session consumable SKU through the IAP verification path
      (`computeBilling` still 402s for pay-per-use in prod, which is now unreachable from the UI)
- [ ] 🟡 Sandbox test end-to-end: buy Plus monthly with a sandbox tester, verify `payments` row +
      `plan_expires_at` ≈ +30 days; kill the app mid-purchase and confirm store re-delivery
- [ ] 🟢 Enroll in the App Store Small Business Program (15% commission)

## 6. Store compliance & listing 🟡

- [ ] 🔴 **Host a privacy policy + terms at a public URL** and enter them in App Store Connect.
      Pages exist in the portal (`therapist-portal/src/app/privacy`, `/terms`) — confirm the deployed URLs and use those
- [x] ✅ **Privacy Policy / Terms links added to Settings** (2026-07-20) — point at https://dreamlog.app/privacy|terms;
      you must make that domain serve the portal's pages (see Division of work #7)
- [ ] 🔴 **App Privacy "nutrition label"**: data linked to user — audio → transcripts, email, health & fitness, mental-health content
- [ ] 🔴 **App Review notes** for the mental-health surface: two-stage crisis detection, "AI-assisted reflection, not therapy"
      positioning, in-app crisis resources (Settings → Get help now), demo account credentials (LAUNCH_CHECKLIST §2d)
- [ ] 🟡 **Activate + seed the demo account**: confirm the email, then record 3–5 entries so reviewers see reflections/mood/timeline
- [ ] 🟡 Store listing assets: 6.7" + 5.5" screenshots, description, keywords, support URL/email
      (talktoode.dev@gmail.com is referenced in-app — make sure the inbox exists)
- [ ] 🟡 Age rating questionnaire (mental-health content → likely 12+/17+; answer the "medical" questions carefully)
- [ ] 🟢 After approval: set `IOS_STORE_URL` on the backend (verified default is `""` today) so the
      force-update gate can deep-link iOS users to the store

## 7. Production backend env 🟡 (shared with Android — see LAUNCH_CHECKLIST §3)

- [ ] `STUB_AI_ANALYSIS=false` + real `ANTHROPIC_API_KEY`
- [ ] `OPENAI_API_KEY` + `WHISPER_API_URL` (real Whisper)
- [ ] `AZURE_TTS_KEY` + `AZURE_TTS_REGION` (therapy voice)
- [ ] `FCM_CREDENTIALS_JSON` + `FCM_PROJECT_ID` on API **and** worker
- [ ] `SENTRY_DSN` on API **and** worker
- [ ] `SUPABASE_URL`, storage → R2, strong `JWT_SECRET`, `MASTER_ENCRYPTION_KEY`, `CORS_ALLOWED_ORIGINS`
- [x] ✅ **Deploy-target discrepancy resolved (2026-07-20)**: Railway is production (health 200);
      `dreamlog-api.fly.dev` does not resolve. Set all env vars on **Railway** (API + worker).
      🟡 Decide: remove or repoint the dead Fly deploy jobs in `.github/workflows/deploy.yml`

## 8. Build, test, submit 🟡

- [ ] Register a test iPhone: `make mobile-device-ios` → `eas build --profile development --platform ios` → install via EAS link
- [ ] Add `SENTRY_AUTH_TOKEN` to EAS env (`eas env:create`) so release builds upload source maps
      (builds succeed without it, but stack traces stay minified)
- [ ] Production build: `make mobile-build-prod-ios` (bump `version` in `mobile/app.json` if needed — shared with Android)
- [ ] `make mobile-submit-ios` → TestFlight
- [ ] 🟢 TestFlight soak: run the build with internal testers ≥1 week before public release
- [ ] Submit for App Review with the notes from §6

---

## Known-incomplete features (decide before submission)

| Feature | State | iOS impact |
|---|---|---|
| Therapy pay-per-use purchase | ✅ Hidden from UI 2026-07-20 (Pro-only v1); SKU wiring is a fast follow | None for launch |
| Google Fit (Android) | Stub | None for iOS (HealthKit works) |
| DLQ admin endpoint | Not built | None — ops fast-follow |
| CI test gate (`go test -race`, vet, tsc) | ✅ Added 2026-07-20 (`.github/workflows/test.yml`) | — |
| Privacy/Terms links in Settings | ✅ Added 2026-07-20 | Domain must serve the pages |
