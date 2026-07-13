# DreamLog API Contract

**Base URL (dev):** `http://localhost:8080`
**Auth:** All endpoints except `/health`, `/version`, `/auth/register`, `/auth/login`, `GET /share/:token`, and `GET /journeys` require `Authorization: Bearer <jwt>`

Claude: Read this file before touching any handler in `backend/internal/handlers/`. The request/response shapes here are the contract the mobile app is built against - do not change field names or types without updating both sides.

---

## Health

### GET /health
No auth required.

Response `200`:
```json
{ "status": "ok" }
```

### GET /version
No auth required. Force-update gate: the mobile app calls this on every cold start and
compares its installed version (from `app.json`) against `minimum_version`. If the
installed version is lower, the app shows a blocking screen with an "Update Now" button
that opens the store listing. The check is fail-open on the client - if this endpoint is
unreachable, the app launches normally.

Response `200`:
```json
{
  "minimum_version": "1.0.0",      // env MINIMUM_APP_VERSION; semver-style "x.y.z"
  "android_store_url": "string",   // env ANDROID_STORE_URL; Play Store listing
  "ios_store_url": "string"        // env IOS_STORE_URL; "" until the app is on the App Store
}
```

---

## Auth (Local Path)

> **Rate limiting:** `POST /auth/login`, `POST /auth/register`, and the public
> `GET /share/:token` are rate-limited per client IP (HTTP `429` with a
> `Retry-After` header when exceeded). Additionally, a share link locks itself
> for 15 minutes after 5 consecutive incorrect passcodes (`429`), independent of
> source IP.

### POST /auth/register
```json
// Request
{ "email": "string", "password": "string", "name": "string" }

// Response 201
{ "token": "string", "user": { "id": "uuid", "email": "string", "name": "string" } }

// Errors
400 - missing fields
409 - email already registered
```

### POST /auth/login
```json
// Request
{ "email": "string", "password": "string" }

// Response 200
{ "token": "string", "user": { "id": "uuid", "email": "string", "name": "string" } }

// Errors
400 - missing fields
401 - invalid credentials
```

---

## Billing / Subscription

### GET /billing/plan
Returns the authenticated user's current plan and all plan limits.

Response `200`:
```json
{
  "plan": "free | plus | pro | b2b",
  "plan_expires_at": "RFC3339 | null",
  "limits": {
    "plan": "free",
    "monthly_entries": 10,        // -1 = unlimited
    "monthly_shares": 0,          // -1 = unlimited; 0 = not allowed
    "has_pdf_export": false,
    "has_weekly_review": false,
    "has_mood_history": false,
    "has_hindi": false,
    "has_all_modes": false,
    "has_streak_freeze": false,
    "has_therapist_share": false,
    "display_name": "Free",
    "price": "₹0 · $0"
  },
  "all_plans": { /* same structure for free, plus, pro, b2b */ }
}
```

### POST /billing/upgrade
Called after an In-App Purchase completes in the store SDK (`expo-iap`). The
backend **verifies the purchase with the store server-side** (Apple
`verifyReceipt` / Google Play Developer API) before granting anything - it
never trusts the client's claim that payment happened. Plan expiry is set
server-side to now + 30 days (monthly) or now + 365 days (annual) — purchases
are one-time consumable passes, not auto-renewing subscriptions. Pricing is
store-managed per product, so there is no amount check server-side.

```json
// Request
{
  "plan":           "free | plus | pro",
  "period":         "monthly | annual",      // optional, defaults to "monthly"
  "platform":       "ios | android",
  "product_id":     "com.dreamlog.app.plus.monthly",
  "purchase_token": "..."   // iOS: base64 app receipt · Android: Play purchase token
}

// Response 200
{ "plan": "plus", "plan_expires_at": "RFC3339 | null", "limits": { /* PlanLimits */ } }

// Errors
400 - invalid or missing plan; invalid period; missing/invalid platform;
      missing product_id or purchase_token; unknown product_id; product does
      not match the requested plan or billing period; b2b requested (b2b is
      provisioned by sales, not self-serve)
402 - the store rejected the purchase (not purchased, pending, refunded, or forged)
409 - store transaction already used to grant a plan (replay protection)
500 - store verification unavailable (retry later; the purchase is safe with the store)
```

**Product catalogue** (must match App Store Connect / Play Console SKUs and
`backend/internal/services/iap.go`; display prices in `docs/PRICING.md`):
| Product ID | Grants |
|---|---|
| `com.dreamlog.app.plus.monthly` | Plus, 30-day pass |
| `com.dreamlog.app.plus.annual`  | Plus, 365-day pass |
| `com.dreamlog.app.pro.monthly`  | Pro, 30-day pass |
| `com.dreamlog.app.pro.annual`   | Pro, 365-day pass |

Rules:
- `plan: "free"` (self-downgrade) needs no purchase and clears the expiry.
- For `plus`/`pro` the purchase is verified with the store for the exact
  `product_id`, and the product must map to the requested plan + period.
- Each store transaction grants a plan exactly once (`payments` table, unique
  on `transaction_id` - Apple transaction_id / Play orderId).
- The mobile app finishes (consumes) the store transaction only **after** this
  endpoint succeeds, so an interrupted flow is re-delivered by the store.
- Dev mode (no `APPLE_SHARED_SECRET` / `GOOGLE_PLAY_CREDENTIALS_JSON`):
  verification is skipped; paid plans are granted with a server-set 30- or
  365-day expiry so the local stack needs no external APIs.

**Plan gating:**

All gating uses the **effective plan**: a paid plan whose `plan_expires_at`
is in the past is treated as `free` everywhere (including `GET /billing/plan`).

- `GET /mood/history` - requires `plus` or higher; returns `403` otherwise
- `GET /reviews/weekly` and `GET /reviews/weekly/latest` - requires `plus` or higher
- `POST /share` - requires `plus` (5/month) or `pro`/`b2b` (unlimited); `free` gets `403`
- `GET /export/pdf` - requires `plus` or higher (Plus is the complete journal product)
- `POST /entries` - `free` plan capped at 10 entries/month (returns `409` at limit)

---

## User

### GET /me
Auto-provisions user if first request with this JWT.

Response `200`:
```json
{
  "id": "uuid",
  "email": "string",
  "name": "string",
  "preferred_name": "string | null",  // shown in AI reflections instead of name if set
  "timezone": "string",               // IANA timezone e.g. "Asia/Kolkata"
  "fcm_nudge_hour": 8,                // 0-23, local hour for morning nudge
  "nudge_auto_time": true,            // true = nudges sent at the user's learned typical recording hour
                                      // (falls back to fcm_nudge_hour until enough entries exist)
  "goal": "stress | anxiety | grief | depression | trauma | relationships | career | curious | null",
  "age_range": "under_18 | 18_24 | 25_34 | 35_44 | 45_plus | null",
  "voice_language": "auto | <language>",  // therapy TTS voice; auto = follow detected speech language.
                                          // <language> is any key in models.SupportedVoiceLanguages (30 languages:
                                          // english, hindi, spanish, french, german, ..., each mapped to an Azure voice)
  "created_at": "RFC3339"
}
```

### PUT /me
```json
// Request (all fields optional; at least one required)
{
  "name": "string",
  "preferred_name": "string",
  "timezone": "string",
  "fcm_nudge_hour": 8,
  "nudge_auto_time": true,
  "goal": "stress | anxiety | grief | depression | trauma | relationships | career | curious",
  "age_range": "under_18 | 18_24 | 25_34 | 35_44 | 45_plus",
  "voice_language": "auto | <language>"   // any key in models.SupportedVoiceLanguages; 400 on any other value
}

// Response 200 - same shape as GET /me
```

Setting `fcm_nudge_hour` without also sending `nudge_auto_time` flips
`nudge_auto_time` to `false` server-side - an explicit hour choice wins over
learned timing until the user re-enables auto.

---

## Entries

### POST /entries/presign
```json
// Request
{ "filename": "string", "content_type": "audio/aac" }

// Response 200
{
  "upload_url": "string",    // pre-signed PUT URL, expires in 15 min
  "audio_key": "string"      // object key to pass to POST /entries
}
```

### POST /entries
```json
// Request
{
  "audio_key": "string",
  "duration_sec": 120.5,
  "audio_size_bytes": 204800,
  "mode": "processing"    // optional: processing (default) | rant | gratitude | decision
}

// Response 201
{
  "id": "uuid",
  "status": "pending",
  "mode": "processing",
  "created_at": "RFC3339"
}
```

### GET /entries
Paginated list, newest first.

Query params: `?page=1&limit=20`

Response `200`:
```json
{
  "entries": [
    {
      "id": "uuid",
      "status": "completed",         // pending | processing | completed | failed
      "duration_sec": 120.5,
      "transcript": "string | null",
      "language": "string | null",
      "created_at": "RFC3339"
    }
  ],
  "total": 42,
  "page": 1,
  "limit": 20
}
```

### GET /entries/:id
Response `200` - same shape as single item in list above, plus:
```json
{
  "error_msg": "string | null",
  "retry_count": 0
}
```

Errors: `404` if not found or belongs to different user.

### GET /entries/search?q=
Full-text search via PostgreSQL tsvector.

Query params: `?q=anxiety+work&page=1&limit=20`

Response `200` - same shape as GET /entries list.

---

## Analysis

### GET /entries/:id/analysis
Only returns data when entry `status=completed`.

Response `200`:
```json
{
  "id": "uuid",
  "entry_id": "uuid",
  "mood_score": 65,                          // 1-100
  "emotional_tone": [
    { "emotion": "cautious hope", "intensity": 0.7 },
    { "emotion": "warmth", "intensity": 0.5 }
  ],
  "topics": ["family connection", "physical movement"],
  "key_quotes": ["I forgot how much I like it", "genuinely okay"],
  "summary": "string",                       // 2-3 factual sentences
  "reflection": "string",                    // 3-5 sentences + open question
  "morning_nudge": "string",                 // 1 sentence
  "is_crisis": false,
  "connection_insight": "string",            // optional; cross-entry pattern ("this is the 3rd time X
                                             // has come up in the last month"); omitted when no
                                             // recurring topic crossed the threshold
  "dream_symbols": ["snake", "river"],       // only present when entry.mode = "dream"; 3-6 symbols
  "dream_type": "vivid",                     // only present when mode = "dream"; nightmare|lucid|recurring|vivid|surreal|mundane
  "psychological_lens": "string",            // only present when mode = "dream"; Jungian / depth-psychology reading
  "vedic_lens": "string",                    // only present when mode = "dream"; Vedic Svapna Shastra reading
  "created_at": "RFC3339"
}
```

Errors: `404` if entry not found; `409` if entry not yet completed.

### GET /entries/flashback
Time capsule: resurfaces a past entry from roughly one year ago (±7 days), or
failing that roughly one month ago (±3 days). Crisis entries are never
resurfaced.

Response `200`:
```json
{
  "entry_id": "uuid",
  "label": "one_year_ago | one_month_ago",
  "date": "RFC3339",
  "summary": "string",          // the AI summary of that entry
  "mood_score": 58,
  "topics": ["relocation"]
}
```

Errors: `404` when neither window has a completed, non-crisis entry.

### POST /entries/:id/checkin
Self-set trigger: schedules a "check in on this tomorrow" push nudge for the
entry, delivered tomorrow at the user's nudge hour. The message reuses the
entry's `morning_nudge` (falls back to a topic-based line). One pending
check-in per entry.

Response `201`:
```json
{ "entry_id": "uuid", "scheduled_at": "RFC3339" }
```

Errors: `400` invalid UUID · `404` entry not found or belongs to different
user · `409` entry not completed, check-in already scheduled, or nudges
disabled.

---

## Timeline

### GET /timeline
Entries combined with their analyses. Newest first.

Query params: `?page=1&limit=20`

Response `200`:
```json
{
  "items": [
    {
      "entry": { /* same as GET /entries/:id */ },
      "analysis": { /* same as GET /entries/:id/analysis, or null if not complete */ }
    }
  ],
  "total": 42,
  "page": 1,
  "limit": 20
}
```

---

## Conversations

### POST /entries/:id/conversation
Idempotent - returns existing conversation if already created for this entry.

Response `200` or `201`:
```json
{
  "id": "uuid",
  "entry_id": "uuid",
  "turn_count": 0,
  "is_closed": false,
  "messages": [],             // array of existing messages if conversation existed
  "created_at": "RFC3339"
}
```

Errors: `404` if entry not found or not completed.

### POST /conversations/:id/messages
```json
// Request
{ "content": "string" }

// Response 201
{
  "user_message": {
    "id": "uuid",
    "role": "user",
    "content": "string",
    "created_at": "RFC3339"
  },
  "assistant_message": {
    "id": "uuid",
    "role": "assistant",
    "content": "string",
    "created_at": "RFC3339"
  },
  "turn_count": 1,
  "is_closed": false          // true when turn_count reaches 3 after this message
}
```

Errors:
- `404` - conversation not found
- `409` - conversation is already closed (turn_count = 3)
- `400` - empty content

---

## Mood

### GET /mood/weekly
Average mood score per day for the last 7 days. Excludes crisis entries (`is_crisis=true`).

Response `200`:
```json
{
  "days": [
    { "date": "2026-05-21", "avg_mood": 65, "entry_count": 2 },
    { "date": "2026-05-22", "avg_mood": null, "entry_count": 0 },
    ...
  ]
}
```

### GET /mood/streak
Response `200`:
```json
{
  "current_streak": 5,        // consecutive days with at least one completed entry
  "longest_streak": 21,
  "total_days": 34,           // total distinct days with entries ever
  "next_milestone": 7,        // next milestone (7/21/50/100); 0 if all reached
  "freeze_count": 1           // available streak freezes (0-3)
}
```

### GET /mood/history
30 / 90 / 365-day mood trendline with period-over-period comparison.

Query params: `?range=30d` (default) | `90d` | `365d`

Response `200`:
```json
{
  "days": [
    { "day": "2026-04-28", "avg_mood": 68, "entry_count": 2 }
  ],
  "range": "30d",
  "avg_mood": 70,             // weighted average over the period; null if no data
  "prev_avg_mood": 62,        // average for prior equal period; null if no data
  "mood_delta": 8,            // avg_mood - prev_avg_mood; null if insufficient data
  "top_emotions": ["hopeful", "anxious", "calm"],
  "entry_count": 14           // total entries in the period
}
```

Errors: `400` if range is not one of the allowed values.

### GET /mood/patterns
Top-8 emotions with frequency and intensity data for the Pattern Radar. Available to all plans.

Query params: `?range=30d` (default) | `90d` | `365d`

Response `200`:
```json
{
  "range": "30d",
  "emotions": [
    {
      "emotion": "hopeful",
      "frequency": 8,           // entries where this emotion appeared
      "avg_intensity": 0.72,    // average intensity 0.0–1.0
      "score": 1.0              // normalized combined score (0.0–1.0) for radar axis
    }
  ],
  "total_entries": 12,
  "mood_distribution": {
    "high": 6,                  // mood_score >= 70
    "neutral": 5,               // 40–69
    "low": 1                    // < 40
  }
}
```

Errors: `400` if range is not one of the allowed values.

### POST /streak/freeze
Uses one streak freeze to protect a missed day (the frozen date is treated as an active day in streak calculation).

```json
// Request
{ "freeze_date": "2026-05-27" }   // YYYY-MM-DD, the day to protect

// Response 200
{ "freeze_count": 0, "freeze_date": "2026-05-27" }

// Errors
400 - missing or invalid freeze_date
409 - no streak freezes remaining
```

---

## Shareable Insight Cards

Available to all plans (no gate) - maximises viral sharing.

### GET /insights/card
Returns everything needed to render the week's shareable insight card.

Response `200`:
```json
{
  "week_label": "May 26 – Jun 1",
  "week_start": "2026-05-26",
  "mood_arc": [
    { "date": "2026-05-26", "avg_mood": 72 }
  ],
  "top_emotions": ["hopeful", "calm", "anxious"],
  "streak": 5,
  "entry_count": 3,
  "share_count": 1
}
```

### POST /insights/share
Records that the user shared their insight card. Returns updated all-time share count.

```json
// Request (body optional)
{ "week_start": "2026-05-26" }   // YYYY-MM-DD; defaults to current week if omitted

// Response 201
{
  "total_shares": 4,
  "week_start": "2026-05-26"
}
```

---

## Weekly Reviews

### GET /reviews/weekly/latest
Returns the most recent completed weekly review for the authenticated user.

Response `200`:
```json
{
  "id": "uuid",
  "user_id": "uuid",
  "week_start": "2026-05-25",         // date string (Sunday)
  "narrative": "string",              // Claude-generated paragraph
  "top_emotions": ["hopeful", "anxious", "calm"],
  "mood_arc": [
    { "date": "2026-05-19", "avg_mood": 65 },
    { "date": "2026-05-20", "avg_mood": 72 }
  ],
  "entry_count": 5,
  "status": "completed",
  "scheduled_at": "RFC3339",
  "generated_at": "RFC3339",
  "created_at": "RFC3339"
}
```

Errors: `404` if no completed weekly review exists yet.

### GET /reviews/weekly
Returns the 10 most recent completed weekly reviews for the authenticated user.

Response `200`:
```json
{
  "reviews": [ /* same shape as GET /reviews/weekly/latest */ ]
}
```

---

## Annual Reviews

Requires `plus` or higher plan. Generated automatically once per year (Jan 1). Claude produces a full-year narrative + monthly mood arc.

### GET /reviews/annual/latest
Returns the most recent completed annual review for the authenticated user.

Response `200`:
```json
{
  "id": "uuid",
  "user_id": "uuid",
  "year": 2025,
  "narrative": "string",              // Claude-generated paragraph
  "top_emotions": ["hopeful", "anxious", "calm"],
  "top_topics": ["work", "family", "health"],
  "mood_arc": [
    { "month": "2025-01", "avg_mood": 65, "entry_count": 8 },
    { "month": "2025-02", "avg_mood": 72, "entry_count": 11 }
  ],
  "entry_count": 87,
  "avg_mood": 68,
  "status": "completed",
  "scheduled_at": "RFC3339",
  "generated_at": "RFC3339",
  "created_at": "RFC3339"
}
```

Errors: `403` if plan below Plus · `404` if no completed annual review exists.

### GET /reviews/annual
Returns all completed annual reviews for the authenticated user.

Response `200`:
```json
{ "reviews": [ /* same shape as GET /reviews/annual/latest */ ] }
```

Errors: `403` if plan below Plus.

---

## Life Chapters

User-defined named time periods (e.g. "My Bangalore Chapter", "Grad School"). Claude can generate a narrative summary of any chapter on demand.

### GET /chapters
Response `200`:
```json
{
  "chapters": [
    {
      "id": "uuid",
      "user_id": "uuid",
      "title": "string",
      "description": "string",
      "start_date": "2024-01-01",     // YYYY-MM-DD
      "end_date": "2024-12-31",       // YYYY-MM-DD or null (ongoing)
      "emoji": "🌱",
      "color": "#7C3AED",
      "summary": "string",            // empty until POST /chapters/:id/summarize is called
      "created_at": "RFC3339",
      "updated_at": "RFC3339"
    }
  ]
}
```

### POST /chapters
```json
// Request
{
  "title": "string",                  // required
  "description": "string",
  "start_date": "2024-01-01",        // required, YYYY-MM-DD
  "end_date": "2024-12-31",          // optional; omit for ongoing
  "emoji": "🌱",
  "color": "#7C3AED"
}

// Response 201 - LifeChapter (same shape as list item)
```

Errors: `400` missing title or start_date.

### GET /chapters/:id
Response `200` - LifeChapter (same shape as list item).

Errors: `404` not found or belongs to different user.

### PUT /chapters/:id
All fields optional; only provided fields are updated.

```json
// Request
{
  "title": "string",
  "description": "string",
  "end_date": "2025-06-30",   // pass "" to clear end_date (make ongoing again)
  "emoji": "🌲",
  "color": "#059669"
}

// Response 200 - updated LifeChapter
```

Errors: `404` not found.

### DELETE /chapters/:id
Response `204`.

### GET /chapters/:id/detail
Returns the chapter enriched with aggregated entry data for the date range.

Response `200`:
```json
{
  "id": "uuid",
  "title": "string",
  "description": "string",
  "start_date": "2024-01-01",
  "end_date": "2024-12-31",
  "emoji": "🌱",
  "color": "#7C3AED",
  "summary": "string",
  "entry_count": 42,
  "avg_mood": 68,                     // null if no entries
  "top_emotions": ["hopeful", "calm"],
  "mood_arc": [
    { "date": "2024-01-15", "avg_mood": 65 }
  ],
  "created_at": "RFC3339",
  "updated_at": "RFC3339"
}
```

### POST /chapters/:id/summarize
Generates a Claude narrative for the chapter using all entries within its date range. Stores result in `summary`. Idempotent - calling again regenerates.

Response `200`:
```json
{ "summary": "string" }
```

Errors: `404` chapter not found · `500` if Claude is unavailable.

---

## Relationship Map

Automatically populated by Claude during the entry analysis pipeline. No user action required - people mentioned in journal entries are extracted and tracked. The top extracted people (with sentiment lean) are also injected into Therapy Mode's context snapshot at session start.

### GET /relationships
Returns all **non-hidden** people extracted from the user's entries.

Response `200`:
```json
{
  "people": [
    {
      "id": "uuid",
      "user_id": "uuid",
      "name": "string",
      "role": "family | friend | colleague | romantic | other",
      "mention_count": 12,
      "positive_count": 8,
      "negative_count": 2,
      "last_mentioned_at": "RFC3339",
      "hidden": false,
      "created_at": "RFC3339",
      "updated_at": "RFC3339"
    }
  ]
}
```

### GET /relationships/:id
Returns a single person with their recent mention history.

Response `200`:
```json
{
  "person": { /* same shape as list item */ },
  "mentions": [
    {
      "id": "uuid",
      "person_id": "uuid",
      "entry_id": "uuid",
      "user_id": "uuid",
      "sentiment": "positive | neutral | negative",
      "context": "string",            // excerpt from the entry mentioning this person
      "created_at": "RFC3339"
    }
  ]
}
```

Errors: `400` invalid UUID · `404` person not found or belongs to different user.

### PATCH /relationships/:id
Rename, re-categorize, or hide/unhide a person. All fields optional; at least one required. Hidden people are excluded from `GET /relationships` (and from Therapy context) but their journal entries are untouched.

```json
// Request (any subset)
{
  "name": "string",
  "role": "family | friend | colleague | romantic | other",
  "hidden": true
}

// Response 200 - Person (same shape as list item)
```

Errors: `400` invalid UUID / empty name / invalid role / no fields · `404` not found · `409` a person with that name already exists (use merge instead).

### POST /relationships/:id/merge
Folds another person (`source_id`) into this one (`:id`): the source's mentions are reassigned to the target, the target's counts are recomputed, and the source is deleted. Used to clean up duplicates (e.g. "Mom" and "mother").

```json
// Request
{ "source_id": "uuid" }

// Response 200 - the updated target Person
```

Errors: `400` invalid UUID / `source_id` equals `:id` · `404` either person not found or belongs to different user.

---

## Export

### GET /export/pdf?period=monthly|yearly
Generates and streams a PDF export of the user's emotional journal.

Query params: `?period=monthly` (default, last 30 days) | `yearly` (last 365 days)

Response `200`:
- Content-Type: `application/pdf`
- Content-Disposition: `attachment; filename="dreamlog-monthly-2026-05.pdf"`
- Body: binary PDF

PDF includes:
- Cover page (user name, period, avg mood, entry count, top emotions)
- Mood overview (avg mood card, entry count, trend vs prior period, daily bar chart, emotion bars)
- Entry summaries (date, AI summary, mood score badge, key quote, topics)

Errors: `400` if period is not `monthly` or `yearly`.

---

## B2B Corporate Wellness

### POST /b2b/companies/:slug/join
Adds the authenticated user to a company by its slug. Idempotent.

Response `200`:
```json
{ "company_id": "uuid", "company_name": "string", "role": "member" }
```

Errors: `404` company not found · `409` seat limit reached.

### GET /b2b/companies/:slug/mood?range=30d|90d
Anonymised team mood summary. Requires the requesting user to be a company **admin**.

Response `200`:
```json
{
  "company_id": "uuid",
  "company_name": "string",
  "total_members": 47,
  "days": [
    { "day": "2026-05-01", "avg_mood": 64, "active_members": 12, "entry_count": 18 }
  ],
  "avg_mood": 64,
  "prev_avg_mood": 58,
  "mood_delta": 6,
  "alert_threshold": 40,
  "is_alerted": false
}
```

Errors: `403` not an admin · `404` company not found.

---

## Guided Journeys

### GET /journeys
Returns all available journey templates. **Public** (no auth) - the catalogue is
static seeded content, so guests can browse journeys before signing in. Starting a
journey (`POST /journeys/:journeyID/start`) still requires auth.

Response `200`:
```json
{
  "journeys": [
    {
      "id": "stress_relief",
      "title": "Stress Relief",
      "description": "string",
      "step_count": 3,
      "estimated_minutes": 15,
      "tags": ["stress", "calm"],
      "prompts": ["string", "string", "string"]
    }
  ]
}
```

### POST /journeys/:journeyID/start
Starts a new session for the given journey template.

Response `201` - JourneySession:
```json
{
  "id": "uuid",
  "user_id": "uuid",
  "journey_id": "stress_relief",
  "journey_title": "Stress Relief",
  "current_step": 0,
  "total_steps": 3,
  "status": "in_progress",
  "steps": [
    { "step_index": 0, "prompt": "string", "entry_id": null, "completed": false }
  ],
  "created_at": "RFC3339",
  "updated_at": "RFC3339"
}
```

Errors: `404` journey ID not found.

### GET /journeys/sessions
Lists the user's journey sessions, newest first (max 20).

Response `200`:
```json
{ "sessions": [ /* JourneySession */ ] }
```

### GET /journeys/sessions/:sessionID
Returns a single session with all step states.

Response `200` - JourneySession (same shape as above).

Errors: `400` invalid UUID · `404` session not found or belongs to another user.

### POST /journeys/sessions/:sessionID/advance
Records an entry for the current step and advances to the next. Returns the updated session.

```json
// Request
{ "entry_id": "uuid" }

// Response 200 - updated JourneySession
```

Errors: `400` missing entry_id or invalid session UUID · `409` session already completed.

---

## Therapist Dashboard

### POST /therapists/register
```json
// Request
{ "name": "string", "email": "string", "credentials": "PhD, Clinical Psychology" }

// Response 201 - Therapist object
{ "id": "uuid", "user_id": "uuid", "name": "string", "email": "string",
  "credentials": "string", "plan": "trial", "created_at": "RFC3339" }
```

### POST /therapists/clients/link
Requests a link to a client by their DreamLog user ID (shared out-of-band). The
link is created in `pending` state and **grants no access to the client's data**
until the client approves it (see the client-facing consent endpoints below).
This prevents a self-registered "therapist" from reading a client's journal
summaries merely by knowing their UUID.

```json
// Request
{ "client_id": "uuid" }

// Response 200
{ "therapist_id": "uuid", "client_id": "uuid", "status": "pending" }
```

Errors: `403` caller is not a registered therapist · `400` invalid UUID.

### DELETE /therapists/clients/:clientID
Soft-revokes the link. Response `204`.

---

## Therapist Link Consent (client-facing)

These endpoints are called by the **client** (any authenticated user) to control
which therapists may access their journal data.

### GET /therapists/requests
Lists pending therapist link requests awaiting the client's approval.

Response `200`:
```json
{
  "requests": [
    {
      "therapist_id": "uuid",
      "therapist_name": "string",
      "credentials": "string",
      "requested_at": "RFC3339"
    }
  ]
}
```

### POST /therapists/requests/:therapistID/approve
Client consents to a pending link, activating the therapist's access.

Response `200`:
```json
{ "therapist_id": "uuid", "status": "active" }
```

Errors: `400` invalid therapist UUID · `404` no pending request from this therapist.

### POST /therapists/requests/:therapistID/decline
Client rejects a pending link (or revokes an active one). Access is removed.

Response `200`:
```json
{ "therapist_id": "uuid", "status": "revoked" }
```

Errors: `400` invalid therapist UUID · `404` no request or link from this therapist.

### GET /therapists/clients
```json
// Response 200
{
  "clients": [
    {
      "client_id": "uuid",
      "name": "string",
      "linked_at": "RFC3339",
      "last_entry_at": "RFC3339 | null",
      "avg_mood_30d": 65,
      "entry_count": 14
    }
  ]
}
```

### GET /therapists/clients/:clientID/brief
Generates a real-time Claude pre-session brief for the specified client.

Response `200`:
```json
{
  "client_id": "uuid",
  "client_name": "string",
  "generated_at": "RFC3339",
  "brief": "3-sentence pre-session brief generated by Claude",
  "top_emotions": ["hopeful", "anxious", "calm"],
  "mood_trend": "improving | declining | stable | insufficient_data",
  "avg_mood_7d": 68,
  "entry_count": 42,
  "recent_entries": [
    {
      "date": "RFC3339",
      "summary": "string",
      "mood_score": 72,
      "topics": ["work", "family"],
      "key_quote": "string"
    }
  ]
}
```

Errors: `403` not a registered therapist · `404` client not linked.

---

## Therapist Workspace (external clients + session notes)

In-app + portal workspace for therapists to manage their own (non-app) clients and
digitize handwritten session notes. Sensitive fields (names, notes, summaries) are
encrypted at rest per-therapist (see docs/THERAPIST_PORTAL.md). All routes require a
therapist profile; creating clients/notes additionally requires the client-data consent.

### GET /therapists/me
Returns the caller's therapist profile; the mobile login pill uses this to route.

```json
// Response 200
{ "therapist": { /* Therapist */ }, "client_consent_accepted": true }

// Errors
404 - the authenticated user has no therapist profile
```

### POST /therapists/consent
Records acceptance of the client-data responsibility terms. Required before creating
external clients or uploading notes.

```json
// Request (optional)
{ "version": "1.0" }

// Response 200
{ "client_consent_accepted": true, "version": "1.0" }
```

### GET /therapists/overview
```json
// Response 200
{
  "external_clients": 12,
  "linked_clients": 3,
  "sessions_this_week": 9,
  "sessions_this_month": 31,
  "total_sessions": 214,
  "last_session_at": "RFC3339 | null"
}
```

### POST /therapists/external-clients
```json
// Request
{ "name": "Asha K", "role": "client" }   // role optional, defaults to "client"

// Response 201 - ExternalClient
{
  "id": "uuid", "therapist_id": "uuid", "name": "Asha K", "role": "client",
  "archived": false, "session_count": 0, "last_session_at": null,
  "created_at": "RFC3339", "updated_at": "RFC3339"
}

// Errors
400 - missing/empty name · 403 - consent not accepted
```

### GET /therapists/external-clients?include_archived=true
Response `200`: `{ "clients": [ /* ExternalClient */ ] }` — newest activity first.

### GET /therapists/external-clients/:id
Response `200` - ExternalClient. Errors: `400` invalid UUID · `404` not found or another therapist's.

### PATCH /therapists/external-clients/:id
```json
// Request (any subset, at least one)
{ "name": "string", "role": "string", "archived": true }

// Response 200 - updated ExternalClient
```

### DELETE /therapists/external-clients/:id
Deletes the client **and all their sessions** (cascade). Response `204`.

### POST /therapists/sessions/presign
```json
// Request
{ "filename": "notes.jpg", "content_type": "image/jpeg" }  // jpeg | png | webp only

// Response 200
{ "upload_url": "string", "image_key": "notes/{therapistID}/{uuid}.jpg" }

// Errors
400 - unsupported content_type · 403 - consent not accepted
```

### POST /therapists/sessions
Create a session for exactly one client (external or linked) with exactly one notes
source: `image_key` (photo → async OCR) or `bullets` (typed → stored immediately).

```json
// Request - photo path
{ "external_client_id": "uuid", "image_key": "notes/…", "session_date": "2026-07-07" }

// Request - typed path
{ "linked_client_id": "uuid", "bullets": ["discussed sleep", "set boundary"] }

// Response 201 - ClientSession
{
  "id": "uuid", "therapist_id": "uuid",
  "external_client_id": "uuid | absent", "linked_user_id": "uuid | absent",
  "session_date": "2026-07-07",
  "status": "pending | processing | completed | failed",
  "raw_text": "string (OCR reference, once completed)",
  "bullets": ["string"],
  "summary": "string (only after summarize)",
  "error_msg": "string (only when failed)",
  "created_at": "RFC3339", "updated_at": "RFC3339"
}

// Errors
400 - zero/both client refs; zero/both note sources; bad date; image_key not owned by caller
403 - consent not accepted
404 - external client not found, or linked client has no active (approved) link
```

Photo sessions start `pending`; the client polls `GET /therapists/sessions/:id` until
`completed` (bullets populated, photo deleted) or `failed` (`error_msg` set).

### GET /therapists/sessions?external_client_id=&linked_client_id=
Response `200`: `{ "sessions": [ /* ClientSession */ ] }` — newest first, max 50.
With neither param: the therapist's most recent sessions across all clients.

### GET /therapists/sessions/:id
Response `200` - ClientSession. Errors: `400` · `404`.

### PATCH /therapists/sessions/:id
Replaces the editable bullet list.

```json
// Request
{ "bullets": ["edited point", "new point"] }

// Response 200 - updated ClientSession
// Errors: 400 empty bullets · 404
```

### POST /therapists/sessions/:id/summarize
Generates (and stores, encrypted) a 3-5 sentence AI summary of the session bullets.
The client's identity is never sent to the AI. Idempotent - regenerates on repeat call.

Response `200` - ClientSession with `summary` set.
Errors: `404` · `409` session not completed / has no notes.

### DELETE /therapists/sessions/:id
Deletes the session; any not-yet-processed note photo is also removed from storage.
Response `204`.

---

## Terms of Service Acceptance

### POST /me/accept-terms
Records the authenticated user's ToS/privacy acceptance (any account type).

```json
// Request (optional)
{ "version": "1.0" }

// Response 200
{ "tos_accepted": true, "version": "1.0" }
```

### GET /me/terms
```json
// Response 200
{
  "tos_accepted_at": "RFC3339 | null",
  "tos_version": "string | null",
  "current_version": "1.0"    // re-prompt when != tos_version
}
```

---

## Devices (Push Notifications)

### POST /devices
Register or update FCM token. Upserts on `fcm_token`.

```json
// Request
{ "fcm_token": "string", "platform": "ios" }   // platform: "ios" | "android"

// Response 201
{ "id": "uuid", "created_at": "RFC3339" }
```

---

## Therapy Mode

Real-time AI-assisted voice/text conversation sessions. Sessions are standalone - not tied to a journal entry. Crisis detection runs on every user message.

**Billing:** first session ever is free. Pro plan includes 1 session/month; beyond that Pro members pay the member price (₹299/session) and everyone else pays the standalone price (₹499/session), charged at session start. `402` returned if no sessions remaining and no session credits.

### POST /therapy/sessions
Start a new therapy session. Loads the user's journal context (last 30d mood avg, top emotions, top topics, last 5 entry summaries, last 3 past session summaries) and snapshots it for the session.

```json
// Request (all fields optional)
{
  "persona": "comforting | rational | cbt | mindful"  // default: "comforting"
}

// Response 201
{
  "id": "uuid",
  "status": "active",
  "persona": "comforting",
  "started_at": "RFC3339",
  "expires_at": "RFC3339",        // started_at + 1 hour, enforced server-side
  "context_loaded": true,         // false if user has no prior entries
  "has_session_history": false,   // true if past session summaries were injected
  "billing_amount_paise": 49900   // 0 if covered by Pro plan
}

// Errors
402 - no session credits and not on Pro plan
400 - invalid persona value
```

**Persona options:**

| Persona | Style |
|---|---|
| `comforting` | Warm, validating, feelings-first |
| `rational` | Structured, Socratic, logic-grounded |
| `cbt` | CBT-informed, identifies thought patterns and distortions |
| `mindful` | Grounding, present-moment, breath-aware |

### POST /therapy/sessions/:id/presign
Get a pre-signed PUT URL for uploading a voice turn. Same pattern as journal entry presign.

```json
// Request
{ "filename": "string", "content_type": "audio/aac" }

// Response 200
{ "upload_url": "string", "audio_key": "string" }

// Errors
404 - session not found or belongs to different user
409 - session is not active (expired, completed, or crisis_detected)
```

### POST /therapy/sessions/:id/messages
Send a user message (voice or text) and receive an AI response.

```json
// Request - voice input
{ "audio_key": "string", "input_mode": "voice" }

// Request - text input
{ "content": "string", "input_mode": "text" }

// Response 201
{
  "user_message": {
    "id": "uuid",
    "role": "user",
    "content": "string",          // transcribed text if voice input
    "input_mode": "voice | text",
    "created_at": "RFC3339"
  },
  "assistant_message": {
    "id": "uuid",
    "role": "assistant",
    "content": "string",
    "tts_url": "string | null",   // pre-signed URL to AI voice audio; null if TTS disabled
    "created_at": "RFC3339"
  },
  "session_state": {
    "status": "active | completed | expired | crisis_detected",
    "turn_count": 3,
    "time_remaining_sec": 3200,
    "is_crisis": false,
    "crisis_warnings": 0    // 0 = none, 1 = de-escalating (one more detection → hard stop)
  }
}

// Errors
400 - missing both audio_key and content; or content is empty
404 - session not found or belongs to different user
409 - session is not active (expired, completed, or crisis_detected)
410 - session expired (time limit reached)
```

**Crisis response - two-stage:**
- **First detection (`crisis_warnings = 0`):** `session_state.is_crisis` becomes `true`, `session_state.crisis_warnings` becomes `1`, but `session_state.status` remains `active`. The assistant response attempts de-escalation (grounding, validation, breathing). Session stays open.
- **Second detection (`crisis_warnings >= 1`):** `session_state.status` becomes `crisis_detected`. Response contains crisis hotlines + "I can't help further - please reach out to a professional." Session cannot accept further messages.

### POST /therapy/sessions/:id/end
End a session early. Triggers Claude to generate a 3-sentence post-session summary.

```json
// Response 200
{
  "session_id": "uuid",
  "status": "completed",
  "duration_sec": 1842,
  "turn_count": 14,
  "post_session_summary": "string"   // Claude-generated 3-sentence summary
}

// Errors
404 - session not found or belongs to different user
409 - session already ended
```

### GET /therapy/sessions/:id
Get current session state and full message history.

```json
// Response 200
{
  "id": "uuid",
  "status": "active | completed | expired | crisis_detected",
  "started_at": "RFC3339",
  "expires_at": "RFC3339",
  "ended_at": "RFC3339 | null",
  "duration_sec": 1842,
  "turn_count": 14,
  "time_remaining_sec": 1758,         // 0 if expired or ended
  "post_session_summary": "string | null",
  "messages": [
    {
      "id": "uuid",
      "role": "user | assistant",
      "content": "string",
      "input_mode": "voice | text | null",
      "created_at": "RFC3339"
    }
  ]
}
```

### GET /therapy/sessions
List the authenticated user's therapy sessions, newest first (max 20).

```json
// Response 200
{
  "sessions": [
    {
      "id": "uuid",
      "status": "completed",
      "started_at": "RFC3339",
      "ended_at": "RFC3339 | null",
      "duration_sec": 1842,
      "turn_count": 14,
      "post_session_summary": "string | null"
    }
  ]
}
```

---

## Error Response Format

All errors use this shape:
```json
{
  "error": "human-readable message",
  "code": "MACHINE_READABLE_CODE"    // optional
}
```

Common HTTP status codes:
- `400` Bad Request - validation failure
- `401` Unauthorized - missing or invalid JWT
- `403` Forbidden - valid JWT but wrong user
- `404` Not Found
- `409` Conflict - e.g. conversation closed, email already exists
- `500` Internal Server Error
