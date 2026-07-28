# Store Submission Content (draft)

Ready-to-paste content for App Store Connect and the Play Console.
Drafted 2026-07-20 — review the wording before pasting; anything in [brackets] needs your input.

---

## 1. App identity

| Field | Value |
|---|---|
| Name | Ode — Voice Journal |
| Subtitle (iOS, 30 chars) | Speak. Reflect. Understand. |
| Short description (Play, 80 chars) | Voice journaling with AI reflections that understand your emotional patterns. |
| Bundle / package | com.ode.app |
| Category | Health & Fitness (primary) · Lifestyle (secondary) |
| Support email | talktoode.dev@gmail.com (inbox must exist before submission) |
| Privacy policy URL | https://dreamlog.app/privacy |
| Terms URL | https://dreamlog.app/terms |
| Marketing URL | https://dreamlog.app |

---

## 2. Full description (both stores)

> **Your voice, heard.**
>
> Ode is a voice journal. Talk for a minute or thirty — about your day, a decision,
> a dream, something you can't stop thinking about. Ode listens, transcribes, and
> reflects back what it heard: your mood, the feelings underneath, the patterns that
> keep coming up.
>
> **What Ode gives you**
> - 🎙 Voice-first journaling — no typing, just talk
> - 🪞 A thoughtful AI reflection after every entry, grounded in your recent history
> - 📈 Mood tracking, streaks, and a Life Graph of your emotional trends
> - 🌙 Dream Decoder — psychological and Vedic readings of your dreams
> - 🧭 Guided journeys for stress, gratitude, grief, and big decisions
> - 📅 Weekly and yearly reviews written from your own words
> - 🗣 Therapy Sessions — AI-assisted voice conversations that know your journal (not a therapy replacement)
> - 🔒 Private by design — audio is deleted right after transcription
>
> **Speaks your language.** English, Hindi, and Hinglish supported.
>
> **Your safety matters.** Every entry passes through crisis screening. If Ode ever
> detects that you might be in crisis, it responds with real helplines instead of an
> AI reflection.
>
> Ode is a space for reflection, not medical care. It does not diagnose, treat, or
> replace a mental-health professional.

Keywords (iOS, 100 chars):
`voice journal,journaling,mood tracker,mental health,diary,reflection,dream,gratitude,wellbeing`

---

## 3. App Review notes (Apple) / review notes (Play)

Paste into App Store Connect → App Review Information → Notes:

> Ode is a voice-journaling app with AI-generated reflections. It is positioned as
> "AI-assisted reflection, not therapy" — this wording appears in-app on the therapy
> screens and in our disclaimer.
>
> SAFETY DESIGN (mental-health surface):
> 1. Every journal entry and every therapy-session message passes two-stage crisis
>    screening (deterministic keyword match, then an AI confirmation). The system
>    fails safe: if the AI is unreachable, the content is treated as a crisis.
> 2. When a crisis is detected, the app does NOT show an AI reflection. It shows
>    structured crisis resources with tap-to-call helplines (iCall, Vandrevala
>    Foundation, 988), plus links to find professional therapists.
> 3. Crisis resources are always reachable from Settings → "Get help now".
> 4. AI therapy sessions are hard-capped at 1 hour server-side and end immediately
>    with helpline resources if crisis signals repeat.
>
> PRIVACY DESIGN:
> - Audio recordings are transcribed and then permanently deleted from storage;
>   only the transcript is retained.
> - Account deletion (with all data) is available in Settings.
>
> DEMO ACCOUNT (pre-seeded with journal entries):
> - Email: bharatbanthia2207+tester@gmail.com
> - Password: DreamTest!2026
>
> IN-APP PURCHASES: consumable 30/365-day plan passes (Ode+ and Ode Pro), verified
> server-side. They do not auto-renew, and the app says so on the purchase screen.
>
> HEALTHKIT (iOS): after a completed entry the app writes a Mindful Session
> (write-only; nothing is read from HealthKit).

---

## 4. Apple App Privacy "nutrition label"

Data types collected, all **linked to identity** (account-based app), none used for tracking:

| Data type | Collected? | Purpose | Notes |
|---|---|---|---|
| Contact info → Email address | Yes | App functionality (account) | Supabase auth |
| User content → Audio data | Yes | App functionality | Transcribed then deleted from storage |
| User content → Other (journal transcripts, AI reflections) | Yes | App functionality | Core product data |
| Health & fitness | Yes | App functionality | Mood scores, mental-health content; HealthKit write-only |
| Identifiers → User ID | Yes | App functionality | Account UUID; FCM device token |
| Usage data → Product interaction | Yes | Analytics | First-party analytics_events table only (IDs/metadata, no content) |
| Diagnostics → Crash data | Yes | App functionality | Sentry, PII disabled (`sendDefaultPii: false`) |
| Location / purchases / browsing / search history / contacts | No | — | Coarse region only from device locale (not collected server-side beyond country on payment) |

Tracking (ATT): **No** — no cross-app tracking, no ad SDKs. Answer "No" to the tracking question.

---

## 5. Play Console Data Safety form

**Data collected & why:**
- Personal info → Email address: app functionality, account management. Encrypted in transit. Deletable.
- Audio → Voice/sound recordings: app functionality. Ephemeral — deleted after transcription (mark "processed ephemerally" for the audio itself; transcripts are stored).
- Health info → mental-health journal content, mood data: app functionality. Encrypted in transit. Deletable.
- App activity → in-app interactions: analytics (first-party). 
- App info & performance → crash logs: Sentry.
- Device IDs → FCM push token: app functionality (push notifications).

**Security practices:**
- Data encrypted in transit: Yes (HTTPS everywhere).
- Users can request deletion: Yes — in-app account deletion (Settings → Delete all data).
- Independent security review: No.

**Health apps declaration:** the app handles mental-health data → complete the
Play "Health apps" declaration accurately; the privacy policy must cover it.

**Data shared with third parties:** transcripts are processed by AI providers
(Anthropic, OpenAI Whisper, Azure) as service providers for app functionality —
declare as "data shared for app functionality" if the form's definition of
sharing covers processors, else document in the privacy policy. [Confirm with
the current form wording at submission time.]

---

## 6. Age rating guidance

- Both questionnaires ask about mature/medical themes. The app contains
  user-generated mental-health content and references to crisis/self-harm
  *resources* (not depictions). Expected outcome: **12+ (Apple)** / **Teen (Play)**.
- Answer "No" to gambling, violence, sexual content; answer honestly on
  "frequent/intense medical or treatment information" — the crisis-resource
  surface is a safety feature, not treatment.

---

## 7. Screenshot shot-list (both stores)

1. Record screen mid-recording (waveform + orb) — "Just talk"
2. Reflection screen with a warm AI reflection — "Be understood"
3. Mood screen with Life Graph + streak — "See your patterns"
4. Timeline with entries + analyses — "Your story, kept"
5. Therapy session (voice orb) — "A space to talk" (include the not-a-replacement disclaimer visibly)
6. Dream Decoder result — "Decode your dreams"

Sizes: iOS 6.7" (1290×2796) + 5.5" (1242×2208); Play phone + 7" & 10" tablet, plus 1024×500 feature graphic.

---

## 8. Things to verify before pasting

- [ ] https://dreamlog.app/privacy and /terms actually resolve on the production domain (pages exist in `therapist-portal/src/app/`; the domain must point at the Firebase Hosting site, project `dreamlog-48f94`)
- [ ] talktoode.dev@gmail.com inbox exists
- [ ] Demo account email confirmed + 3–5 entries recorded
- [ ] Privacy policy text actually covers: audio processing, AI processors (Anthropic/OpenAI/Azure), health data, retention, deletion, children's policy
