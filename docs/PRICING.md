# DreamLog Pricing & Unit Economics

Final pricing model decided 2026-06-11. This is the single source of truth for prices,
margins, infrastructure costs, and the analytics we track to validate the model.

All INR figures assume **₹96/USD** (update the Unit Economics section if the rate moves
materially — AI costs are dollar-denominated, India revenue is rupee-denominated, so a
weaker rupee compresses India margins).

---

## 1. Final Price Structure

Two subscription tiers split by **product line** (journal vs journal + therapy), each sold
as a monthly or annual pass, plus one pay-per-use consumable. No session packs. B2B stays
sales-led and off-store.

| SKU | India | Global | Store product type |
|---|---|---|---|
| **Free** | ₹0 | $0 | — |
| **Plus** (journal only) | ₹249/mo | $5.99/mo | Auto-renewing subscription |
| **Plus — annual** | ₹1,999/yr (~₹167/mo, save 33%) | $39.99/yr (~$3.33/mo, save 44%) | Auto-renewing subscription |
| **Pro** (journal + therapy) | ₹499/mo | $9.99/mo | Auto-renewing subscription |
| **Pro — annual** | ₹4,499/yr (~₹375/mo, save 25%) | $79.99/yr (~$6.67/mo, save 33%) | Auto-renewing subscription |
| **Therapy session — Pro member price** | ₹299 | $4.99 | Consumable IAP |
| **Therapy session — standalone** | ₹499 | $7.99 | Consumable IAP |
| **B2B Wellness** | ₹199/employee/mo (min 50) | custom | Invoiced, not in stores |

EUR mirrors USD (€5.99 / €39.99 / €9.99 / €79.99; sessions €7.99 / €4.99).
Plan passes are sold as **consumable In-App Purchases** (App Store / Google Play,
verified server-side): monthly = 30-day pass, annual = 365-day pass, expiry always
set server-side. Stripe was removed 2026-07-13; store pricing is configured per
product in App Store Connect / Play Console (the tables above are the list prices
to configure there).

### What each tier includes

```
Free
  10 entries/month · basic reflection · 7-day mood chart · 3-turn follow-up

Plus — the complete journal, no therapy
  Unlimited entries · Hindi/Hinglish · all prompt modes (incl. Dream Decoder)
  Life Graph · Weekly Review · Annual Review · streak freeze
  therapist share (5/mo) · PDF export · Apple Health sync

Pro — everything in Plus, plus therapy
  1 therapy session/month included
  Extra sessions at member price (₹299 / $4.99)
  Unlimited therapist share · priority processing

Therapy session (standalone, any plan incl. Free)
  Up to 1 hour · voice or text · AI voice output · post-session summary
  Journal-context-aware · crisis detection always active
```

### Why this shape (decisions log)

- **Plus is journal-only, not "Pro minus features".** Journal-only users won't pay ₹499
  to fund therapy AI costs they never use. Splitting by product line is something users
  understand in one read; the old Plus/Pro feature split was not.
- **Plus at ₹249 not ₹199**: at ₹96/USD a heavy journal user (30 entries/mo) costs
  ~₹112/mo to serve; ₹199 nets ~₹143 → 22% worst-case margin, too thin against further
  INR weakening. ₹249 nets ~₹179 → 37% floor.
- **Pro includes 1 session, not 2** (changed from the Phase 6 design): 2 included
  sessions at ₹499 produced a ~12% worst-case margin at ₹96/USD and would go negative if
  the rupee slides further. With 1 included session the floor is ~41%, and heavy therapy
  users buying extras are profitable per-session instead of margin-destroying.
- **No session packs** (5-pack / 12-pack removed): pre-purchase commitments create IAP
  refund/restore/expiry complexity and the heavy user they target should be on Pro.
  Revisit only with real demand data.
- **Standalone session price = Pro monthly price (₹499)**: deliberate anchor. One
  standalone session costs the same as a month of Pro that includes a session — the
  upgrade pitch writes itself.
- **Sell via IAP at the 15% small-business rate** (decision per LAUNCH_CHECKLIST §1):
  web-only Stripe keeps ~8 more points of revenue but forbids any in-app purchase path
  (Guideline 3.1.1) and makes us merchant of record for Indian GST. Not worth it
  pre-traction. Web checkout can be added later as a parallel channel (never linked from
  the iOS app). **Implemented 2026-07-13**: direct store IAP (`expo-iap` on mobile,
  server-side receipt verification on the backend) — no RevenueCat dependency; plan
  passes stay consumables, not auto-renewing subscriptions.
- **Annual passes added 2026-07-02** — upfront cash, locked-in retention, and a
  permanent *honest* "% off" badge (annual-vs-monthly is the legitimate version of
  discount anchoring; we do NOT inflate list prices to fake a permanent discount —
  misleading-pricing rules + trust erosion in a mental-health brand).
- **India annual discount is deliberately shallower than global** (33%/25% vs 44%/33%):
  India margins are FX-exposed (costs in USD, revenue in INR). At a 40%-off Plus annual
  (₹1,799) a heavy user (30 entries/mo) is breakeven-to-negative; at ₹1,999 the floor
  stays positive. Global margins are FX-immune, so global can afford the deeper discount.
- **Price-increase playbook**: launch prices are framed as founding-member pricing.
  Raising prices later = raise for new users only, grandfather existing subscribers.
  No fake strikethrough pricing, ever.

---

## 2. Revenue Stack (what we actually receive)

India listed prices are GST-inclusive; Apple/Google are merchant of record and remit the
18% GST. Both stores take 15% commission under their Small Business Programs (<$1M/yr —
**enrollment in both programs is mandatory before first sale**, see Action Items).

Net multiplier India = 0.85 / 1.18 ≈ **0.72** · Global ≈ **0.85**

| SKU | Listed (INR) | Net to us (INR) | Listed (USD) | Net to us (USD) |
|---|---|---|---|---|
| Plus | ₹249 | **₹179** | $5.99 | **$5.09** |
| Plus annual | ₹1,999 | **₹1,439/yr (₹120/mo)** | $39.99 | **$33.99/yr ($2.83/mo)** |
| Pro | ₹499 | **₹359** | $9.99 | **$8.49** |
| Pro annual | ₹4,499 | **₹3,239/yr (₹270/mo)** | $79.99 | **$67.99/yr ($5.67/mo)** |
| Session (member) | ₹299 | **₹215** | $4.99 | **$4.24** |
| Session (standalone) | ₹499 | **₹359** | $7.99 | **$6.79** |

At the standard 30% commission (if we fail to enroll) the India multiplier drops to 0.59
and the Pro worst case goes negative. Enrollment is not optional.

---

## 3. Variable Costs (per-user AI costs, ₹96/USD)

| Activity | USD | INR | Basis |
|---|---|---|---|
| Journal entry (3 min audio) | ~$0.035 | ~₹3.4 | Whisper $0.006/min + Sonnet analysis w/ context (cached) |
| Therapy session — worst case | ~$1.05 | ~₹101 | Full 1 hr, 30 turns, TTS every turn (see ARCHITECTURE.md cost table) |
| Therapy session — realistic | ~$0.55–0.70 | ~₹53–67 | ~30 min average, prompt caching active |
| Weekly review + nudges + misc | — | ~₹5–10/mo | Per active user |

Real session costs depend on **prompt caching being active in production** — the $0.70
Claude figure assumes it. Verify in the Anthropic console after launch (cache-read tokens
should dominate).

---

## 4. Margins

### Per-tier monthly margin (India, net revenue vs serve cost)

| Tier / scenario | Net revenue | Cost to serve | Margin |
|---|---|---|---|
| Plus — average (10 entries) | ₹179 | ~₹40 | **78%** |
| Plus — heavy (30 entries) | ₹179 | ~₹112 | **37%** (floor) |
| Pro — average (10 entries + 1 session) | ₹359 | ~₹145 | **60%** |
| Pro — heavy (30 entries + 1 full session) | ₹359 | ~₹213 | **41%** (floor) |
| Extra session — member ₹299 | ₹215 | ~₹101 | **53%** |
| Standalone session ₹499 | ₹359 | ~₹101 | **72%** |

### Global (immune to FX — revenue and costs both in USD)

| Tier / scenario | Net revenue | Cost to serve | Margin |
|---|---|---|---|
| Plus — heavy | $5.09 | ~$1.15 | **77%** |
| Pro — heavy | $8.49 | ~$2.20 | **74%** |
| Standalone session | $6.79 | ~$1.05 | **85%** |

### Annual passes (net monthly-equivalent revenue vs serve cost)

| Tier / scenario | Net rev /mo | Cost to serve /mo | Margin |
|---|---|---|---|
| Plus annual India — average (10 entries) | ₹120 | ~₹42 | **65%** |
| Plus annual India — heavy (30 entries) | ₹120 | ~₹110 | **~8%** (floor) |
| Pro annual India — average | ₹270 | ~₹145 | **46%** |
| Pro annual India — heavy (30 entries + full 1 hr session) | ₹270 | ~₹213 | **21%** (floor) |
| Pro annual India — heavy, realistic session (~30 min) | ₹270 | ~₹170 | **~37%** |
| Plus annual global — heavy | $2.83 | ~$1.15 | **59%** |
| Pro annual global — heavy | $5.67 | ~$2.20 | **61%** |

The India annual floors are thin **on paper** but assume a p99 user sustaining maximum
usage for 12 consecutive months — and the cash is banked upfront, so annual cohort
blended margin lands near the average case (journaling intensity decays after months
1–3). Watch the `entries per paying user p90` metric (§6b); if the heavy tail grows,
cut serve cost (cheaper STT, batched analysis) before touching prices.

Key property of this structure: **margins improve with engagement** (extra sessions are
profitable per-unit) instead of degrading, and no SKU has a negative worst case.

---

## 5. Fixed Costs (tech stack)

| Service | Plan | Monthly (USD) | Notes |
|---|---|---|---|
| Railway (API + worker + Postgres + Redis) | Hobby/Pro | $20–40 | Scales with usage |
| Supabase (auth) | Free → Pro | $0 → $25 | Free tier fine until ~50K MAU |
| Cloudflare R2 (audio) | Free tier | ~$0 | Audio deleted after transcription (ADR-005); storage ~nil |
| Firebase FCM (push) | Free | $0 | |
| Sentry (errors, 3 projects) | Developer | $0 | ~5K events/mo free |
| UptimeRobot | Free | $0 | |
| RevenueCat (IAP) | Free tier | $0 → 1% | Free up to $2.5K/mo tracked revenue |
| Expo EAS builds | Free tier | $0–19 | Free tier covers launch cadence |
| Domain + email | — | ~$2 | |
| **Total fixed** | | **~$25–60/mo (₹2.5–6K)** | |

One-time / annual: Apple Developer **$99/yr**, Google Play **$25 once**.

### Break-even

At worst-case fixed costs (~₹6K/mo): **~17 Pro subscribers** or **~34 Plus subscribers**
(India net margins) cover the entire infrastructure. Everything beyond that is
contribution margin.

### Illustrative monthly profit (India-weighted mix, average usage)

| Scale | Mix | Net revenue | AI costs | Fixed | Profit |
|---|---|---|---|---|---|
| 100 paying (70 Plus / 30 Pro) | +20 standalone sessions/mo | ₹30.5K | ₹9.2K | ₹6K | **~₹15K** |
| 500 paying (300 Plus / 200 Pro) | +100 sessions/mo | ₹161K | ₹46K | ₹8K | **~₹107K** |
| 2,000 paying (1,000 / 1,000) | +500 sessions/mo | ₹717K | ₹196K | ₹15K | **~₹506K** |

---

## 6. Analytics & Metrics (what we must track)

The pricing model embeds bets that need measurement. Three layers:

### 6a. Purchase / revenue metrics (RevenueCat + `payments` table)

RevenueCat dashboards give most of this free; mirror every transaction into the backend
`payments` table (migration 000026 — extend with `store`, `product_id`, `currency`,
`amount`, `country`) so revenue queries can join against usage.

- MRR, ARPU, ARPPU; revenue split India vs global (FX exposure)
- Free → Plus and Free → Pro conversion rate; Plus → Pro upgrade rate
- Trial/paywall view → purchase funnel; standalone session purchases by plan
- Churn + reactivation per tier; refund rate
- **Effective store fee** (verify 15% small-business rate is actually applied)

### 6b. The margin-critical bets (compute monthly from our own DB)

| Metric | Why it matters | Source |
|---|---|---|
| **Session redemption rate** (% of Pro users using their included session) | The Pro margin floor assumes ≤100%; if redemption is low, included sessions could go back to 2 as a retention lever | `therapy_sessions` joined to `users.plan` |
| **Avg session duration + turns + TTS chars** | Validates the $0.55–1.05 cost range | `therapy_sessions.duration_sec`, `turn_count` |
| **Entries per paying user/month** (p50 / p90 / max) | Validates the heavy-user cost ceiling | `entries` |
| **AI cost per user/month** | Direct margin tracking | Anthropic/OpenAI/Azure usage exports |
| **Prompt cache hit rate** | The $0.70 Claude session cost assumes caching works | Anthropic console |
| **Extra-session attach rate** (Pro users buying ₹299 sessions) | Whether the member price earns its IAP SKU complexity | `payments` |

### 6c. Product analytics (event stream)

Add a lightweight append-only `analytics_events` table (`user_id`, `event_name`,
`properties JSONB`, `created_at`) written from existing handlers/services — no external
SDK needed at launch; export to PostHog/Amplitude later if slicing in SQL gets painful.

Minimum event set:

- `signup`, `onboarding_step_completed` (per step), `onboarding_completed`
- `entry_recorded`, `entry_completed`, `entry_failed` (with mode)
- `reflection_viewed`, `followup_started`, `followup_turn`
- `therapy_session_started` / `ended` (persona, duration, turns, input modes)
- `paywall_viewed` (which screen: upgrade / therapy-pricing / 402-redirect)
- `purchase_initiated`, `purchase_completed`, `purchase_failed` (SKU, currency)
- `plan_changed` (from → to, source), `entry_limit_hit` (the Free 10/mo wall)
- `share_created`, `insight_card_shared`, `export_downloaded`
- Retention inputs: app-open events for D1/D7/D30 cohorts (mobile-side, batched)

Privacy rule: events carry IDs and metadata only — **never transcript or reflection
content, ever**.

---

## 7. Implementation Changes Required (tracking list)

Code/doc changes to land the new pricing:

- [x] Pro included sessions: 2 → **1**/month (`models.TherapyProMonthlyAllowance`,
      `TherapyService.computeBilling` + tests) ✅ 2026-06-11
- [x] Member extra-session price ₹299 (`models.TherapyMemberSessionPricePaise`);
      standalone ₹499 unchanged ✅ 2026-06-11
- [x] Remove 5-pack / 12-pack from `app/therapy/pricing.tsx` (Single + "Get Pro" only,
      Pro is the highlighted BEST VALUE card) ✅ 2026-06-11
- [x] `app/upgrade.tsx` + `src/services/region.ts`: Plus repositioned as journal-only
      at ₹249/$5.99 (PDF export, Health sync, annual review under Plus); Pro =
      "journal + therapy" at ₹499/$9.99 ✅ 2026-06-11
- [x] Backend plan limits: `HasPDFExport` true for Plus; `/export/pdf` gate lowered
      Pro → Plus ✅ 2026-06-11
- [x] `POST /billing/create-payment-intent` amounts: plus 24900/599/€599, pro
      49900/999/€999 (superseded later by IAP migration) ✅ 2026-06-11
- [x] Docs synced: API_CONTRACT.md, ROADMAP.md, ARCHITECTURE.md, TESTING.md,
      legal/TERMS_AND_CONDITIONS.md ✅ 2026-06-11
- [x] IAP migration: 4 consumable plan-pass SKUs, both stores; `expo-iap` client +
      server-side receipt verification (`services/iap.go`); Stripe removed ✅ 2026-07-13
- [ ] Create the 4 products in App Store Connect + Play Console
      (`com.dreamlog.app.{plus,pro}.{monthly,annual}`) at the list prices above
- [ ] Enroll: App Store Small Business Program + Play 15% tier
- [ ] Therapy pay-per-use charge server-side (`computeBilling` still returns 402 in
      prod for non-included sessions - wire a therapy-session consumable SKU through
      the same IAP verification path)
- [x] `analytics_events` migration (000028) + `services/analytics.go` + `repositories/analytics.go` wired via `handlers/router.go` ✅ 2026-06-11
- [x] Extend `payments` table: `store`, `product_id`, `country` — migration 000029 ✅ 2026-06-11
- [x] Annual passes: backend `period` on `/billing/upgrade` (365-day server-set
      expiry), annual amounts + tests ✅ 2026-07-02 (Stripe verification since
      replaced by store IAP verification, 2026-07-13)
- [x] Mobile upgrade screen: Monthly/Annual segmented toggle (annual default,
      savings badges, per-month equivalents); annual price constants in
      `src/services/region.ts` ✅ 2026-07-02
- [x] Website: Monthly/Annual toggle on `/pricing` + homepage pricing section;
      EUR aligned to canonical (€5.99/€9.99, session €7.99, member €4.99);
      terms page pass copy updated ✅ 2026-07-02
- [ ] Future option: convert plan passes from consumables to real auto-renewing
      subscriptions (would require subscription SKUs + renewal webhooks; the
      current one-time-pass model needs neither)

---

*Last updated: 2026-07-02 (annual passes added) · FX assumption ₹96/USD · Owner: pricing
decisions in this file override the older monetization table in ROADMAP.md.*
