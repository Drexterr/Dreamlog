# Backend White-Box and Black-Box 80% Plan

Date: 2026-10-15

White-box baseline: 4976/6533 statements (76.16715%). Target at least 80% on a fresh whole-backend integration-tag profile, retaining all packages.

Completed checkpoint: 5424/6547 statements (82.84711%), 1452 named passes, zero failures/skips, and whole-backend vet passed. Black-box manifest v3: 221/255 (86.66667%), 34 planned cases retained, 80/86 enabled endpoints exercised beyond authentication. See `ODE_DUAL_80_CHECKPOINT.md` for feature outcomes, reproduced fixes and remaining work.

Black-box target: at least 80% of the versioned API contract scenario manifest in scripts/regression/backend-blackbox-matrix.json. The manifest is frozen before the first scenario execution; planned/blocked cases remain in its denominator. This measures the stated API regression scope, not every possible input, mobile behavior, production load or provider accuracy.

## Work Batches

1. Parent: real loopback HTTP API harness, real PostgreSQL/Redis, account/entry/conversation/consent/notes/share/therapy contracts, endpoint inventory and reporting.
2. Independent agent: therapist-note handler/service success and error paths for source coverage.
3. Independent agent: workers, re-engagement, retry and cleanup paths.
4. Independent agent: storage service/S3 adapter against a local HTTP server.
5. Independent agent: black-box chapters/relationships/journeys/reviews/mood/insight/company contracts.
6. Consolidate full backend tests and vet, independently report source coverage and manifest scenario completion, preserve failed runs and reproduce/fix defects.

## Measurement Rules

- A passed status must match the expected contract; positive lifecycle and ownership scenarios additionally verify returned identity/data or absence of foreign data.
- Authentication checks for every protected route count as one boundary scenario. They cannot substitute for a feature's successful behavior.
- Black-box assertions observe HTTP only. Database access prepares isolated synthetic fixtures; production handlers/services/repositories are used.
- AI and storage boundaries use controlled local responses. Real provider behavior is separately planned and cannot earn completion from a stub.
- Each manifest case must map to a named test. A failed parent feature test prevents that feature's child checks from being certified in the summary.
- No phone is required. Local services use loopback ports 54331/63791 and unique qa_bb_* schemas, cleaned after each test.
- Source coverage and black-box scenario percentages have separate denominators and are never averaged.
