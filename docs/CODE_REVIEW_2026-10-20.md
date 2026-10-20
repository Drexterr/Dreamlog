Full Codebase Review — 2026-10-19
(updated 2026-10-20)
Scope: main @ c4508bc — backend/ (API + worker), mobile/ , therapist-portal/ ,
.github/workflows/ , scripts/regression/ . Reviewed against: CLAUDE.md , backend/CLAUDE.md ,
mobile/CLAUDE.md , docs/DECISIONS.md (ADR-001 … ADR-020), docs/API_CONTRACT.md ,
docs/TESTING.md . Baseline: go vet ./... is clean and go test -race ./... passes (unit tier; the
-tags=integration Postgres/Redis tier was not run here). Mobile results are in Verification. No
code was changed by this review.
History. The first pass of this review ran on f3da32c (2026-10-19) and reported 3 critical, 14
high, 18 medium and 8 low findings. Since then, 26 commits on main address those findings by
ID (C1…L8) and add two features: typed journals ( POST /entries/text ) and continuous therapy
chat. This version does three things:
1. Checks each earlier finding against the new code (status table).
2. Reports what is still open or only partly fixed.
3. Reviews the code added since the first pass.
Severity scale:
Critical: user safety or data-exposure bug reachable in production today.
High: breaks a documented invariant or loses money/data in realistic conditions.
Medium: correctness/security weakness with a narrower trigger.
Low: rule violations, hygiene, doc drift.
Summary
No critical findings remain. All three first-pass criticals are fixed: the crisis flag is now honoured,
register no longer reactivates deleted accounts, and B2B mood is k-anonymised. Most high
findings are fixed and come with tests. What is left:
ID Severity Finding
N1 High 
Production (Railway) deploys straight from GitHub, so the new test gate
doesn't apply to it
N2 High 
Accounts soft-deleted before the hard-delete change keep all their data, and
linked therapists can still read it
N3 Medium 
Therapy status writes are unconditional, so a crisis stop can still be overwritten
by a concurrent end or expiry
N4 Medium 
Therapy turns time out on the client before the server finishes; "safe retry" re-
sends with no idempotency key
N5 Medium Concurrent same-user replays of one purchase can extend a pass twice
N6 Medium 
The unverified email from public local registration is trusted for B2B join, the
sandbox allowlist, and can block Supabase sign-in
--- page 1 ---
ID Severity Finding
N7 Medium 
Storage objects outlive their rows: over-quota uploads, abandoned uploads,
deleted accounts, TTS audio
N8 Medium The production-safety guard is opt-in ( APP_ENV defaults to development )
N9 Medium 
Under a queue backlog the reconciler marks entries failed that were never
attempted
N10 Medium 
POST /entries still maps every service error to 400, checks quota before
idempotency, and duplicates under concurrency
N11–
N17 
Low See Low
High
N1. Production deploys bypass the new test gate
Where: .github/workflows/deploy.yml:3-8 ("Production runs on Railway … which deploys
from GitHub itself. The Fly.io jobs below … only run when … ENABLE_FLY_DEPLOY is 'true'").
Failure scenario: H13 was fixed for the Fly jobs: they now run on workflow_run of Test with
conclusion == success . But per the same file and LAUNCH_CHECKLIST, production is
Railway, and Railway's GitHub integration deploys every push to main as soon as it lands,
in parallel with CI. A commit that fails go test -race or the crisis coverage gate still reaches
production API and worker. The gate only protects the target that isn't used.
Fix: In Railway, enable "Wait for CI" (service → Settings → Deploy) so a deploy waits for
GitHub checks to pass. Alternatively, disable auto-deploy and trigger railway up from the
workflow_run job. Add a LAUNCH_CHECKLIST item, because the setting lives outside the
repo.
N2. Accounts soft-deleted before the hard-delete change keep all their data,
and linked therapists can still read it
Where: backend/migrations/000037_account_hard_delete.up.sql (no purge of existing
is_deleted rows), backend/internal/repositories/users.go:175-187 (purge only when that
same identity signs in again), backend/internal/services/auth.go:67-71 (purge only when
someone registers that email), backend/internal/repositories/therapist.go:153-170
( ListClients ) and the other client queries, none of which filter is_deleted .
Failure scenario: Every user who tapped "Delete account" before 219dadd still has a
users row with is_deleted=true , plus their transcripts, analyses, therapy messages and
people. Deletion is now lazy and only happens if that person returns, so data for users who
never come back stays forever. If such a user had approved a therapist link, the link is still
active , and the therapist dashboard, brief and portal keep serving their journal summaries.
H1 is fixed for new deletions only.
Fix: Add a migration (or a one-off job run at API start, like the cipher upgrade): DELETE FROM
users WHERE is_deleted = true . The cascades already cover every user table. Then drop the
is_deleted / deleted_at columns and the lazy-purge branches, so the soft-delete state can't
come back.
--- page 2 ---
Medium
N3. Unconditional therapy status writes: a crisis stop can still be overwritten,
and replies are appended after a hard stop
Where: backend/internal/repositories/therapy.go:124-134 ( UpdateStatus … WHERE id = $1 ),
callers at backend/internal/services/therapy.go:138, 405, 436 . Also therapy.go:126-226 :
status is read once at the start of SendMessage and never re-checked before the reply is
stored.
Failure scenario: H10 added a status check in EndSession , but the write after it is
unconditional and runs after a multi-second TherapySummary call. The user sends a crisis
message, then taps End while it's pending:
1. EndSession reads active and starts summarising.
2. SendMessage hard-stops and writes crisis_detected .
3. EndSession writes completed .
The crisis status is lost, a crisis transcript is summarised, and the summary enters session
memory (ADR-016). The same check-then-write applies to the expiry update. Separately, a
normal message already past its status check keeps going after a concurrent hard stop, so
its AI reply is appended after the crisis resources.
Fix: Make transitions conditional: UPDATE … SET status=$2 … WHERE id=$1 AND status='active'
RETURNING status . If no row is updated, return ErrTherapyAlreadyEnded and discard the
summary. Re-check status='active' in the same transaction that inserts the assistant
message.
N4. Therapy turns time out on the client before the server finishes, and the
new retry duplicates turns
Where: mobile/src/api/client.ts:56 (global 30 s axios timeout, also used by
sendTherapyMessage ), mobile/app/therapy/session.tsx:323-345 ( setDraft(content) on any
error, then the user re-sends), backend/internal/services/therapy.go:126-270 (budget:
Whisper 30 s + Claude up to 60 s + TTS 20 s; no idempotency key).
Failure scenario: A slow voice or text turn takes 35 s. The client times out and restores
the draft, which invites a resend, while the server finishes and stores both messages. The
resend stores the turn again, makes a second Claude call, synthesises TTS again and
increments turn_count twice. If the message contained crisis language, the duplicate
counts as a second detection and hard-stops the session (ADR-014) for what was really a
single message. The commit title says "safe retries", but only typed journals got an
idempotency key.
Fix: Give therapy messages a client request_id , the same pattern as POST /entries/text .
Store it with a unique constraint and return the existing pair on retry. Set a per-call timeout
on sendTherapyMessage above the server budget (e.g. 120 s). On timeout, re-fetch the
session before offering a resend.
N5. Concurrent same-user replays of one purchase can extend a pass twice
Where: backend/internal/handlers/billing.go:220-275 . Record (insert or lookup),
UpgradePlan and MarkGranted are three separate statements.
--- page 3 ---
Failure scenario: The fix for H11 made same-user replays idempotent by checking
granted_at . Two requests for the same transaction can run concurrently. The launch-time
recoverUnfinishedPurchases and an upgrade-screen retry is a realistic pair. Request A
inserts. Request B sees Inserted=false, Granted=false (A hasn't marked yet) and "falls
through and grants". Both call UpgradePlan , and because the same plan now extends from
the current expiry (M14 fix), B adds another 30/365 days on top of A's. One payment buys
two passes.
Fix: Make the grant the claim. In one transaction: insert or select the payment row FOR
UPDATE , return early if granted_at is set, then UpgradePlan , then SET granted_at = NOW() ,
then commit. Or claim first with UPDATE payments SET granted_at = NOW() WHERE
transaction_id=$1 AND granted_at IS NULL RETURNING and grant only if a row came back,
rolling back on failure.
N6. Unverified emails from public local registration are still trusted
Where: backend/internal/handlers/router.go:100-101 ( /auth/register and /auth/login are
still mounted in production), backend/internal/handlers/b2b.go:51 ( sameEmailDomain ),
backend/internal/handlers/billing.go:210 ( SandboxAllowed(currentUser.Email) ),
backend/internal/repositories/users.go:175-200 (Supabase insert collides with a local row
on users.email UNIQUE ).
Failure scenario: POST /auth/register lets anyone claim any email with no verification.
Several new checks treat users.email as proof of identity:
1. B2B: register anyone@acme.com , then join Acme. The M12 domain gate is bypassed:
the attacker takes a seat and their entries feed the team dashboard.
2. Sandbox allowlist: if an allowlisted tester email hasn't signed in yet, register it
locally, buy a free TestFlight or sandbox product, and get a real plan. The H12 fix is
bypassed.
3. Squatting (M4, still open): pre-register victim@x.com . When the victim later signs
in with Supabase, the insert hits the unique email constraint and every request
returns 500.
Fix: In production, mount local auth only for the therapist portal. Better, move the portal to
Supabase and stop mounting local auth at all (C2 recommended this). Wherever email is
used for authorization, require a verified email: Supabase's email_verified , or a
users.email_verified column set only from verified sources. On a Supabase insert conflict
with an unverified local row, take the email over instead of failing.
N7. Storage objects outlive their rows
Where: No sweeper or lifecycle rule anywhere in backend/ (grep for
sweep|orphan|lifecycle finds only the note-photo cleanup). Sources of orphans:
backend/internal/handlers/entries.go:55 checks quota at register, after the upload.
backend/internal/services/tts.go:166 writes tts/{session}/{message}.mp3 and never
deletes it (M16, still open).
UserRepository.Delete cascades rows only.
mobile/src/services/upload.ts:46-90 : uploadedKey is local, so an offline-queue retry
presigns a new key.
Failure scenario: A free user at 10 entries this month records an eleventh. The audio
uploads, POST /entries returns 409, and the object stays in R2 forever (ADR-005). It's the
same for any upload whose registration keeps failing, a recording never transcribed before
--- page 4 ---
the DLQ, pending uploads and note photos of a user who deletes their account, and every
TTS reply ever synthesised.
Fix:
Check quota at POST /entries/presign .
Add R2 lifecycle rules: tts/ 1 day; audio/ and therapy/ e.g. 2 days. Live keys are
deleted much sooner, so this only catches orphans.
Delete the user's prefixes ( audio/{id}/ , therapy/{id}/ , notes/{therapistID}/ ) on
account deletion.
Persist uploadedKey in the offline-queue item.
N8. The production-safety guard is opt-in
Where: backend/internal/config/config.go:159 ( APP_ENV defaults to development ),
config.go:329-355 , fly.api.toml / fly.worker.toml [env] (no APP_ENV ),
LAUNCH_CHECKLIST item still unchecked.
Failure scenario: M9 added ProductionProblems , but it only runs when APP_ENV=production
is set by hand. Forget it on Railway, or on just one of the two services, and the API starts
with no store credentials. /billing/upgrade then grants paid plans for free, and notes are
encrypted with the JWT-derived key. That's the exact silent failure M9 was meant to
prevent.
Fix: Fail closed. Treat any environment that isn't explicitly development / test as production,
or require APP_ENV to be set (with no default) and refuse to start without it. Add APP_ENV =
"production" to both Fly configs.
N9. Under a queue backlog the reconciler marks entries failed that were never
attempted
Where: backend/internal/workers/entry_reconciler.go:16-20 ( stalePendingAge = 15m , 5-min
cycle), backend/internal/repositories/entries.go:285-325 ( ReclaimStuck increments
retry_count on every requeue and fails rows at maxRetries ).
Failure scenario: On a morning spike the queue is 20 minutes deep, with 4 workers per
replica and Whisper running a few minutes per long entry. Every entry still pending after 15
minutes is requeued, adding a duplicate job, and its retry_count goes up. After 3 cycles
(about 30 minutes) entries still waiting are marked failed ("processing was interrupted too
many times") without a worker ever touching them. The client stops polling on failed .
Fix: Don't count reconciler requeues of pending rows as attempts. Only processing rows
lost by a worker should count. Check queue depth ( LLEN ) before requeueing pending rows,
or record enqueued_at and requeue only rows that aren't in the queue. Log the queue depth
on every reconciler cycle.
N10. POST /entries still maps every service error to 400, checks quota before
idempotency, and duplicates under concurrency
Where: backend/internal/handlers/entries.go:55-66 ,
backend/internal/services/entries.go:63-70 ,
backend/migrations/000040_entries_audio_key_index.up.sql (a non-unique index).
Failure scenarios:
1. M5, part fixed: the handler still does c.Error(apierr.BadRequest(err.Error())) . A
storage or DB outage comes back as a 400 carrying the wrapped internal message,
--- page 5 ---
which the client treats as permanent ( upload.ts:95 ). CheckEntryQuota errors also
become 409 with internal text.
2. A free user's 10th registration succeeds but the response is lost. The retry hits the
quota check before the idempotency lookup and gets 409, even though the entry
exists. The text path handles this case ( entries.go:89-91 comment), but the audio
path doesn't.
3. Two concurrent registrations of the same key both miss GetByAudioKey , creating two
rows and two jobs. The first job deletes the audio, so the second fails and shows the
user a failed duplicate.
Fix: Return service apierr values unchanged and wrap everything else as InternalErr .
Run the idempotency lookup before the quota check. Make (user_id, audio_key) a UNIQUE
index and use INSERT … ON CONFLICT DO NOTHING RETURNING , the same as CreateText .
Low
N11. Typed journals in Hindi are analysed with the English prompt
Where: backend/internal/repositories/entries.go CreateText (stores no language ),
backend/internal/services/prompts.go:39-42 ( detectScriptLanguage returns "en" unless
Whisper said "hi" ).
Failure scenario: A Devanagari or Hinglish typed journal gets the English system prompt
and an English reflection, unlike the same words spoken.
Fix: Detect the script from the text when there's no Whisper language (the existing
DetectTextLanguage used by therapy), and store it on the entry.
N12. The follow-up crisis reply shows hotlines as plain text and the crisis state
isn't persisted
Where: mobile/app/followup/[id].tsx:186 , backend/internal/services/conversation.go:205
( IsCrisis is only set on the send response).
Failure scenario: The crisis resources render as an ordinary bubble with no tap-to-call
cards like the reflection screen has. On reopening, the conversation shows "You've reached
the end of this conversation", because is_crisis isn't returned by GetOrCreate . The parent
entry also stays non-crisis in mood stats.
Fix: Persist a crisis flag on the conversation and render the same hotline card component.
N13. Session-storage chunk writes aren't atomic
Where: mobile/src/lib/secureStorage.ts:41-48 .
Failure scenario: setItem deletes the chunk count first and rewrites it last. A concurrent
getItem , or an app kill in between, sees count 0, returns null and signs the user out. The
chunk size is 1800 UTF-16 code units, so a session with non-ASCII metadata (e.g. a Hindi
full_name ) can exceed SecureStore's 2048-byte Android limit.
Fix: Write chunks under a new generation key, then switch a single pointer key. Size chunks
by UTF-8 bytes (≤ 2000).
N14. The therapist-portal CSP allows 'unsafe-inline' scripts
Where: therapist-portal/firebase.json:38 .
--- page 6 ---
Failure scenario: script-src 'self' 'unsafe-inline' lets injected inline script run, so the
M18 CSP does little against XSS. The bearer token is still JS-readable, now with a 24 h
lifetime.
Fix: Use script hashes for Next's inline bootstrap (a hash list generated at build time) and
remove 'unsafe-inline' .
N15. Legacy v1 ciphertexts are accepted forever
Where: backend/pkg/crypto/envelope.go ( Open falls back to Decrypt ).
Failure scenario: After the startup re-seal, any remaining or restored v1 ciphertext can
still be moved between rows and decrypt. That's the swap attack M10 closed for v2.
Fix: Once UpgradeLegacyCiphertexts reports 0 rows in production, add a flag that rejects v1
in Open .
N16. A manual OTA can publish an unreviewed branch
Where: .github/workflows/deploy.yml:65-93 .
Failure scenario: workflow_dispatch runs on whatever ref the dispatcher picks and
publishes to the production channel, with no test gate. runtimeVersion is still static ( 1.0.1 ),
so native compatibility still depends on someone following the comment.
Fix: Add if: github.ref == 'refs/heads/main' and needs on a test job. Use runtimeVersion:
{"policy": "fingerprint"} .
N17. Remaining rule-hygiene items
mobile/app/therapist/add-session.tsx:89-90 : direct fetch for the photo PUT (L1, partly
fixed). It's a storage upload, not an API call, but upload.ts routes the same kind of request
through a service.
mobile/app/relationships.tsx:356 : as never cast on a route.
The Google Play verification doesn't bind a purchase to the account (no
obfuscatedExternalAccountId check), so a purchase token can be redeemed by whichever
account submits it first.
On iOS, recoverUnfinishedPurchases redeems to whoever is currently signed in. A purchase
left unfinished by account A is granted to account B after a sign-in switch.
Status of the first-pass findings
Status is from reading the fix on c4508bc and its tests. "Partial" means part of the finding
remains, cross-referenced to the new ID.
ID Finding Status
C1 
Model crisis flag
discarded
Fixed: ClaudeAnalysisOutput.Crisis , worker branches to
storeCrisis ( transcription.go:285-290 ), empty analysis is an
error
C2 
Register reactivates
deleted accounts 
Fixed (purges and creates fresh); local auth still public, see N6
C3 
B2B mood not k-
anonymous 
Fixed: TeamMinGroupSize = 5 , join-date filter (migration 000038)
--- page 7 ---
ID Finding Status
H1 Soft delete only Partial: new deletions are hard; legacy rows remain (N2)
H2 Crisis audio kept Fixed: audio deleted right after transcript persists
H3 
Audio deleted after
analysis 
Fixed: same; retries reuse the transcript
H4 
Follow-up not crisis-
screened 
Fixed: screened and closed on detection; display gap (N12)
H5 Stage 2 fails open 
Fixed: only "no" clears; rune-safe 6000-rune excerpt around the
match
H6 Keyword normalisation Fixed: quote folding, suicid , Hindi/Hinglish, kms
H7 Unreliable queue 
Fixed via reconciler, shutdown grace, detached ctx; backlog side
effect (N9)
H8 Retry ordering race Fixed: SetRetrying (pending) before enqueue; ctx-aware backoff
H9 3-turn cap race Fixed: conditional UPDATE claims the turn before Claude
H10 Therapy crisis races 
Partial: atomic warning claim and EndSession check; status writes
still unconditional (N3)
H11 
IAP recovery /
idempotency
Partial: launch recovery and same-user replay; concurrent double
grant (N5)
H12 Apple bundle / sandbox 
Fixed: bundle_id check, sandbox gated by allowlist; allowlist
trusts unverified email (N6)
H13 Deploy without tests Partial: Fly gated; Railway is not (N1)
H14 
Static runtimeVersion +
auto OTA 
Mitigated: OTA manual only (N16)
M1 JWT exp/aud/iss Fixed
M2 JWKS under lock Fixed: fetch outside the lock, 5 s timeout, 1/min refresh throttle
M3 Spoofable XFF Fixed: TRUSTED_PROXIES / trusted platform header
M4 
Upsert per request /
squatting 
Partial: read-first; squatting still possible (N6)
M5 Entry errors → 400 Partial: service returns apierr; handler re-wraps as 400 (N10)
M6 
Upload orphans /
duplicates
Partial: same-key retry within one upload; offline retry and
concurrency (N7, N10)
M7 
Offline queue drops
silently 
Fixed: failures surfaced, never dropped
M8 
Session in AsyncStorage /
no refresh
Fixed: keychain storage, AppState refresh, 401 retry; atomicity
(N13)
M9 No prod guard Partial: guard exists but is opt-in (N8)
M10 Derived key / no AAD 
Fixed: required in prod, AAD v2 with re-seal; v1 still accepted
(N15)
M11 
Classifier prompt outside
prompts.go
Fixed: buildCrisisClassifier* with the guard; de-escalation
guarded
M12 B2B self-join 
Partial: domain gate and advisory-lock seats; domain from
unverified email (N6)
M13 Therapy keys unbound Fixed: therapy/{user}/{session}/
M14 Pass time discarded Fixed for same plan; switch is warned in the UI
--- page 8 ---
ID Finding Status
M15 Share passcode Fixed: 8-char alphanumeric, lifetime cap, header transport
M16 TTS never deleted Open (N7)
M17 
Raw model output in
errors 
Fixed
M18 Portal token / CSP Partial: CSP and 24 h token; 'unsafe-inline' (N14)
L1 Mobile rule violations Mostly fixed: 0 hex literals, 0 any ; N17
L2 
Empty name in therapy
prompt 
Fixed
L3 
Check-ins on crisis
entries 
Fixed
L4 Upload proxy Fixed: prod guard refuses it
L5 
Login enumeration /
timing 
Fixed: dummy bcrypt
L6 
Committed demo
password 
Fixed
L7 Contract drift Fixed
L8 Robustness items Fixed (summarize now rate-limited per user)
Test gaps
1. N3: a concurrent EndSession and crisis SendMessage must leave crisis_detected , and a
reply after a hard stop must not be stored. Needs the integration tier.
2. N4: a therapy message retried with the same request_id returns the original pair and does
not count as a second crisis detection.
3. N5: two parallel /billing/upgrade calls with the same transaction extend expiry exactly
once.
4. N2: a migration test asserting no is_deleted=true rows survive, and therapist queries
never return a deleted user.
5. N9: the reconciler with a deep queue does not increase retry_count for unstarted entries.
6. N10: POST /entries retry at the quota boundary returns 201 with the existing entry, and
parallel registration of one key yields one row.
7. N11: a typed Devanagari journal selects the Hindi prompt.
What looks solid
The crisis pipeline is now layered and fail-safe end to end:
normalised Stage 1;
an inverted Stage 2 in which only an explicit "no" clears the entry;
the analysis model's own flag is honoured;
follow-up and therapy messages are screened;
crisis.go coverage is enforced at ≥ 90% in CI.
--- page 9 ---
Audio is deleted immediately after the transcript is persisted, and retries reuse the
transcript.
The 3-turn cap and the therapy de-escalation slot are claimed atomically in the database.
POST /entries/text has a well-designed idempotency key: deterministic per-user UUID,
quota checked after the retry lookup, and conflicting reuse rejected.
The JWT and JWKS handling, envelope encryption with row/column AAD, and the share-link
and login lockouts are all careful, and the new integration suite covers the earlier races.
Verification
Backend: go vet ./... clean; go test -race ./... passes (unit tier, Go 1.26.4).
Mobile: npx tsc --noEmit clean; npm test -- --runInBand 29 suites, 304 tests passed.
Earlier version of this report (first pass on f3da32c ): local tag review-2026-10-19-v1 .
--- page 10 ---
