# Workers Coverage Expansion

## Scope

Only new worker tests and `test-results/workers-expansion-*` evidence are owned by
this change. Existing dirty work, including production transcription changes,
is preserved. No production changes were made.

The parent whole-backend baseline is 76.17%, with an 80% target. This independent
task measures worker-package improvement, not a new whole-backend percentage.

## Added Regression Coverage

- Re-engagement eligibility query uses the configured 26-hour lapse threshold;
  only repository-returned candidates are dispatched. Actual SQL eligibility
  filtering remains the repository integration suite's responsibility.
- Re-engagement row metadata, curated stable message selection, all-device push
  payloads, and creation-before-push ordering.
- Empty candidate batches, fetch/create/token errors, no-device consumption,
  all-device and mixed-device push failures, and status-update failures.
- Immediate scheduler tick and cancellation with the original context.
- Transcription claim/fetch/download/empty-transcript/persistence/context errors,
  ordinary and crisis completion failures, and retry-enqueue failure.
- Optional connection-insight, nudge scheduling, audio cleanup, and person-store
  errors remain nonfatal; successful connection insights are persisted.
- OCR claim/state/download/read/empty-image/encryption/result-store errors retain
  retry artifacts. Failed readers are closed.
- Oversized OCR input fails permanently before the OCR client; raw-only output
  becomes a trimmed bullet, and cleanup failure does not undo completion.
- Malformed OCR payloads retain original bytes in the DLQ; retry-enqueue failure
  marks the session finally failed.
- Both worker loops stop after empty/error dequeues cancel their contexts.

All dependencies are in-memory fakes, except local envelope encryption. No
provider, network service, real queue, database, or push is used. Test commands
also disable module-network access with `GOPROXY=off` and `GOSUMDB=off`.

## Verification

Use the cached Go 1.26.4 binary because the default launcher is Go 1.26.1 while
the module requires 1.26.4. Set `GOMAXPROCS=2`, `GOTOOLCHAIN=local`, `GOPROXY=off`,
and `GOSUMDB=off`; run from `backend` with `-p 1`.

```powershell
go test -p 1 ./internal/workers -run '^TestExpansion' -count=1 -timeout=90s '-coverprofile=../test-results/workers-expansion-focused.out'
go test -p 1 ./internal/workers -skip '^TestExpansion' -count=1 -timeout=120s '-coverprofile=../test-results/workers-expansion-before.out'
go test -p 1 ./internal/workers -count=1 -timeout=120s '-coverprofile=../test-results/workers-expansion-after.out'
```

| Run | Result | Time | Worker coverage |
| --- | --- | --- | --- |
| Expansion tests only | PASS | 9.590s | 33.8% |
| Existing tests, excluding expansion | PASS | 13.084s | 442/648 statements, 68.21% |
| Combined worker suite | PASS | 26.029s | 552/648 statements, 85.19% |

The expansion adds **110 previously uncovered production statements**, exceeding
the requested 65-statement aim:

| Production file | Newly covered statements |
| --- | ---: |
| `reengagement_scheduler.go` | 46 |
| `transcription.go` | 32 |
| `note_ocr.go` | 32 |

Combined coverage reaches 100% for re-engagement `tick`/`pickMessage`,
transcription `handle`/`processJob`, and OCR `handle`/`processJob`/`contentTypeForKey`.
The remaining gaps include long-running/timer branches and other schedulers;
this change does not simulate minute-long ticker events.

Logs, profiles, and exact statement deltas are under
`test-results/workers-expansion-*`. Reproduce the comparison from the workspace
root with:

```powershell
./test-results/workers-expansion-compare.ps1
```

The machine-readable result is `test-results/workers-expansion-summary.json`.
No new whole-backend run was performed; the parent should merge these statements
into its final coverage run rather than treating worker coverage as backend-wide.

The before run excludes only the new `TestExpansion*` tests, leaving source and
other tests unchanged. The comparison rejects changed coverage block layouts,
so concurrent production changes cannot silently contaminate the delta.
