# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Required Reading Before Making Changes

Always read these files before touching code. They define contracts, invariants, and decisions that must not be broken silently.

- @docs/ARCHITECTURE.md - system design, data flow, full layer map, DB schema
- @docs/API_CONTRACT.md - request/response shapes for every endpoint
- @docs/DECISIONS.md - why things are built the way they are (read before proposing changes to core patterns)
- @docs/TESTING.md - what must be tested, how to structure tests
- @docs/ROADMAP.md - phase status and what's planned next
- @docs/LAUNCH_CHECKLIST.md - everything left before App Store / Play Store launch (read before release/store work)
- @ANTIGRAVITY.md - rules, guidelines, and commands for the Antigravity AI coding assistant

When working in the backend: also read @backend/CLAUDE.md
When working in the mobile: also read @mobile/CLAUDE.md

## Project Overview

DreamLog is a voice journaling app with AI-powered reflections. Users record audio journals; the backend transcribes them, runs crisis screening, builds context from recent entries, and calls Claude to generate personalized reflections. Users can follow up with a 3-turn conversation.

## Commands

### Backend (Go)

```bash
go run ./cmd/api        # Start the HTTP API server
go run ./cmd/worker     # Start the async job processor
go mod tidy             # Tidy dependencies
go test ./...           # Run all tests
go test ./internal/services/...  # Run tests in a specific package
```

### Mobile (React Native / Expo)

```bash
cd mobile
npm install --legacy-peer-deps   # Install dependencies (flag required - see mobile/CLAUDE.md)
npx expo start          # Start dev server
npx expo start --android
npx expo start --ios
```

### Release builds (EAS - one codebase, both stores)

```bash
make mobile-build-prod        # Android production build (.aab)
make mobile-build-prod-ios    # iOS production build (.ipa, cloud macOS worker - no Mac needed)
make mobile-submit-android    # Upload latest build to Play Console
make mobile-submit-ios        # Upload latest build to TestFlight
make mobile-device-ios        # Register an iPhone UDID for development builds
make mobile-versions          # Show remote versionCode / buildNumber counters
```

Versioning: bump `version` in `mobile/app.json` once per release (shared by both platforms);
per-platform build numbers are auto-incremented remotely by EAS. See docs/LAUNCH_CHECKLIST.md.

### Docker / Make (recommended for full-stack dev)

```bash
make dev                # Start all services (Postgres, Redis, MinIO, Whisper, API, Worker)
make dev-stop           # Stop all services
make dev-logs           # Tail logs
make dev-restart        # Restart all services

make db-migrate         # Run pending migrations
make db-reset           # Drop and re-migrate (destructive)
make db-psql            # Open psql shell

make mobile-start       # Start Expo dev server
make mobile-android
make mobile-ios
```

Copy `.env.example` to `.env` before running - it's pre-filled for local dev (no external API keys required).

## Architecture

### Request Flow

```
Mobile → POST /entries/presign → backend returns { upload_url, audio_key }
Mobile → PUT audio directly to MinIO/R2 (pre-signed URL, no backend bandwidth)
Mobile → POST /entries { audio_key, ... } → backend creates DB row, pushes job to Redis
Worker → BRPOP job → transcribe → crisis screen → build context → Claude analyze → store analysis
Mobile → polls GET /entries/:id until status='completed'
Mobile → GET /entries/:id/analysis → display reflection
Mobile → POST /entries/:id/conversation + POST /conversations/:id/messages (max 3 turns)
```

### Backend (`backend/`)

- **`cmd/api`** - Gin HTTP server entry point; runs migrations on startup
- **`cmd/worker`** - Long-running worker; BRPOPs from Redis, processes entries sequentially
- **`internal/handlers/`** - Thin HTTP handlers (router.go wires everything via DI)
- **`internal/services/`** - All business logic; key files:
  - `claude.go` - Anthropic API calls (entry analysis + conversation turns)
  - `transcription.go` - Whisper API (or local faster-whisper-server in dev)
  - `crisis.go` - Two-stage detection: keyword match → Claude confirmation; fails safe (treats uncertain as crisis)
  - `context_builder.go` - Fetches last 5 entries to inject into Claude prompts
  - `prompts.go` - All Claude system prompts live here
  - `auth.go` - Local email/password auth: bcrypt hashing + JWT minting (separate from Supabase path)
  - `fcm.go` - Firebase Cloud Messaging via HTTP v1 API; skips gracefully when credentials absent
  - `nudge.go` - Morning nudge scheduling logic (timezone-aware, uses `users.fcm_nudge_hour`)
- **`internal/workers/transcription.go`** - Full processing pipeline per entry
- **`internal/workers/nudge_scheduler.go`** - Separate goroutine; polls every minute for due nudges and dispatches FCM
- **`internal/repositories/`** - pgx queries; entries use PostgreSQL `tsvector` + GIN index for full-text search; `nudges.go` handles device tokens and nudge lifecycle
- **`pkg/queue/`** - Redis LPUSH/BRPOP abstraction
- **`pkg/storage/`** - S3-compatible client (MinIO locally, Cloudflare R2 in prod)
- **`migrations/`** - SQL files managed by golang-migrate; run automatically on startup

**Auth:** Two paths coexist. `POST /auth/register` and `POST /auth/login` use local bcrypt + JWT (`internal/services/auth.go`). All other routes use `internal/middleware/auth.go` which validates Supabase JWTs (HS256) and auto-provisions users on first request.

### Mobile (`mobile/`)

- **`app/`** - expo-router file-based routes; `(tabs)/` is the bottom tab group
- **`src/api/client.ts`** - Typed Axios instance; reads JWT from expo-secure-store and injects `Authorization` header
- **`src/hooks/useRecorder.ts`** - Audio recording state machine (expo-av); AAC 44.1kHz mono, max 30 min
- **`src/services/upload.ts`** - Orchestrates presign → PUT → POST with 3-attempt exponential backoff
- **`src/services/offlineQueue.ts`** - AsyncStorage-based queue; auto-flushes on reconnect, max 5 retries
- **`src/services/push.ts`** - FCM push registration (permission → token → `POST /devices`); fail-silent, called from `app/_layout.tsx` on auth
- **`src/theme.ts`** - Design tokens (dark purple palette, CormorantGaramond + Nunito fonts)

No Redux or global state manager - component state via React hooks, persistence via AsyncStorage or SecureStore.

## Key Design Decisions

- **Workers are stateless and horizontally scalable** - Redis queue handles coordination; audio is deleted from storage after processing.
- **Crisis detection fails safe** - If Claude can't confirm non-crisis, the entry is treated as a crisis.
- **Dev environment requires no external APIs** - Claude calls are stubbed, local Whisper server is used, MinIO replaces R2.
- **Follow-up conversations are capped at 3 user turns** (`is_closed` flag on `conversations` table).
- **Typed routes** - `experiments.typedRoutes: true` in `app.json`; use typed `href` from `expo-router`.

## Environment

All config is injected via environment variables (see `.env.example`). Key vars:

| Variable | Purpose |
|---|---|
| `DATABASE_URL` | PostgreSQL connection string |
| `REDIS_ADDR` | Redis address |
| `JWT_SECRET` | Supabase JWT secret |
| `ANTHROPIC_API_KEY` | Claude API (blank = stubbed in dev) |
| `STORAGE_ENDPOINT` | MinIO (dev) or R2 endpoint |
| `WHISPER_API_URL` | Local Whisper server or OpenAI API URL |
| `AZURE_TTS_KEY` / `AZURE_TTS_REGION` | Azure Speech TTS for therapy voice (blank = OpenAI fallback or skipped in dev) |
| `EXPO_PUBLIC_API_URL` | Backend URL consumed by mobile |
