# Backend - Claude Guidance

Read this before touching any Go code in this directory.

Also read:
- `../docs/ARCHITECTURE.md` - system design, data flow, layer map
- `../docs/API_CONTRACT.md` - request/response shapes (do not break these)
- `../docs/DECISIONS.md` - why things are built the way they are
- `../docs/TESTING.md` - what must be tested before any change ships

---

## Go Patterns Used Here

### Dependency Injection
ALL wiring happens in `internal/handlers/router.go`. This is the only place where repositories and services are instantiated and passed into handlers.

- Handlers receive pre-wired service structs - not config, not `os.Getenv`
- Services receive repository structs - not DB connections directly
- Add a new endpoint: create handler → register in router.go → done

### Repository Pattern
- `internal/repositories/` contains ALL pgx queries - no SQL in services or handlers
- Return domain types from `internal/models/`, not raw pgx rows
- No business logic in repositories - pure data access

### Error Handling
- Use `pkg/apierr` typed errors for API-facing errors
- `middleware/errors.go` converts these to JSON responses
- Don't call `c.JSON` with raw error strings in handlers - always go through apierr
- Log errors at the service layer, not the handler layer

### Context
- Always thread `context.Context` as the first argument through service and repository calls
- Use `ctx` from the Gin request: `c.Request.Context()`

---

## Invariants - Do Not Break These

1. **Crisis detection fails safe** - if Claude is unreachable during Stage 2, the entry MUST be treated as a crisis. See `services/crisis.go`. Do not add any code path that bypasses this.

2. **3-turn conversation cap** - enforced in `services/conversation.go`. The cap is a product decision. Do not remove or make it configurable without an explicit instruction.

3. **Audio deleted after transcription** - `workers/transcription.go` deletes the audio from storage after Whisper completes. Do not skip this. Do not move it after other steps.

4. **All prompts in prompts.go** - never put Claude prompt strings in handlers, workers, or other service files. `internal/services/prompts.go` is the single source of truth.

5. **Workers are stateless** - no in-memory caching in worker processes. All state goes through PostgreSQL or Redis.

---

## Adding a New API Endpoint

1. Add handler function in the appropriate `internal/handlers/*.go` file (or create a new one)
2. Register the route in `internal/handlers/router.go`
3. If it needs DB access: add query to `internal/repositories/*.go`
4. If it needs business logic: add to `internal/services/*.go`
5. Update `../docs/API_CONTRACT.md` with the new endpoint's request/response shape
6. Write integration test following patterns in `../docs/TESTING.md`

---

## Adding a Database Migration

1. Create `migrations/000004_description.up.sql` and `migrations/000004_description.down.sql`
2. Number must be sequential - check existing files first
3. The down migration must fully reverse the up migration
4. Migrations run automatically on API startup - test locally with `make db-migrate`
5. NEVER modify an existing migration file - always add a new one

---

## Adding a New Claude Prompt

1. Add a new build function in `internal/services/prompts.go` (e.g. `buildWeeklyReviewPrompt()`)
2. Add a new method to the Claude service in `internal/services/claude.go`
3. Define the expected JSON output struct in `internal/models/`
4. Write a unit test in `internal/services/claude_test.go` with a mock response

---

## Key Files Quick Reference

| File | Purpose |
|---|---|
| `cmd/api/main.go` | API entry point - config load, DB connect, router setup |
| `cmd/worker/main.go` | Worker entry point - starts nudge scheduler + BRPOP loop |
| `internal/handlers/router.go` | All DI wiring + route registration |
| `internal/services/claude.go` | Anthropic API calls (analysis + conversation) |
| `internal/services/prompts.go` | ALL Claude prompt strings |
| `internal/services/crisis.go` | Safety-critical crisis detection - read DECISIONS.md ADR-002 before touching |
| `internal/services/context_builder.go` | Builds prompt context from last 5 entries |
| `internal/workers/transcription.go` | Full pipeline: transcribe → crisis → context → claude → store |
| `internal/workers/nudge_scheduler.go` | Polls nudges table every 60s, sends FCM |
| `migrations/` | SQL schema - add new files, never modify existing |
| `pkg/apierr/errors.go` | Typed API errors |

---

## Running Locally (Without Docker)

```bash
# Requires Postgres + Redis running separately
go run ./cmd/api
go run ./cmd/worker

# Tests
go test ./...
go test -race ./...

# Tidy
go mod tidy
```

## Environment Variables

Do not hardcode any config. All config comes from environment variables via `internal/config/config.go`. Key vars:

```
DATABASE_URL        PostgreSQL connection string
REDIS_ADDR          Redis host:port
REDIS_PASSWORD      Redis auth
JWT_SECRET          Supabase JWT secret (shared with mobile)
ANTHROPIC_API_KEY   Leave blank in dev; set for real Claude calls
STUB_AI_ANALYSIS    true in dev (returns canned response), false in prod
WHISPER_API_URL     Local whisper server or OpenAI endpoint
STORAGE_ENDPOINT    MinIO (dev) or R2 endpoint
FCM_CREDENTIALS_JSON  Firebase service-account JSON content (blank in dev = push skipped silently)
FCM_PROJECT_ID        Firebase project ID (needed on API and worker - nudges send from the worker)
```
