# DreamLog — AI Design Prompts

Two prompts to hand to an emergent AI design tool (v0, Bolt, Lovable, etc.).
Goal: give the AI enough product context to invent a fresh UI direction — not recreate what exists.

---

## PROMPT 1 — Mobile App

Design the mobile app for **DreamLog** — a voice journaling app that uses AI to turn spoken thoughts into emotional reflections.

### What the app does

Users open the app, tap a record button, and speak freely for up to 30 minutes. They don't type anything. The app sends the audio to an AI pipeline that transcribes it, screens for emotional crisis, and generates a personalized reflection — a warm 3–5 sentence response that names what the user seems to be feeling, highlights patterns across past entries, and ends with an open question to sit with. The whole process takes under a minute.

After reading their reflection, users can have a short 3-turn follow-up conversation with the AI about what came up.

### Core features

- **Voice journaling** — one-tap record, any length, no typing
- **AI reflections** — mood score (1–100), emotional tones with intensity, topics, key quotes pulled from the audio, and a personalized written reflection
- **Journal modes** — users pick a mode before recording: Processing (default), Rant, Gratitude, Decision, or Dream. Dream mode triggers a special dual-lens analysis (a Jungian psychological reading and a Vedic/Indian spiritual reading of the dream symbols)
- **Mood tracking** — 7-day strip, 30/90/365-day life graph, streak counter, emotion pattern radar
- **Therapy Mode** — a separate real-time AI conversation (voice or text) of up to 1 hour, grounded in the user's journal history. The AI knows the user's mood trends and emotional patterns from past entries before the first word. Four AI personas: Comforting, Rational, CBT-informed, Mindful
- **Guided Journeys** — structured multi-step prompting programs (e.g. Stress Relief, Grief Processing, Decision Clarity) that guide the user through a series of journal entries
- **Life Chapters** — user-defined named time periods (e.g. "My Bangalore chapter") with AI-generated narrative summaries of that period's entries
- **Relationship Map** — AI automatically extracts people mentioned across entries and tracks sentiment over time
- **Weekly & Annual Reviews** — Claude-generated narrative reviews of the user's emotional arc over a week or year
- **Morning nudges** — personalized push notifications based on yesterday's entry, delivered at the user's chosen time in their local timezone
- **Crisis safety** — every entry and therapy message is screened for crisis signals. If detected, the app shows emergency resources (hotlines, tap-to-call) and routes the user to professional help

### The user

Someone who wants to process their inner life but finds writing too slow or effortful. They speak the way they think. The app is a private companion — not a social platform, not a productivity tool. Users typically journal at night or during commutes. They come in because of stress, anxiety, grief, relationship difficulty, career transitions, or just curiosity about their inner patterns.

### Emotional goal system

During onboarding, users pick their primary emotional goal (Stress / Anxiety / Grief / Depression / Relationships / Career / Trauma / Curious). This shifts the app's entire color theme — each goal has its own dark palette — and personalizes the AI's tone in reflections.

### Tone and feel

Private. Safe. Unhurried. The app should feel like opening a journal at night — it doesn't rush you. It's not clinical. It's not a productivity dashboard. The emotional vocabulary in the UI ("I'm listening", "speak freely", "your reflection is ready") matters as much as the layout.

### What to design

Design the complete mobile app: onboarding, home screen, recording, processing/waiting, reflection view, follow-up chat, timeline/history, mood and insights, therapy mode, settings. Show how the app handles the recording-to-reflection flow as a coherent journey. The design should feel fresh and considered — this is a deeply personal tool, and the UI should reflect that.

---

## PROMPT 2 — Product Website (with Therapist Portal)

Design the website for **DreamLog** — a voice journaling app with AI-powered emotional reflections. The website serves two purposes in one: a public-facing product site that acquires users and communicates the product's value, and a logged-in therapist portal where mental health professionals manage their clients' data. Both live under the same domain and share the same visual identity.

---

### Part A — Public Product Website

This is what a potential user or therapist sees before they sign up. It needs to make someone feel something — not just read a list of features.

**What the site communicates:**

DreamLog is a private voice journaling app. You speak, the AI listens, and it reflects back what it hears — your emotional patterns, recurring themes, the things you keep coming back to. It is not a chatbot. It is not therapy. It is a mirror for your inner life, built for people who think faster than they type.

**The audience is two groups:**

1. **Individual users** — people dealing with stress, anxiety, grief, relationship difficulty, career transitions, or just wanting to understand themselves better. They want something private, non-judgmental, and actually useful — not another journaling app that's just a text editor with a dark mode.

2. **Therapists and counsellors** — mental health professionals who want to stay connected to their clients' emotional state between sessions. DreamLog gives them a data layer: mood trends, AI summaries of entries, and a pre-session brief generated before each appointment.

**Key things the homepage should convey:**

- The core loop is simple: speak → AI reflects → you understand yourself better
- The AI knows your history — reflections get more personal over time as patterns emerge
- There is a Therapy Mode: a real-time AI conversation (up to 1 hour) that the AI enters already knowing your emotional context from past entries. Four AI companion styles: Comforting, Rational, CBT-informed, Mindful
- Crisis safety is built in — every entry and therapy session is screened; if distress signals are detected, the app surfaces professional resources immediately
- Pricing: Free tier (10 entries/month), DreamLog+ (₹249/month · $5.99 — unlimited journaling, all modes, weekly reviews, mood history), DreamLog Pro (₹499/month · $9.99 — everything plus 1 therapy session/month), Pay-per-session therapy (₹499 · $7.99 each)
- Available on iOS and Android

**Pages to design:**

- **Homepage** — hero, how it works (3-step flow: speak → reflect → understand), feature highlights, social proof / testimonial area, pricing, download CTA
- **Features page** — deeper look at: voice journaling, AI reflections, Therapy Mode, Dream Decoder (dual Jungian + Vedic analysis of dreams), Guided Journeys, Life Chapters, Relationship Map, Weekly/Annual Reviews, mood tracking
- **Pricing page** — Free, Plus, Pro, Therapy sessions, B2B/corporate wellness
- **For Therapists landing page** — targeted at mental health professionals; explains the therapist portal, the data they get access to, the privacy model (no raw transcripts, client-consented sharing), and a CTA to register
- **Login / Sign up page** — clean auth page that routes users to the app and therapists to the portal

---

### Part B — Therapist Portal (logged-in area)

Once a therapist signs in, they enter a dashboard that lives within the same site. The portal is a B2B2C tool where therapists monitor their clients' emotional wellbeing data between sessions.

**How it works:**

Therapists register on the site and link clients who use the DreamLog mobile app. The client shares a UUID from their app settings — no email lookup, preserving privacy. Once linked, the therapist sees aggregated emotional data from the client's journal history.

**What therapists see:**

- **Client list** — all linked clients with 30-day average mood, trend direction (improving / declining / stable), last entry date, low-mood alerts
- **Client detail** — AI-generated 3-sentence pre-session brief (the centrepiece), 30-day mood chart, top emotions, recent entry summaries (AI-summarized, never raw transcripts), any crisis events
- **Pre-session brief** — written by AI on demand before a session: what the client has been going through, dominant emotional themes, a relevant quote from a recent entry. Therapists use this to walk into a session already oriented

**Privacy:** Therapists never see raw audio, full transcripts, or the AI's reflective text. Only: mood scores, AI summaries, topics, key quotes. The client actively chooses to share.

**Who uses it:** Therapists with 10–40 clients, on a laptop before their first session or a tablet in the therapy room. Needs to work at a glance.

---

### Overall tone and feel

The public site and the portal should feel like one continuous product — not a marketing site bolted onto a dashboard. Dark, calm, considered. Warm without being soft. This is for people doing real emotional work, and the design should carry that weight without being heavy. Think of the difference between a clinical EHR and a tool a therapist would actually want to open.

### What to design

Design the full website: homepage, features page, pricing, for-therapists landing page, login/signup, and the logged-in therapist portal (client list, client detail with pre-session brief and mood chart, add-client flow). The design should work at full desktop width and scale gracefully to tablet. Everything — the marketing pages and the portal — should feel like one product.
