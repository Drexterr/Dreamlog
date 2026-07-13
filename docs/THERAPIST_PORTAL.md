# Ode Therapist Workspace

The therapist workspace lets a practicing therapist use Ode as their daily practice tool —
in the mobile app **and** the web portal (`therapist-portal/`). It serves two kinds of clients:

1. **External clients** — the therapist's own offline clients who do not use Ode.
   The therapist manages them entirely: adds them, photographs handwritten session notes,
   edits the extracted bullets, and generates AI session summaries.
2. **Linked app clients** — Ode users who explicitly consented (in their app) to share
   journal *summaries* with the therapist. Read-only mood trends + AI pre-session briefs
   (Phase 5g). The therapist can also attach their own session notes to a linked client.

A therapist is a normal Ode user with a `therapists` row (`therapists.user_id → users.id`).
Same Supabase JWT, same auth middleware — the "I'm a therapist" pill on the mobile auth screen
only changes *where the user lands after login*, not how they authenticate. Therapists get the
full journaling product too ("My journal →" on the dashboard).

---

## Feature Overview

| Feature | Mobile | Web portal |
|---|---|---|
| Therapist registration + consent | `app/therapist/register.tsx` | `/login` → register |
| Dashboard (caseload metrics) | `app/therapist/index.tsx` | `/dashboard` |
| External client CRUD | `app/therapist/clients.tsx` | `/dashboard/notes` |
| Photo-of-notes → OCR → bullets | `app/therapist/add-session.tsx` (camera/gallery) | `/dashboard/notes` (file upload) |
| Typed notes (no photo) | ✅ | via bullet editor |
| Edit bullets | `app/therapist/session/[id].tsx` | `/dashboard/notes` |
| AI session summary | ✅ | ✅ |
| Linked app clients + AI brief | list in `clients.tsx` | `/dashboard/clients/[id]` |
| Use the journal as a user | "My journal →" | — |

## The Note Pipeline

```
Therapist photographs handwritten notes after a consultation
  │
  ├─ POST /therapists/sessions/presign   { content_type: image/jpeg }
  │     ← { upload_url, image_key }      # key namespaced notes/{therapistID}/…
  ├─ PUT photo → R2/MinIO (direct, pre-signed)
  ├─ POST /therapists/sessions           { external_client_id, image_key }
  │     backend: INSERT client_sessions (status=pending), LPUSH note OCR job
  │
  │   [Worker: NoteOCRWorker]
  │     → download photo
  │     → Claude vision OCR → { raw_text, bullets[] }   (NO crisis screening — ADR-018)
  │     → encrypt raw_text + bullets with the therapist's data key
  │     → store ciphertext, status=completed
  │     → DELETE photo from storage      (most sensitive artifact; ADR-005 spirit)
  │
  ├─ client polls GET /therapists/sessions/:id until status=completed
  ├─ PATCH /therapists/sessions/:id      { bullets: [...] }        # edit / add / remove
  └─ POST /therapists/sessions/:id/summarize                        # AI summary, stored encrypted
```

Typed-notes path: `POST /therapists/sessions { external_client_id, bullets }` → stored
immediately (`status=completed`), no OCR job.

## Encryption Model (ADR-017)

All sensitive therapist-workspace data — client names, raw OCR text, bullets, AI summaries —
is encrypted **at the application layer** before touching Postgres:

- Per-therapist 32-byte **data key (DEK)**, generated on first use.
- Fields encrypted with AES-256-GCM under the DEK (`pkg/crypto`).
- The DEK is stored **wrapped** (encrypted by the master key) in `therapist_keys`.
- The **master key** lives only in the server environment: `MASTER_ENCRYPTION_KEY`
  (64 hex chars or base64 of 32 bytes). When unset, a key is derived from
  `SUPABASE_JWT_SECRET` (domain-separated SHA-256) so dev needs no setup —
  **set it explicitly in production.**

What this protects: a stolen DB dump, leaked backup, or compromised DB credentials yields
only ciphertext. What it does **not** claim: end-to-end encryption. The server decrypts
transiently in memory to serve the authenticated therapist and to run AI processing
(OCR, summaries). Never describe this as "end-to-end" in user-facing copy.

Additional data-minimisation choices:

- Note **photos are deleted** from storage immediately after successful OCR (and on
  unreadable-photo failures). Only ciphertext text remains.
- The AI summary prompt receives the bullets and an anonymous label ("the client") —
  **the client's name is never sent to the AI**.
- Deleting a client cascades to all their sessions; deleting a session also removes any
  not-yet-processed photo.

## Consent Model

Two separate consents, both versioned:

1. **User ToS** (`users.tos_accepted_at` / `tos_version`): checkbox at email signup;
   a one-time `/accept-terms` screen catches Google/Apple sign-ins and existing users
   after a terms-version bump (`models.CurrentToSVersion`).
2. **Therapist client-data consent** (`therapists.client_consent_accepted_at`):
   "I have my clients' consent and am responsible for the client data I upload."
   Accepted at therapist registration (mobile) or on first visit to Session Notes
   (portal). **Enforced server-side**: creating clients or uploading notes returns
   `403` until accepted.

Linked app clients additionally require the client's own in-app approval of the link
(Phase 5g consent gate) before the therapist can reference them in a session.

## API Surface

All routes require auth; all `/therapists/*` routes below also require a therapist profile.
Full request/response shapes: `docs/API_CONTRACT.md` § Therapist Workspace.

```
GET    /therapists/me                          profile + consent state (404 = not a therapist)
POST   /therapists/consent                     accept client-data terms
GET    /therapists/overview                    caseload metrics
POST   /therapists/external-clients            create (403 until consent)
GET    /therapists/external-clients            list (?include_archived=true)
GET    /therapists/external-clients/:id
PATCH  /therapists/external-clients/:id        rename / role / archive
DELETE /therapists/external-clients/:id        cascades to sessions
POST   /therapists/sessions/presign            note photo PUT URL (jpeg/png/webp)
POST   /therapists/sessions                    create (image_key XOR bullets)
GET    /therapists/sessions                    list (?external_client_id= | ?linked_client_id=)
GET    /therapists/sessions/:id
PATCH  /therapists/sessions/:id                replace bullets
POST   /therapists/sessions/:id/summarize      AI summary
DELETE /therapists/sessions/:id
POST   /me/accept-terms                        record ToS acceptance (any user)
GET    /me/terms                               acceptance state + current version
```

Security properties enforced in code (and covered by tests):

- **Ownership isolation**: every query is scoped by `therapist_id`; therapist B gets 404
  for therapist A's clients/sessions (`therapist_notes_test.go`).
- **Image-key namespacing**: `image_key` must start with `notes/{therapistID}/` — a
  therapist cannot point the OCR pipeline at another therapist's (or any other) object.
- **Content-type allowlist** for uploads (jpeg/png/webp) and a 10 MB photo cap
  (double-enforced in the worker).
- **No crisis screening on therapist notes** (ADR-018) — these are clinical records
  *about* a client, read by the treating professional.

## Dashboard Metrics

`GET /therapists/overview`: external client count, active linked clients, sessions this
week / this month / total, last-session timestamp. Per-client: session count + last session
date (in the client list), plus the Phase 5g mood stats for linked app clients.

## Storage & Env

| Concern | Value |
|---|---|
| Photo storage key | `notes/{therapistID}/{uuid}.{jpg\|png\|webp}` |
| OCR job queue | Redis list `dreamlog:notes:queue` (`WORKER_NOTES_QUEUE_KEY`), DLQ `dreamlog:notes:dlq` |
| Worker | `NoteOCRWorker` goroutine inside `cmd/worker` |
| Encryption master key | `MASTER_ENCRYPTION_KEY` (set on API **and** worker) |
| Dev stubs | `STUB_AI_ANALYSIS=true` stubs OCR + summaries (no API key needed) |

## Operational Notes

- **Key rotation**: rotate by re-wrapping DEKs — decrypt each `therapist_keys.wrapped_dek`
  with the old master key, re-encrypt with the new, update the row. Field ciphertext does
  not change (it's encrypted with DEKs, not the master key). No tooling for this yet —
  write a one-off migration command when first needed.
- **Lost master key = lost notes.** Back the key up in a secrets manager, separately from
  DB backups (storing them together defeats the design).
- **DLQ**: failed OCR jobs land in `dreamlog:notes:dlq` and mark the session `failed`
  with a therapist-visible `error_msg`; the therapist can delete + retake the photo, or
  type the notes manually.
- Sessions with an unprocessed photo that are deleted before the worker runs: the worker
  detects the missing session and deletes the orphaned photo.

## Pricing Hook (future)

`therapists.plan` exists (`trial`) since Phase 5g. A natural gate: free ≤ 2 external
clients, paid unlimited + AI summaries. Not enforced anywhere yet — product decision
pending (see docs/PRICING.md when it lands).
