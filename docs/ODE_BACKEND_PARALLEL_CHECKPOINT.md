# Backend Parallel Export Checkpoint

Date: 2026-10-14. Scope: isolated PostgreSQL export data filtering and aggregation. No mobile, ADB, live API, deployment, purchases, audio uploads, account deletion, or secrets were used. Existing dirty changes were preserved. Parent-owned checkpoint documents and coverage ledger were not edited.

## Changes

Appended three top-level integration tests (seven named Go checks including four subtests) and a fixture helper to `backend/internal/repositories/regression_integration_test.go`. Its pre-existing tests and schema-isolation harness remain intact. No production code changed.

- Exact inclusive start/exclusive end boundaries for current and equally sized prior windows, including microsecond-adjacent fixtures.
- Owner isolation and exclusion of crisis entries, future/old records, and pending records without analysis.
- Exact daily counts, chronological summaries, weighted mood average, prior average and delta.
- Emotion intensity threshold at 0.5 versus 0.49, frequency ranking without tied counts, topic preservation, first quote selection and absent quotes.
- Empty user, unknown user, prior-only/current-only periods and nullable metrics.
- Malformed legacy emotion JSON shape does not discard valid summary/mood data; canceled query returns a wrapped cancellation error and no partial success.

Fixtures use fixed UTC dates and randomly named `qa_*` schemas through `regressionDB`. Only the loopback `ode_test` database and isolated Redis at 127.0.0.1:63791 were used. Audio keys are inert fixture labels, never uploaded. Test cleanup drops only each test's own generated schema, not accounts or shared tables.

## Steps And Outcomes

| Step | Outcome | Coverage |
| --- | --- | --- |
| Docker verification | Initial sandbox permission denial; approved read-only retry found PostgreSQL and Redis healthy | Not measured |
| Initial baseline attempt | Silent sandbox run interrupted; not counted as passing | Not measured |
| Diagnostic profile test | Failed before assertions: required `ODE_TEST_DATABASE_URL` omitted in diagnostic invocation | Not measured; retained in diagnostic log |
| Initial successful pre-change backend run | 836 named passes, zero failures/skips | Package percentages emitted, but unquoted PowerShell coverage argument produced no profile; exact total not claimed from this run |
| Initial focused export run | Seven named passes, zero failures/skips | Repository package reported 3.7%; no profile from unquoted argument |
| Corrected comparable baseline | 836 named passes, zero failures/skips; only new export tests excluded | Whole backend 3,355/6,514 = 51.50445194964691% |
| Corrected focused export run | Three top-level tests plus four subtests passed | Repository package 68/1,846 = 3.6836403033586134%; `ExportData` 56/64 = 87.5% |
| Fresh whole backend including integration | 843 named passes (813 top-level, 30 subtests), zero failures/skips | 3,422/6,514 = 52.53300583358919% |
| Backend vet including integration | Passed, exit 0, empty diagnostic log | Not measured; does not change coverage |
| Whitespace/format check | `gofmt` and `git diff --check` passed; existing CRLF warnings are not failures | Not measured |

Historical backend statement reference: **51.50445194964691%**. The fresh baseline exactly matches it. Final delta versus both historical and fresh baseline: **+1.0285538839422799 percentage points**. Denominator unchanged at 6,514 statements; 67 newly covered statements, zero lost, zero counter-identity/size mismatches. Focused coverage is not comparable to the whole-backend percentage and is not a regression in overall coverage. Counts include parent tests and subtests, not distinct product scenarios.

## Reproduction

Run from `backend/` in PowerShell. Fixture credentials below are the public local-only values in `scripts/regression/compose.yml`, not secrets. No `.env` files were read.

```powershell
$env:ODE_TEST_DATABASE_URL='postgres://ode_test:regression-local-only@127.0.0.1:54331/ode_test?sslmode=disable'
$env:ODE_TEST_REDIS_ADDR='127.0.0.1:63791'
go test -tags=integration ./... -count=1 -skip '^TestIntegration_Export' '-coverprofile=../test-results/backend-parallel-baseline.cover' -json
go test -tags=integration ./internal/repositories -run '^TestIntegration_Export' -count=1 '-coverprofile=../test-results/backend-parallel-focused.cover' -json
go test -tags=integration ./... -count=1 '-coverprofile=../test-results/backend-parallel-after.cover' -json
go vet -tags=integration ./...
go tool cover '-func=../test-results/backend-parallel-focused.cover'
```

Executed JSON streams were redirected to the corresponding separate evidence files; commands exited 0. The initial diagnostic used `go test -tags=integration ./internal/repositories -run '^TestIntegration_UserProfilePersistenceAndConstraints$' -count=1 -v` without fixture env and exited 1. Initial unquoted `-coverprofile=...` commands passed assertions but their profiles were unavailable; corrected reruns supersede only their coverage measurements, not their retained history. No existing `backend/coverage` file was changed.

## Evidence And Limits

`test-results/backend-parallel-results.json` is the step manifest. Separate `backend-parallel-*.jsonl` files contain named test outcomes, `.cover` files contain statement counters, and diagnostic/vet logs retain operational outcomes.

`ExportData` still has eight uncovered statements out of 64, mostly query/scan/row-error paths. Other export limits remain: plan/auth enforcement, PDF content/layout validation, very long/multilingual summaries, native share/export UI, and network recovery. These tests do not audit analyzed-but-noncompleted records or NULL mood/summary legacy records, nor validate true unrounded aggregate arithmetic. No full-scenario completion or 70/80/90 milestone is claimed. No production defect was reproduced in this chosen scope.

Files changed by this task: `backend/internal/repositories/regression_integration_test.go`, this document, and the eleven `test-results/backend-parallel-*` evidence files enumerated in the manifest. All mobile and parent-owned files remain outside this task's edits.
