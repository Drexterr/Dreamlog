# Backend Analytics And Account Regression Checkpoint

Date: 2026-10-14

## Scope And Plan

Owned source: `backend/internal/repositories/analytics_account_regression_integration_test.go`.
Owned evidence: `test-results/backend-analytics-account-*`.
No production files, existing tests, shared ledgers, mobile files, deployment settings,
or other agents' files were edited. No Docker containers were started or stopped.

Read backend guidance, testing strategy, architecture, API contract, decisions,
`analysis.go`, `users.go`, the existing `regressionDB(t)` helper, and relevant migrations.
Use the existing real PostgreSQL service, isolated per-test migrated schemas, explicit
loopback fixture credentials, and only the `TestIntegration_AnalyticsAccount` prefix.
Implement positive, negative, ownership, boundary, empty, and canceled-query assertions;
measure file coverage against the supplied baseline without claiming whole-backend results.

## Completed Features

| Feature | Assertions | Final Result |
| --- | --- | --- |
| Mood aggregation | Journal plus completed therapy, daily ordering/counts, weighted average, previous-period delta, crisis/foreign/null-mood exclusion, empty/current-only/prior-only histories, malformed legacy tone | PASS |
| Pattern radar | Intensity 0.3 inclusion and 0.299 exclusion, frequency/average/normalization, mood boundaries 39/40/69/70, completion/time/ownership filters, invalid numeric intensity, empty response | PASS |
| Weekly/year review summaries | Inclusive start/exclusive end, exact annual timestamp vs weekly date, thresholds 0.5 vs 0.4, chronology, topics, empty summaries, unfinished/crisis/foreign exclusion, malformed tone, inverted window | PASS |
| Streaks/freezes | Empty/today/yesterday/expired streaks, longest separated streak, milestones, freeze bridge, exhaustion, unknown users, weekly idempotence/cap, rollback after actual PostgreSQL insert rejection | PASS |
| Account lifecycle | Local hash persistence, duplicate rejection, lookup paths, social upsert identity/name behavior, null/missing hash, rename, soft deletion, active lookup exclusion, retained audit state, repeat reactivation counters, missing users | PASS |
| Preferences/plans/counts | Every optional field, false/zero/empty-string values, partial/no-op update preservation, foreign-user nonmutation, persisted preference endpoints, schema-invalid goal/age/language/plan, paid plans and expiry, downgrade clearing, month start/end ownership | PASS |
| Analysis round trip | Full optional dream metadata, stable identity/read-back, rejected NaN JSON and invalid dream type, unchanged row after failed writes, clearing optional metadata, missing analysis | PASS |
| Auth counters | Threshold transition, reset-on-lock counter, 15-minute expiry bounds, persisted lock, below-threshold preservation, successful reset, unknown-email no-ops | PASS |
| Cancellation | 32 repository operations preserve `context.Canceled`; canceled mutations leave account name/deletion/plan unchanged | PASS |
| Context queries | 30-day journal/therapy average, recent summary selection/reversal/limits, ranked/truncated emotions/topics, malformed tone, case-insensitive distinct topic occurrences, self exclusion, inclusive since, empty/unknown results | PASS |
| Flashback | Month fallback, yearly priority, nearest yearly candidate, crisis/pending/foreign/out-of-window exclusion, empty result | PASS |

Final run: **11 top-level tests + 46 subtests passed, 0 failures, 0 skips**.
Final repository-package focused coverage: **25.8%**. This is not whole-backend coverage.

## Coverage Evidence

Baseline: `test-results/backend-parallel-after.cover`.
New focused profile: `test-results/backend-analytics-account-final.cover`.
Match blocks by exact `file:start,end`, weight by each block's statement count,
and use boolean coverage union (not addition of percentages or execution counts).

| File | Total Statements | Baseline Covered | Focused Covered | Baseline + Focused Union | Newly Covered | Union Coverage |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| `analysis.go` | 375 | 93 | 292 | 348 | 255 | 92.8% |
| `users.go` | 200 | 48 | 187 | 187 | 139 | 93.5% |
| Combined owned targets | 575 | 141 | 479 | 535 | **394** | 93.0435% |

The original eight-feature retest added 302 baseline-missing statements (163 analysis,
139 users); the round-trip/context/flashback additions brought the final gain to 394.
The parent owns actual whole-backend union and regression consolidation.

Function-level details: `test-results/backend-analytics-account-functions.txt`.
Exact file metrics: `test-results/backend-analytics-account-coverage.json`.

## Runs And Failures

1. `backend-analytics-account-first.log`: sandbox blocked loopback socket access before
   fixture setup; temporary Go coverage metadata rename was also denied. No successful
   behavioral validation and no usable profile from this run.
2. `backend-analytics-account-focused.log`: authorized local connection succeeded.
   Seven original feature groups passed; pattern score expected 0.13 but computed 0.12
   at a binary floating-point rounding tie. Corrected the test expectation, no production
   edit. The unquoted PowerShell coverage option did not yield the intended profile.
3. `backend-analytics-account-retest.log` / `.cover`: all original eight groups passed,
   19.5% focused package coverage. Quoted absolute coverage path used thereafter.
4. `backend-analytics-account-roundtrip.log` / `.cover`: optional-analysis round-trip
   group passed, 2.3% focused package coverage. This isolated percentage is not additive.
5. `backend-analytics-account-final.log` / `.cover`: all final 11 groups passed, 25.8%.
6. `backend-analytics-account-race.log`: invocation blocked before execution because
   `-race` requires CGO. `go env CGO_ENABLED CC` returned `0` and `gcc`; neither `gcc`
   nor `clang` was available on PATH. Race validation is **not claimed**.
7. `backend-analytics-account-vet.log`: package vet passed (no command output, exit 0;
   evidence file records the command and result).

No migration or fixture DDL failure occurred on the connected runs. The intentional
freeze-date CHECK rejection was asserted as a negative case and verified credit rollback.

## Reproduction Commands

From `backend/` in PowerShell, with local socket permissions:

```powershell
$env:ODE_TEST_DATABASE_URL='postgres://ode_test:regression-local-only@127.0.0.1:54331/ode_test?sslmode=disable'
$env:ODE_TEST_REDIS_ADDR='127.0.0.1:63791'
gofmt -w internal/repositories/analytics_account_regression_integration_test.go
go test -tags=integration ./internal/repositories -run '^TestIntegration_AnalyticsAccount' -count=1 -v '-coverprofile=C:/Users/bharat.jain/Desktop/per/dream/test-results/backend-analytics-account-final.cover'
go tool cover '-func=../test-results/backend-analytics-account-final.cover'
go vet -tags=integration ./internal/repositories
go test -race -tags=integration ./internal/repositories -run '^TestIntegration_AnalyticsAccount' -count=1 -v '-coverprofile=C:/Users/bharat.jain/Desktop/per/dream/test-results/backend-analytics-account-race.cover'
```

The final command requires a CGO-capable Go toolchain and C compiler; it did not run here.

## Limits

- `ExportData` intentionally untouched; existing baseline export tests supply its coverage.
- `ListWithRecentEntries` remains uncovered. Its unqualified selected columns in a join
  appear ambiguous by inspection; this was not executed or treated as a proven defect.
- Remaining analytics gaps are principally row scan/iteration errors and later-stage
  multi-query failures. No fake rows, mocked database, or generic invocation-only tests.
- `UseStreakFreeze` update-error path and duplicate-date/concurrent-spend behavior were
  not validated. Insert-error rollback, exhaustion, caps, and successful commits were.
- There is no role-update method in `users.go`; role behavior is outside this owned scope.
- Nudge-hour bounds are not database CHECK constraints in the migrations; tests assert
  valid hour endpoints, not a nonexistent database rejection contract.
- UTC-relative history/streak fixtures may be sensitive to crossing midnight during a run;
  current fixtures are placed at noon on earlier days wherever possible.
- CodeRabbit CLI was unavailable. No external review or full suite/race success is claimed.

## Changed Files

- `backend/internal/repositories/analytics_account_regression_integration_test.go`
- `docs/ODE_BACKEND_ANALYTICS_ACCOUNT_CHECKPOINT.md`
- `test-results/backend-analytics-account-*` logs, profiles, function table, and JSON metrics
  (evidence directory is ignored by Git; artifacts remain on disk for consolidation).
