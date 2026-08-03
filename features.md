Ode: Full Entrepreneurial & Strategic Analysis

  Wearing the hat of someone who's built and killed startups for 10+ years across consumer, B2B SaaS,
  and health tech.

  ---

  ## Index

  - [Build status update (read this first)](#build-status-update-read-this-first)
  - [1. Honest Market Reality Check](#1-honest-market-reality-check)
  - [2. Competitive Map - Who You're Actually Fighting](#2-competitive-map---who-youre-actually-fighting)
  - [3. What Ode Actually Has That's Valuable](#3-what-ode-actually-has-thats-valuable-be-honest)
  - [4. The Strategic Repositioning You Need](#4-the-strategic-repositioning-you-need)
  - [5. Features That Will Actually Move the Needle](#5-features-that-will-actually-move-the-needle)
  - [6. Monetization Architecture](#6-monetization-architecture)
  - [7. What VCs Actually Want to See](#7-what-vcs-actually-want-to-see-brutally-honest)
  - [8. The Rocket Journal Problem](#8-the-rocket-journal-problem---your-response-strategy)
  - [9. The One Big Bet You Should Consider](#9-the-one-big-bet-you-should-consider)
  - [10. Immediate Action Plan](#10-immediate-action-plan)

  ---

  ## Build status update (read this first)

  This document was written as a pre-launch strategic analysis, when Ode was "a technically impressive
  prototype without a story" (section 7's own words). **That is no longer true of the build.** As of
  2026-08-03, all 9 planned build phases in [docs/ROADMAP.md](docs/ROADMAP.md) are complete, and nearly
  every feature recommended below has since shipped - including several (Therapy Mode, the full Therapist
  Workspace with envelope-encrypted notes, Guided Journeys, Life Chapters, Relationship Map) that go well
  beyond what this document originally asked for.

  What has **not** changed since this was written:
  - **No real usage data yet.** The app has not had a public launch; DAU, D7/D30 retention, and outcome
    metrics referenced in section 7 are still unmeasured. Current work is Store Launch Prep - see
    [docs/LAUNCH_CHECKLIST.md](docs/LAUNCH_CHECKLIST.md).
  - **Clinical validation study (§5, item 12)** - not started; a business-development track, not a code
    deliverable.
  - **Apple Watch / Wear OS quick entry (§5, item 14)** - not built.
  - **Google Fit integration** - stubbed on Android; iOS HealthKit works (Phase 5e).
  - **Multi-regional-language beyond Hindi** (Tamil, Bengali, etc.) - not built; Hindi/Hinglish shipped
    (Phase 4e), and Therapy Mode TTS separately supports 30 voice languages.

  Each feature section below is now tagged with its actual status. The monetization numbers in section 6
  are the original *proposal* - the real, decided pricing model lives in
  [docs/PRICING.md](docs/PRICING.md) and differs from what's drafted here (see the note in that section).

  ---
  1. Honest Market Reality Check

  Global Mental Health App Landscape (2025-2026)

  Mental health tech raised $352M in 2025 - a 150% jump from $138M in 2024. The money is flowing. But
  here's what the data actually tells you that most founders miss: the number of deals dropped from 20
  to 13 even as total capital more than doubled. Translation - VCs are writing bigger checks into fewer,
   more established companies. Seed-stage journaling/wellness apps are fighting for a shrinking slice of
   attention.

  The global mental health apps market hits $22.73B by 2030 (MarketsandMarkets). India specifically:
  $497.9M in 2024 → $1.4B by 2030 at 18.5% CAGR. India is the fastest-growing segment in Asia-Pacific.
  That's your headline number for a pitch deck.

  Critical nuance VCs know: Clinical Mental Health Apps have a cap-to-deal ratio of 0.23x - meaning
  therapy/journaling apps are raising far smaller rounds than clinical/psychiatric platforms. The big
  money is in clinical validation, not consumer apps. You need to understand which game you're actually
  playing.

  ---
  2. Competitive Map - Who You're Actually Fighting

  Rosebud (Your Global Benchmark)

  - Funding: $6M seed (June 2025), led by Bessemer Venture Partners, Tim Ferriss, Initialized Capital,
  776
  - Metrics: 500M words journaled, 7,500+ paying customers, 30M mindful minutes, 75% of users report
  meaningful improvement in 30 days
  - Pricing: $12.99/month or $107.99/year
  - Moat: Bessemer-backed means serious capital, distribution, and credibility. Tim Ferriss means
  influencer access. They have a CBT-template library, adaptive follow-up questions, and voice
  journaling in 20+ languages
  - Weakness: Still US/Western-centric. The 20+ language claim is surface-level - the AI prompt
  engineering is not culturally localized

  Rocket Journal (Your India Direct Competitor - Most Dangerous)

  - Backer: Rocket Health (founded 2021, 200,000+ therapy sessions delivered, 100+ licensed
  psychologists)
  - Launch: September 2025, immediately broke into India Top 10 Health & Fitness within 48 hours
  - Distribution advantage: They're not a journaling startup - they're a therapy company with an
  existing emotionally-committed user base that built a journaling app. That's not marketing, that's a
  structural moat
  - Features: Rant Mode (free-form venting) + Structured check-in prompts
  - Platform: iOS only (Android + Apple Watch 2026)
  - Weakness: They are a therapy company first. Their journaling UX will always be secondary priority.
  They don't have Ode's pipeline sophistication (crisis detection, context builder, follow-up
  conversations). Also - they're owned by a clinical entity, which creates HIPAA-adjacent liability and
  conservative product decisions

  Day One

  - Scale: 10M+ users, profitable, acquired by Automattic (WordPress parent company)
  - Reality: Not your competitor. People using Day One are text journalers who want a beautiful diary.
  Different use case, different psychographic

  Reflect / Reflection.app, Mindsera, Glimmo, Lound

  - All US-based, text-first or lightly voice-enabled, minimal India presence, no clinical backing
  - These are the apps you can beat on execution

  The Real Insight on Rocket Journal

  They beat you to market in India and have a clinical network. You cannot win on "we're also a voice
  journaling app in India." You need a clear wedge.

  > **Status note:** this competitive map has not been re-verified since the doc was written - re-check
  > Rocket Journal's Android launch status and any new entrants before using this in a live pitch.

  ---
  3. What Ode Actually Has That's Valuable (Be Honest)

  Before building anything, recognize what's genuinely differentiated in the current architecture:

  Real competitive edges (at time of writing):
  1. Context builder (last 5 entries) - this is embryonic longitudinal intelligence. No consumer
  journaling app does this well
  2. Two-stage crisis detection that fails safe - this is better than anything Rosebud or Rocket Journal
   has documented publicly. This is an ethical infrastructure advantage
  3. 3-turn follow-up conversations grounded in the original entry - the prompt engineering here (not
  repeating the reflection, not using therapy language, ending gracefully) is thoughtful
  4. Zero-cost dev environment - you can move fast and iterate without burning capital on API calls
  5. Clean, scalable backend architecture - most consumer apps at this stage are held together with duct
   tape. Yours can handle worker scaling, has a dead letter queue, has full-text search. This is a
  hiring/investor signal

  What was actually weak at the time - **and current status**:
  1. No retention mechanics beyond streaks → ✅ shipped: streak freeze/forgiveness, weekly review,
     re-engagement + streak-risk pushes, connection insights, flashback time capsule, self-set check-ins
     (Phase 4)
  2. No outcomes measurement → 🚧 partial: `analytics_events` event stream exists (Phase 7a/PRICING.md
     §6c) tracking product events, but no user-facing self-reported-improvement metric has been added
  3. No onboarding / personalization → ✅ shipped: goal selection, preferred name, age range onboarding
     flow injecting into the AI system prompt (Phase 4a, UX Polish)
  4. Single language (English) → ✅ Hindi/Hinglish shipped (Phase 4e); other regional languages
     (Tamil, Bengali) still not built
  5. No viral/sharing loop → ✅ shipped: shareable insight cards + milestone celebration sharing
     (Phase 4d/4c)
  6. No B2B angle → ✅ shipped: B2B Corporate Wellness with anonymized team dashboard (Phase 5c)
  7. Mobile only → still true; no web app for the end-user product (the therapist portal is a separate
     Next.js web app, Phase 5g/9)

  ---
  4. The Strategic Repositioning You Need

  Stop thinking of Ode as a "voice journaling app." That category is crowded and underfunded.
  Reposition as:

  ▎ "Longitudinal Emotional Intelligence - the first app that understands how you feel across months and
  ▎  years, not just today."

  Fitbit didn't win by saying "it counts your steps." It won by showing you your sleep patterns, your
  resting heart rate over 6 months, your active minutes this year vs. last year. The data across time is
   the product. Every other journaling app shows you today's reflection. None of them do a compelling
  job of showing you March 2025 you vs. May 2026 you.

  Your context builder is the seed of this. Water it aggressively.

  > **Status note:** this repositioning is now backed by real features, not just intent - Life Graph
  > (30/90/365-day trendlines), Weekly Review, Annual Review, Life Chapters, and the Relationship Map
  > (Phases 4g, 7c, 7d, 7e) are exactly this "longitudinal" product surface, built and shipped.

  ---
  5. Features That Will Actually Move the Needle

  Ranked by impact-to-effort ratio as originally assessed. Asterisked ones (*) were flagged VC
  pitch-worthy differentiators. Status reflects the actual codebase as of 2026-08-03.

  Tier 1 - Originally "Build in the Next 60 Days" (Retention & Engagement)

  **1. Structured Onboarding with Goal Selection *** — ✅ **Shipped** (Phase 4a; extended with age range
  and a Journal/Therapy gate in Phase 8f)
  When a user opens the app for the first time, ask them 3 questions:
  - What brought you here? (stress, anxiety, grief, relationships, career pressure, just curious)
  - How much time do you want to spend? (2 min quick vent / 10 min deep reflection)
  - What name should we call you?

  This does two things: it personalizes the AI system prompt immediately, and it gives you segmentation
  data. A user who says "grief" needs different reflection language than one who says "career pressure."

  **2. Weekly "Emotional Review" Push Notification*** — ✅ **Shipped** (Phase 4b)
  Every Sunday at 10 AM local time, a scheduler generates a Claude narrative from the week's entries and
  delivers it via push; `GET /reviews/weekly/latest` serves it in-app. An Annual Review (Phase 7c) was
  later added on top of the same pattern.

  **3. Streak Mechanics with Forgiveness*** — ✅ **Shipped** (Phase 4c)
  - Streak Freeze (1/week auto-granted, max 3, applied via `POST /streak/freeze`)
  - "Comeback" card with non-guilt language when a streak breaks
  - Milestone celebrations at 7/21/50/100 days with native Share API

  **4. Shareable Insight Cards (Non-Content)** — ✅ **Shipped** (Phase 4d)
  Anonymized mood-arc + top-emotion card generated client-side and shared via the native share sheet
  (`GET /insights/card`, `POST /insights/share`). No transcript or reflection content is ever included.

  **5. Hindi + Regional Language Support*** — 🚧 **Partially shipped** (Phase 4e)
  Hindi and Hinglish detection + dedicated system prompts are live. `users.voice_language` additionally
  supports 30 languages for Therapy Mode TTS voice selection. Other regional languages (Tamil, Bengali,
  etc.) for the journaling reflection prompt itself are not built.

  ---
  Tier 2 - Originally "Build in 60-120 Days" (Differentiation & Monetization)

  **6. The Life Graph - 30/90/365 Day Emotional Trajectory*** — ✅ **Shipped** (Phase 4g)
  `GET /mood/history?range=30d|90d|365d` returns daily mood, weighted average, prior-period comparison,
  and top emotions; `GET /mood/patterns` adds an emotion-frequency/intensity radar. Rendered in
  `mobile/app/(tabs)/mood.tsx`.

  **7. Therapist Sharing Mode*** — ✅ **Shipped** (Phase 5a) — and substantially exceeded by Phase 9
  Passcode-protected, 72-hour-TTL read-only share links (`POST /share`, `GET /share/:token`) showing
  30-day mood arc, AI summaries, and top emotions - never raw transcripts. The original ask ("share with
  my therapist") has since grown into a full **Therapist Workspace** (Phase 9): therapist accounts,
  client-consent-gated links to app users, a therapist's own external (non-app) clients, photo-of-notes
  OCR, AI session summaries, and envelope-encrypted storage of everything sensitive - well beyond the
  "read-only export" originally scoped here.

  **8. Crisis → Care Bridge*** — ✅ **Shipped** (Phase 5b)
  Crisis reflection screen shows structured hotline cards (iCall, Vandrevala, 988), tap-to-call, a
  "Find a therapist near you" CTA (Practo affiliate with UTM tracking), and a YourDOST online-therapy
  option. The revenue-share affiliate model proposed here is implemented as an affiliate link; actual
  bounty/referral economics were not separately tracked in this codebase.

  **9. Prompt Modes / Templates** — ✅ **Shipped** (Phase 4f) — plus one mode not originally proposed
  Rant, Gratitude, Decision, and Processing modes all shipped as distinct system prompts in
  `prompts.go`. A fifth mode, **Dream Decoder** (Phase 4f-i), was added beyond the original scope: dual
  psychological (Jungian) and Vedic (Svapna Shastra) interpretive lenses for dream entries.

  **10. Export & Data Portability** — ✅ **Shipped** (PDF), 🚧 **partial** (health sync)
  - PDF export (`GET /export/pdf?period=monthly|yearly`) - Phase 5d, gated to Plus+
  - Apple Health sync (`writeMindfulSession`) - Phase 5e, iOS complete via HealthKit
  - Google Fit - stub only, pending Google Fit credentials
  - CSV export of raw mood scores - not built (PDF export was judged sufficient)

  ---
  Tier 3 - Originally "120+ Days" (Scale & VC Narrative)

  **11. B2B Corporate Wellness Play*** — ✅ **Shipped** (Phase 5c)
  `companies` + `company_members` tables, self-serve join by slug (`POST /b2b/companies/:slug/join`),
  and an admin-only anonymized team mood dashboard (`GET /b2b/companies/:slug/mood`) with an alert
  threshold at avg_mood < 40. Pricing at ₹199/employee/month (min 50) as originally proposed - see
  docs/PRICING.md.

  **12. Clinical Validation Study*** — ❌ **Not started**
  This remains a business-development track (contacting NIMHANS/AIIMS), not a code deliverable. No
  action has been taken on this in the repository.

  **13. API for Therapists / Mental Health Professionals** — ✅ **Shipped, and far more built than
  proposed** (Phase 5g + Phase 9)
  The original ask (₹1,999/mo therapist dashboard with pre-session briefs) shipped as Phase 5g
  (`GET /therapists/clients/:id/brief` - a real-time Claude-generated pre-session brief) and was then
  expanded dramatically in Phase 9 into a full practice-management tool: external client CRUD, note
  photo → OCR → editable bullets, AI session summaries, envelope encryption (ADR-017), consent gating,
  and a parity Next.js web portal (`therapist-portal/`).

  **14. Apple Watch / Wear OS Quick Entry** — ❌ **Not built**
  No wearable surface exists in the codebase.

  ---
  6. Monetization Architecture

  > **Superseded.** This section is the *original proposal*. The actual, decided pricing model (finalized
  > 2026-06-11, last updated 2026-07-02) lives in **[docs/PRICING.md](docs/PRICING.md)** and differs in
  > structure: tiers now split by *product line* (journal-only vs journal+therapy) rather than by feature
  > tier, IAP consumable passes replaced Stripe subscriptions, and annual passes were added. Use
  > docs/PRICING.md for anything pricing-related; the table below is left for historical context only.

  Original proposal (do not use for current decisions):

  Free Tier (Forever)
  ├── 10 entries/month
  ├── Basic reflection (standard prompt)
  ├── 7-day mood chart
  └── 3-turn follow-up

  Ode Plus - ₹199/month India | $7.99/month Global
  ├── Unlimited entries
  ├── Hindi + regional language
  ├── Life Graph (30/90/365 day view)
  ├── Weekly Emotional Review
  ├── All prompt modes (Rant, Gratitude, Decision)
  ├── Streak freeze (2x/week)
  └── Therapist share link (5/month)

  Ode Pro - ₹499/month India | $14.99/month Global
  ├── Everything in Plus
  ├── PDF export (monthly reports)
  ├── Apple Health / Google Fit integration
  ├── Unlimited therapist share links
  ├── Priority processing (faster Claude response)
  └── Early access to new features

  B2B Wellness - ₹199/employee/month (min 50 employees)
  ├── All Pro features for employees
  ├── HR dashboard (aggregated only, never individual)
  ├── Monthly wellness report
  └── Dedicated support

  **What actually shipped instead** (see docs/PRICING.md for full detail and unit economics):

  | SKU | India | Global | Includes |
  |---|---|---|---|
  | Free | ₹0 | $0 | 10 entries/mo, basic reflection, 7-day chart, 3-turn follow-up |
  | Plus (journal only) | ₹249/mo · ₹1,999/yr | $5.99/mo · $39.99/yr | Unlimited entries, Hindi/Hinglish, all prompt modes, Life Graph, Weekly + Annual Review, streak freeze, therapist share (5/mo), PDF export, Apple Health |
  | Pro (journal + therapy) | ₹499/mo · ₹4,499/yr | $9.99/mo · $79.99/yr | Everything in Plus + 1 therapy session/mo included, extra sessions at member price, unlimited therapist share, priority processing |
  | Therapy session (standalone) | ₹499 | $7.99 | Pay-per-use, any plan including Free |
  | B2B Wellness | ₹199/employee/mo (min 50) | custom | All Pro features, HR dashboard, monthly report |

  Key differences from the original proposal: Plus is deliberately *journal-only* (not "Pro minus a few
  things") so journal-only users aren't subsidizing therapy AI costs; plan passes are one-time 30/365-day
  consumable IAP purchases rather than auto-renewing subscriptions (a considered decision, see
  docs/PRICING.md "Why this shape"); and pricing is ₹249 not ₹199 for Plus because ₹199 nets a
  worst-case margin of only 22% against a heavy user's serve cost, deemed too thin against further INR
  weakening.

  ---
  7. What VCs Actually Want to See (Brutally Honest)

  At the time of writing you were "a technically impressive prototype without a story." The feature gaps
  called out in that framing are now closed - the traction gaps are not (the app has not yet had a public
  launch, per docs/LAUNCH_CHECKLIST.md).

  ┌───────────────────────┬─────────────────────────────────────┬──────────────────────────────────┐
  │     What VCs Ask      │      Status as of 2026-08-03        │          Still needed            │
  ├───────────────────────┼─────────────────────────────────────┼──────────────────────────────────┤
  │ How many DAUs?        │ 0 (pre-launch)                       │ 500+ DAUs before pitching seed   │
  ├───────────────────────┼─────────────────────────────────────┼──────────────────────────────────┤
  │ What's D30 retention? │ No data (not launched)               │ 25%+ is good; 40%+ is exceptional│
  ├───────────────────────┼─────────────────────────────────────┼──────────────────────────────────┤
  │ What's your outcome   │ Event stream exists                  │ A self-reported improvement      │
  │ data?                 │ (analytics_events); no outcome       │ metric surfaced to users, then   │
  │                       │ metric surfaced yet                  │ measured                          │
  ├───────────────────────┼─────────────────────────────────────┼──────────────────────────────────┤
  │ Who else is on the    │ Unknown / not tracked in this repo   │ Clinical advisor OR therapist    │
  │ team?                 │                                      │ co-founder                       │
  ├───────────────────────┼─────────────────────────────────────┼──────────────────────────────────┤
  │ Why can't Rosebud     │ Now much stronger: Hindi + Life      │ Real usage data to prove the     │
  │ copy you?             │ Graph + B2B + full Therapy Mode +    │ moat converts to retention        │
  │                       │ Therapist Workspace, all shipped     │                                   │
  ├───────────────────────┼─────────────────────────────────────┼──────────────────────────────────┤
  │ What's the            │ Freemium + B2B + IAP billing all    │ Revenue proof at scale             │
  │ monetization?         │ implemented (docs/PRICING.md)        │                                   │
  ├───────────────────────┼─────────────────────────────────────┼──────────────────────────────────┤
  │ Why India now?        │ Good answer (Rocket Journal is       │ Need a slide on it; re-verify     │
  │                       │ weaker than they look) - unverified  │ their current Android/feature      │
  │                       │ since original analysis              │ status before using in a pitch     │
  └───────────────────────┴─────────────────────────────────────┴──────────────────────────────────┘

  The most important thing to do now is the same as it was: get real users post-launch and measure their
  D7/D30 retention. Every VC conversation will start with that number, and no amount of shipped
  feature-work substitutes for it.

  ---
  8. The Rocket Journal Problem - Your Response Strategy

  Rocket Journal's moats: existing therapy user base, clinical credibility, first-mover in India, Rocket
   Health brand.

  Ode's counter-strategy - don't fight on their ground:

  1. Go deeper on AI quality - Rocket Journal's reflections are generic (I'd bet on this). Your prompt
  engineering is sophisticated. Make Ode's reflections visibly, noticeably better. Show this in
  marketing: "Read a Ode reflection vs. a competitor's reflection. Spot the difference."
  2. Own the non-clinical user - Rocket Journal will always feel like it's connected to therapy/mental
  illness. Stigma is real in India. Ode should feel like a personal growth tool for
  well-functioning people who want to understand themselves. "Not because something's wrong - because
  you want to know yourself better."
  3. Win on language - Hindi, Tamil, Bengali. Rocket Journal won't move fast here because their
  therapist network and clinical content is English-first. (Hindi/Hinglish shipped; Tamil/Bengali still
  open.)
  4. Win on data ownership narrative - "Your journal stays yours. We never share it. No therapist reads
  it unless you choose to share." This is a trust story that a therapy-company-owned app can never fully
   tell. (Now backed in code by the consent-gated therapist link model, Phase 5g/9.)
  5. Win on B2B - Rocket Health is a B2C therapy platform. Corporate wellness is not their core motion.
  You can own it. (Shipped, Phase 5c.)

  ---
  9. The One Big Bet You Should Consider

  After 10+ years of building startups, the companies that get funded aren't the ones with the best
  features - they're the ones with the clearest story about why they win at scale.

  Ode's big bet story:

  ▎ "Every other mental health app knows how you feel today. We're building the first system that
  ▎ understands how you feel across years - detecting emotional patterns before you're aware of them,
  ▎ connecting triggers to outcomes, and giving you a longitudinal map of your inner life. We start with
  ▎  voice journaling because it's the lowest-friction way to collect this data. The data is the moat.
  ▎ The insights are the product."

  This story is about becoming the mental health data layer - not a journaling app. The journaling is
  just how you collect the emotional signal. If you can build longitudinal emotional intelligence that's
   genuinely predictive ("the last 4 times your mood dropped below 35, a major work event preceded it by
   3 days"), you have something defensible, scientifically interesting, and fundable.

  > **Status note:** the predictive/pattern-detection piece exists today only as `connection_insight`
  > (deterministic recurring-topic detection, Phase 4). True predictive pattern-matching across mood
  > drops and preceding events, as described in the example above, has not been built.

  ---
  10. Immediate Action Plan (Next 30 Days) — as originally written, with outcomes

  1. Soft launch to 100 beta users → ❌ not done; app has not had a public/beta launch yet
  2. Build the Weekly Emotional Review → ✅ done (Phase 4b)
  3. Add Hindi language support → ✅ done (Phase 4e)
  4. Build the shareable milestone card → ✅ done (Phase 4c/4d)
  5. Add outcome measurement (post-10th-entry mood-comparison prompt) → ❌ not done; `analytics_events`
     tracks product events but no user-facing outcome-measurement prompt exists
  6. Write 3 tests (crisis detection, worker pipeline, conversation turn cap) → ✅ far exceeded - 600+
     tests now exist across the backend (see docs/TESTING.md)
  7. Get a clinical advisor → unknown/not tracked in this repository

  **The current, real action plan is [docs/LAUNCH_CHECKLIST.md](docs/LAUNCH_CHECKLIST.md)**: IAP store
  setup, store compliance (privacy policy, data-safety forms), iOS launch readiness, production env
  verification, and monitoring. Items 1, 5, and 7 above (beta launch, outcome measurement, clinical
  advisor) remain genuinely open and are not tracked anywhere else in the docs - worth deciding whether
  they belong on the launch checklist or a separate growth/traction plan.

  ---
  Bottom line (original): Ode has excellent technical bones, a genuinely thoughtful AI layer, and is
  entering a real market at the right time. What it doesn't have yet is users, retention data, or a moat
  story that's compelling under VC scrutiny. The Life Graph + Hindi + B2B corporate wellness is the
  differentiation triangle that makes the story work. Build those three things, get 500 real users,
  measure your D30, and you have a credible seed pitch.

  **Bottom line (now):** the differentiation triangle got built - and then some (Therapy Mode, Therapist
  Workspace, Guided Journeys, Life Chapters, Relationship Map all shipped beyond the original ask). The
  gap that remains is exactly the one this document couldn't close by itself: real users and real
  retention data. That's the next milestone, not another feature.
