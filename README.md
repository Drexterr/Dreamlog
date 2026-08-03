# Ode

A production-grade voice journaling app. Speak your day - Ode transcribes it, reflects back what it hears, screens for crisis, and tracks your emotional patterns over time. Includes a real-time AI Therapy Mode and a full therapist workspace (portal + in-app) for professionals.

All 9 build phases are complete; the project is in **Store Launch Prep** - see [docs/LAUNCH_CHECKLIST.md](docs/LAUNCH_CHECKLIST.md).

---

## Index

- [Architecture](#architecture)
- [Tech stack](#tech-stack)
- [Pipelines](#pipelines)
  - [Recording → Reflection](#recording--reflection)
  - [Follow-up conversation](#follow-up-conversation)
  - [Therapy Mode](#therapy-mode)
  - [Therapist Note OCR](#therapist-note-ocr)
  - [Habit loop](#habit-loop-hook-model)
- [Feature map by phase](#feature-map-by-phase)
- [Quick start](#quick-start)
- [Environment variables](#environment-variables)
- [API reference](#api-reference)
- [Make commands](#make-commands)
- [Dev cost: zero](#dev-cost-zero)
- [Database schema (summary)](#database-schema-summary)
- [Crisis detection](#crisis-detection)
- [Security & privacy](#security--privacy)
- [Further reading](#further-reading)

---

## Architecture

```
dream/
├── backend/              Go 1.23 · Gin · PostgreSQL 16 · Redis · MinIO/R2
│   ├── cmd/
│   │   ├── api/          HTTP server (runs migrations on startup)
│   │   └── worker/       Transcription/AI worker + all scheduler goroutines
│   ├── internal/
│   │   ├── config/       Env-var loading, panics on missing required vars
│   │   ├── handlers/     HTTP routes - entries, analysis, conversations, mood,
│   │   │                 billing, weekly/annual reviews, insights, share, export,
│   │   │                 b2b, journeys, life chapters, relationships, therapist
│   │   │                 dashboard + workspace (notes/OCR), therapy, users, auth
│   │   ├── middleware/   JWT auth (Supabase HS256/ES256 + local), CORS, logging,
│   │   │                 error formatting, Sentry capture
│   │   ├── models/       Domain types (pure structs, no methods beyond EffectivePlan)
│   │   ├── repositories/ pgx/v5 queries - no business logic
│   │   ├── services/     Business logic - claude.go, crisis.go, transcription.go,
│   │   │                 context_builder.go, prompts.go, auth.go, fcm.go, tts.go,
│   │   │                 iap.go, therapy.go, therapist_notes.go, journey.go,
│   │   │                 connection_insight.go, analytics.go, subscription.go
│   │   └── workers/      Async jobs: transcription pipeline, note OCR, and
│   │                     scheduler goroutines (nudge / re-engagement /
│   │                     streak-risk / weekly-review / year-in-review / plan-expiry)
│   ├── migrations/       golang-migrate SQL files (35+, auto-run on API startup)
│   └── pkg/
│       ├── apierr/       Typed API errors
│       ├── queue/        Redis job queue (BRPOP/LPUSH)
│       └── storage/      S3-compatible client (MinIO dev / Cloudflare R2 prod)
├── mobile/                React Native 0.81 · Expo 54 · expo-router 6
│   ├── app/
│   │   ├── (tabs)/        Home, Timeline, Mood, Settings
│   │   ├── record.tsx, processing/[id].tsx, reflection/[id].tsx, followup/[id].tsx
│   │   ├── therapy/        index, persona-picker, pricing, session, summary/[id]
│   │   ├── therapist/      index, register, clients, client/[id], add-session,
│   │   │                   session/[id] (therapist workspace)
│   │   ├── journeys.tsx, journeys/[sessionId].tsx
│   │   ├── relationships.tsx, share/[id].tsx, export.tsx, upgrade.tsx
│   │   ├── onboarding.tsx, accept-terms.tsx, change-goal.tsx, entries.tsx
│   │   └── auth/, auth.tsx, therapist-requests.tsx
│   └── src/
│       ├── api/          Axios client + typed endpoints (client.ts)
│       ├── hooks/        useRecorder (audio state machine)
│       ├── services/     upload, offlineQueue, push (FCM), health (Apple Health/Fit)
│       ├── context/      auth/session context providers
│       ├── lib/          Supabase client
│       ├── config/       region/currency detection
│       ├── theme.ts      Design tokens (Erode + Hanken Grotesk, espresso palette)
│       └── types/        Shared TypeScript types
└── therapist-portal/      Next.js 14 · TypeScript · Tailwind · Recharts
    └── src/app/           /login, /dashboard, /dashboard/clients/:id,
                           /dashboard/notes (therapist web workspace)
```

### Tech stack

| Layer | Tech |
|---|---|
| API | Go 1.23, Gin, `pgx/v5`, `go-redis/v9`, `aws-sdk-go-v2` |
| Auth | Supabase JWT (HS256 dev / ES256+JWKS prod) + local bcrypt path, `golang-jwt/jwt/v5` |
| DB | PostgreSQL 16 - migrations via `golang-migrate` (35+ files) |
| Queue | Redis list (`BRPOP` / `LPUSH`) - async job pipeline + note-OCR queue |
| Storage | MinIO (dev) / Cloudflare R2 (prod) - pre-signed PUT URLs |
| Transcription | faster-whisper-server (local dev) / OpenAI Whisper API (prod) |
| AI analysis | Claude Sonnet 4.6 via Anthropic Messages API (stub mode in dev) |
| TTS | Azure Speech - Therapy Mode AI voice (empathetic SSML styles, Hindi + DragonHD voices); OpenAI TTS fallback; skipped in dev |
| IAP | `expo-iap` client + server-side Apple `verifyReceipt` / Google Play Developer API verification |
| Encryption | AES-256-GCM envelope encryption (per-therapist DEK wrapped by a master key) for therapist notes |
| Error tracking | Sentry (backend `pkg/monitoring`, mobile `@sentry/react-native`, portal `@sentry/nextjs`) |
| Mobile | React Native 0.81, Expo 54, expo-router 6, expo-av |
| Fonts | Erode (serif headings, bundled TTFs) + Hanken Grotesk (sans body) |
| Therapist portal | Next.js 14, TypeScript, Tailwind CSS, Recharts |

---

## Pipelines

### Recording → Reflection

```
Mobile                     Backend                    Workers
  │                           │                          │
  ├─ POST /entries/presign ──►│ returns pre-signed URL   │
  ├─ PUT audio ─────────────► MinIO/R2                   │
  ├─ POST /entries ─────────►│ creates DB row            │
  │                           │ pushes job → Redis ──────►│
  │                           │                          │ BRPOP job
  │                           │                          │ Whisper transcribe
  │                           │                          │ Crisis screen (keyword + Claude, fail-safe)
  │                           │                          │ Context builder (last 5 entries)
  │                           │                          │ Claude AnalyzeEntry (+ dream dual-lens if mode=dream)
  │                           │                          │ Person extraction → relationship map
  │                           │                          │ Connection insight (recurring-topic check)
  │                           │                          │ Store analysis
  │                           │                          │ Schedule morning nudge (adaptive hour)
  │                           │                          │ Delete audio
  ├─ GET /entries/:id ───────►│ polls status             │
  ├─ GET /entries/:id/analysis►│ returns reflection       │
```

### Follow-up conversation

```
POST /entries/:id/conversation   → get or create Conversation
POST /conversations/:id/messages → send user turn, get Claude reply
                                   (max 3 turns, then is_closed = true)
```

### Therapy Mode

Real-time AI-assisted voice/text conversation. Unlike journal entries (async, worker-processed), therapy sessions are synchronous - the response comes back in the same HTTP request (ADR-011). Sessions are pre-loaded with the user's journal history (mood trends, top emotions, recent entry summaries, past session summaries) so Claude starts with context, and support 4 selectable personas (comforting / rational / cbt / mindful).

```
Mobile                     API (synchronous - no worker)
  │                           │
  ├─ POST /therapy/sessions ─►│ INSERT session, load journal + past-session context, charge billing
  │                           │ ← { session_id, persona, expires_at }
  │                           │
  ├─ POST /therapy/sessions/:id/presign ─►│ ← { upload_url, audio_key }
  ├─ PUT audio ─────────────► MinIO/R2
  │                           │
  ├─ POST /therapy/sessions/:id/messages ─►│
  │                           │ Whisper transcribe (synchronous)
  │                           │ Crisis detection (Stage 1 + Stage 2, fail-safe, layered de-escalate→hard-stop)
  │                           │ Claude therapy turn (persona prompt + journal context + wind-down + history)
  │                           │ TTS synth (Azure, persona voice) if enabled
  │                           │ Delete audio from storage
  │                           │ ← { user_message, assistant_message, session_state }
  │                           │
  ├─ POST /therapy/sessions/:id/end ─►│ Generate post-session summary
  │                           │ ← { summary, duration_sec }
```

**1-hour time limit** enforced server-side via `expires_at` (ADR-012). Crisis detection is mandatory on every message and uses the same two-stage fail-safe as journal entries (ADR-013), with a layered de-escalate-then-hard-stop response (ADR-014).

### Therapist Note OCR

Runs as a goroutine inside the worker process, consuming a separate Redis queue. Lets therapists photograph handwritten session notes and get an editable, encrypted bullet list back - **no crisis screening** (ADR-018, these are clinical records about a client, not first-person journaling).

```
POST /therapists/sessions { image_key } → INSERT client_sessions (pending) → LPUSH job
Worker: download photo → Claude vision OCR (raw_text + bullets)
      → encrypt with per-therapist DEK → store ciphertext, status=completed
      → DELETE photo from storage immediately (ADR-019)
```

Sensitive fields (client names, note text, bullets, AI summaries) are encrypted at rest via envelope encryption (ADR-017): a per-therapist AES-256-GCM data key wrapped by `MASTER_ENCRYPTION_KEY` - the database only ever holds ciphertext. Full doc: `docs/THERAPIST_PORTAL.md`.

### Habit loop (Hook Model)

Retention features built around the Trigger → Action → Variable Reward → Investment loop:

- **Adaptive nudge timing** - the morning nudge is delivered at the user's
  *learned typical recording hour* (modal local hour of their last 20 entries)
  instead of a fixed clock time. Manually setting `fcm_nudge_hour` via
  `PUT /me` turns the learning off (`users.nudge_auto_time`).
- **Streak-at-risk push** - users with an active streak (≥3 days) and no entry
  today get a caring 21:00-local reminder stating the streak length.
- **Re-engagement push** - warm, non-guilt nudge after 26h+ of silence.
- **Plan-expiry push** - since Ode's paid plans are one-time 30/365-day IAP
  passes (not auto-renewing subscriptions), `plan_expiring_soon` fires within 3
  days of expiry and `plan_expired` fires shortly after lapse - the only place
  a lapsing plan is surfaced to the user.
- **Connection insights** - the worker deterministically detects recurring
  topics (3rd+ occurrence in 30 days) and stores a one-line pattern insight
  (`entry_analysis.connection_insight`) shown with the reflection - an
  intentionally intermittent, variable reward.
- **Flashback time capsule** - `GET /entries/flashback` resurfaces an entry
  from ~1 year (or ~1 month) ago on the home screen.
- **Self-set check-in** - "Check in on this tomorrow" button on the reflection
  screen (`POST /entries/:id/checkin`) schedules the next day's nudge from the
  entry's own morning-nudge line - the user authors their next trigger.
- **One-tap record** - tapping any nudge notification deep-links straight to
  the record screen (or the upgrade screen for plan-expiry nudges).
- **Home-screen shortcut** - long-press the app icon → "Record a moment"
  (`expo-quick-actions`); on Android the action can be pinned to the home
  screen as a standalone one-tap record icon. Requires a native build.

---

## Feature map by phase

Full detail and status of every phase lives in [docs/ROADMAP.md](docs/ROADMAP.md). Summary:

| Phase | Feature area | Status |
|---|---|---|
| 1 | Foundation - upload pipeline, Whisper, auth, entry CRUD | ✅ |
| 2 | AI Core - reflections, crisis detection, context builder, mood tracking, push, search | ✅ |
| 3 | Local Auth (dev-only bcrypt path) | ✅ |
| 4 | Retention & Growth - onboarding, weekly review, streaks, insight cards, Hindi/Hinglish, prompt modes, Dream Decoder, Life Graph | ✅ |
| 5 | Scale & B2B - therapist share links, crisis→care bridge, B2B wellness, PDF export, Apple Health/Google Fit, therapist dashboard API | ✅ |
| 6 | Therapy Mode - real-time voice/text sessions, journal-aware context | ✅ |
| 7 | Longitudinal Intelligence - billing/plans, guided journeys, annual review, life chapters, relationship map | ✅ |
| 8 | Enhanced Therapy Mode - 4 personas, session memory, wind-down, layered crisis | ✅ |
| 9 | Therapist Workspace - external clients, note OCR, envelope encryption, consent layer, web portal | ✅ |
| - | Store Launch Prep | 🚧 in progress - see LAUNCH_CHECKLIST.md |

---

## Quick start

### Prerequisites

- Docker Desktop
- Node.js 20+ (for mobile)
- Expo Go app on your phone **or** Android/iOS simulator

### 1 - Clone and configure

```bash
git clone <repo>
cd dream
cp .env.example .env   # already pre-filled for local dev
```

### 2 - Start the stack

```bash
make dev
# or
docker compose up --build -d
```

Services that start:

| Service | URL | Notes |
|---|---|---|
| API | http://localhost:8080 | `{"status":"ok"}` at `/health` |
| MinIO Console | http://localhost:9001 | admin / minioadmin_secret |
| PostgreSQL | localhost:5432 | auto-migrated on API start |
| Redis | localhost:6379 | |
| Whisper | localhost:9002 | downloads model on first start (~60 s) |

### 3 - Run the mobile app

```bash
make mobile-install   # npm install --legacy-peer-deps
make mobile-start     # npx expo start
```

Press `a` for Android emulator, `i` for iOS simulator, `w` for Expo Web.

**Set the API URL in `mobile/.env`:**

```env
# Android emulator
EXPO_PUBLIC_API_URL=http://10.0.2.2:8080

# iOS simulator or Expo Web
# EXPO_PUBLIC_API_URL=http://localhost:8080
```

### 4 - Authenticate (dev mode)

The app opens a JWT paste screen. Generate a test token at **[jwt.io](https://jwt.io)**:

| Field | Value |
|---|---|
| Algorithm | HS256 |
| Payload | `{"sub":"test-user-001","email":"test@dreamlog.dev"}` |
| Secret | your `SUPABASE_JWT_SECRET` from `.env` |

Copy the encoded token and paste it into the app.

---

## Environment variables

All variables live in `.env` at the project root (used by Docker Compose). See `backend/CLAUDE.md` for the full annotated list; key ones:

```env
# Server
PORT=8080

# Database
POSTGRES_USER=dreamlog
POSTGRES_PASSWORD=dreamlog_secret
POSTGRES_DB=dreamlog
DATABASE_URL=postgres://dreamlog:dreamlog_secret@postgres:5432/dreamlog?sslmode=disable

# Redis
REDIS_ADDR=redis:6379
REDIS_PASSWORD=redis_secret

# Storage (MinIO / R2)
STORAGE_ENDPOINT=http://minio:9000
STORAGE_ACCESS_KEY_ID=minioadmin
STORAGE_SECRET_ACCESS_KEY=minioadmin_secret
STORAGE_BUCKET=dreamlog-audio
STORAGE_USE_PATH_STYLE=true       # required for MinIO

# Auth
SUPABASE_JWT_SECRET=<your-secret>

# Transcription - points to local Whisper in dev
OPENAI_API_KEY=ignored
OPENAI_BASE_URL=http://whisper:8000/v1
WHISPER_MODEL=Systran/faster-whisper-base

# AI analysis - stub mode = no API calls, zero cost in dev
ANTHROPIC_API_KEY=                # leave blank in dev
STUB_AI_ANALYSIS=true             # set false + add key for real analysis

# Therapy Mode TTS - blank = skipped/OpenAI fallback in dev
AZURE_TTS_KEY=
AZURE_TTS_REGION=

# Push notifications - blank = skipped silently in dev
FCM_CREDENTIALS_JSON=
FCM_PROJECT_ID=

# IAP receipt verification - blank = /billing/upgrade grants without verification in dev
APPLE_SHARED_SECRET=
GOOGLE_PLAY_CREDENTIALS_JSON=

# Therapist note encryption - blank in dev = derived from JWT secret; set explicitly in prod
MASTER_ENCRYPTION_KEY=

# Error tracking - blank = disabled entirely
SENTRY_DSN=

# Worker
WORKER_CONCURRENCY=4
WORKER_MAX_RETRIES=3
```

---

## API reference

Base URL (dev): `http://localhost:8080`. All endpoints require `Authorization: Bearer <jwt>` except `/health`, `/version`, `/auth/register`, `/auth/login`, `GET /share/:token`, and `GET /journeys`. Full request/response shapes: [docs/API_CONTRACT.md](docs/API_CONTRACT.md).

### Auth & user

| Method | Path | Description |
|---|---|---|
| `POST` | `/auth/register`, `/auth/login` | Local email/password path (dev-friendly; mobile uses Supabase) |
| `GET`/`PUT` | `/me` | Get or update profile (name, timezone, nudge hour, goal, age range, voice language) |
| `POST` | `/me/accept-terms` | Record ToS acceptance |
| `GET` | `/me/terms` | Current ToS acceptance status |
| `POST` | `/devices` | Register FCM token for push |

### Entries, analysis & timeline

| Method | Path | Description |
|---|---|---|
| `POST` | `/entries/presign` | Get pre-signed PUT URL for audio upload |
| `POST` | `/entries` | Create entry + queue transcription job (supports `mode`) |
| `GET` | `/entries`, `/entries/:id`, `/entries/search?q=` | List / get / full-text search |
| `GET` | `/entries/:id/analysis` | AI analysis (mood, emotions, topics, reflection, dream lenses if mode=dream) |
| `GET` | `/entries/flashback` | Time capsule: entry from ~1 year / ~1 month ago |
| `POST` | `/entries/:id/checkin` | Schedule a "check in on this tomorrow" nudge |
| `GET` | `/timeline` | Entries + analyses combined (paginated) |

### Conversations & mood

| Method | Path | Description |
|---|---|---|
| `POST` | `/entries/:id/conversation`, `/conversations/:id/messages` | Follow-up conversation (max 3 turns) |
| `GET` | `/mood/weekly`, `/mood/streak` | 7-day mood + streak stats |
| `GET` | `/mood/history?range=`, `/mood/patterns?range=` | 30/90/365-day trendline + emotion radar |
| `POST` | `/streak/freeze` | Protect a missed day with a streak freeze |

### Billing

| Method | Path | Description |
|---|---|---|
| `GET` | `/billing/plan` | Current plan, expiry, limits, all-plan comparison |
| `POST` | `/billing/upgrade` | Grant a plan after store-verified IAP purchase |

### Insights, reviews, chapters, relationships, journeys

| Method | Path | Description |
|---|---|---|
| `GET`/`POST` | `/insights/card`, `/insights/share` | Weekly shareable insight card |
| `GET` | `/reviews/weekly`, `/reviews/weekly/latest` | Claude-generated weekly narrative |
| `GET` | `/reviews/annual`, `/reviews/annual/latest` | Yearly narrative (Plus+) |
| `GET`/`POST`/`PUT`/`DELETE` | `/chapters`, `/chapters/:id`, `/chapters/:id/detail`, `/chapters/:id/summarize` | Life Chapters (user-defined time periods) |
| `GET`/`PATCH`/`POST` | `/relationships`, `/relationships/:id`, `/relationships/:id/merge` | Auto-extracted relationship map |
| `GET` | `/journeys` (public) | Guided journey template catalogue |
| `POST`/`GET` | `/journeys/:id/start`, `/journeys/sessions*`, `/journeys/sessions/:id/advance` | Guided journey session flow |

### Sharing, export, B2B

| Method | Path | Description |
|---|---|---|
| `POST` | `/share` · `GET` | `/share/:token` (public) | Therapist share links (passcode-protected) |
| `GET` | `/export/pdf?period=` | Streams a PDF journal export (Plus+) |
| `POST`/`GET` | `/b2b/companies/:slug/join`, `/b2b/companies/:slug/mood` | Corporate wellness enrollment + anonymized dashboard |

### Therapist dashboard & workspace

| Method | Path | Description |
|---|---|---|
| `POST` | `/therapists/register` | Therapist self-registers |
| `POST`/`DELETE` | `/therapists/clients/link`, `/therapists/clients/:id` | Link/unlink an app-user client (consent-gated) |
| `GET`/`POST` | `/therapists/requests*` | Client-facing consent approve/decline for pending links |
| `GET` | `/therapists/clients`, `/therapists/clients/:id/brief` | Client list + Claude pre-session brief |
| `GET` | `/therapists/me` | Therapist profile (mobile login routing) |
| `POST` | `/therapists/consent` | Accept client-data responsibility terms |
| `GET` | `/therapists/overview` | Dashboard metrics |
| CRUD | `/therapists/external-clients*` | Therapist's own (non-app) clients |
| `POST` | `/therapists/sessions/presign` | Pre-signed URL for a note photo |
| CRUD | `/therapists/sessions*` | Session notes: create (photo OCR or typed), edit bullets, summarize, delete |

### Therapy Mode

| Method | Path | Description |
|---|---|---|
| `POST` | `/therapy/sessions` | Start a session (persona, loads journal context, charges billing) |
| `POST` | `/therapy/sessions/:id/presign` | Pre-signed URL for voice turn upload |
| `POST` | `/therapy/sessions/:id/messages` | Send turn (voice or text), get AI response |
| `POST` | `/therapy/sessions/:id/end` | End session, generate post-session summary |
| `GET` | `/therapy/sessions/:id`, `/therapy/sessions` | Session state/history + list |

---

## Make commands

Run `make` with no arguments to print all available targets.

### Dev lifecycle

```bash
make dev              # build images + start all services (detached)
make dev-stop         # stop + remove containers (volumes kept)
make down             # stop + remove containers AND volumes (full wipe)
make dev-restart      # rebuild changed images + restart
make dev-status       # show container status (alias: make ps)
make health           # curl http://localhost:8080/health
```

### Logs

```bash
make dev-logs         # tail API + worker together
make logs-api         # API only
make logs-worker      # worker only
make logs-whisper     # faster-whisper-server
make logs-postgres    # PostgreSQL
make logs-redis       # Redis
```

### Build

```bash
make build            # rebuild all images (--no-cache)
make build-api        # rebuild API image only
make build-worker     # rebuild worker image only
make restart          # interactive: pick a service to force-recreate
```

### Shells

```bash
make shell-api        # sh into the running API container
make shell-postgres   # sh into PostgreSQL container
make shell-redis      # redis-cli (reads password from .env)
```

### Database

```bash
make db-migrate       # apply pending migrations (runs inside API container)
make db-migrate-down  # roll back last migration
make db-reset         # ⚠ drop DB + re-apply all migrations (destroys data)
make db-psql          # open psql session
```

### Scaling

```bash
make scale-worker N=3   # run 3 worker replicas (default N=2)
```

### Local Go (without Docker)

```bash
make api              # go run ./cmd/api
make worker           # go run ./cmd/worker
make tidy             # go mod tidy
```

### Mobile

```bash
make mobile-install   # npm install --legacy-peer-deps
make mobile-start     # expo start (choose platform interactively)
make mobile-android   # expo start --android
make mobile-ios       # expo start --ios
make mobile-web       # expo start --web
```

### Release builds (EAS)

```bash
make mobile-build-prod        # Android production build (.aab)
make mobile-build-prod-ios    # iOS production build (.ipa, cloud macOS worker - no Mac needed)
make mobile-submit-android    # Upload latest build to Play Console
make mobile-submit-ios        # Upload latest build to TestFlight
make mobile-device-ios        # Register an iPhone UDID for development builds
make mobile-versions          # Show remote versionCode / buildNumber counters
```

---

## Dev cost: zero

During local development no paid APIs are called:

| Service | Dev | Prod |
|---|---|---|
| Transcription | faster-whisper-server (local CPU) | OpenAI Whisper API |
| AI analysis | Stubbed (`STUB_AI_ANALYSIS=true`) | Anthropic Claude API |
| TTS (Therapy Mode) | Skipped / stubbed | Azure Speech TTS (OpenAI TTS fallback) |
| IAP verification | Skipped - plans granted without proof | Apple `verifyReceipt` + Google Play Developer API |
| Push notifications | Skipped (no FCM credentials) | Firebase Cloud Messaging |
| Storage | MinIO (local Docker) | Cloudflare R2 |
| Auth | Manually generated JWT | Supabase Auth |

To enable real AI analysis: set `ANTHROPIC_API_KEY=<key>` and `STUB_AI_ANALYSIS=false` in `.env`, then `docker compose up -d --force-recreate worker`.

---

## Database schema (summary)

```
users                    - supabase_id, email, name, preferred_name, timezone, fcm_nudge_hour,
                           nudge_auto_time, goal, age_range, voice_language, plan, plan_expires_at,
                           streak_freeze_count, tos_accepted_at
entries                  - user_id, audio_key, duration_sec, status, transcript, search_vector
entry_analysis           - entry_id, mood_score, emotional_tone (JSONB), topics[], key_quotes[],
                           reflection, morning_nudge, is_crisis, connection_insight,
                           dream_symbols[] / dream_type / psychological_lens / vedic_lens (dream mode)
conversations            - entry_id, user_id, turn_count, is_closed
conversation_messages    - conversation_id, role, content
user_devices             - user_id, fcm_token, platform
nudges                   - user_id, entry_id, message, status, scheduled_at,
                           nudge_type (morning | reengagement | streak_risk | checkin |
                           plan_expiring_soon | plan_expired)
dead_letter_jobs         - entry_id, payload, error, attempt (failed worker jobs)
payments                 - user_id, transaction_id (unique), plan, store, product_id, country
analytics_events         - user_id, event_name, properties (JSONB) - privacy-safe product events
weekly_reviews           - user_id, week_start, narrative, top_emotions[], mood_arc (JSONB)
annual_reviews           - user_id, year, narrative, top_emotions[], top_topics[], mood_arc (JSONB)
journey_templates        - id, title, description, step_count, tags[], prompts[]
journey_sessions         - user_id, journey_id, current_step, status
journey_steps            - session_id, step_index, entry_id, completed
life_chapters            - user_id, title, description, start_date, end_date, emoji, color, summary
people / person_mentions - relationship map: extracted persons + per-entry sentiment mentions
therapy_sessions         - user_id, status, persona, started_at, expires_at,
                           crisis_warnings, context_snapshot (JSONB), post_session_summary
therapy_session_messages - session_id, role, content, input_mode
therapist_keys           - therapist_id, wrapped_dek (envelope encryption)
external_clients         - therapist_id, name_enc (AES-256-GCM), archived
client_sessions          - therapist_id, external_client_id/linked_user_id, status,
                           raw_text_enc, bullets_enc, summary_enc (all encrypted)
companies / company_members - B2B corporate wellness tenancy
share_links              - therapist-sharing tokens (passcode-hashed, 72h TTL)
```

Migrations are in `backend/migrations/` (35+ files) and run automatically on API startup. Full column-level detail: [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md).

---

## Crisis detection

Two-stage pipeline - fast and safe, applied to journal entries **and** every therapy session message:

1. **Stage 1 - keyword match** (`<1 ms`): 20+ high-certainty phrases → immediate crisis response with hotline numbers
2. **Stage 2 - Claude confirmation** (`~1 s`): ambiguous phrases → Claude yes/no prompt; fail-safe defaults to crisis if Claude is unreachable or times out

Crisis entries skip the regular reflection flow and are never included in mood statistics. In Therapy Mode, crisis handling is layered: the first detection triggers a de-escalating response and keeps the session open (`crisis_warnings=1`); a second detection hard-stops the session with crisis resources. See ADR-002, ADR-013, ADR-014 in `docs/DECISIONS.md`.

---

## Security & privacy

- **Audio is deleted** from storage immediately after successful transcription (journal entries and therapy voice turns) - see ADR-005.
- **Note photos are deleted** immediately after OCR extraction succeeds - see ADR-019.
- **Therapist notes are encrypted at rest** via per-therapist envelope encryption (AES-256-GCM DEK wrapped by `MASTER_ENCRYPTION_KEY`) - not end-to-end; see ADR-017.
- **Therapist access to a client's journal requires the client's explicit consent** - a link request grants no data access until approved.
- **Pre-signed URLs** mean audio/photo bytes never transit the API server - see ADR-008.

---

## Further reading

| Doc | Purpose |
|---|---|
| [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) | Full system design, data flow, layer map, DB schema |
| [docs/API_CONTRACT.md](docs/API_CONTRACT.md) | Exact request/response shapes for every endpoint |
| [docs/DECISIONS.md](docs/DECISIONS.md) | ADRs - why things are built the way they are |
| [docs/TESTING.md](docs/TESTING.md) | Test priorities, coverage, mock patterns |
| [docs/ROADMAP.md](docs/ROADMAP.md) | Phase-by-phase status and shipped features |
| [docs/LAUNCH_CHECKLIST.md](docs/LAUNCH_CHECKLIST.md) | Everything left before App Store / Play Store launch |
| [docs/THERAPIST_PORTAL.md](docs/THERAPIST_PORTAL.md) | Therapist workspace feature doc |
| [docs/PRICING.md](docs/PRICING.md) | Plan tiers, unit economics, pricing decisions |
| [ANTIGRAVITY.md](ANTIGRAVITY.md) | Rules for the Antigravity AI coding assistant |
| [backend/CLAUDE.md](backend/CLAUDE.md) | Backend Go patterns, invariants, env vars |
| [mobile/CLAUDE.md](mobile/CLAUDE.md) | Mobile React Native patterns and rules |
