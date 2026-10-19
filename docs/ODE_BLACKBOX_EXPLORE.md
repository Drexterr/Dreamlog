# Explore Black-Box Scenarios

Plan frozen before the first scenario run. Scope: `backend/tests/blackbox/explore_test.go`.
Every name below is a literal `h.request` case name. Requests use the common
loopback router/database harness. SQL is fixture setup only; all behavioral
assertions use HTTP responses. Each domain gets an isolated harness/schema.

## Frozen Manifest

86 unique literal request names across seven domain tests. Manifest membership
and uniqueness checked statically; no scenario requests executed by this task.

Chapters: `chapter_create`, `chapter_read_persisted`, `chapter_list_contains_created`,
`chapter_other_list_excludes_owner`, `chapter_other_read_hidden`,
`chapter_other_update_rejected`, `chapter_other_delete_noop`,
`chapter_owner_survives_foreign_delete`, `chapter_update`, `chapter_update_persisted`,
`chapter_detail_aggregates_entries`, `chapter_missing_title`, `chapter_invalid_id`,
`chapter_delete`, `chapter_deleted_read`.

Relationships: `relationship_map_contains_seed`, `relationship_detail_mentions`,
`relationship_foreign_read_hidden`, `relationship_foreign_update_rejected`,
`relationship_rename_and_role`, `relationship_edit_persisted`,
`relationship_empty_edit`, `relationship_blank_name`, `relationship_invalid_role`,
`relationship_duplicate_name`, `relationship_hide`, `relationship_hidden_map_excludes`,
`relationship_unhide`, `relationship_self_merge`, `relationship_foreign_source_merge`,
`relationship_merge`, `relationship_merged_mentions`, `relationship_source_removed`,
`relationship_other_map_isolated`.

Journeys: `journey_public_catalogue`, `journey_unknown_template`, `journey_start`,
`journey_read_initial_steps`, `journey_owner_sessions`, `journey_other_sessions_empty`,
`journey_foreign_read_hidden`, `journey_advance_requires_entry`, `journey_advance_first`,
`journey_advance_second`, `journey_advance_complete`, `journey_completed_persisted`,
`journey_completed_rejects_advance`, `journey_foreign_entry_session`,
`journey_rejects_foreign_entry`, `journey_foreign_entry_did_not_advance`.

Reviews: `weekly_latest_empty`, `annual_latest_empty`, `weekly_completed_only`,
`weekly_latest_completed`, `annual_completed_only`, `annual_latest_completed`,
`weekly_free_plan_denied`, `annual_free_plan_denied`,
`weekly_other_paid_list_isolated`, `annual_other_paid_list_isolated`,
`weekly_other_paid_latest_missing`, `annual_other_paid_latest_missing`.

Mood: `mood_weekly_aggregates_noncrisis`, `mood_other_weekly_isolated`,
`mood_history_paid_aggregate`, `mood_history_invalid_range`, `mood_history_free_denied`,
`mood_patterns_exclude_crisis_and_noise`, `mood_patterns_invalid_range`,
`mood_other_patterns_isolated`, `mood_streak_completed_entries`.

Insights: `insight_initial_card`, `insight_share_explicit_week`,
`insight_share_default_week`, `insight_card_counts_shares`,
`insight_other_card_isolated`, `insight_other_share_own_counter`,
`insight_owner_counter_unchanged`.

B2B: `b2b_nonmember_dashboard_denied`, `b2b_join`, `b2b_member_dashboard_denied`,
`b2b_admin_dashboard_aggregate`, `b2b_admin_dashboard_90d`,
`b2b_join_full_company`, `b2b_unknown_company`, `b2b_other_company_admin_denied`.

## Assertions And Boundaries

- Chapters: persisted fields, exact list ownership, foreign mutations leave owner
  data intact, date-bounded entry count/mood, deletion and validation.
- Relationships: fixture IDs and mention contexts, trimmed rename and role,
  hide/unhide, duplicate-name conflict, merge deduplication and recomputed counts,
  foreign-source rejection and source removal.
- Journeys: public template prompts, persisted step-to-entry mapping, cursor and
  completed status, conflict after completion, session and entry ownership.
- Reviews: completed-only ordering, narrative/IDs, newer pending records excluded,
  paid/free gates and empty paid other-user results.
- Mood: numerical averages/counts, crisis exclusion, low-intensity emotion
  exclusion, cross-user isolation, range validation and completed-entry streak.
- Insights: explicit/default week and per-user share counters through card reads.
- B2B: join membership reflected in admin totals, member/admin distinction,
  seat-limit conflict, aggregate mood and anonymized daily response keys.

Potential production defect identified during inspection (not yet reproduced):
journey advancement checks session ownership but does not appear to validate the
submitted entry's owner. The foreign-entry case intentionally expects 404 and an
unchanged cursor. Do not weaken this expectation or change production code before
recording the first-run reproduction. HTTP follow-up runs even if rejection fails.

## Verification

Compile only: `go test -tags=integration ./tests/blackbox -run '^$'` from `backend`.
Compile-only command attempted locally but did not return output before it was
interrupted; compilation is not yet verified. `gofmt` completed successfully.
Scenario execution pending parent authorization:
`go test -tags=integration ./tests/blackbox -run TestBlackboxExplore`.
No production files, shared helpers, or shared ledgers are owned by this task.
