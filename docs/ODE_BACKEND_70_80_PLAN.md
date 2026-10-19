# Backend Regression Plan: 70-80%

Baseline: 3422/6514 statements (52.53301%) in backend-parallel-after.cover, with 843 passing named checks. Keep the same whole-backend denominator and integration build tag; never exclude hard-to-test files to improve the percentage. Counts include parent tests/subtests, not unique product scenarios.

## Targets and Gates

- 70%: at least 4560 covered statements, 1138 more than baseline.
- 80%: at least 5212 covered statements, 1790 more than baseline.
- Every feature batch needs meaningful positive/negative assertions, retained failures, a focused run and a fresh whole-backend checkpoint. Focused file percentages are not overall coverage.
- Final gate: all backend unit/integration tests pass, go vet passes, ownership/safety regressions pass; remaining untested paths are documented. Statement coverage alone does not certify branch, provider quality, security or whole-product scenarios.

## Feature Batches

| Order | Scope | Positive cases | Negative/regression cases | Status |
| --- | --- | --- | --- | --- |
| 1 | Accounts and analytics | Persisted fields, history/streak statistics, subscription/account transitions | Other-user isolation, empty data, date boundaries, duplicate/missing/cancelled operations | Focused scope passed; scheduler follow-up also fixed |
| 2 | Therapist and therapy data | Clients, sessions, notes, archive/restore, consent/session persistence | Ownership, absent links, invalid transitions, duplicates, missing rows, cancellation | Focused scope passed twice; error-handling defect fixed |
| 3 | Relationships and life chapters | Mentions, merge, rename/hide, chapter CRUD and scoped statistics | Retry idempotency, collisions, ownership, empty ranges, malformed/cancelled operations | Passed; concurrent retries passed three runs |
| 4 | Journeys, reviews, conversations, nudges | Ordered persisted state, completed steps/reviews, due-message scheduling | Missing/duplicate records, wrong-user access, bounded results, failure/cancellation | Focused scopes passed; company/insight checks also passed |
| 5 | Services, handlers and external adapters | Real local HTTP contracts and controlled mocked provider success | Auth/entitlement denial, storage/provider failures, safety fail-safe, retries and response errors | Router public/auth/rate-limit scope passed; wider provider/service work remains |
| 6 | Consolidation and remaining gaps | Fresh full integration suite and vet | Lost-coverage audit, race checks on touched concurrency, uncaught fixture failures | Completed at 76.17%; full suite and vet passed; race unavailable |

Priority comes from uncovered statement counts plus risk, not just easy lines. Largest initial gaps: analysis 282, therapist-notes handler 161, users 152, therapist-notes repository 145, life chapters 137, therapist repository 132, relationships 120, therapy repository 114 and nudges 104. These are maximum available headroom, not promised gains.

## Isolation and Ownership

Use existing healthy ode-regression containers on loopback ports 54331/63791, isolated ode_test database and a unique qa_* schema per test. No real backend mutations, phone, audio upload, purchases, deployments or credentials are needed. Do not mock PostgreSQL. Agents own separate new integration test files; parent owns remaining test files and shared ledger/docs. Production fixes require a reproduced defect and scoped regression; preserve all existing worktree changes.

## Progress

Planning/infra step: backend coverage remains 52.53301%; no new code is exercised yet. The default compose project initially collided with the existing test ports. Existing ode-regression services were confirmed healthy, and only the newly created duplicate regression containers were removed.

Step 1: focused relationships/chapters passed after two reproduced production fixes: duplicate mention retries inflated counters, and PostgreSQL DATE values could not be scanned into chapter strings. Focused file coverage 106/125 (84.8%) relationships and 132/139 (94.96%) chapters. First fresh whole-backend checkpoint backend-target-step1.cover: 4440/6521 (68.08772%), 959 named passes, zero failures, one known-defect test skipped. The denominator increased by seven real production statements; this is not an identical-denominator gain calculation. The skipped cancellation regression is being closed and does not earn completion credit.

Step 2 in progress: weekly/annual review lifecycle tests reproduced inability to scan NULL pending narratives. Reproduction evidence backend-reviews-isolated-reproduction.jsonl retains both failures; nullable scanners are being corrected while preserving empty-string API/model representation. Initial focused run was also build-blocked by an unfinished agent test file; that separate evidence is retained.

Step 2 completed focused checks: four journey/conversation/weekly/annual lifecycle tests passed in backend-review-journey-conversation-fixed.jsonl. Review nullable scanners are fixed. Last full measurement remains 68.08772%; no inferred overall gain is reported from this focused run.

Step 3 completed focused checks: two notification tests passed in backend-nudges-retry.jsonl after a reproduced lapse/expiry SQL integer parameter-binding defect was fixed with typed interval arithmetic. An initial invalid-type/platform expectation was corrected because the repository/schema do not promise that validation; missing-owner foreign-key cases now test the actual contract. No notifications were sent. Two company/insight tests passed in backend-company-insight-focused.jsonl. Last full measurement remains 68.08772% until consolidation.

Parallel analytics/account tests passed with 394 previously uncovered statements in their measured baseline/profile union; focus percentages are not substituted for whole-backend coverage. Therapist cancellation regression is unconditional and initially failed because MoodStats swallowed query errors; a narrowly authorized fix is in progress. CGO/compiler is unavailable for race mode and must be reported, not treated as a pass.

Step 4 whole-backend checkpoint: backend-target-final.cover measured 4965/6533 (75.99878%), 1068 named passes, zero failures/skips. Whole-backend vet passed. The 70% target is achieved; the denominator includes legitimate fixes, not exclusions. New router public/auth/rate-limit checks and all completed integration batches are included. Therapist MoodStats now propagates database errors and handles absent averages; its unconditional cancellation regression passes.

Step 5 final follow-up: a real scheduler-user query failed with SQLSTATE 42702 because shared user columns were ambiguous in a join. It is changed to an EXISTS filter, preserving unique eligible users and date/status behavior. Reproduction backend-recent-users-reproduction.jsonl is retained. Concurrent mention retries passed three focused runs without race mode.

Final verified checkpoint: backend-target-verified.cover measured 4976/6533 statements (76.16715%). All 1070 named tests passed with zero failures/skips, and whole-backend go vet passed. This is +23.63415 percentage points and 1554 covered statements versus the valid 3422/6514 baseline; the denominator grew by 19 legitimate production statements. The 70-80% objective is achieved without excluding packages. Reaching exactly 80% on the current denominator requires 5227 covered statements, or 251 additional statements. The largest remaining practical seams are therapist-note handlers/services, context and provider services, schedulers/workers, and startup wiring. Race mode remains unverified because a CGO C compiler is unavailable.
