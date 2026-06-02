# DreamLog Roadmap

## Phase Status

| Phase | Name | Status |
|---|---|---|
| 1 | Foundation | ✅ Complete |
| 2 | AI Core | ✅ Complete |
| 3 | Local Auth | ✅ Complete |
| 4 | Retention & Growth | ✅ Complete |
| 5 | Scale & B2B | ✅ Complete |
| 6 | Therapy Mode | ✅ Complete |
| 7 | Longitudinal Intelligence | ✅ Complete |
| 8 | Enhanced Therapy Mode | 🚧 In Progress |
| — | UX Polish | ✅ Complete |

---

## Phase 1 — Foundation ✅
Migration: `000001_init.up.sql`

Built:
- Users, entries, dead letter queue schema
- Audio upload pipeline (presign → MinIO PUT → entry creation → Redis job)
- Whisper transcription worker
- Supabase JWT auth + auto-provisioning
- Basic entry CRUD with pagination
- Health check endpoint

---

## Phase 2 — AI Core ✅
Migration: `000002_phase2.up.sql`

Built:
- Full `entry_analysis` schema (mood_score, emotional_tone, topics, key_quotes, reflection, morning_nudge, is_crisis)
- Two-stage crisis detection (keyword + Claude confirmation, fail-safe)
- Context builder (last 5 entries → emotion trend + topic trend injected into prompts)
- Claude reflection generation (structured JSON output, 7-field schema)
- Follow-up conversations (3-turn cap, grounded in original entry)
- Mood tracking: GET /mood/weekly + GET /mood/streak
- Full-text search via tsvector + GIN index
- FCM push notifications + user_devices table
- Morning nudge scheduler (timezone-aware, polls every 60s)
- Timeline endpoint (entries + analyses combined)

---

## Phase 3 — Local Auth ✅
Migration: `000003_auth.up.sql`

Built:
- `password_hash` column on users
- POST /auth/register + POST /auth/login via bcrypt + JWT
- Dev-friendly: no Supabase setup required

---

## Phase 4 — Retention & Growth ✅

### 4a — Onboarding ✅
- Goal selection screen on first open (stress / anxiety / grief / relationships / career / curious)
- Goal stored in users table (new column)
- Goal injects into Claude system prompt to personalize reflection tone
- "What name should I call you?" during onboarding

### 4b — Weekly Emotional Review ✅
- Hourly scheduler: schedules pending row on last Sunday 10 AM per user's timezone (idempotent)
- Claude generates a "week in review" narrative from the 7-day entry summaries
- FCM push delivered on completion; narrative stored in `weekly_reviews` table
- GET /reviews/weekly/latest and GET /reviews/weekly API endpoints
- Weekly review card displayed on mobile Mood screen

### 4c — Streak Mechanics Overhaul ✅
- Streak freeze: `streak_freeze_count` on users (max 3); auto-granted 1 per week via WeeklyReviewScheduler; POST /streak/freeze to apply
- Freeze days stored in `streak_freeze_days` table; included in streak calculation (union with entry days)
- GET /mood/streak returns `next_milestone` and `freeze_count`
- "Comeback" card when streak == 0 (non-guilt language + freeze button if available)
- Milestone modal celebration at 7 / 21 / 50 / 100 days with share via native Share API

### 4d — Shareable Insight Cards ✅
- Not the journal content — anonymized mood arc + top emotion for the week
- Generate as image on mobile, share via native share sheet
- Zero-cost viral acquisition loop

### 4e — Hindi + Hinglish Support ✅
- Whisper already detects language and returns it in `entries.language`
- New Hindi system prompt in prompts.go
- Hinglish detection (mixed Hindi-English) → use Hinglish prompt variant
- Language auto-detected, no user setting needed

### 4f — Prompt Modes ✅
- Rant Mode: no analysis, just transcription + acknowledgement
- Gratitude Mode: AI asks 3 specific gratitude follow-ups after listening
- Decision Mode: Socratic follow-up for decisions
- Processing Mode: current default
- Mode selection on record screen

### 4g — Life Graph ✅
- 30 / 90 / 365 day mood trendline with range selector
- Period-over-period mood delta (avg_mood vs prior equal window)
- Top-3 emotion aggregation across the selected window
- GET /mood/history?range=30d|90d|365d — daily avg, weighted overall avg, prior-period avg, top emotions
- LifeGraph component in mood.tsx replaces Pattern Radar placeholder

---

## Phase 5 — Scale & B2B ✅
Migrations: `000008_phase5a.up.sql`, `000009_phase5c.up.sql`, `000010_phase5g.up.sql`

### 5a — Therapist Sharing Mode ✅
- `share_links` table: 32-byte random token + bcrypt-hashed 4-digit passcode, 72h TTL
- `POST /share` (auth) → create link, returns plaintext passcode once
- `GET /share/:token?p=xxxx` (public) → validate passcode, return anonymised view
- Mobile: `app/share/[id].tsx` — generate link, show passcode, send via native Share sheet
- View includes: 30-day mood arc, AI entry summaries, top emotions; NO transcripts or reflections

### 5b — Crisis → Care Bridge ✅
- Crisis reflection screen completely redesigned: structured hotline cards (iCall, Vandrevala, 988)
- Tap-to-call via `Linking.openURL('tel:...')`
- "Find a therapist near you" CTA → Practo affiliate URL with UTM tracking
- YourDOST online therapy option
- `crisisResponse()` returns plain text, rendering is entirely in mobile

### 5c — B2B Corporate Wellness ✅
- `companies` + `company_members` tables (migration 000009)
- `v_team_daily_mood` view: fully anonymised, never exposes individual identities
- `POST /b2b/companies/:slug/join` — user self-enrolls
- `GET /b2b/companies/:slug/mood?range=30d|90d` — admin-only aggregated mood dashboard
- Threshold alert: `is_alerted=true` when avg_mood < 40

### 5d — PDF Export ✅
- `GET /export/pdf?period=monthly|yearly` — streams binary PDF (auth-protected)
- Uses `go-pdf/fpdf` for pure-Go A4 generation
- PDF structure: cover page → mood overview (stats cards + bar chart + emotion bars) → entry summaries
- Mobile: `app/export.tsx` — period picker, `expo-file-system.downloadAsync` with auth header, `expo-sharing.shareAsync`
- Linked from Settings → Export my data

### 5e — Apple Health / Google Fit Integration ✅
- `src/services/health.ts`: `writeMindfulSession({ startDate, endDate })`
- iOS: HealthKit via `react-native-health`, requests `MindfulSession` write permission
- Android: Google Fit stub (same interface, implementation pending Google Fit credentials)
- Called from `app/processing/[id].tsx` the moment entry status=completed
- Fails silently — health write errors never surface to the user

### 5f — Clinical Validation
- Partner with NIMHANS or AIIMS psychology department
- 90-day study: does daily voice journaling reduce self-reported anxiety?
- Outcome: peer-reviewed study, "clinically validated" positioning
- (Not a software deliverable — business development track)

### 5g — Therapist Dashboard API (B2B2C) ✅
- `therapists` + `client_therapist_links` tables (migration 000010)
- `POST /therapists/register` — therapist self-registers
- `POST /therapists/clients/link` — link a client by UUID (client shares ID from app settings)
- `GET /therapists/clients` — list clients with 30d avg mood + last entry date
- `GET /therapists/clients/:id/brief` — Claude-generated 3-sentence pre-session brief
- Brief includes: 7d avg mood, trend (improving/declining/stable), top emotions, last 5 entry summaries
- Prompt lives in `services/prompts.go::BuildTherapistBriefPrompt`
- **Therapist web portal** (`therapist-portal/`): Next.js 14 app (separate from mobile)
  - `/login` — JWT auth via existing `/auth/login`
  - `/dashboard` — client list, add by UUID, remove
  - `/dashboard/clients/:id` — full brief with mood stats, trend icon, entry cards
  - Stack: Next.js 14, TypeScript, Tailwind CSS, Recharts

---

## Phase 6 — Therapy Mode ✅
Migration: `000018_therapy_mode.up.sql`

Real-time AI-assisted voice/text conversation sessions. Synchronous (not worker-queued). Journal-context-aware — Claude knows the user's mood history, emotional patterns, and recent entries before the first word is spoken.

Positioned as: **"AI-assisted reflection session grounded in your journal history"** — not a therapy replacement.

Built:
- `therapy_sessions` + `therapy_messages` tables + `therapy_session_status` ENUM (active | completed | expired | crisis_detected)
- `internal/repositories/therapy.go` — session CRUD, message storage, context snapshot
- `internal/services/therapy.go` — full session lifecycle (start, send message, end, expiry check, billing)
- `buildTherapyModeSystemPrompt()` + `buildTherapyPostSessionPrompt()` in `prompts.go`
- `internal/handlers/therapy.go` — all 6 endpoints registered in router
- Crisis detection runs on every message (Stage 1 + Stage 2, fail-safe) per ADR-013
- Context snapshot loaded at session start: last 5 entry summaries, top emotions, top topics, 30-day avg mood
- Session hard-cap 1 hour enforced server-side via `expires_at` per ADR-012
- Audio deleted after transcription per ADR-005
- Mobile: `app/therapy/index.tsx`, `app/therapy/session.tsx`, `app/therapy/summary/[id].tsx`
- All 6 therapy API functions in `src/api/client.ts`, all 11 therapy types in `src/types/index.ts`
- Billing: first session free; Pro plan 2/month; ₹499/session otherwise (402 returned when no credits)

**Not yet built:**
- Voice input on mobile (presign API exists, `useRecorder` hook exists; wiring pending)
- OpenAI TTS voice output (server plumbing ready; mobile audio playback not wired)

### Cost Per Session (text mode)
| | |
|---|---|
| Claude Sonnet with prompt caching | ~$0.70 |
| Whisper (if voice enabled) | ~$0.18 |
| **Total (text only)** | **~$0.70** |

Gross margin at ₹499 (text mode): ~88%.

---

## Phase 7 — Longitudinal Intelligence ✅
Migrations: `000011_monetization.up.sql`, `000013_guided_journeys.up.sql`, `000014_year_in_review.up.sql`, `000015_life_chapters.up.sql`, `000016_dream_decoder.up.sql`, `000017_relationship_map.up.sql`

Features built beyond the original roadmap phases:

### 7a — Subscription & Billing API ✅
- `plan TEXT` + `plan_expires_at TIMESTAMPTZ` columns added to `users`
- `GET /billing/plan` — returns current plan + all plan limits; mobile reads this on Settings screen
- `POST /billing/upgrade` — sets plan (stub in dev, payment gateway in prod)
- Plan gating enforced in handlers: `mood/history`, `reviews/weekly`, `reviews/annual`, `share`, `export/pdf`

### 7b — Guided Journeys ✅
- `journey_templates` + `journey_sessions` + `journey_steps` tables (migration 000013)
- Seeded templates: Stress Relief, Gratitude Practice, Decision Clarity, Grief Processing, Self-Compassion
- `GET /journeys` — list templates; `POST /journeys/:id/start` — create session
- `GET /journeys/sessions`, `GET /journeys/sessions/:id` — session state + step prompts
- `POST /journeys/sessions/:id/advance` — submit entry for current step, advance to next
- Mobile: `app/journeys.tsx` (list + active sessions) and `app/journeys/[sessionId].tsx` (step view)

### 7c — Annual Year in Review ✅
- `annual_reviews` table (migration 000014); auto-scheduled yearly for each user
- Claude generates a narrative + top emotions + top topics + monthly mood arc (JSONB)
- `GET /reviews/annual/latest` and `GET /reviews/annual` — requires Plus or higher
- Displayed in `app/(tabs)/mood.tsx` below the Life Graph

### 7d — Life Chapters ✅
- `life_chapters` table (migration 000015) — user-defined named time periods with start/end dates
- `GET /chapters`, `POST /chapters`, `GET /chapters/:id`, `PUT /chapters/:id`, `DELETE /chapters/:id`
- `GET /chapters/:id/detail` — enriched with entry count, avg mood, top emotions, daily mood arc
- `POST /chapters/:id/summarize` — generates Claude narrative for the chapter period; stored in `summary` column

### 7e — Relationship Map ✅
- `people` + `person_mentions` tables (migration 000017) — extracted automatically from entry transcripts by Claude

---

## Phase 8 — Enhanced Therapy Mode 🚧
Migration: `000020_therapy_personas.up.sql`

Upgrades the existing Therapy Mode (Phase 6) with personas, session memory, graceful wind-down, layered crisis handling, and an onboarding gate.

### Sub-steps

#### 8a — DB Schema + Models 🔲
- Add `persona TEXT NOT NULL DEFAULT 'comforting'` to `therapy_sessions`
- Add `crisis_warnings INT NOT NULL DEFAULT 0` to `therapy_sessions`
- Update `TherapyContextSnapshot` to carry `past_session_summaries []string` (last 3 post-session summaries, fetched at session start — no new column, sourced from existing `post_session_summary` rows)
- Update `TherapySession` model + repo interface with new fields

#### 8b — Therapist Personas 🔲
Four personas, each with a distinct system prompt and mapped TTS voice:

| Persona | Character | TTS Voice |
|---|---|---|
| `comforting` | Warm, validating, gentle — focuses on feelings first | `nova` |
| `rational` | Logical, structured, Socratic — facts before feelings | `onyx` |
| `cbt` | CBT-informed — identifies thought patterns, challenges distortions | `alloy` |
| `mindful` | Grounding, present-moment, breath-aware | `shimmer` |

- Four new `buildTherapyPersonaSystemPrompt_*(ctx, timeRemainingSec)` functions in `prompts.go`
- `POST /therapy/sessions` accepts `persona` field (defaults to `comforting`)
- Persona stored in `therapy_sessions.persona`; used on every turn

#### 8c — Session Continuity 🔲
- At session start, fetch last 3 completed session `post_session_summary` values for the user
- Inject into `TherapyContextSnapshot.PastSessionSummaries`
- System prompt includes a "Memory from past sessions" section so the AI can reference prior work naturally
- This is what makes the AI feel like it actually remembers you

#### 8d — Graceful Wind-Down 🔲
- `time_remaining_sec` injected into the system prompt on every `SendMessage` call
- When `time_remaining_sec < 600` (10 min): prompt instructs Claude to start bringing the conversation to a natural close
- When `time_remaining_sec < 120` (2 min): prompt instructs Claude to wrap up in this turn
- Mobile timer already exists; server now drives the wind-down tone

#### 8e — Layered Crisis Handling 🔲
Current behaviour: crisis detected → immediate hard stop, crisis resources shown.

New two-stage behaviour:
1. **Stage 1 (first detection):** AI sends a grounding de-escalation response (breathing, safety check, validation); session stays active; `crisis_warnings` incremented to 1
2. **Stage 2 (second detection, or if user explicitly confirms intent):** session hard-stopped, status = `crisis_detected`; response includes crisis helplines + "I'm not able to help further — please reach out to a professional" message

- If the de-escalation response itself contains another crisis signal → immediately escalate to Stage 2
- `crisis_warnings` column tracks attempts per session

#### 8f — Onboarding Gate: Journal vs Therapy 🔲
- After goal selection in onboarding, add a screen: "How would you like to begin?" with two cards: Journal and Start a Therapy Session
- "Journal" → existing record flow
- "Start a Therapy Session" → persona picker → POST /therapy/sessions

**Not in Phase 8:**
- Voice input wiring on mobile (presign + useRecorder exists; wiring tracked separately)
- OpenAI TTS audio playback on mobile (server maps persona→voice; playback pending)
- Each person tracked with role (family/friend/colleague/romantic/other), mention count, positive/negative sentiment counts
- `GET /relationships` — full map of all people mentioned; `GET /relationships/:id` — person + recent mention context
- Claude extracts people during the worker pipeline (in `workers/transcription.go` after analysis)

---

## UX Polish ✅
Migration: `000021_age_range.up.sql`

Incremental UX improvements shipped outside phase gates.

### Age Range Collection ✅
- Optional `age_range` column on `users` table (CHECK constraint: `under_18 | 18_24 | 25_34 | 35_44 | 45_plus`)
- Collected as step 3 of onboarding between the name step and the Journal/Therapy gate
- Displayed as a tap-to-select list; tapping a selected range deselects it (skip support without a separate button)
- Button label changes to "Skip" when nothing selected, "Continue" when a range is selected
- Exposed on `GET /me` and accepted on `PUT /me` with enum validation (400 on invalid value)
- Used for internal analytics cohorts; never shown in AI prompts or reflections
- `AgeRange` type added to `src/types/index.ts`; `updateMe` in `src/api/client.ts` accepts the field

### Settings — Profile Modal ✅
- Profile card in Settings now shows `"N entries · Plan name"` instead of email address
- Card is a `TouchableOpacity` with a `›` chevron; tapping opens a bottom-sheet profile modal
- Modal displays: avatar with initials, display name, email, full name, and age range ("Not set" in muted text if absent)
- "Change email address" row → Alert directing user to support@dreamlog.app
- Consistent with existing modal patterns (BlurView + sheet)

### Greeting Splash ✅
- On every cold app open for an authenticated, post-onboarding user, a full-screen overlay fades in with "Hello, [name]"
- Uses `preferred_name` if set, falls back to `name`
- Animation: fade in 400ms → hold 1400ms → fade out 400ms (total ~2.2s)
- Implemented as an `Animated.View` with `absoluteFill` and `zIndex: 999` inside `ThemeProvider` in `app/_layout.tsx`
- Uses `CormorantGaramond_300Light` 36px matching the app's serif palette
- Does not show during onboarding, on auth screen, or if user has no name

### Therapy Session Screen — Voice-First Redesign ✅
- Screen rebuilt to mirror the journal record screen aesthetic: pulsating mic orb as the primary UI element
- `SessionOrb` (160px): idle breathing pulse (1.03x), strong pulse + glow + red tint while recording, `ActivityIndicator` overlay while waiting for AI response
- `Waveform` component (9 animated bars with staggered delays) shown while recording — matches `record.tsx` pattern
- Dual input modes controlled by local state:
  - **Voice mode** (default): header with persona/timer/End button; message scroll limited to `maxHeight: 38%` above orb; "Chat" button opens text mode
  - **Text mode**: full-height chat list, keyboard-aware text input + send; auto-returns to voice after send
- Crisis and session-ended states preserved from original implementation

### Therapy Pricing Screen ✅ (`app/therapy/pricing.tsx`)
- New dedicated screen for therapy session purchase options: Single, 5-Pack, 12-Pack, Pro plan
- Currency auto-detected via `detectAndCacheRegion()` on mount — same pattern as `app/upgrade.tsx`; no manual toggle
- Shows `ActivityIndicator` while region is loading, then renders prices in the detected currency (INR for India, USD elsewhere)
- `OptionCard` component: badge, title, subtitle, price, per-session breakdown, saving badge, feature checklist, CTA
- Persona chip horizontal scroll to preview companion styles before purchasing
- "Every session includes" info box + safety disclaimer at bottom
- Pro CTA → `app/upgrade.tsx`; session pack CTAs → `app/therapy/persona-picker.tsx`

### Therapy Index Screen — Hero Redesign ✅ (`app/therapy/index.tsx`)
- Complete redesign replacing the minimal "Start a Session" landing with a rich hero screen
- `AmbientGlow` pulsating blob behind the hero (position absolute, does not scroll with content)
- Hero: small-caps eyebrow label, 38px `CormorantGaramond_300Light` heading, supporting copy
- Feature chip strip: 5 horizontal scroll pills (Voice or text, Up to 60 min, Journal-aware AI, Post-session summary, Crisis detection)
- Active session resume banner: shown only when a session has `status === 'active'`; taps straight into that session
- Companion styles preview: horizontal scroll of 4 persona cards (emoji, name, tagline)
- Stats bar: total sessions / total turns / completed count; hidden when user has no session history
- `SessionCard` redesign: left accent bar (brand color = active, muted = otherwise), status badge, persona emoji + name, date, 2-line summary, turn count + duration meta

### Therapy — 402 Credit Redirect ✅
- When `POST /therapy/sessions` returns 402 (no session credits), `app/therapy/persona-picker.tsx` now calls `router.replace('/therapy/pricing')` instead of showing an Alert
- User lands directly on the pricing screen where they can purchase a session pack or upgrade to Pro

---

## Monetization Tiers (Target)

```
Free
  10 entries/month | basic reflection | 7-day mood chart | 3-turn follow-up

DreamLog+ — ₹199/month India | $7.99/month Global
  Unlimited entries | Hindi support | Life Graph | Weekly Review
  All prompt modes | Streak freeze | Therapist share (5/month)

DreamLog Pro — ₹499/month India | $14.99/month Global
  Everything in Plus | PDF export | Apple Health sync
  Unlimited therapist share | Priority processing
  2 Therapy Sessions/month

Therapy Session — ₹499/session (pay-per-use)
  Journal-context-aware AI conversation | Up to 1 hour
  Voice or text input | Optional AI voice output
  Post-session summary | Crisis detection active

B2B Wellness — ₹199/employee/month (min 50 employees)
  All Pro features | HR dashboard | Monthly wellness report
```
