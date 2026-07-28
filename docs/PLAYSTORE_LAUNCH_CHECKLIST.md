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
1. Play Console account ($25) + app record + internal testing track
2. Create the 4 IAP consumable products; service account with financial-data access →
   `GOOGLE_PLAY_CREDENTIALS_JSON` on Railway
3. `FCM_CREDENTIALS_JSON` + `FCM_PROJECT_ID` + `SENTRY_DSN` + real AI keys on Railway (API **and** worker)
4. Point https://dreamlog.app at the portal hosting so /privacy and /terms resolve; create support@ inbox
5. Data Safety form + Health apps declaration + content rating (drafted answers in `docs/STORE_SUBMISSION.md`)
6. Screenshots + feature graphic; seed demo account
7. `eas build` (native modules: push, IAP, Sentry need a rebuild — OTA is not enough) → license-tester
   IAP test → internal track soak → staged production rollout

---

## 1. Code readiness — ✅ done (verified in repo)

- [x] Package `com.ode.app` set in `app.json` → `android.package`
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

- [ ] 🔴 **Play Console developer account** ($25 one-time) + create the app (package `com.ode.app`)
- [ ] 🔴 **Upload signing**: let EAS manage the upload keystore; opt in to Play App Signing on first upload
- [ ] 🟡 Set up an **internal testing track** first — required anyway for IAP license testers

## 3. Payments (IAP) 🔴

- [ ] 🔴 **Create the 4 consumable products in Play Console** with these exact IDs
      (must match `backend/internal/services/iap.go` + `mobile/src/services/iap.ts`):
      `com.ode.app.plus.monthly`, `com.ode.app.plus.annual`,
      `com.ode.app.pro.monthly`, `com.ode.app.pro.annual` — prices per docs/PRICING.md
- [ ] 🔴 **Set `GOOGLE_PLAY_CREDENTIALS_JSON` on the backend** — service account with Play Console
      "View financial data" permission (can reuse the FCM service account if granted).
      Optional `GOOGLE_PLAY_PACKAGE_NAME` (defaults to `com.ode.app`).
      While unset, `/billing/upgrade` grants plans WITHOUT verification — do not launch without it
- [x] ✅ **Therapy pay-per-use gap defused for v1 (2026-07-20)**: unbuyable "Single Session" card hidden;
      therapy ships free-first-session + Pro-only. 🟢 Fast follow: wire a session consumable SKU through IAP
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

- [ ] 🔴 **Host privacy policy + terms at a public URL** and add to the Play Console store listing.
      Pages exist in the portal (`therapist-portal/src/app/privacy`, `/terms`) — use the deployed URLs
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
- [ ] 🟢 Soak on internal/closed track ≥1 week; watch Android Vitals + Sentry
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
| Therapy pay-per-use purchase | ✅ Hidden from UI 2026-07-20 (Pro-only v1); SKU wiring is a fast follow | None for launch |
| Google Fit mindfulness sync | **Stub** (`src/services/health.ts` — `writeGoogleFitSession` is a no-op) | Silent no-op; not a blocker, but don't claim Fit sync in the listing |
| DLQ admin endpoint | Not built | Ops fast-follow |
| Privacy/Terms links in Settings | ✅ Added 2026-07-20 | Domain must serve the pages |
