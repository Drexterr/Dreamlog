# Storage Coverage Expansion

Date: 2026-10-15

## Scope

Only new whitebox tests in `backend/pkg/storage/s3_test.go` and
`backend/internal/services/storage_test.go`; no production changes.
Existing dirty work is preserved.

All object requests target local `httptest` servers. Credentials are synthetic,
AWS metadata discovery is disabled, and SDK shared configuration/credentials
paths point to nonexistent files inside test temporary directories. Presigning
must not contact either the internal or public HTTP server. No live provider,
database, Redis, deployment, or user data is involved.

## Test Coverage Scope

- S3 upload/download/delete/exists success, exact bucket/key routing, authorization,
  content type/length, buffered body bytes, caller-owned download bodies, missing
  objects, and idempotent deletion.
- Upload reader failure with error identity and no HTTP request.
- Provider failures and pre-canceled contexts for object operations; wrapped
  cancellation identity and no requests from canceled operations.
- Internal/public endpoint selection, escaped object keys, signatures, configured
  upload expiry, per-call download expiry, invalid expiry, canceled credential
  retrieval, and AWS configuration-load errors.
- Service canonical audio keys, fresh UUIDs, therapy filename sanitization,
  traversal/control-character inputs, leading dots, randomized fallback names,
  128-byte filename truncation, exact server-generated keys, proxy query escaping,
  expiry conversion, IO delegation, and error wrappers.

## Behaviors For Parent Verification

Code inspection identified these behaviors, and the new tests encode them.
Runtime reproduction remains pending because test execution did not complete.
The assertions do not endorse or fix these behaviors.

1. `Client.Exists` returns `(false, nil)` for forbidden and canceled HEAD
   operations, not only missing objects. A caller cannot distinguish absence
   from a permission failure or cancellation.
2. `StorageService.PresignPut` ignores its `contentType` and `expiry` arguments.
   Supplying a 93-second expiry and `audio/custom` produces the configured
   12-second URL without content-type binding. The expiry and signed-header
   assertions make this behavior explicit.

## Verification

Both test files were formatted with `gofmt`.

Focused command (resource-limited, no broad backend rerun):

```powershell
$env:GOMAXPROCS = '2'
go test -p 1 ./pkg/storage ./internal/services -run TestStorageExpansion -count=1 -timeout=90s -coverprofile=../test-results/storage-expansion-focused.cover
```

The initial agent commands stalled before emitting test/build output and were
stopped. Parent verification subsequently passed 19 named checks with zero
failures/skips in `test-results/storage-expansion-parent.jsonl`. Exact source
coverage is being measured in the consolidated whole-backend profile.

Baseline evidence `test-results/backend-target-verified.cover` has zero coverage
for both target production files (44 adapter statements + 47 service statements).
The baseline whole backend is 4976/6533 = 76.16715%. A 70-statement increase would
produce 5046/6533 = 77.23863%, leaving 181 statements to reach 80%. This is only
arithmetic, not a measured result; the independent storage contribution cannot
alone achieve the whole-backend target.
