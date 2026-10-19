# Ode Backend 70-80% Checkpoint

Date: 2026-10-14

## Verified Result

- Backend statements: 4976/6533 (76.16715%).
- Named tests: 1070 passed, 0 failed, 0 skipped.
- Static check: whole-backend `go vet` passed.
- Change from valid baseline: 3422/6514 (52.53301%) to 76.16715%, +23.63415 percentage points and +1554 covered statements.
- Isolation: local PostgreSQL and Redis only; no live backend, deployment, phone, purchase, audio upload or user data mutation.

Evidence: `test-results/backend-target-verified.cover`, `test-results/backend-target-verified.jsonl`, and `test-results/backend-target-verified-vet.log`.

## Regressions Found and Fixed

1. Retrying a relationship mention inflated aggregate counters; counter updates now occur only when a new mention is inserted, including concurrent retry coverage.
2. PostgreSQL DATE values could not scan into life-chapter string fields; dates now use a typed scanner and stable API formatting.
3. Pending weekly and annual reviews failed on NULL narratives; nullable database values now map to the model's empty representation.
4. Nudge interval queries failed integer binding; interval arithmetic now uses typed PostgreSQL integers.
5. Therapist mood statistics swallowed database cancellation/query errors and mishandled absent averages; errors now propagate and nullable averages are supported.
6. The recent-user scheduler query used ambiguous joined columns; an EXISTS filter now preserves unique eligible users without ambiguous selection.

## Remaining to 80%

At the current 6533-statement denominator, 80% requires 5227 covered statements: 251 more. The largest remaining files include therapist-note handlers (161), API startup wiring (94), therapist-note services (83), worker startup wiring (80), Claude-note services (66), the re-engagement scheduler (47), storage service (47), S3 adapter (44), context builder (43), and transcription worker (38).

These are statement gaps, not promised gains. Provider error contracts, startup wiring, race detection, and real external integrations require separate evidence. Race mode is currently blocked by the missing CGO C compiler.
