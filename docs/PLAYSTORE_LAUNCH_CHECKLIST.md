# Play Store Launch Checklist (Android)

Status as of **2026-07-20**, verified against the actual codebase.
Companion file: `docs/IOS_LAUNCH_CHECKLIST.md`. Cross-platform payments/compliance detail
lives in `docs/LAUNCH_CHECKLIST.md` — this file is the Android-specific execution list.

Legend: ✅ done · 🔴 blocker (broken feature or store rejection) · 🟡 required before submission · 🟢 fast follow

---

## Division of work (updated 2026-07-20)

**Done by Claude in the codebase (2026-07-20)** — full detail in `docs/IOS_LAUNCH_CHECKLIST.md` → Division of work:
therapy pricing made Pro-only (unbuyable card hidden) · Privacy/Terms links in Settings ·
CI test gate (`.github/workflows/test.yml`) · portal build artifacts untracked + gitignored ·
store content drafted (`docs/STORE_SUBMISSION.md`) · confirmed **Railway is production** (Fly target is dead).

**Only you can do (accounts/dashboards):**
1. ✅ Play Console account ($25) — done. Still needed: app record + internal testing track.
   If it's a Personal account, also start the closed-test 12-tester/14-day clock ASAP (§2).
2. Create the 4 IAP **one-time products** (not subscriptions — see docs/PRICING.md) as
   Play Console **In-app products**; service account with financial-data access →
   `GOOGLE_PLAY_CREDENTIALS_JSON` on Railway
3. `FCM_CREDENTIALS_JSON` + `FCM_PROJECT_ID` + `SENTRY_DSN` + real AI keys on Railway (API **and** worker)
4. ✅ Privacy/terms already resolve at https://dreamlog-48f94.web.app/privacy + /terms — use
   as-is. Fast follow (not a blocker): point https://dreamlog.app at the portal hosting; create
   talktoode.dev@gmail.com support inbox
5. Data Safety form + Health apps declaration + content rating (drafted answers in `docs/STORE_SUBMISSION.md`)
6. Screenshots + feature graphic; seed demo account
7. `eas build` (native modules: push, IAP, Sentry need a rebuild — OTA is not enough) → license-tester
   IAP test → closed-test tester recruitment (if Personal account) → internal track soak → staged
   production rollout

---

## 1. Code readiness — ✅ done (verified in repo)

- [x] Package `com.talktoode.app` set in `app.json` → `android.package`
- [x] `google-services.json` present (gitignored, injected at build) — Android Firebase config works
- [x] Permissions declared: `RECORD_AUDIO`, `MODIFY_AUDIO_SETTINGS`, `POST_NOTIFICATIONS` (Android 13 runtime prompt handled in `src/services/push.ts`)
- [x] Push client code done: FCM token fetch + `POST /devices` + token-rotation re-registration, fail-silent
- [x] Backend FCM HTTP v1 OAuth exchange implemented (`services/fcm.go` + tests)
- [x] IAP client code done: `expo-iap` + `mobile/src/services/iap.ts`; server-side Google Play Developer API
      verification in `backend/internal/services/iap.go`; purchase token consumed only after backend grants
- [x] Adaptive icon + Breath Line brand assets at all densities; boot splash on espresso `#18150f`
- [x] Account deletion in Settings (Play data-safety requirement)
- [x] Sentry mobile SDK wired (DSN in eas.json preview/production)
- [x] Force-update gate via `GET /version`; `ANDROID_STORE_URL` default already points at the Play listing
- [x] EAS release pipeline: `make mobile-build-prod` (.aab) + `make mobile-submit-android`
- [x] Demo/tester account created in production Supabase (LAUNCH_CHECKLIST §2d)

## 2. Play Console setup 🔴

- [x] ✅ **Play Console developer account** ($25 one-time) — approved.
      If this was a **Personal** account (created after Nov 13, 2023), Google requires a
      **closed test with ≥12 testers, opted in continuously for 14 days**, before granting
      production/public-listing access. Start recruiting testers now (§7 below) - this is
      the long pole in the whole timeline, not code or store-listing work. Organization
      accounts are exempt from this.
- [ ] 🔴 **Create the app record** in Play Console (package `com.talktoode.app`)
- [ ] 🔴 **Upload signing**: let EAS manage the upload keystore; opt in to Play App Signing on first upload
- [ ] 🟡 Set up an **internal testing track** first — required anyway for IAP license testers
      (internal testing has no tester-count/duration requirement; it's the *closed* track that
      needs the 12/14-day minimum before production access)

## 3. Payments (IAP) 🔴

- [ ] 🔴 **Create the 4 consumable products in Play Console** with these exact IDs
      (must match `backend/internal/services/iap.go` + `mobile/src/services/iap.ts`):
      `com.talktoode.app.plus.monthly`, `com.talktoode.app.plus.annual`,
      `com.talktoode.app.pro.monthly`, `com.talktoode.app.pro.annual` — prices per docs/PRICING.md
- [ ] 🔴 **Set `GOOGLE_PLAY_CREDENTIALS_JSON` on the backend** — service account with Play Console
      "View financial data" permission (can reuse the FCM service account if granted).
      Optional `GOOGLE_PLAY_PACKAGE_NAME` (defaults to `com.talktoode.app`).
      While unset, `/billing/upgrade` grants plans WITHOUT verification — do not launch without it
- [x] ✅ **Therapy pay-per-use gap defused for v1 (2026-07-20)**: unbuyable "Single Session" card hidden;
      therapy ships free-first-session + Pro-only. ✅ Session SKUs wired 2026-10-23 (`POST /billing/therapy-credits`);
      the pricing screen sells single sessions again. 🔴 Create the consumables
      `com.talktoode.app.therapy.single` + `com.talktoode.app.therapy.member` (In-app products) before release
- [ ] 🟡 Test end-to-end with **license testers** on the internal track: buy Plus monthly, verify
      `payments` row (store='google', orderId) + expiry ≈ +30 days; kill the app mid-flow and
      confirm Play re-delivers the unconsumed purchase
- [ ] 🟢 Apply for the Play 15% reduced-commission tier

## 4. Push notifications — code done, needs env + verification 🔴

- [ ] 🔴 Set `FCM_CREDENTIALS_JSON` + `FCM_PROJECT_ID` on the production backend — **both** the API
      and the worker process (the nudge scheduler runs in the worker)
- [ ] 🔴 **Rebuild the app** (`eas build`) — push needs native modules; OTA/JS update is not enough, Expo Go won't work
- [ ] 🟡 Quick test: Firebase console → Messaging → test message → device token from `user_devices` table
- [ ] 🟡 End-to-end test: record entry → `nudges` row → `UPDATE nudges SET scheduled_at = NOW()` →
      worker sends within 60s (row flips to `status='sent'`)

## 5. Store compliance & listing 🟡

- [x] ✅ **Privacy policy + terms are live** at https://dreamlog-48f94.web.app/privacy and /terms
      (pages exist in `therapist-portal/src/app/`) — use these URLs in the Play Console store
      listing now. 🟢 Fast follow: point `dreamlog.app` at Firebase Hosting and swap to the
      branded domain later; not a blocker.
- [x] ✅ **Privacy Policy / Terms links added to Settings** (2026-07-20) — domain must serve the portal pages
- [ ] 🔴 **Data Safety form**: audio collected (processed, then deleted), transcripts stored, health data,
      email, mental-health content, account deletion available in-app
- [ ] 🔴 **Health apps declaration** — Play has extra policy requirements for apps handling
      health/mental-health data; declare accurately
- [ ] 🟡 Content rating questionnaire (IARC)
- [ ] 🟡 Store listing assets: phone + 7"/10" tablet screenshots, **feature graphic (1024×500 — Play-only)**,
      short + full description, support email (talktoode.dev@gmail.com inbox must exist)
- [ ] 🟡 Seed the demo account with 3–5 entries for Play pre-launch report / reviewers
- [ ] 🟡 Target API level: verify the Expo SDK's compile/target SDK meets Google's current requirement at submission time

## 6. Production backend env 🟡 (shared with iOS — see LAUNCH_CHECKLIST §3)

- [ ] `STUB_AI_ANALYSIS=false` + real `ANTHROPIC_API_KEY`
- [ ] `OPENAI_API_KEY` + `WHISPER_API_URL`
- [ ] `AZURE_TTS_KEY` + `AZURE_TTS_REGION`
- [ ] `SENTRY_DSN` on API **and** worker; `SUPABASE_URL`; storage → R2; strong `JWT_SECRET`; `MASTER_ENCRYPTION_KEY`
- [x] ✅ **Deploy-target discrepancy resolved (2026-07-20)**: Railway is production; Fly target is dead.
      🟡 Remove or repoint the Fly jobs in `.github/workflows/deploy.yml`

## 7. Build, test, submit 🟡

- [ ] Add `SENTRY_AUTH_TOKEN` to EAS env so release builds upload source maps
- [ ] Production build: `make mobile-build-prod` (.aab)
- [ ] `make mobile-submit-android` → internal testing track
- [ ] 🔴 **If the Play Console account is Personal**: create a **closed testing** track,
      add ≥12 testers by email, and get them to actually install + open the app and stay
      opted in continuously for 14 days. This is a hard Play requirement before production
      access is granted (see §2) — start this the moment you have a working internal build;
      it's the longest lead time in this whole checklist, longer than any code work left.
- [ ] 🟢 Soak on internal/closed track ≥1 week; watch Android Vitals + Sentry
- [ ] Apply for production access on the Play Console dashboard once the closed-test
      requirement is met (skip if Organization account)
- [ ] Promote to production rollout (staged %, e.g. 20% → 100%)

## 8. Repo hygiene / CI (pre-launch should-do) 🟡

- [x] ✅ CI test workflow added (2026-07-20): `.github/workflows/test.yml` — `go vet` + `go test -race` +
      mobile/portal `tsc --noEmit` on push/PR (crisis tests are in the backend suite, so blocking)
- [x] ✅ `therapist-portal/out/` + `.firebase/` gitignored and untracked (2026-07-20)
- [ ] Rotate any secrets ever committed (`google-services.json` was tracked before removal — verify Firebase key restrictions or rotate)
- [ ] Demo-account password is committed to the repo — never reuse it; rotate if repo goes public

---

## Known-incomplete features (decide before submission)

| Feature | State | Android impact |
|---|---|---|
| Therapy pay-per-use purchase | ✅ Wired 2026-10-23 (prepaid credits via IAP) | Needs the 2 therapy consumables created in Play Console |
| Google Fit mindfulness sync | **Stub** (`src/services/health.ts` — `writeGoogleFitSession` is a no-op) | Silent no-op; not a blocker, but don't claim Fit sync in the listing |
| DLQ admin endpoint | Not built | Ops fast-follow |
| Privacy/Terms links in Settings | ✅ Added 2026-07-20 | Domain must serve the pages |
