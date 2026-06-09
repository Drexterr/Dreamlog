# DreamLog - 5-Phase Development Plan

**From Zero to Working Prototype to Full Product**  
Phase 1: Foundation → Phase 2: Working Prototype → Phase 3: Launchable MVP  
Phase 4: Growth Engine → Phase 5: Scale & Monetize

*Version 2.0 · May 2026 · Solo Developer Build Plan - Enhanced with Strategic Analysis*

---

## Development Philosophy

Build the smallest thing that works, use it yourself, prove it matters, then expand. Every phase delivers a usable product that layers complexity on top of a proven foundation.

---

## Phase Map

| Phase | Name | Duration | What You Have | Status |
|---|---|---|---|---|
| 1 | Foundation | Weeks 1–2 | Backend + recording + transcription | ✅ Complete |
| 2 | Working Prototype | Weeks 3–5 | Record → AI reflection → timeline | ✅ Complete |
| 3 | Launchable MVP | Weeks 6–9 | Onboarding, Mood Map, notifications, paywall | ✅ Complete |
| 4 | Growth Engine | Weeks 10–15 | Pattern Radar, Life Chapters, social sharing | 🔄 In Progress |
| 5 | Scale & Monetize | Weeks 16–24+ | Dream Decoder, Relationship Map, B2B, API | ⏳ Planned |

By the end of Phase 2, you have a working prototype you use every night. By the end of Phase 3, you have a product ready for the App Store with a subscription model. Phases 4 and 5 are about making it sticky, viral, and profitable.

---

## Strategic Context

> Stop thinking of DreamLog as a "voice journaling app." That category is crowded and underfunded. Reposition as: **"Longitudinal Emotional Intelligence - the first app that understands how you feel across months and years, not just today."**

Every other journaling app shows you today's reflection. DreamLog shows you March 2025 you vs. May 2026 you. The data across time is the product. The context builder is the seed of this - water it aggressively.

**Core Competitive Edges (Do Not Dilute):**
1. Context builder (last 5 entries) - embryonic longitudinal intelligence
2. Two-stage crisis detection that fails safe - better than anything Rosebud or Rocket Journal has
3. 3-turn follow-up conversations grounded in the original entry
4. Clean, scalable architecture with worker scaling, dead letter queue, full-text search

---

## PHASE 1: FOUNDATION

**Timeline: Weeks 1–2 (14 days)**  
**Goal:** Set up the entire backend, get voice recording and transcription working end-to-end. No UI polish - just the engine.

### 1.1 Backend Infrastructure

Everything starts here. Build the server, database, and storage layer that will power the entire app.

1. Initialize Go project with Gin framework. Set up project structure: `cmd/api/`, `internal/handlers/`, `internal/services/`, `internal/models/`, `internal/middleware/`, `pkg/`.
2. Design and implement the PostgreSQL database schema. Core tables: `users` (id, email, name, timezone, preferences, created_at), `entries` (id, user_id, audio_url, transcript, duration_secs, recorded_at), `entry_analysis` (entry_id, emotional_tone, topics, mood_score, reflection_text, summary), `emotional_metrics` (user_id, date, mood_score, dominant_emotions).
3. Set up Cloudflare R2 (or AWS S3) bucket for audio file storage. Configure: private bucket, server-side encryption enabled, pre-signed upload URLs (15 min expiry), pre-signed download URLs for playback.
4. Set up Redis for session caching and background job queue. Async AI processing - user records audio, uploads it, and the transcription + analysis happens in a background worker, not blocking the request.
5. Create Docker Compose file for local development: Go API server, PostgreSQL 16, Redis 7, MinIO (local S3-compatible storage for testing without cloud costs).
6. Build core API endpoints (all behind auth middleware): `POST /entries`, `GET /entries`, `GET /entries/:id`, `GET /me`, `PUT /me`.
7. Write database migration scripts. Use golang-migrate for version-controlled schema changes.

### 1.2 Authentication

1. Set up Supabase project (free tier sufficient for MVP). Configure Google OAuth and Apple Sign-In providers.
2. Implement auth middleware in Go: validate Supabase JWT tokens on every API request, extract user_id, reject expired/invalid tokens.
3. Build sign-up/sign-in flow in React Native using `@supabase/supabase-js`. Support Google sign-in (both platforms), Apple sign-in (iOS), and email + magic link (fallback).
4. Implement auto-creation of user record in PostgreSQL on first sign-in (triggered on first API call).

**Two auth paths coexist:** Supabase JWT (production mobile app) + Local bcrypt/JWT (dev environment, no Supabase required). Both paths produce identical JWT tokens - middleware doesn't know which path minted the token.

### 1.3 React Native Project Setup

1. Initialize Expo project (managed workflow). Install core dependencies: `expo-av`, `expo-router`, `@supabase/supabase-js`, `react-native-reanimated`.
2. Set up app navigation: Auth screens, Main tab navigator (Home/Record, Timeline, Insights, Settings), Entry detail screen, Recording screen (full-screen modal).
3. Configure dark theme as default. Define color palette, typography scale, and spacing constants in `src/theme.ts`.
4. Set up EAS Build for iOS and Android development builds. Configure `app.json` with bundle identifiers, microphone permissions, and splash screen.

### 1.4 Voice Recording & Transcription Pipeline

This is the core technical pipeline. By the end of this section, a user can record their voice and get a text transcript back.

1. Build the recording screen: full-screen dark UI with one large circular record button. Live audio waveform visualization (react-native-reanimated) and elapsed time counter. Stop button and cancel button.
2. Implement audio recording using expo-av: format AAC, channels mono, sample rate 44100, bit rate 64000. ~500KB per minute of speech.
3. Build the upload flow: pre-signed upload URL from Go API → upload AAC directly to R2/S3 → send audio URL to Go API to trigger processing.
4. Build the transcription worker (Go background job): receive audio URL, download from R2, send to OpenAI Whisper API (model: whisper-1, response_format: verbose_json). Store transcript and language. Delete audio from R2 after successful transcription.
5. Handle edge cases: accept any recording length, show "Take your time" at 15 minutes, hard cap at 30 minutes. AI adapts reflection depth to entry length. Handle upload failures with retry (3 attempts with exponential backoff). Queue recordings offline for upload on reconnect.
6. Test the full pipeline end-to-end: record on phone → upload → transcribe → store.

> **✅ MILESTONE:** You can record your voice on the phone and see the text transcript appear in the database. The engine works.

---

## PHASE 2: WORKING PROTOTYPE

**Timeline: Weeks 3–5 (21 days)**  
**Goal:** Add the AI brain - reflections that reference your history, a browsable timeline of past entries, and basic mood tracking. By the end of this phase, you use the app every night and it feels valuable.

### 2.1 AI Analysis Engine

1. Build the Claude API integration service in Go. Use Claude Sonnet for all nightly analysis. Structured JSON response fields: `emotional_tone` (array with intensity 0–1), `topics` (key subjects), `mood_score` (1–100), `key_quotes` (2–3 notable phrases), `summary` (2–3 sentences), `reflection` (3–5 sentence personalized reflection).
2. Design and iterate on the master analysis prompt - the most important piece of engineering in the entire product. Instruct Claude to: analyze emotional content, identify key themes, reference previous entries, write warmly but not patronizingly, never diagnose or prescribe, ask exactly one thought-provoking question at the end. **Spend real time on this.**
3. Implement entry summarization: 2–3 sentence summary per entry. Stored alongside full transcript. These summaries become the context building blocks.
4. Build the context assembly pipeline: user profile (name, account age), summaries of last 5 entries, full transcript of last entry (continuity), notable patterns. Keep total context under 4,000 tokens at this stage.
5. Implement crisis detection (non-negotiable, must be present from first user version). Before generating reflection: run a screening pass. Check for mentions of self-harm, suicidal ideation, harm to others, severe substance abuse crisis. If detected: skip normal reflection, show compassionate message + localized crisis resources (988 Lifeline in US, iCall/Vandrevala in India).
6. Build the full processing pipeline: audio uploaded → Whisper transcription → crisis screening → context assembly → Claude analysis → store analysis → push notification. Total processing time target: under 15 seconds.

### 2.2 Reflection Display & Goodnight Flow

1. Build the reflection screen. Dark calming background, reflection text in a clean readable font, gentle fade-in animation. Show mood score as a subtle color indicator (not a number - keep it emotional, not clinical).
2. Add two action buttons below reflection: "Goodnight" (closes app with gentle animation + optional sleep sound) and "Tell me more" (opens a brief 2–3 exchange follow-up conversation).
3. Implement the "Tell me more" flow - a short, focused conversation, not an open-ended chatbot. AI continues from the reflection's closing question. User replies via voice (preferred) or text. After 2–3 exchanges, AI gently wraps up: "Something to sit with tonight. Sleep well."
4. Build the "processing" waiting screen shown while AI analyzes. Show a calming animation (breathing circle, gentle pulse, stars) - not a loading spinner. The wait should feel intentional.

### 2.3 Entry Timeline

1. Build the timeline screen: scrollable list of all past entries, newest first. Each entry card shows: date and time, mood indicator (color dot derived from mood_score), 1-line summary preview, topics as small tags.
2. Build the entry detail screen. Tap any entry to see: full transcript (scrollable), AI reflection, mood score, emotional tones, topics. If "Tell me more" conversation happened, show that exchange.
3. Implement basic keyword search across all entry transcripts using PostgreSQL `tsvector/tsquery`.

### 2.4 Basic Mood Tracking

1. Store daily `mood_score` (1–100) from each entry's AI analysis in the `emotional_metrics` table.
2. Build a simple 7-day mood chart on the home screen. Line graph showing last 7 days of mood scores. Tap any day to jump to that entry.
3. Add a daily "streak" counter on the home screen: "5-day streak 🔥". Simple but surprisingly effective for building the daily habit.

### 2.5 Morning Nudge (Basic)

1. After each nightly analysis, generate a morning nudge: a single personalized sentence for push notification the next morning. Store in database with scheduled delivery time (user's timezone + preferred morning time, default 7:30am).
2. Set up Firebase Cloud Messaging (FCM) in the React Native app. Request notification permissions during onboarding.
3. Build simple notification scheduler in Go: cron job that runs every minute, checks for due nudges, sends via FCM, marks as sent.

> **✅ MILESTONE:** You have a working prototype. Every night: open app → talk for 2 minutes → see a personalized reflection that references past entries → wake up to a morning nudge. Share with 10 friends/testers for feedback.

---

## PHASE 3: LAUNCHABLE MVP

**Timeline: Weeks 6–9 (28 days)**  
**Goal:** Polish the UX, add onboarding, build the full Mood Map, implement subscriptions via RevenueCat, and prepare for App Store submission.

### 3.1 Onboarding Flow

First impression determines whether users stay. The onboarding must be fast (under 90 seconds), warm, and lead to an immediate "aha" moment with their first reflection.

1. **Screen 1 - Welcome:** "DreamLog is your nightly companion. Talk, and it listens. Over time, it'll understand you better than you understand yourself." Calming animation.
2. **Screen 2 - Sign In:** Google / Apple / email. One tap, no friction.
3. **Screen 3 - Quick Setup:** Name (pre-filled from auth), timezone (auto-detected), usual bedtime, preferred morning nudge time (default 7:30am).
4. **Screen 4 - Goal Selection + Adaptive Theme** *(see Section 3.1a below)*
5. **Screen 5 - Emotional Baseline:** "In one sentence, how would you describe where you are in life right now?" Voice or text input. Seeds the AI's initial context so even the first reflection feels personal.
6. **Screen 6 - First Recording:** "Take a moment. How was your day? What's on your mind tonight?" Soft prompt to reduce blank-canvas anxiety.
7. **Screen 7 - First Reflection:** Show the AI's reflection for their first entry. **This is the make-or-break moment.** The AI must reference their baseline statement to feel personal even on Day 1.
8. **Screen 8 - Habit Setup:** Confirm notification permissions. Explain 7-day free trial of Pro. Done.

---

### 3.1a Adaptive Color Theming by Emotional Goal

**Feature:** When a user selects their primary reason for using the app, the entire app UI dynamically transitions to a color palette that is psychologically optimized for their specific emotional condition - based on established research in color psychology, chromotherapy, and therapeutic environment design.

**Why this matters:** Color has measurable physiological and psychological effects. Hospital environments, therapy rooms, and crisis centers all use evidence-based color selection. No journaling app has brought this level of intentionality to their UI. This is a visible, felt differentiation from day one - users will notice the app "feels right" for them without knowing why.

**How it works:**
1. User selects their primary focus during onboarding.
2. App sets a `theme_key` on the user's profile (`users.theme`).
3. `src/theme.ts` exports a `getThemeForGoal(goal)` function that returns the appropriate color set.
4. All colors in the app (backgrounds, accents, button fills, chart lines, mood indicators) derive from the active theme.
5. User can change their theme at any time in Settings → Theme.
6. Theme smoothly transitions (300ms ease) when changed.

**Goal Selection Options & Psychological Color Palettes:**

---

#### Grief / Loss
> *Supporting someone navigating loss, bereavement, or the end of something significant.*

**Psychological basis:** Blues and soft lavenders create contemplative, calm environments that support healthy grief processing. These tones are widely used in hospice and bereavement settings because they reduce agitation while allowing emotional openness. Muted rose undertones add warmth without the forced positivity of bright warm colors, which can feel jarring during grief.

| Role | Color | Hex | Notes |
|---|---|---|---|
| Background | Deep Slate Blue | `#1A2535` | Replaces default deep purple - quieter, more still |
| Surface | Muted Blue-Grey | `#253347` | Cards and panels |
| Primary Accent | Soft Periwinkle | `#8FA8C4` | Interactive elements, highlights |
| Secondary Accent | Muted Lavender | `#B5A8C8` | Secondary actions, tags |
| Warm Touch | Dusty Rose | `#C4A5A8` | Occasional warmth, prevents coldness |
| Text Primary | Warm Off-White | `#EDE8F0` | Easier on eyes than pure white |
| Mood Indicator Low | Soft Violet | `#8B7AAB` | Non-alarming crisis indicator |
| Mood Indicator High | Pale Lavender-Blue | `#A8C0D8` | Gentle positive |
| Chart Line | Periwinkle | `#9BB5CC` | Mood chart strokes |

---

#### Anxiety
> *Managing worry, overthinking, nervousness, or general anxious tendencies.*

**Psychological basis:** Green is the most extensively researched color for anxiety reduction. Studies in environmental psychology (Ulrich 1984, subsequent replications) show that green environments measurably reduce cortisol and stimulate the parasympathetic nervous system. Sage green in particular - unlike bright greens - triggers the nervous system's "rest and digest" mode rather than alertness. Soft mint accents add freshness without over-stimulation.

| Role | Color | Hex | Notes |
|---|---|---|---|
| Background | Deep Forest Night | `#1A2B20` | Dark green base - grounding |
| Surface | Deep Sage | `#233428` | Cards and panels |
| Primary Accent | Sage Green | `#7B9E87` | Interactive elements |
| Secondary Accent | Soft Mint | `#A8C5B0` | Secondary actions |
| Warm Touch | Pale Sage | `#C5D9CE` | Highlights |
| Text Primary | Cream White | `#F0EDEA` | Warm, not cold white |
| Mood Indicator Low | Muted Olive | `#7A8F6A` | Non-alarming |
| Mood Indicator High | Clear Mint | `#90C5A0` | Clean positive signal |
| Chart Line | Sage | `#8FB89A` | Soft and natural |

---

#### Depression / Low Mood
> *Navigating persistent low energy, sadness, emotional flatness, or lack of motivation.*

**Psychological basis:** Warm amber and golden tones are used in chromotherapy (colour therapy) and seasonal affective disorder (SAD) treatment as lower-intensity warm-spectrum alternatives to light therapy. Unlike bright yellows (which can feel manic or hollow when depressed), warm ambers and ochres provide gentle energizing warmth. Research on SAD light therapy and interior design for depression recovery consistently favors these tones. Avoid cool blues, which can amplify low mood.

| Role | Color | Hex | Notes |
|---|---|---|---|
| Background | Deep Warm Brown | `#201A12` | Very dark amber - warm cocoon |
| Surface | Dark Ochre | `#2E2318` | Cards and panels |
| Primary Accent | Warm Amber | `#C8965A` | Interactive elements |
| Secondary Accent | Golden Wheat | `#E8C878` | Secondary actions |
| Warm Touch | Pale Gold | `#F0D898` | Highlights |
| Text Primary | Warm Ivory | `#F5EDD8` | Warm not clinical |
| Mood Indicator Low | Muted Bronze | `#A07840` | Gentle, not alarming |
| Mood Indicator High | Bright Amber | `#E8A840` | Warm positive signal |
| Chart Line | Amber | `#C8905A` | Warm tracking line |

---

#### Stress / Burnout
> *Managing work pressure, overwhelm, exhaustion from doing too much.*

**Psychological basis:** Cool blues have the most documented physiological stress-reduction effects of any color - measurably reducing heart rate, blood pressure, and respiratory rate in multiple controlled studies. Blue light therapy is used clinically. For an app used at the end of a stressful day, a blue-dominant palette creates an immediate psychological signal of relief and decompression. Ocean-inspired blues (vs. clinical blues) add depth and avoid the sterility of healthcare environments.

| Role | Color | Hex | Notes |
|---|---|---|---|
| Background | Deep Ocean | `#0F1E2E` | Dark navy - deep water calm |
| Surface | Steel Blue-Grey | `#1A2D40` | Cards and panels |
| Primary Accent | Ocean Blue | `#5B8DB8` | Interactive elements |
| Secondary Accent | Sky Blue | `#7AAECC` | Secondary actions |
| Warm Touch | Pale Blue | `#B0D0E8` | Highlights, breathing room |
| Text Primary | Cool White | `#E8F0F5` | Clean, clear |
| Mood Indicator Low | Muted Blue | `#5A7A98` | Calm indicator |
| Mood Indicator High | Bright Sky | `#70B8D8` | Clear positive signal |
| Chart Line | Ocean | `#5890B8` | Flowing tracking line |

---

#### Relationship Issues / Loneliness
> *Processing relationship dynamics, social pain, disconnection, or loneliness.*

**Psychological basis:** Pink and rose tones are associated with nurturing, empathy, and social connection. Research in behavioral psychology (notably the Baker-Miller Pink / "drunk tank pink" studies) shows that pink environments reduce aggressive arousal and promote calm connectedness. Dusty/muted pinks - unlike saturated hot pinks - provide a sense of warmth and being held, which directly counteracts the emotional experience of loneliness and social pain.

| Role | Color | Hex | Notes |
|---|---|---|---|
| Background | Deep Rose-Brown | `#221518` | Dark warm rose - intimate |
| Surface | Deep Dusty Rose | `#32201E` | Cards and panels |
| Primary Accent | Dusty Rose | `#C87F7F` | Interactive elements |
| Secondary Accent | Peach Coral | `#E0A898` | Secondary actions |
| Warm Touch | Pale Peach | `#EED0C8` | Highlights |
| Text Primary | Warm Pink-White | `#F5EAE8` | Soft and warm |
| Mood Indicator Low | Muted Mauve | `#A87070` | Gentle |
| Mood Indicator High | Warm Coral | `#E89080` | Inviting positive signal |
| Chart Line | Dusty Rose | `#C88080` | Warm tracking line |

---

#### Career / Work Pressure
> *Navigating professional stress, ambition, career transitions, or purpose questions.*

**Psychological basis:** Earth tones and forest greens create a "grounding" effect - reducing the existential weightlessness of career anxiety. Warm browns and muted greens are used in executive coaching and leadership therapy environments because they convey stability, reliability, and organic growth. The contrast with stress blues is intentional: where stress blues create distance from activation, earth tones create a feeling of standing on solid ground.

| Role | Color | Hex | Notes |
|---|---|---|---|
| Background | Dark Earth | `#181D15` | Very dark forest - grounded |
| Surface | Deep Forest | `#222B1C` | Cards and panels |
| Primary Accent | Forest Green | `#5C7A62` | Interactive elements |
| Secondary Accent | Warm Brown | `#8C7055` | Secondary actions |
| Warm Touch | Warm Sand | `#C0A880` | Highlights |
| Text Primary | Warm Off-White | `#F0EDE8` | Comfortable |
| Mood Indicator Low | Muted Olive | `#6A7A58` | Grounded |
| Mood Indicator High | Clear Forest | `#70A878` | Growth signal |
| Chart Line | Earth Green | `#6A8A65` | Natural tracking line |

---

#### Trauma / Difficult Past
> *Processing past experiences, working through difficult memories, healing.*

**Psychological basis:** Soft warm neutrals are recommended in trauma-informed design. Research in trauma therapy environments (SAMHSA guidelines, PTSD treatment centers) emphasizes non-triggering sensory environments - avoiding high-contrast, high-saturation colors which can stimulate hypervigilance. Warm taupes and soft greys with warm undertones create psychological safety: low arousal, non-threatening, and deeply calming without coldness.

| Role | Color | Hex | Notes |
|---|---|---|---|
| Background | Deep Warm Grey | `#1C1A18` | Very dark warm grey |
| Surface | Warm Charcoal | `#282420` | Cards and panels |
| Primary Accent | Warm Taupe | `#A09080` | Soft interactive elements |
| Secondary Accent | Warm Beige | `#C8B898` | Secondary actions |
| Warm Touch | Pale Cream | `#E0D0B8` | Highlights |
| Text Primary | Soft Ivory | `#F0E8D8` | Gentle |
| Mood Indicator Low | Muted Warm Grey | `#907868` | Non-triggering |
| Mood Indicator High | Warm Gold | `#B8A070` | Gentle positive |
| Chart Line | Taupe | `#A89070` | Soft tracking line |

---

#### Just Curious / Self-Discovery
> *No specific issue - using the app for introspection, self-understanding, or growth.*

**Psychological basis:** Purples and indigos are culturally and psychologically associated with introspection, wisdom, and the search for meaning. This is the DreamLog default theme (dark purple palette). Purple is used across meditation apps, spiritual practices, and self-reflection tools because it occupies a middle position between the physical warmth of reds and the cool clarity of blues - the perfect tonal space for turning inward.

| Role | Color | Hex | Notes |
|---|---|---|---|
| Background | Deep Purple | `#1A1025` | Default DreamLog bg |
| Surface | Dark Violet | `#221530` | Default cards |
| Primary Accent | Medium Purple | `#7B6FA0` | Default accent |
| Secondary Accent | Soft Violet | `#9B8EC4` | Default secondary |
| Warm Touch | Soft Lilac | `#C0B0E0` | Default highlights |
| Text Primary | Cool White | `#F0EDF5` | Default text |
| Mood Indicator Low | Muted Purple | `#7060A0` | Default indicator |
| Mood Indicator High | Bright Violet | `#9878D8` | Default positive |
| Chart Line | Purple | `#8878C0` | Default chart |

---

**Implementation Notes:**
- Store user's `goal` and `theme` in the `users` table (new column: `goal TEXT`, `theme TEXT`)
- `src/theme.ts` exports `THEMES: Record<GoalKey, ThemeColors>` and `getTheme(goal: GoalKey): ThemeColors`
- Use React Context (`ThemeContext`) to make active theme available throughout the app
- All hardcoded colors in components must be replaced with `theme.background`, `theme.accent`, etc.
- The transition animation (300ms ease-in-out) should be visible and intentional - let the user feel the app change for them
- In Settings, allow re-selection with a description of what each theme is for (without being clinical)

---

### 3.2 Full Mood Map

1. Extend mood tracking beyond 7 days. Store and display the full emotional timeline from the user's first entry.
2. Build the Mood Map screen: interactive line chart showing `mood_score` over time. Pinch to zoom between day/week/month/all-time views. Tap any data point to see that day's entry summary in a bottom sheet.
3. Add emotion tags below the chart: show dominant emotions for the visible time period as colored tags ("anxious" in theme-appropriate yellow, "hopeful" in theme-appropriate green). Makes mood data feel human, not just a number.
4. Build correlation hints: simple overlays showing day-of-week patterns. "Your mood tends to dip on Mondays and peak on Fridays." Computed server-side after 3+ weeks of data.
5. Add exportable mood summary (plain text or screenshot) that users can share with a therapist. "Share my last 30 days" generates a clean image.

### 3.3 Notification System (Full)

1. Build complete notification system with all types: evening reminder (30 min before bedtime), morning nudge (personalized to last entry), streak-at-risk (if no entry for 2 days - gentle, never guilt-inducing), weekly report ready (Sunday evening), milestone celebrations (7, 30, 90, 365 days).
2. Implement smart evening reminders that vary nightly. Rotate between: a question, a gentle nudge, a pattern tease. Repetitive notifications get ignored; varied ones get opened.
3. Build notification preferences screen: toggle each type, set custom times for evening and morning, choose quiet days.

### 3.4 Context Window Optimization

As users accumulate entries, you need a smarter system for giving the AI relevant context without exceeding token limits or API costs.

1. Implement weekly digest generation: every Sunday, compress that week's entry summaries into a single "weekly digest" (3–5 sentences capturing the week's emotional arc, key themes, notable moments). Store in `weekly_digests` table.
2. Implement monthly profile generation: compress weekly digests into a "monthly emotional profile." Store in `monthly_profiles` table.
3. Update context assembly pipeline: for nightly reflections, send Claude: user profile + current month's weekly digests + last 3 entry summaries + last full entry transcript + active patterns. Target: under 6K tokens total.
4. Set up pgvector extension on PostgreSQL. Generate embeddings for each entry summary. When assembling context, find the 2 most semantically similar past entries to tonight's transcript and include their summaries.

### 3.5 Subscription System

1. Set up RevenueCat. Configure products: `dreamlog_pro_monthly` ($7.99 / ₹199), `dreamlog_pro_annual` ($59.99 / ₹1,990), `dreamlog_premium_monthly` ($14.99 / ₹499), `dreamlog_premium_annual` ($99.99 / ₹3,990).
2. Integrate RevenueCat SDK in React Native. Check subscription status on app launch, cache locally for offline access, listen for changes.
3. Build the paywall screen. Clear comparison of Free vs Pro vs Premium. Include price, what's included, annual savings, "Start 7-day free trial" button. Clean, not aggressive.
4. Implement smart trial trigger: 7-day Pro trial appears after user's **2nd entry** (not at sign-up). User has experienced the core loop before being asked to pay. Much higher conversion.
5. Build feature gating. Free tier: 5 entries/month, basic reflection (no history references), 7-day mood chart only. Pro tier: unlimited entries, deep reflections with full history, full Mood Map, Pattern Radar, weekly reports. Premium tier: everything in Pro plus Dream Decoder, Relationship Map, unlimited deep dives, therapist reports.
6. Implement trial-end conversion screen: personalized summary - "In 7 days, DreamLog tracked your mood across X entries, learned about [topics], and gave you N personalized reflections. Want to keep going?" Include mood chart from trial period.
7. Set up regional pricing: India/SEA/LATAM at ~40% of US price. Student discount at 50% off.
8. Build subscription management screen. Make cancellation easy and non-guilt-inducing.

### 3.6 App Store Preparation

1. Design App Store screenshots (6 screens): recording screen, reflection example, mood map, pattern tease, morning nudge example, "your AI listener" headline.
2. Record App Store preview video (30 seconds): show the core loop.
3. Write App Store listing optimized for: journaling, mental health, mood tracker, AI journal, voice journal.
4. Add required legal: privacy policy (audio deletion policy, encryption), terms of service, mental health disclaimer ("DreamLog is a wellness companion, not a replacement for professional care").
5. Implement "Get Help Now" button accessible from all screens - links to crisis hotlines. Required for App Store approval.
6. Submit to both App Store and Google Play.

> **✅ MILESTONE:** DreamLog is LIVE on the App Store and Google Play. Users can download, sign up, use the free tier, start a Pro trial, and subscribe. You have a real product generating real revenue.

---

## PHASE 4: GROWTH ENGINE

**Timeline: Weeks 10–15 (42 days)**  
**Goal:** Add the features that make DreamLog genuinely irreplaceable - Pattern Radar, Life Chapters, weekly reports, Hindi support, and viral sharing mechanics.

### 4.1 Pattern Radar

The single most differentiating feature in the product. After 2–3 weeks of entries, the AI starts surfacing emotional patterns the user cannot see themselves.

1. Build the pattern detection pipeline. Weekly background job for each user with 10+ entries. Sends full history (monthly profiles + weekly digests + recent entry summaries) to Claude Opus. Identify patterns across 6 categories:
   - **Recurring triggers:** "You feel most anxious on Sunday evenings - 6 of the last 8 Sundays"
   - **People patterns:** "You consistently feel drained after interactions with [person]"
   - **Contradictions:** "You say you're over the breakup but you've mentioned your ex in 9 of 12 entries"
   - **Avoidance:** "You've mentioned wanting to confront your manager 7 times but always change the subject"
   - **Growth moments:** "One month ago you were paralyzed about the move. This week you're actively planning it."
   - **Cyclical:** "Your mood dips in the last week of every month - could be deadline-related or hormonal"
2. Store detected patterns in `patterns` table: type, description, confidence score, evidence (array of entry IDs), detected_at, status (new/seen/dismissed).
3. Build the Pattern Radar screen: card-based feed of detected patterns. Pattern type icon, description, "Evidence: based on X entries over Y weeks", "See entries" link. New patterns have a glowing indicator.
4. Add pattern push notification: "I found a new pattern in your recent entries." Pro/Premium only. Highest open rate of any notification because it triggers curiosity.
5. Implement pattern feedback: user marks each pattern as "Spot on" or "Not quite right." Feed back into future detection prompts to improve accuracy.
6. Build the locked pattern teaser for free users. After enough entries, show: "We've detected 2 patterns in your entries" with a blurred card. This is the highest-converting paywall trigger.

### 4.2 Life Chapters

1. Build the chapter detection algorithm. Use entry embeddings (pgvector) to cluster entries by emotional themes and topics. When a cluster of 5+ entries shares a dominant theme, create a Life Chapter.
2. Auto-generate chapter titles and summaries using Claude. Examples: "The Job Transition" (March–May: 47 entries, dominant emotion: anxiety → excitement), "The Friendship Shift."
3. Build the Life Chapters screen: visual timeline showing chapters as segments with mood arcs. Tap a chapter to see: summary, mood trajectory, key turning points, and entry list.
4. Implement auto-closing of chapters: when AI detects user has moved on from a theme (no related entries for 2+ weeks), chapter is marked as closed with a wrap-up summary.
5. Premium feature: allow users to manually create chapters and tag entries.

### 4.3 Weekly Reports

1. Build the weekly report generator. Runs every Sunday evening. Produces: emotional summary of the week (3–4 sentences), mood trend visualization (mini chart), top 3 themes, notable moments (specific quotes or insights), one pattern observation, and a "focus for next week" suggestion.
2. Design the weekly report screen: beautiful, scrollable, card-based report that feels like a personal newsletter about your inner life. The user's own words quoted back to them.
3. Build a shareable report card: single image (optimized for Instagram Stories format, 1080×1920) showing the week's mood arc, dominant emotion, and one sanitized insight. Users can share without revealing private details. **This is the primary organic growth driver.**
4. Send weekly report notification on Sunday evening. Email delivery option for lapsed user re-engagement.

### 4.4 The Life Graph (30/90/365 Day Emotional Trajectory)

This is your moat. Build a visualization that shows:

1. **Mood score trendline** over 30/90/365 days (`mood_score` per entry, already in DB).
2. **Recurring emotional tones** across the year (aggregate `emotional_tone` JSONB).
3. **Topic clusters that repeat** (aggregate `topics[]`): "For the past 6 weeks, 'work' appears in 80% of your entries. In January, it was 30%."
4. Add **month-over-month narrative**: "Comparing this month to last month: your average mood is up 8 points. Anxiety appeared 40% less often. The topic 'family' appeared for the first time since October."
5. New API endpoint: `GET /mood/history?range=90d`

This is what no one else does. Rosebud shows weekly summaries. Day One shows nothing. You can show someone 2 years of their emotional life. That's not a journaling app - that's a personal longitudinal emotional health record.

### 4.5 Hindi + Regional Language Support

India has 300M+ Hindi speakers on smartphones. Rocket Journal is English-only for now. DreamLog can own this wedge.

1. Whisper already detects language and returns it in `entries.language` - no new infrastructure needed.
2. Build Hindi system prompts in `internal/services/prompts.go`: same emotional intelligence, different language. Add `buildHindiSystemPrompt()` and `buildHinglishSystemPrompt()`.
3. Add Hinglish detection (mixed Hindi-English, extremely common - 200M Indians use it daily). Auto-detect from language field, no user setting required.
4. Language auto-selected for all Claude calls based on detected transcript language.
5. This is a 2–3 week build with outsized strategic impact. Rocket Journal will take 3–6 months to catch up because their therapist network thinks in English.

### 4.6 Streak Mechanics with Forgiveness

Current streaks are punishing. People miss one day and they're done.

1. Add **Streak Freeze**: one automatic per week, two more purchasable. Duolingo's most important retention mechanic.
2. Add **"Comeback" language** when streak breaks - not guilt, encouragement. "You're back. That matters more than the number."
3. Add **milestone celebrations** at 7, 21, 50, 100 days with a shareable card (this is the viral hook).
4. Store `streak_freezes_remaining` and `streak_freeze_used_this_week` in `users` table.

### 4.7 Shareable Insight Cards

NOT sharing the journal entry (that kills privacy trust). Share a beautiful, anonymized visual:
- "21 days of reflecting. My most common emotion: cautious hope." [share card]
- Mood arc graphic for the week, no text content

Build with `react-native-view-shot`, export as image. Zero-cost viral acquisition loop. One share on Instagram Stories = 3–5 new app opens.

### 4.8 Prompt Modes / Templates

Not everyone can free-form journal. Add structured modes alongside free recording:

1. **Rant Mode:** "Just talk. No analysis, just get it out." Shorter reflection, no mood tracking. Different system prompt in `prompts.go`.
2. **Gratitude Mode:** AI asks 3 specific gratitude follow-up questions after listening.
3. **Decision Mode:** "I have a decision to make" → AI helps think it through via Socratic follow-up questions.
4. **Processing Mode:** Current default behavior.

Mode selection on the record screen. Each mode is 2–3 days to build well with its own prompt in `prompts.go`.

### 4.9 Marketing Push

- **TikTok + Instagram Reels:** Content pillars: "My AI noticed a pattern I couldn't see," "What DreamLog told me after 30 days," "The pattern that changed my life." Post 3–5 times/week.
- **Product Hunt launch:** Prepare compelling copy, demo video, maker story. Target a Monday launch.
- **Reddit seeding:** Authentic posts in r/selfimprovement, r/mentalhealth, r/journaling, r/productivity, r/getdisciplined.
- **Influencer outreach:** 20–50 wellness and self-improvement influencers for beta access.
- **SEO content:** Articles on "voice journaling benefits," "how to track your mental health," "why you can't stick to journaling," "AI therapy alternatives."
- **Landing page:** Hero video, feature showcase, beta user testimonials, download links, email capture.

> **✅ MILESTONE:** DreamLog has Pattern Radar, Life Chapters, weekly reports, and viral sharing. Users who stay past 2 weeks rarely churn because the data moat is too valuable. Target: $5K–$10K MRR.

---

## PHASE 5: SCALE & MONETIZE

**Timeline: Weeks 16–24+ (Ongoing)**  
**Goal:** Add premium differentiators (Dream Decoder, Relationship Map, Therapist Prep), build additional revenue streams (Guided Journeys, Year in Review, B2B), and optimize for profitability.

### 5.1 Dream Decoder

1. Build dream detection: when AI analyzes a nightly entry, check if user described a dream.
2. Implement dream analysis prompt using established psychological frameworks (Jungian archetypes, common dream symbolism research). Connect dream themes to user's recent emotional patterns. "You dreamed about being lost in a building. You've been talking about feeling directionless at work for two weeks."
3. Build dream journal section: separate tab filtering only dream entries. Each shows: dream narrative, interpretation, and connection to waking emotional themes.
4. Premium-only feature.

### 5.2 Relationship Map

1. Build person extraction from transcripts. Use Claude to detect names, nicknames ("my boss," "mom"), and emotional context. Store in `person_mentions` table with sentiment score per mention.
2. Build the Relationship Map visualization: interactive graph where each person is a node. Node size = mention frequency. Node color = average sentiment (green = positive, yellow = mixed, red = negative). Tap a person to see their detail view.
3. Build per-person detail view: sentiment history over time, notable quotes about them, AI-generated relationship insight.
4. Implement "missing connection" alerts: if a frequently mentioned person disappears from entries for 3+ weeks, note it.
5. Premium-only feature.

### 5.3 Therapist Prep Report + Therapist Sharing Mode

1. Build the therapist prep report generator: user taps "Prepare for therapy session" and selects date range. AI generates: top 3 themes to discuss, emotional trend summary, specific moments worth exploring, detected patterns, and suggested questions for the therapist.
2. Export as clean PDF with DreamLog branding. User can email it to therapist or read during session.
3. Build read-only shareable link (72-hour expiry, passcode-protected) that shows: last 30 days of mood scores (graph), AI-generated summaries only (not raw transcripts unless user opts in), top recurring emotional tones and topics.
4. Charge ₹99/export or include in premium tier. Therapists who start looking at DreamLog data in sessions become a distribution channel.
5. Build a therapist-facing landing page: explain DreamLog, show a sample prep report, encourage recommending it to clients.

### 5.4 Guided Journeys (Premium Content)

1. Build the Journey framework: multi-week structured experience with specific nightly prompts, tailored analysis, and milestone reflections. Users opt into a journey and receive a specific prompt each night.
2. Launch initial journeys: "Processing Grief" (8 weeks), "Career Clarity" (6 weeks), "Relationship Audit" (4 weeks), "Anxiety Toolkit" (4 weeks), "New Parent Adjustment" (6 weeks). Each designed with input from psychological frameworks.
3. Monetize as one-time purchases ($4.99 each) or included in Premium tier.

**Note:** Guided Journeys tie directly into the Adaptive Color Theming feature from Phase 3 - users in the "Processing Grief" journey should be offered the Grief theme automatically, creating a cohesive therapeutic environment.

### 5.5 Year in Review

1. Build the annual review generator: comprehensive AI-written narrative of the user's year. Life chapters timeline, emotional arc visualization, top patterns discovered, growth milestones, relationship shifts, and a personalized letter from the AI.
2. Offer as beautifully designed digital PDF ($9.99) or printed hardcover book shipped to the user ($29.99 via Lulu or Blurb).
3. This is DreamLog's "Spotify Wrapped" moment - an annual event that drives massive social sharing and re-engagement. Launch in December.

### 5.6 Crisis → Care Bridge

Right now, when you detect a crisis, you show hotline numbers. That's the legally safe minimum. The monetizable version:

1. "It sounds like you're going through something heavy. Would you like to speak with a therapist today?"
2. Integration with Practo, MindPeers, YourDOST, or Rocket Health via affiliate link.
3. Revenue share: ₹200–500 per successful therapy booking.

This converts safety infrastructure into a revenue line. You're not just detecting crisis - you're bridging to care.

### 5.7 Voice Tone Analysis

1. Build a Python microservice using librosa or pyAudioAnalysis that analyzes raw audio (before deletion) for: speaking pace, pitch variability, energy levels, pause frequency, vocal tension.
2. Feed voice metrics into the mood scoring model: someone who speaks quickly with short pauses and high pitch is likely anxious, even if their words sound calm.
3. Display voice insights in entry detail: "Your voice was notably quieter and slower tonight compared to your usual pace."

### 5.8 B2B Corporate Wellness

1. Build anonymized team mood dashboard for HR/people teams. Companies purchase DreamLog as an employee wellness benefit at $5/employee/month. Employees get free Pro access; HR sees only aggregate, anonymized mood trends.
2. Dashboard shows: team-level mood trends over time, stress spikes correlated with company events, engagement scores, anonymous sentiment themes.
3. Target: IT companies in Bangalore/Hyderabad with 200+ employees. These companies already spend ₹500–2000/employee/month on wellness perks.
4. One pilot at 500 employees = ₹1L/month ARR + a logo for your pitch deck + a testimonial.

### 5.9 Clinical Validation Study

1. Contact psychology department at NIMHANS (Bangalore) or AIIMS (Delhi). Offer free premium access for a study: "Does 90 days of voice journaling with AI reflection reduce self-reported anxiety scores?"
2. If the study shows 20%+ improvement (likely, based on Rosebud's self-reported data), you get: a peer-reviewed study to cite in pitch decks, "clinically validated" positioning, and a reason for therapists to recommend you.
3. Cost: zero cash. Time: 3–6 months. Payout: category-defining.

### 5.10 Platform & API

- Build a public REST API for the emotional analysis engine: send text, get back emotional tone, mood score, topics, and patterns. License to other wellness apps, therapy platforms, HR tools.
- Apple Watch companion app: quick 30-second voice check-in from your wrist, haptic morning nudge.
- Integration with Apple Health (mood tracking category) and Google Health Connect.
- Explore Alexa/Google Home integration: "Hey Google, talk to DreamLog."

### 5.11 Export & Data Portability

- **PDF export:** Monthly/yearly "Emotional Journal" - beautiful formatted PDF with mood graphs, top quotes, key moments. Charge ₹49/export or include in premium.
- **Apple Health / Google Fit integration:** Write MindfulSession events after each entry. Gets you into the health ecosystem and Apple's mental wellness narrative.
- **CSV export** of mood scores + dates for power users.

> **✅ MILESTONE:** DreamLog is a full-featured mental wellness platform with multiple revenue streams: consumer subscriptions, premium content (Journeys), annual products (Year in Review), B2B licensing, and API access. Target: $30K–50K+ MRR.

---

## Monetization Architecture

**Free Tier (Forever)**
- 5 entries/month
- Basic reflection (no history references)
- 7-day mood chart
- 3-turn follow-up

**DreamLog Plus - ₹199/month India | $7.99/month Global**
- Unlimited entries
- Deep reflections with full history context
- Hindi + regional language support
- Life Graph (30/90/365 day view)
- Weekly Emotional Review
- All prompt modes (Rant, Gratitude, Decision)
- Streak freeze (2×/week)
- Therapist share link (5/month)
- Adaptive Color Themes

**DreamLog Pro - ₹499/month India | $14.99/month Global**
- Everything in Plus
- Pattern Radar
- Life Chapters
- PDF export (monthly reports)
- Apple Health / Google Fit integration
- Unlimited therapist share links
- Priority processing (faster Claude response)
- Early access to new features

**B2B Wellness - ₹199/employee/month (min 50 employees)**
- All Pro features for employees
- HR dashboard (aggregated only, never individual)
- Monthly wellness report
- Dedicated support

---

## Complete Task Reference

Every task across all 5 phases. Priority: P0 = required for that phase, P1 = high value, P2 = important but can ship without.

| # | Task | Pri | Effort | Gate | Notes |
|---|---|---|---|---|---|
| 1.1 | Go backend + Gin + API scaffolding | P0 | 3 hrs | Proto | Foundation for everything |
| 1.2 | PostgreSQL schema (users, entries, analysis, metrics) | P0 | 5 hrs | Proto | Design for future extensibility |
| 1.3 | R2/S3 encrypted audio storage + signed URLs | P0 | 3 hrs | Proto | Privacy from Day 1 |
| 1.4 | Redis cache + background job queue | P0 | 2 hrs | Proto | Async AI processing |
| 1.5 | Docker Compose local dev environment | P1 | 2 hrs | Proto | Go + PG + Redis + MinIO |
| 1.6 | Core API endpoints (CRUD entries + user) | P0 | 4 hrs | Proto | REST API foundation |
| 1.7 | DB migration scripts (golang-migrate) | P1 | 2 hrs | Proto | Version-controlled schema |
| 1.8 | Supabase Auth setup + JWT middleware | P0 | 4 hrs | Proto | Google + Apple + email sign-in |
| 1.9 | React Native Expo project init + nav | P0 | 3 hrs | Proto | Tabs, screens, dark theme |
| 1.10 | EAS Build config (iOS + Android) | P1 | 2 hrs | Proto | Dev builds for testing |
| 1.11 | Recording screen + waveform visualization | P0 | 6 hrs | Proto | Core UX - make it beautiful |
| 1.12 | Audio recording (expo-av, AAC mono 64kbps) | P0 | 4 hrs | Proto | Test across 5+ devices |
| 1.13 | Background upload with retry + offline queue | P0 | 4 hrs | Proto | Reliability is non-negotiable |
| 1.14 | Whisper API transcription integration | P0 | 4 hrs | Proto | Language auto-detect |
| 1.15 | Full transcription pipeline (upload→transcribe→store→delete) | P0 | 4 hrs | Proto | Privacy-first by default |
| 1.16 | Edge cases (silence, noise, short audio) | P1 | 3 hrs | Proto | Graceful error handling |
| 2.1 | Claude API integration + structured JSON output | P0 | 6 hrs | Proto | Core intelligence layer |
| 2.2 | Master analysis prompt (tone, topics, mood, reflection) | P0 | 10 hrs | Proto | Most important engineering work |
| 2.3 | Entry summarization (2–3 sentences per entry) | P0 | 3 hrs | Proto | Building block for context |
| 2.4 | Context assembly pipeline (history-aware) | P0 | 6 hrs | Proto | What makes reflections personal |
| 2.5 | Crisis detection screening | P0 | 6 hrs | Proto | Non-negotiable safety feature |
| 2.6 | Full processing pipeline (audio→transcript→screen→analyze→notify) | P0 | 4 hrs | Proto | End-to-end pipeline |
| 2.7 | Reflection display screen + Goodnight/Tell me more | P0 | 5 hrs | Proto | The magic moment UI |
| 2.8 | Tell me more flow (2–3 voice/text exchanges) | P1 | 5 hrs | Proto | Optional depth after reflection |
| 2.9 | Processing waiting screen (calming animation) | P1 | 2 hrs | Proto | Make the wait feel intentional |
| 2.10 | Entry timeline (scrollable list + cards) | P0 | 4 hrs | Proto | Browse past entries |
| 2.11 | Entry detail screen (transcript + reflection) | P0 | 3 hrs | Proto | Full entry view |
| 2.12 | Keyword search across entries | P2 | 3 hrs | Proto | PostgreSQL full-text search |
| 2.13 | 7-day mood chart on home screen | P1 | 3 hrs | Proto | Visual emotional snapshot |
| 2.14 | Streak counter on home screen | P1 | 1 hr | Proto | Habit motivation |
| 2.15 | Morning nudge generation + scheduling | P0 | 5 hrs | Proto | Personalized next-morning push |
| 2.16 | FCM push notification integration | P0 | 3 hrs | Proto | Delivery infrastructure |
| 2.17 | Basic notification scheduler (cron) | P0 | 3 hrs | Proto | Morning nudge + evening reminder |
| 3.1 | Onboarding flow (8 screens) | P0 | 8 hrs | MVP | First impression is everything |
| 3.1a | Goal selection + adaptive color theme system | P0 | 8 hrs | MVP | Unique differentiator - felt immediately |
| 3.1b | `users.goal` + `users.theme` DB columns + migration | P0 | 1 hr | MVP | Schema update |
| 3.1c | `src/theme.ts` - 8 complete theme palettes | P0 | 4 hrs | MVP | Psychology-backed color sets |
| 3.1d | ThemeContext + app-wide theme wiring | P0 | 4 hrs | MVP | Replace all hardcoded colors |
| 3.1e | Theme settings screen (allow re-selection) | P1 | 2 hrs | MVP | User control |
| 3.2 | Full Mood Map (interactive, zoomable, all-time) | P0 | 8 hrs | MVP | Major feature for marketing |
| 3.3 | Emotion tags on Mood Map | P1 | 3 hrs | MVP | Human-readable mood data |
| 3.4 | Day-of-week correlation hints | P2 | 3 hrs | MVP | Simple pattern preview |
| 3.5 | Exportable mood summary for therapists | P2 | 2 hrs | MVP | Screenshot + share |
| 3.6 | Full notification system (5 types + variance) | P0 | 6 hrs | MVP | Retention driver |
| 3.7 | Notification preferences screen | P1 | 3 hrs | MVP | User control |
| 3.8 | Weekly digest generation (background job) | P0 | 4 hrs | MVP | Context window management |
| 3.9 | Monthly profile generation | P1 | 3 hrs | MVP | Long-term memory compression |
| 3.10 | pgvector setup + entry embeddings | P1 | 4 hrs | MVP | Semantic retrieval for old entries |
| 3.11 | Updated context assembly with digests + vectors | P0 | 4 hrs | MVP | Smarter, cheaper AI calls |
| 3.12 | RevenueCat integration (iOS + Android) | P0 | 6 hrs | MVP | Payment infrastructure |
| 3.13 | Paywall screen (Free/Pro/Premium comparison) | P0 | 4 hrs | MVP | Conversion UI |
| 3.14 | 7-day trial (triggered after 2nd entry) | P0 | 3 hrs | MVP | Smart trial timing |
| 3.15 | Feature gating system | P0 | 5 hrs | MVP | Lock features by tier |
| 3.16 | Trial-end conversion screen (personalized) | P0 | 4 hrs | MVP | Highest-converting moment |
| 3.17 | Regional pricing + student discount | P1 | 3 hrs | MVP | Expand market |
| 3.18 | Subscription management screen | P0 | 3 hrs | MVP | Upgrade/cancel/restore |
| 3.19 | App Store screenshots + preview video | P0 | 4 hrs | MVP | ASO critical |
| 3.20 | App Store listing + keywords | P0 | 2 hrs | MVP | Discoverability |
| 3.21 | Privacy policy + terms + disclaimers | P0 | 3 hrs | MVP | Legal requirement |
| 3.22 | Get Help Now button + crisis resources | P0 | 2 hrs | MVP | App Store requirement |
| 3.23 | Submit to App Store + Google Play | P0 | 2 hrs | MVP | Go live |
| 4.1 | Pattern detection pipeline (6 types) | P0 | 12 hrs | Growth | Crown jewel feature |
| 4.2 | Pattern storage + evidence linking | P0 | 3 hrs | Growth | Data layer for patterns |
| 4.3 | Pattern Radar UI (cards + evidence) | P0 | 6 hrs | Growth | Compelling display |
| 4.4 | Locked pattern teaser for free users | P0 | 2 hrs | Growth | Highest-converting paywall |
| 4.5 | Pattern notifications + feedback | P1 | 3 hrs | Growth | Engagement + accuracy |
| 4.6 | Chapter detection algorithm (embedding clusters) | P1 | 8 hrs | Growth | AI narrative segmentation |
| 4.7 | Chapter auto-titling + summaries | P1 | 4 hrs | Growth | AI-generated story |
| 4.8 | Life Chapters screen + chapter detail | P1 | 6 hrs | Growth | Visual life story |
| 4.9 | Weekly report generator | P0 | 8 hrs | Growth | Sunday retention event |
| 4.10 | Shareable report card (Stories format) | P0 | 4 hrs | Growth | Organic growth driver |
| 4.11 | Weekly report email delivery | P1 | 3 hrs | Growth | Re-engagement channel |
| 4.12 | Life Graph (30/90/365 day view) | P0 | 8 hrs | Growth | Longitudinal moat |
| 4.13 | Month-over-month narrative API | P1 | 4 hrs | Growth | `GET /mood/history?range=90d` |
| 4.14 | Hindi system prompts in prompts.go | P0 | 6 hrs | Growth | India moat |
| 4.15 | Hinglish detection + auto-language routing | P0 | 3 hrs | Growth | Mixed Hindi-English support |
| 4.16 | Streak freeze mechanic | P0 | 3 hrs | Growth | Retention - Duolingo-proven |
| 4.17 | Comeback language + milestone celebrations | P1 | 4 hrs | Growth | Emotional design |
| 4.18 | Shareable insight cards (non-content) | P0 | 4 hrs | Growth | Viral acquisition loop |
| 4.19 | Rant Mode prompt + UI | P1 | 3 hrs | Growth | Different journaling style |
| 4.20 | Gratitude Mode prompt + UI | P1 | 3 hrs | Growth | Positive reinforcement mode |
| 4.21 | Decision Mode prompt + UI | P2 | 3 hrs | Growth | Socratic follow-up |
| 4.22 | Referral program (2 weeks free each) | P1 | 4 hrs | Growth | Viral acquisition |
| 4.23 | 30-Day Challenge feature | P2 | 4 hrs | Growth | Engagement campaign |
| 4.24 | Landing page / marketing website | P0 | 6 hrs | Growth | SEO + conversion |
| 4.25 | Social media content strategy | P0 | Ongoing | Growth | TikTok, Reels, Reddit |
| 4.26 | Product Hunt launch prep | P1 | 4 hrs | Growth | One-time event |
| 5.1 | Dream detection from transcripts | P1 | 4 hrs | Scale | Premium differentiator |
| 5.2 | Dream analysis prompt + UI | P1 | 6 hrs | Scale | Fascinating and shareable |
| 5.3 | Dream journal section | P2 | 3 hrs | Scale | Filtered dream timeline |
| 5.4 | Person mention extraction (NLP) | P1 | 6 hrs | Scale | Names + sentiment |
| 5.5 | Relationship Map visualization | P1 | 8 hrs | Scale | Interactive graph |
| 5.6 | Per-person detail + insights | P2 | 4 hrs | Scale | Relationship intelligence |
| 5.7 | Missing connection alerts | P2 | 2 hrs | Scale | Proactive relationship care |
| 5.8 | Therapist prep report generator | P1 | 6 hrs | Scale | PDF export |
| 5.9 | Therapist sharing mode (72h expiry link) | P1 | 4 hrs | Scale | Clinical channel |
| 5.10 | Therapist landing page | P2 | 4 hrs | Scale | Professional referral channel |
| 5.11 | Guided Journey framework | P1 | 8 hrs | Scale | Multi-week structured experiences |
| 5.12 | 5 initial Journeys (content + prompts) | P1 | 10 hrs | Scale | Premium content revenue |
| 5.13 | Year in Review generator | P2 | 10 hrs | Scale | Annual Spotify Wrapped moment |
| 5.14 | Crisis → Care Bridge (Practo/YourDOST affiliate) | P1 | 4 hrs | Scale | Convert safety into revenue |
| 5.15 | Voice tone analysis microservice | P2 | 8 hrs | Scale | Audio-level emotion signals |
| 5.16 | B2B team mood dashboard | P2 | 16 hrs | Scale | Enterprise revenue stream |
| 5.17 | Clinical validation study (NIMHANS/AIIMS) | P1 | - | Scale | Zero cost, category-defining |
| 5.18 | Outcome measurement (every 10th entry survey) | P1 | 3 hrs | Scale | VC data requirement |
| 5.19 | PDF export (monthly/yearly) | P1 | 6 hrs | Scale | Premium content |
| 5.20 | Apple Health / Google Fit integration | P2 | 6 hrs | Scale | Health ecosystem |
| 5.21 | Public API for emotional analysis | P3 | 12 hrs | Scale | Platform licensing |
| 5.22 | Apple Watch companion | P3 | 8 hrs | Scale | Wrist-level check-ins |

---

## Timeline Summary

| Week | Phase | What Gets Done | Users | Revenue |
|---|---|---|---|---|
| 1–2 | Phase 1 | Backend, auth, recording, transcription pipeline | Just you | $0 |
| 3–4 | Phase 2a | AI reflections, reflection screen, timeline | You (daily) | $0 |
| 5 | Phase 2b | Mood chart, morning nudge, search | 10 testers | $0 |
| 6–7 | Phase 3a | Onboarding, goal selection + adaptive themes, full Mood Map, notifications | 50 beta | $0 |
| 8–9 | Phase 3b | Subscriptions, App Store prep, launch | 500+ | $500 MRR |
| 10–12 | Phase 4a | Pattern Radar, Life Chapters, weekly reports, Life Graph | 2,000+ | $3K MRR |
| 13–15 | Phase 4b | Hindi, streak mechanics, sharing, referrals, prompt modes, marketing push | 5,000+ | $10K MRR |
| 16–20 | Phase 5a | Dream Decoder, Relationship Map, Therapist Prep, Guided Journeys | 10,000+ | $20K MRR |
| 21–24+ | Phase 5b | Year in Review, B2B, Clinical validation, API, Watch app | 20,000+ | $50K+ MRR |

---

*Phase 1–2: Make it work.*  
*Phase 3: Make it launchable.*  
*Phase 4: Make it grow.*  
*Phase 5: Make it a business.*

Start tonight. Record your first entry. Build what you need.
