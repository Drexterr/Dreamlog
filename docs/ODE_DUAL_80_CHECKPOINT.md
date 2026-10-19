# Backend White-Box and Black-Box Checkpoint

Date: 2026-10-15

## Verified Outcome

| Metric | Result |
| --- | --- |
| Whole-backend statement coverage | 5424/6547 = 82.84711% |
| Previous passing statement coverage | 4976/6533 = 76.16715% |
| Gain | +6.67995 percentage points; +448 covered statements |
| Named Go tests, including parent tests/subtests | 1452 passed, 0 failed, 0 skipped |
| Black-box API manifest v3 | 221/255 = 86.66667%; 34 planned, 0 failed |
| Endpoint reach beyond authentication | 80/86 enabled endpoints = 93.02326% |
| Static checks | Whole-backend integration-tag go vet passed |
| Race detector | Not run; previously unavailable CGO C compiler remains a separate prerequisite |

Both independent 80% targets are achieved. Source denominator increased by 14 legitimate statements from reproduced fixes; no production packages were excluded. No existing file lost covered-statement count in the per-file comparison. Source coverage measures statements, not branch coverage. Black-box percentage measures this versioned scenario matrix, not every possible API input or whole-product behavior.

Evidence: `test-results/backend-dual80-final.cover`, `test-results/backend-dual80-final.jsonl`, `test-results/backend-dual80-vet-fixed.log`, `test-results/backend-dual80-blackbox-summary.json`, `test-results/backend-dual80-summary.json`.

## Feature Results

These are declared API contract scenarios; all assertions observe HTTP. SQL only prepares synthetic fixtures. Feature parents must also pass before their cases earn credit.

| Feature | Passed / declared executable cases |
| --- | ---: |
| Account/profile/terms | 10/10 |
| Billing plan/validation | 4/4 |
| Entries/history/search/analysis/check-in | 20/20 |
| Device registration | 3/3 |
| Follow-up conversations and ownership | 12/12 |
| Chapters | 15/15 |
| Relationships | 19/19 |
| Journeys and entry ownership | 16/16 |
| Weekly reviews | 6/6 |
| Annual reviews | 6/6 |
| Mood/streak/patterns | 9/9 |
| Insight cards/shares | 7/7 |
| Company membership/anonymized mood | 8/8 |
| Therapist workspace/notes | 39/39 |
| Sharing/passcode/revocation | 10/10 |
| Therapy session/text lifecycle | 10/10 |
| Public API contracts | 5/5 |
| Local authentication/validation | 6/6 |
| Security boundary/header/CORS checks | 3/3 |
| Therapist/client consent lifecycle | 13/13 |
| Total executed API scenarios | 221/221 |
| Remaining declared scenarios | 34 planned |

White-box expansion also covers notes encryption/error handling, re-engagement dispatch, pipeline retry/cleanup and local S3 SDK HTTP behavior. Notes handler/service coverage is 477/481 (99.17%); storage service 47/47 (100%), S3 adapter 43/44 (97.73%). Focused storage verification passed 19 named checks. Four independent agents completed their work and were closed after integration.

## Defects Reproduced and Fixed

1. Journey advancement accepted another user's entry. The repository now checks that entry and session owners match inside the write transaction, and the handler preserves the typed 404 response. The regression verifies both rejection and unchanged step state.
2. Conversation opening lacked entry ownership validation. A foreign caller could open an existing conversation or create a conversation against someone else's entry. The insert now selects only caller-owned entries and the conflict update also enforces conversation ownership; foreign requests return 404. Existing mismatched production records, if present, need a separately reviewed audit/remediation before release.
3. Profile updates saved unknown timezones. The handler now validates the timezone before any profile mutation and returns 400 for an unsupported/empty value.

Reproduction evidence is retained in `backend-blackbox-first.jsonl`, `backend-blackbox-second.jsonl`, and `backend-blackbox-conversation-reproduction.jsonl`. The first/second report checkpoints measured 109/225 (48.44%) and 159/225 (70.67%) under manifest v1. Manifest v2 added 18 consent/ownership cases; v3 added 12 positive/negative placeholders for the six untouched enabled endpoints. No case was removed to meet the target.

Other initial failures were harness issues: unsupported language key `hi` instead of `hindi`, incorrect therapy response status, shared auth rate bucket exhaustion, wrong device-token field, and PostgreSQL array fixture syntax. A vet failure caused by formatting a mutex-containing fake by value was corrected. All original runs remain available; final assertions were checked against actual API/model contracts.

## Remaining Work

Twelve planned cases cover success and negative behavior for the six untouched enabled endpoints: account deletion, entry flashback, streak freeze, PDF export, chapter summarization, and therapy audio presigning. Unit/integration evidence elsewhere does not replace these missing API cases.

Twenty-two more declared cases cover real Supabase/JWKS behavior, reference English/Hindi/noisy audio, provider timeout/malformed responses, queued OCR completion/failure, Apple/Google receipt verification/replay, dependency outages/recovery, concurrent API workload, PDF content validation, voice round-trip and notification delivery. Each exact case and requirement is retained in `scripts/regression/backend-blackbox-matrix.json` and the summary JSON. Existing controlled AI/storage responses do not complete these real-provider cases. Reference audio upload remains deferred; store sandbox configuration is unavailable.

Two storage behaviors were characterized but remain open follow-ups: `Exists` suppresses all HEAD errors as absence, and `PresignPut` ignores per-call content type/expiry. See `ODE_STORAGE_EXPANSION.md`. Neither was silently changed while broadening coverage.

## Repeat

```powershell
docker compose -p ode-regression -f scripts/regression/compose.yml up -d --wait
node scripts/regression/run-backend-dual80.mjs
```

The runner retains a fresh evidence directory and exits zero only when tests/vet pass and both percentages reach 80%. PostgreSQL/Redis use isolated loopback services, unique schemas/keys and synthetic accounts. No phone, mobile build, live backend mutation, provider charge, deployment, store purchase or personal-device access was performed.
