# DreamLog Architecture

## System Overview

DreamLog is a voice journaling app. The core pipeline: user speaks → audio uploaded to object storage → worker transcribes → AI analyzes → reflection stored → mobile displays.

Two independent runtimes:
- **API** (`cmd/api`) — HTTP server, handles all client requests, runs DB migrations on startup
- **Worker** (`cmd/worker`) — long-running process, BRPOPs jobs from Redis, runs the transcription + AI pipeline

They share the same PostgreSQL DB and Redis instance but have no direct code dependency on each other — only through the queue.

---

## Request Flow (Happy Path)

```
Mobile
  │
  ├─ POST /entries/presign
  │     ← { upload_url, audio_key }           # pre-signed PUT URL, no backend bandwidth
  │
  ├─ PUT audio → MinIO/R2 (direct, bypasses backend)
  │
  ├─ POST /entries { audio_key, duration_sec }
  │     backend: INSERT entries (status=pending), LPUSH job to Redis
  │     ← { id, status: "pending" }
  │
  │   [Worker picks up job via BRPOP]
  │     → Whisper transcribe audio
  │     → Update entry: transcript, status=processing
  │     → Stage 1 crisis: keyword match (<1 ms)
  │     → Stage 2 crisis: Claude yes/no if ambiguous (~1 s)
  │     → If crisis: store is_crisis=true, skip reflection, send crisis response
  │     → Context builder: fetch last 5 completed entries for this user
  │     → Claude AnalyzeEntry: structured JSON reflection
  │     → INSERT entry_analysis (mood_score, emotional_tone, topics, key_quotes, reflection, morning_nudge)
  │     → Update entry: status=completed
  │     → Schedule morning nudge (INSERT nudges, scheduled_at = user's local 8 AM next day)
  │     → DELETE audio from storage
  │
  ├─ GET /entries/:id (polls until status=completed)
  │
  ├─ GET /entries/:id/analysis
  │     ← { reflection, mood_score, emotional_tone, topics, key_quotes, morning_nudge }
  │
  ├─ POST /entries/:id/conversation    # creates Conversation row if not exists
  │
  └─ POST /conversations/:id/messages  # up to 3 user turns, then is_closed=true
```

---

## Backend Layer Map

```
cmd/api/main.go
  └─ config.Load()
  └─ DB connect + migrate
  └─ Redis connect
  └─ Storage client init
  └─ handlers.SetupRouter(deps) → gin.Engine

cmd/worker/main.go
  └─ config.Load()
  └─ DB connect
  └─ Redis connect
  └─ Storage client init
  └─ go nudge_scheduler.Run()       # goroutine, polls every 60s
  └─ worker.Run()                   # blocks on BRPOP

internal/handlers/router.go         # ALL dependency injection happens here
  └─ wires repos + services → handlers
  └─ registers all routes with middleware

internal/handlers/router.go         # ALL DI wiring + route registration (single source)
internal/handlers/entries.go        # entry CRUD + presign
internal/handlers/analysis.go       # analysis, timeline, search
internal/handlers/conversations.go  # follow-up conversations
internal/handlers/users.go          # /me GET/PUT
internal/handlers/auth.go           # /auth/register, /auth/login
internal/handlers/billing.go        # /billing/plan, /billing/upgrade
internal/handlers/weekly_reviews.go # /reviews/weekly
internal/handlers/year_in_review.go # /reviews/annual (Plus+ only)
internal/handlers/insight.go        # /insights/card, /insights/share
internal/handlers/share.go          # /share CRUD + public view
internal/handlers/export.go         # /export/pdf (Pro+ only)
internal/handlers/b2b.go            # /b2b/companies
internal/handlers/therapist.go      # /therapists dashboard API
internal/handlers/journey.go        # /journeys templates + sessions
internal/handlers/life_chapters.go  # /chapters CRUD + summarize
internal/handlers/relationships.go  # /relationships map
internal/handlers/therapy.go        # /therapy/sessions (Phase 6)
internal/services/*.go              # all business logic lives here
internal/repositories/*.go          # all DB queries (pgx/v5), no logic
internal/workers/transcription.go   # full pipeline orchestration (calls services)
internal/workers/nudge_scheduler.go # polls nudges table, calls fcm service
internal/models/*.go                # plain Go structs, no methods
internal/middleware/*.go            # auth, cors, logging, error formatting
migrations/*.sql                    # golang-migrate, auto-run on API startup (18 migrations)
pkg/queue/redis.go                  # LPUSH / BRPOP wrapper
pkg/storage/s3.go                   # S3-compatible client (MinIO dev, R2 prod)
pkg/apierr/errors.go                # typed API errors
```

---

## Database Schema

```sql
users
  id UUID PK
  supabase_id TEXT UNIQUE       -- from Supabase JWT sub claim
  email TEXT UNIQUE
  name TEXT
  preferred_name TEXT           -- shown in AI reflections instead of name if set
  timezone TEXT DEFAULT 'UTC'
  fcm_nudge_hour INT DEFAULT 8  -- local hour to send morning nudge (0-23)
  nudge_enabled BOOLEAN DEFAULT true
  goal TEXT                     -- stress|anxiety|grief|depression|relationships|career|trauma|curious
  age_range TEXT                -- under_18|18_24|25_34|35_44|45_plus; nullable (migration 000021)
  plan TEXT DEFAULT 'free'      -- free | plus | pro | b2b (migration 000011)
  plan_expires_at TIMESTAMPTZ   -- NULL = no expiry
  password_hash TEXT            -- only set for local-auth users
  streak_freeze_count INT DEFAULT 0
  streak_freeze_granted_week DATE
  created_at, updated_at TIMESTAMPTZ

entries
  id UUID PK
  user_id UUID FK → users
  audio_key TEXT                -- MinIO/R2 object key (deleted after processing)
  audio_size_bytes BIGINT
  duration_sec FLOAT
  status entry_status           -- ENUM: pending | processing | completed | failed
  transcript TEXT
  language TEXT                 -- detected by Whisper
  search_vector tsvector        -- GIN indexed, auto-updated by trigger on transcript change
  error_msg TEXT
  retry_count INT DEFAULT 0
  created_at, updated_at TIMESTAMPTZ

entry_analysis                  -- 1:1 with entries
  id UUID PK
  entry_id UUID FK → entries UNIQUE
  mood_score INT (1-100)        -- 1=crisis-level, 50=neutral, 100=euphoric
  emotional_tone JSONB          -- [{"emotion": str, "intensity": float}]
  topics TEXT[]                 -- 2-5 concrete topics from the entry
  key_quotes TEXT[]             -- verbatim or near-verbatim phrases
  summary TEXT                  -- factual, third-person, 2-3 sentences
  reflection TEXT               -- warm 3-5 sentences + one open question
  morning_nudge TEXT            -- 1 sentence, specific to this entry
  is_crisis BOOLEAN DEFAULT false
  created_at, updated_at TIMESTAMPTZ

conversations                   -- 1:1 with entries (one follow-up per entry)
  id UUID PK
  entry_id UUID FK → entries UNIQUE
  user_id UUID FK → users
  turn_count INT DEFAULT 0      -- incremented on each user message; max 3
  is_closed BOOLEAN DEFAULT false
  created_at, updated_at TIMESTAMPTZ

conversation_messages
  id UUID PK
  conversation_id UUID FK → conversations
  role TEXT CHECK (role IN ('user','assistant'))
  content TEXT
  created_at TIMESTAMPTZ

user_devices                    -- FCM tokens per device
  id UUID PK
  user_id UUID FK → users
  fcm_token TEXT UNIQUE
  platform TEXT                 -- 'ios' | 'android' | 'unknown'
  created_at, updated_at TIMESTAMPTZ

nudges                          -- morning push notifications
  id UUID PK
  user_id UUID FK → users
  entry_id UUID FK → entries    -- the entry that generated this nudge
  message TEXT
  scheduled_at TIMESTAMPTZ      -- user's local 8 AM
  timezone TEXT
  status nudge_status           -- ENUM: pending | sent | failed
  sent_at TIMESTAMPTZ
  error_msg TEXT
  created_at TIMESTAMPTZ

dead_letter_jobs                -- failed worker jobs for audit / retry
  id UUID PK
  entry_id UUID FK → entries
  payload JSONB
  error TEXT
  attempt INT
  created_at TIMESTAMPTZ

weekly_reviews                  -- Claude-generated weekly narrative
  id UUID PK
  user_id UUID FK → users
  week_start DATE               -- Sunday of the reviewed week
  narrative TEXT
  top_emotions TEXT[]
  mood_arc JSONB                -- [{date, avg_mood}]
  entry_count INT
  status TEXT                   -- pending | completed | failed
  scheduled_at, generated_at TIMESTAMPTZ

annual_reviews                  -- Claude-generated yearly narrative (migration 000014)
  id UUID PK
  user_id UUID FK → users
  year INT UNIQUE per user
  narrative TEXT
  top_emotions TEXT[]
  top_topics TEXT[]
  mood_arc JSONB                -- [{month:"YYYY-MM", avg_mood, entry_count}]
  entry_count INT
  avg_mood INT
  status TEXT                   -- pending | completed | failed
  scheduled_at, generated_at TIMESTAMPTZ

journey_templates               -- seeded guided journey configs (migration 000013)
  id TEXT PK                    -- e.g. "stress_relief"
  title TEXT
  description TEXT
  step_count INT
  estimated_minutes INT
  tags TEXT[]
  prompts TEXT[]

journey_sessions                -- user's progress through a journey
  id UUID PK
  user_id UUID FK → users
  journey_id TEXT FK → journey_templates
  current_step INT DEFAULT 0
  status TEXT                   -- in_progress | completed
  created_at, updated_at TIMESTAMPTZ

journey_steps                   -- one row per step per session
  id UUID PK
  session_id UUID FK → journey_sessions
  step_index INT
  entry_id UUID FK → entries    -- NULL until step completed
  completed BOOLEAN DEFAULT false

life_chapters                   -- user-defined named time periods (migration 000015)
  id UUID PK
  user_id UUID FK → users
  title TEXT
  description TEXT
  start_date DATE
  end_date DATE                 -- NULL = ongoing
  emoji TEXT
  color TEXT DEFAULT '#7C3AED'
  summary TEXT                  -- Claude-generated on demand via POST /chapters/:id/summarize
  created_at, updated_at TIMESTAMPTZ

people                          -- relationship map: persons extracted from entries (migration 000017)
  id UUID PK
  user_id UUID FK → users
  name TEXT                     -- case-insensitive unique per user
  role TEXT                     -- family | friend | colleague | romantic | other
  mention_count INT
  positive_count INT
  negative_count INT
  last_mentioned_at TIMESTAMPTZ
  created_at, updated_at TIMESTAMPTZ

person_mentions                 -- one row per person per entry
  id UUID PK
  person_id UUID FK → people
  entry_id UUID FK → entries
  user_id UUID FK → users
  sentiment TEXT                -- positive | neutral | negative
  context TEXT                  -- excerpt from entry mentioning the person

therapy_sessions                -- therapy mode sessions (migrations 000018, 000020)
  id UUID PK
  user_id UUID FK → users
  status therapy_session_status -- ENUM: active | completed | expired | crisis_detected
  persona TEXT DEFAULT 'comforting' -- comforting | rational | cbt | mindful (migration 000020)
  started_at TIMESTAMPTZ
  expires_at TIMESTAMPTZ        -- started_at + 1 hour, enforced server-side
  ended_at TIMESTAMPTZ
  duration_sec INT
  turn_count INT DEFAULT 0
  crisis_warnings INT DEFAULT 0 -- 0=none; 1=de-escalating (next detection→hard stop) (migration 000020)
  context_snapshot JSONB        -- mood_avg_30d, top_emotions, top_topics, recent_summaries, past_session_summaries
  post_session_summary TEXT
  billing_amount_paise INT      -- 49900 = ₹499; 0 = covered by plan
  created_at TIMESTAMPTZ

therapy_messages
  id UUID PK
  session_id UUID FK → therapy_sessions
  role TEXT CHECK (role IN ('user','assistant'))
  content TEXT
  input_mode TEXT               -- 'voice' | 'text'
  created_at TIMESTAMPTZ

-- Views
v_daily_mood: per-user daily avg mood_score (excludes crisis entries)
```

---

## Auth Architecture

Two paths coexist and must not be conflated:

**Path A — Supabase JWT (all protected routes)**
- Mobile generates JWT via Supabase Auth
- Dev: manually generate at jwt.io with `SUPABASE_JWT_SECRET`
- `internal/middleware/auth.go` validates HS256 signature, extracts `sub` + `email`
- If user doesn't exist in DB yet → auto-provisions (INSERT users)
- Sets `userID` in Gin context for downstream handlers

**Path B — Local email/password (only `/auth/register` and `/auth/login`)**
- `internal/services/auth.go`: bcrypt hash on register, compare on login
- Mints its own JWT with same secret as Supabase (so the middleware can validate it)
- Useful for dev and for users who don't want Supabase

Both paths produce the same JWT format — middleware doesn't know which path minted the token.

---

## Crisis Detection (Safety-Critical)

**Stage 1 — Keyword Match** (`services/crisis.go`)
- ~20 high-certainty phrases checked against transcript (case-insensitive, O(n) string scan)
- Latency: <1 ms
- No AI call — deterministic
- If match: immediately mark `is_crisis=true`, return crisis resource message with hotlines

**Stage 2 — Claude Confirmation**
- Triggered when Stage 1 doesn't match but transcript has ambiguous distress signals
- Sends a yes/no prompt to Claude: "Does this transcript indicate active suicidal ideation or intent to harm?"
- If Claude returns "yes" OR if Claude is unreachable (network error, timeout, API error) → treated as crisis
- Fail-safe: uncertain = crisis. This is intentional and must never be changed.

Crisis entries:
- Get `is_crisis=true` in `entry_analysis`
- Are excluded from `v_daily_mood` view (don't skew mood stats)
- Skip regular reflection generation entirely
- Trigger crisis resource message instead of reflection

---

## AI Prompt Architecture

All prompts live in `internal/services/prompts.go`. Never scatter prompt strings elsewhere.

**AnalyzeEntry** — `buildSystemPrompt()` + `buildUserPrompt(input)`
- System: fixed instructions, output schema, few-shot examples, forbidden words
- User: user context (name, account age, emotion trend, topic trend) + last 5 entry summaries + current transcript
- Output: strict JSON with 7 fields (emotional_tone, topics, mood_score, key_quotes, summary, reflection, morning_nudge)

**Follow-up Conversation** — `buildFollowUpSystemPrompt(transcript, reflection)`
- Injects original transcript + original reflection into system prompt
- Stateless per turn — full message history sent each call
- Max 3 user turns enforced in `services/conversation.go`, not in the prompt

**Context Builder** (`services/context_builder.go`)
- Fetches last 5 completed entries for the user
- Derives `EmotionTrend` (most frequent emotions across entries)
- Derives `TopicTrend` (most frequent topics across entries)
- Injects both into `buildUserPrompt` — this is what makes reflections feel personalized

---

## Worker Concurrency

- Single `BRPOP` loop — processes one job at a time per worker process
- Scale horizontally: `make scale-worker N=3` runs 3 worker replicas
- Each worker is stateless — safe to run N replicas
- Redis queue is the coordination point — atomic BRPOP ensures no double-processing
- Failed jobs: after `WORKER_MAX_RETRIES` attempts → inserted into `dead_letter_jobs`

---

## Storage Flow

Audio lifecycle:
1. Client gets pre-signed PUT URL from `POST /entries/presign`
2. Client PUTs audio directly to MinIO/R2 (backend never sees the bytes)
3. Worker downloads audio from storage to process with Whisper
4. Worker deletes audio from storage after successful transcription
5. Audio key remains in `entries.audio_key` for audit but object no longer exists

In dev: MinIO at `http://localhost:9001` (console), bucket `dreamlog-audio`
In prod: Cloudflare R2, same S3-compatible API

---

## Nudge Scheduler

Runs as a goroutine inside the worker process (`workers/nudge_scheduler.go`):
- Polls `nudges` table every 60 seconds
- Finds rows where `status='pending'` AND `scheduled_at <= NOW()`
- Sends FCM push via `services/fcm.go`
- Updates row to `status='sent'` or `status='failed'`

Morning nudge creation: triggered at end of successful entry processing
- `scheduled_at` = next occurrence of `users.fcm_nudge_hour` in `users.timezone`
- One nudge per entry — no duplicates

---

## Therapy Mode Architecture

Therapy Mode is a real-time voice conversation feature. Unlike journal entries (async, worker-processed), therapy sessions are synchronous request-response cycles driven directly by the API server.

### Session Flow

```
Mobile
  │
  ├─ POST /therapy/sessions
  │     backend: INSERT therapy_sessions (status=active), load user journal context
  │     ← { session_id, context_loaded: true, expires_at }
  │
  │   [For each conversation turn:]
  │
  ├─ POST /therapy/sessions/:id/presign   (if voice input)
  │     ← { upload_url, audio_key }       # same presign pattern as journal entries
  │
  ├─ PUT audio → MinIO/R2 (direct)
  │
  ├─ POST /therapy/sessions/:id/messages
  │     { audio_key? OR content, input_mode: "voice"|"text" }
  │     backend:
  │       → if audio_key: Whisper transcribe (synchronous, not queued)
  │       → run crisis detection (Stage 1 + Stage 2, same fail-safe as entries)
  │       → if crisis: mark session crisis_detected, return crisis resources, close session
  │       → check session not expired (started_at + 1hr < NOW())
  │       → fetch session message history
  │       → Claude therapy prompt (with journal context injected at session start)
  │       → if TTS enabled: generate audio via OpenAI TTS
  │       → INSERT therapy_session_messages (user + assistant)
  │       → UPDATE therapy_sessions (turn_count, last_active_at)
  │       → DELETE user audio from storage immediately after transcription
  │     ← { user_message, assistant_message, tts_url?, session_state }
  │
  ├─ POST /therapy/sessions/:id/end       (user ends early)
  │     backend: generate post-session summary via Claude
  │     ← { summary, duration_sec, turn_count }
  │
  └─ GET /therapy/sessions/:id            (poll session state)
       ← { status, turn_count, expires_at, messages[] }
```

### Therapy Session Database Schema

```sql
therapy_sessions
  id UUID PK
  user_id UUID FK → users
  status therapy_session_status    -- ENUM: active | completed | expired | crisis_detected
  started_at TIMESTAMPTZ
  expires_at TIMESTAMPTZ           -- started_at + 1 hour, enforced server-side
  ended_at TIMESTAMPTZ NULL
  duration_sec INT                 -- actual elapsed seconds at end
  turn_count INT DEFAULT 0
  context_snapshot JSONB           -- snapshot of journal context loaded at session start
                                   -- { mood_avg_30d, top_emotions, top_topics, recent_summaries[] }
  post_session_summary TEXT NULL   -- Claude-generated summary after session ends
  billing_amount_paise INT         -- amount charged (49900 = ₹499)
  created_at TIMESTAMPTZ

therapy_session_messages
  id UUID PK
  session_id UUID FK → therapy_sessions
  role TEXT CHECK (role IN ('user','assistant'))
  content TEXT                     -- transcribed text (user) or AI response (assistant)
  input_mode TEXT CHECK (input_mode IN ('voice','text'))  -- how user submitted
  created_at TIMESTAMPTZ
```

### Therapy Prompt Architecture

Therapy Mode adds two new prompt builders in `internal/services/prompts.go`:

- **`buildTherapyModeSystemPrompt(ctx TherapyContext)`** — injects the user's journal context (30-day avg mood, top emotions, top topics, last 5 entry summaries) at session start. Sets conversational therapeutic style: active listening, Socratic reflection, no diagnosis, no prescriptions. Includes mandatory disclaimer in every session start.
- **`buildTherapyPostSessionPrompt(messages []Message)`** — generates 3-sentence session summary after session ends.

Context is injected **once at session start** (stored in `context_snapshot`) and prepended to every subsequent Claude call as a cached system prompt prefix. This keeps per-turn cost predictable and enables Claude's prompt caching to work across all turns.

### Cost Architecture

| Component | Cost/session (1 hr, 30 turns) |
|---|---|
| Whisper (30 min user speech) | ~$0.18 |
| Claude Sonnet (with prompt caching) | ~$0.70 |
| OpenAI TTS (optional, AI voice out) | ~$0.17 |
| **Total** | **~$1.05** |

Priced at ₹499/session or included in Pro plan (2 sessions/month).

### Key Invariants

1. **Crisis detection runs on every user message** — same two-stage fail-safe as journal entries (ADR-002). If crisis detected, session immediately ends and crisis resources are returned. No exceptions.
2. **Session hard-cap is server-side** — `expires_at = started_at + 1hr` checked before every Claude call. Mobile timer is display-only.
3. **Audio deleted immediately after transcription** — same rule as journal entries (ADR-005). No audio retained in storage past the transcription step.
4. **Context injected once, not re-fetched** — journal context is snapshotted at session start into `context_snapshot`. Live journal changes during the session do not affect the active session.

---

## Dev vs. Prod Differences

| Concern | Dev | Prod |
|---|---|---|
| Storage | MinIO (Docker) | Cloudflare R2 |
| Transcription | faster-whisper-server (local CPU) | OpenAI Whisper API |
| AI analysis | Stubbed (`STUB_AI_ANALYSIS=true`) | Anthropic Claude API |
| TTS (Therapy Mode) | Skipped / stubbed | OpenAI TTS API |
| Auth | Manual JWT from jwt.io | Supabase Auth |
| Push notifications | Skipped (no FCM credentials) | Firebase Cloud Messaging |
| Migrations | Auto-run on API startup | Auto-run on API startup |
