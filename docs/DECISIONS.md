# Architecture Decision Records

These explain WHY things are built the way they are. Read before proposing changes to core patterns.

---

## ADR-001: Two Auth Paths (Supabase + Local bcrypt)

**Decision:** Keep both auth paths active simultaneously.

**Why:** The Supabase JWT path is for production (mobile app using Supabase Auth). The local bcrypt path exists so developers can register/login without setting up Supabase — critical for making the dev environment fully self-contained. Both paths produce identical JWT tokens (same secret, same claims), so the middleware doesn't need to know which path minted the token.

**Do not:** Remove either path. Do not try to consolidate them into one — they serve different runtime environments.

---

## ADR-002: Crisis Detection Must Fail Safe

**Decision:** If Claude is unreachable during Stage 2 crisis confirmation, default to treating the entry as a crisis.

**Why:** The cost of a false negative (missing a genuine crisis) is catastrophically higher than the cost of a false positive (showing hotline resources to a non-crisis entry). Network timeouts, API rate limits, and model outages must never result in missed crisis detection.

**Do not:** Add any code path that could cause a crisis entry to be treated as normal when Claude is unavailable. The fail-safe is intentional and safety-critical.

---

## ADR-003: All Claude Prompts in prompts.go

**Decision:** Every string passed to the Anthropic API lives in `internal/services/prompts.go`. No prompt strings anywhere else.

**Why:** Prompts are the product — they determine reflection quality, tone, and safety. Scattering them across files makes them impossible to audit, test, or iterate. Having one file means one place to review for regressions when the model changes.

**Do not:** Inline prompt strings in handlers, workers, or other service files.

---

## ADR-004: Workers Are Stateless

**Decision:** Worker processes share no in-memory state. All coordination goes through Redis (job queue) and PostgreSQL (job state).

**Why:** This makes horizontal scaling trivial (`make scale-worker N=3`). It also means a crashed worker loses nothing — the job is still in Redis for another worker to pick up (BRPOP atomicity guarantees this).

**Do not:** Add any in-memory caching or state to worker processes. If you need caching, use Redis.

---

## ADR-005: Audio Deleted After Transcription

**Decision:** Audio files are deleted from MinIO/R2 immediately after successful Whisper transcription.

**Why:** Audio is the most privacy-sensitive data in the system. Retaining it creates liability, storage cost, and deletion-on-request complexity. The transcript is sufficient for all downstream operations.

**Do not:** Add any feature that requires re-accessing audio after the worker completes. If users need playback, this decision must be revisited with a privacy review.

---

## ADR-006: Follow-up Conversations Capped at 3 User Turns

**Decision:** `conversations.turn_count` max is 3. After the 3rd user message, `is_closed=true`. Enforced in `services/conversation.go`.

**Why:** The follow-up is an intentional, bounded exchange — not an open chatbot. Unlimited conversation would shift the app's identity toward therapy chatbot (liability), increase Claude API costs unboundedly, and dilute the core journaling experience. The cap is a product decision, not a technical constraint.

**Do not:** Remove the cap or make it configurable per-user without a product decision. The system prompt (`buildFollowUpSystemPrompt`) is also written assuming brevity.

---

## ADR-007: No Global State in Mobile

**Decision:** No Redux, no Zustand, no MobX. Component state via React hooks. Persistence via AsyncStorage (non-sensitive) or SecureStore (JWT).

**Why:** The app's data model doesn't require complex cross-component state. Adding a global state manager would add conceptual overhead and boilerplate for no benefit at current scale. If the app grows into a feature set requiring cross-screen state sharing, add Zustand (simplest option) — not Redux.

**Do not:** Install a global state manager without a concrete use case that hooks + prop drilling can't solve cleanly.

---

## ADR-008: Pre-signed URLs for Audio Upload

**Decision:** Mobile uploads audio directly to MinIO/R2 using a pre-signed PUT URL. The API backend never proxies audio bytes.

**Why:** Audio files can be 10-100 MB. Routing them through the API would bottleneck throughput, inflate backend egress costs, and add latency. Pre-signed URLs offload this to the storage layer entirely.

**Do not:** Add any endpoint that accepts audio as a multipart form upload to the API.

---

## ADR-009: Context Builder Uses Last 5 Entries

**Decision:** `services/context_builder.go` fetches the 5 most recent completed entries (excluding crisis entries) to build context for Claude.

**Why:** 5 is enough to detect emotional patterns and recurring topics without exceeding Claude's practical context window constraints or adding latency. Crisis entries are excluded because they represent outlier states that shouldn't bias the emotion trend.

**If changing:** Increasing beyond 10 entries risks making prompts too long for consistent JSON output. Test with the full prompt if you raise this number.

---

## ADR-010: Dependency Injection via router.go

**Decision:** All service and repository instantiation happens in `internal/handlers/router.go`. Handlers receive pre-wired dependencies, not config.

**Why:** Makes testing easy — you can pass mock services to handlers. Makes startup failures explicit and early. Prevents service-layer code from reaching into config.

**Do not:** Use `os.Getenv` or `config.Load()` inside services or repositories. Config belongs in `cmd/*/main.go` → `config.Config` struct → passed into constructors.

---

## ADR-011: Therapy Sessions Are Synchronous, Not Worker-Queued

**Decision:** Therapy session message processing happens synchronously in the API request handler, not via the Redis worker queue.

**Why:** Therapy Mode is a real-time conversation — the user is waiting for a response, staring at the screen. The BRPOP worker pattern adds 100–500ms of queue latency on top of Whisper + Claude latency, which makes the conversation feel unnatural. Journal entries can afford async processing because the user walks away. Therapy turns cannot.

**Consequence:** Whisper transcription and Claude calls happen inline in the HTTP handler. Use generous timeouts (`context.WithTimeout`, 30s for Whisper, 60s for Claude). Handler returns 504 if either times out.

**Do not:** Route therapy session messages through the Redis job queue. Do not reuse `workers/transcription.go` for therapy turns.

---

## ADR-012: Therapy Session Time Limit Is Server-Side, Not Client-Side

**Decision:** The 1-hour session limit is enforced by comparing `therapy_sessions.expires_at` against `NOW()` on every message request. The mobile countdown timer is display-only.

**Why:** Client-side timers are trivially bypassed (background state, clock manipulation, network replay). Any billing or safety invariant enforced only on the client is not enforced. The server is the authority.

**Do not:** Trust any time value sent from the mobile client. Always derive elapsed time from `therapy_sessions.started_at` stored at session creation.

---

## ADR-013: Therapy Mode Crisis Detection Uses the Same Fail-Safe as Journal Entries

**Decision:** Crisis detection (Stage 1 keyword match + Stage 2 Claude confirmation, fail-safe) runs on every user message in a therapy session, identical to the journal entry pipeline.

**Why:** A user in a therapy session is more likely to discuss distressing content than in a brief journal entry. Removing or weakening crisis detection because the context is "therapy-like" would be backwards — the risk is higher, not lower. The cost of a false negative is unchanged regardless of the feature surface.

**Do not:** Skip, short-circuit, or soften crisis detection for therapy sessions. Do not add a "therapy context" branch that treats ambiguous responses as non-crisis. ADR-002 applies here without modification.

---

## ADR-014: Layered Crisis Handling in Therapy Sessions

**Decision:** Crisis detection in therapy sessions follows a two-stage response before hard-stopping: (1) first detection → grounding/de-escalation response, session stays open, `crisis_warnings` incremented; (2) second detection in the same session → hard stop with crisis resources and a clear message that the AI cannot help further.

**Why:** Therapy conversations are more likely to contain distressing content. An immediate hard stop on a first ambiguous signal is disruptive and may feel abandoning to a user who is not in acute crisis. The de-escalation attempt is a proportionate first response. If distress continues or escalates, the second trigger ensures no one in genuine crisis is left with only an AI chatbot.

**Do not:** Allow more than one de-escalation attempt per session (`crisis_warnings >= 1` → always hard-stop on the next detection). Do not use this as a reason to make Stage 1 keyword detection less sensitive — the threshold for detection stays the same; only the *response* is staged.

---

## ADR-015: Therapist Personas Are System Prompt Variants, Not Separate Models

**Decision:** Each persona (comforting, rational, cbt, mindful) is implemented as a distinct `buildTherapyPersonaSystemPrompt_*` function in `prompts.go`. There is no per-persona model selection — all personas use the same Claude model.

**Why:** Personas are tone and style differences, not capability differences. A different system prompt is sufficient and far cheaper than switching models. Using one model also makes crisis detection and output behaviour consistent across all personas.

**Do not:** Route different personas to different Claude models. Do not inline persona prompt strings anywhere outside `prompts.go` (ADR-003 still applies).

---

## ADR-016: Session Continuity via Snapshot Injection, Not Live Queries

**Decision:** Past session summaries (last 3 completed sessions) are fetched once at session start and stored inside `TherapyContextSnapshot`. They are injected into the system prompt from the snapshot, not re-fetched on each turn.

**Why:** Consistency with the existing context snapshot design (ADR-011 rationale: context is snapshotted once, not re-fetched mid-session). This keeps per-turn cost predictable, enables Claude's prompt caching across turns, and prevents a live journal entry added mid-session from changing the session's grounding context.

**Do not:** Query the database for past session summaries on every Claude call. If the user finishes a new session while another is open, the newly-finished summary is not visible until the next session starts — this is intentional.
