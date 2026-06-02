# DreamLog

A production-grade voice journaling app. Speak your day — DreamLog transcribes it, reflects back what it hears, and tracks your emotional patterns over time.

---

## Architecture

```
dream/
├── backend/          Go 1.23 · Gin · PostgreSQL · Redis · MinIO
│   ├── cmd/
│   │   ├── api/      HTTP server
│   │   └── worker/   Transcription + AI analysis worker
│   ├── internal/
│   │   ├── config/
│   │   ├── handlers/     HTTP route handlers
│   │   ├── middleware/   Auth (JWT), CORS, logging, errors
│   │   ├── models/       Domain types
│   │   ├── repositories/ DB queries (pgx/v5)
│   │   ├── services/     Business logic
│   │   └── workers/      Async jobs (transcription, nudge scheduler)
│   ├── migrations/   golang-migrate SQL files
│   └── pkg/
│       ├── queue/    Redis job queue (BRPOP/LPUSH)
│       └── storage/  S3-compatible storage client
└── mobile/           React Native · Expo 51 · expo-router
    ├── app/
    │   ├── (tabs)/   Home, Timeline, Mood, Settings
    │   ├── record.tsx
    │   ├── processing/[id].tsx
    │   ├── reflection/[id].tsx
    │   └── followup/[id].tsx
    └── src/
        ├── api/      Axios client + typed endpoints
        ├── hooks/    useRecorder
        ├── services/ upload, offlineQueue
        ├── theme.ts  Design tokens
        └── types/    Shared TypeScript types
```

### Tech stack

| Layer | Tech |
|---|---|
| API | Go 1.23, Gin, `pgx/v5`, `go-redis/v9`, `aws-sdk-go-v2` |
| Auth | Supabase JWT (HS256), `golang-jwt/jwt/v5` |
| DB | PostgreSQL 16 — migrations via `golang-migrate` |
| Queue | Redis list (`BRPOP` / `LPUSH`) — async job pipeline |
| Storage | MinIO (dev) / Cloudflare R2 (prod) — pre-signed PUT URLs |
| Transcription | faster-whisper-server (local dev) / OpenAI Whisper (prod) |
| AI analysis | Claude Sonnet 4.6 via Anthropic Messages API (stub mode in dev) |
| TTS | OpenAI TTS — Therapy Mode AI voice output (optional, skipped in dev) |
| Mobile | React Native 0.74, Expo 51, expo-router, expo-av |
| Fonts | Cormorant Garamond (serif) + Nunito (sans) |

---

## Pipelines

### Recording → Reflection

```
Mobile                     Backend                    Workers
  │                           │                          │
  ├─ POST /entries/presign ──►│ returns pre-signed URL   │
  ├─ PUT audio ─────────────► MinIO                      │
  ├─ POST /entries ─────────►│ creates DB row            │
  │                           │ pushes job → Redis ──────►│
  │                           │                          │ BRPOP job
  │                           │                          │ Whisper transcribe
  │                           │                          │ Crisis screen (keyword + Claude)
  │                           │                          │ Context builder (last 5 entries)
  │                           │                          │ Claude AnalyzeEntry
  │                           │                          │ Store analysis
  │                           │                          │ Schedule morning nudge
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

### Therapy Mode (Phase 6)

Real-time AI-assisted voice/text conversation. Unlike journal entries (async, worker-processed), therapy sessions are synchronous — response comes back in the same HTTP request. Sessions are pre-loaded with the user's journal history (mood trends, top emotions, recent entry summaries) so Claude starts with context.

```
Mobile                     API (synchronous — no worker)
  │                           │
  ├─ POST /therapy/sessions ─►│ INSERT session, load journal context
  │                           │ ← { session_id, expires_at }
  │                           │
  ├─ POST /therapy/sessions/:id/presign ─►│ ← { upload_url, audio_key }
  ├─ PUT audio ─────────────► MinIO
  │                           │
  ├─ POST /therapy/sessions/:id/messages ─►│
  │                           │ Whisper transcribe (synchronous)
  │                           │ Crisis detection (Stage 1 + Stage 2, fail-safe)
  │                           │ Claude therapy turn (journal context + history)
  │                           │ Delete audio from storage
  │                           │ ← { user_message, assistant_message, session_state }
  │                           │
  ├─ POST /therapy/sessions/:id/end ─►│ Generate post-session summary
  │                           │ ← { summary, duration_sec }
```

**1-hour time limit** enforced server-side via `expires_at`. Crisis detection is mandatory and uses the same two-stage fail-safe as journal entries.

---

## Quick start

### Prerequisites

- Docker Desktop
- Node.js 20+ (for mobile)
- Expo Go app on your phone **or** Android/iOS simulator

### 1 — Clone and configure

```bash
git clone <repo>
cd dream
cp .env.example .env   # already pre-filled for local dev
```

### 2 — Start the stack

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

### 3 — Run the mobile app

```bash
make mobile-install   # npm install
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

### 4 — Authenticate (dev mode)

The app opens a JWT paste screen. Generate a test token at **[jwt.io](https://jwt.io)**:

| Field | Value |
|---|---|
| Algorithm | HS256 |
| Payload | `{"sub":"test-user-001","email":"test@dreamlog.dev"}` |
| Secret | your `SUPABASE_JWT_SECRET` from `.env` |

Copy the encoded token and paste it into the app.

---

## Environment variables

All variables live in `.env` at the project root (used by Docker Compose).

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

# Transcription — points to local Whisper in dev
OPENAI_API_KEY=ignored
OPENAI_BASE_URL=http://whisper:8000/v1
WHISPER_MODEL=Systran/faster-whisper-base

# AI analysis — stub mode = no API calls, zero cost in dev
ANTHROPIC_API_KEY=                # leave blank in dev
STUB_AI_ANALYSIS=true             # set false + add key for real analysis

# Worker
WORKER_CONCURRENCY=4
WORKER_MAX_RETRIES=3
```

---

## API reference

All endpoints (except `/health`) require `Authorization: Bearer <jwt>`.

### User

| Method | Path | Description |
|---|---|---|
| `GET` | `/me` | Get or create authenticated user |
| `PUT` | `/me` | Update display name |

### Entries

| Method | Path | Description |
|---|---|---|
| `POST` | `/entries/presign` | Get pre-signed PUT URL for audio upload |
| `POST` | `/entries` | Create entry record + queue transcription job |
| `GET` | `/entries` | List entries (paginated) |
| `GET` | `/entries/:id` | Get single entry |
| `GET` | `/entries/search?q=` | Full-text search via `tsvector` |

### Analysis & Timeline

| Method | Path | Description |
|---|---|---|
| `GET` | `/entries/:id/analysis` | Get AI analysis for a completed entry |
| `GET` | `/timeline` | Entries with their analyses (paginated) |

### Conversations

| Method | Path | Description |
|---|---|---|
| `POST` | `/entries/:id/conversation` | Get or create follow-up conversation |
| `POST` | `/conversations/:id/messages` | Send a message (max 3 user turns) |

### Mood

| Method | Path | Description |
|---|---|---|
| `GET` | `/mood/weekly` | Average mood per day for last 7 days |
| `GET` | `/mood/streak` | Current streak, longest streak, total days |

### Therapy Mode

| Method | Path | Description |
|---|---|---|
| `POST` | `/therapy/sessions` | Start a session (loads journal context, charges billing) |
| `POST` | `/therapy/sessions/:id/presign` | Pre-signed URL for voice turn upload |
| `POST` | `/therapy/sessions/:id/messages` | Send turn (voice or text), get AI response |
| `POST` | `/therapy/sessions/:id/end` | End session, generate post-session summary |
| `GET` | `/therapy/sessions/:id` | Session state + full message history |
| `GET` | `/therapy/sessions` | Session history list |

### Devices

| Method | Path | Description |
|---|---|---|
| `POST` | `/devices` | Register FCM token for push notifications |

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
make mobile-install   # npm install
make mobile-start     # expo start (choose platform interactively)
make mobile-android   # expo start --android
make mobile-ios       # expo start --ios
make mobile-web       # expo start --web
```

---

## Dev cost: zero

During local development no paid APIs are called:

| Service | Dev | Prod |
|---|---|---|
| Transcription | faster-whisper-server (local CPU) | OpenAI Whisper API |
| AI analysis | Stubbed (`STUB_AI_ANALYSIS=true`) | Anthropic Claude API |
| TTS (Therapy Mode) | Skipped / stubbed | OpenAI TTS API |
| Storage | MinIO (local Docker) | Cloudflare R2 |
| Auth | Manually generated JWT | Supabase Auth |

To enable real AI analysis: set `ANTHROPIC_API_KEY=<key>` and `STUB_AI_ANALYSIS=false` in `.env`, then `docker compose up -d --force-recreate worker`.

---

## Database schema (summary)

```
users                    — supabase_id, email, name, timezone, fcm_nudge_hour
entries                  — user_id, audio_key, duration_sec, status, transcript, search_vector
entry_analysis           — entry_id, mood_score, emotional_tone (JSONB), topics[], reflection, is_crisis
conversations            — entry_id, user_id, turn_count, is_closed
conversation_messages    — conversation_id, role, content
therapy_sessions         — user_id, status, started_at, expires_at, context_snapshot (JSONB), post_session_summary
therapy_session_messages — session_id, role, content, input_mode
user_devices             — user_id, fcm_token, platform
nudges                   — user_id, entry_id, message, status, scheduled_for
```

Migrations are in `backend/migrations/` and run automatically on API startup.

---

## Crisis detection

Two-stage pipeline — fast and safe:

1. **Stage 1 — keyword match** (`<1 ms`): 20+ high-certainty phrases → immediate crisis response with hotline numbers
2. **Stage 2 — Claude confirmation** (`~1 s`): ambiguous phrases → Claude yes/no prompt; fail-safe defaults to crisis if Claude is unreachable

Crisis entries skip the regular reflection flow and are never included in mood statistics.
