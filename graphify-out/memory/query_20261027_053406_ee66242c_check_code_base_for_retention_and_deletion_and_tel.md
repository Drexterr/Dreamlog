---
type: "query"
date: "2026-10-27T05:34:06.358006+00:00"
question: "check code base for retention and deletion and tell me what we are doing in the code"
contributor: "graphify"
outcome: "useful"
source_nodes: ["TranscriptionWorker", "LifecycleRule", "account_delete_test.go", "entries", "Audio Deletion Policy"]
---

# Q: check code base for retention and deletion and tell me what we are doing in the code

## Answer

Expanded from original query via graph vocab: [retention, delete, deletion, entries, transcript, audio, storage, purge, account, privacy, lifecycle]. Current implementation: TranscriptionWorker stores the transcript, then attempts object-storage deletion before crisis/AI analysis; deletion errors are non-fatal. LifecycleRule installs a two-day fallback for orphan audio/therapy objects, but lifecycle installation is fail-soft. Transcripts and analyses stay in PostgreSQL. There is no per-entry DELETE route and no implemented 180-day purge. DELETE /me hard-deletes the user row and cascades user-owned database data, then attempts deletion of audio/therapy/TTS/note prefixes; storage cleanup failure does not reverse account deletion. Therefore the privacy-policy claims of configurable 180-day auto-deletion and delete-anytime are not implemented. Focused services/workers/storage tests passed.

## Outcome

- Signal: useful

## Source Nodes

- TranscriptionWorker
- LifecycleRule
- account_delete_test.go
- entries
- Audio Deletion Policy