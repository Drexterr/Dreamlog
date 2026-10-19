//go:build integration

package blackbox

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
)

func exploreSeed(t *testing.T, h *apiHarness, sql string, args ...any) {
	t.Helper()
	if _, err := h.db.Exec(context.Background(), sql, args...); err != nil {
		t.Fatalf("explore fixture: %v", err)
	}
}

func exploreEqual(t *testing.T, got map[string]any, key string, want any) {
	t.Helper()
	if got == nil {
		t.Fatalf("missing HTTP response while checking %s", key)
	}
	if got[key] != want {
		t.Errorf("%s: got %#v, want %#v", key, got[key], want)
	}
}

func exploreRows(t *testing.T, got map[string]any, key string, count int) []map[string]any {
	t.Helper()
	rows, ok := got[key].([]any)
	if !ok {
		t.Fatalf("%s: expected JSON array, got %#v", key, got[key])
	}
	if len(rows) != count {
		t.Fatalf("%s: got %d rows, want %d: %#v", key, len(rows), count, rows)
	}
	out := make([]map[string]any, len(rows))
	for i, row := range rows {
		var ok bool
		out[i], ok = row.(map[string]any)
		if !ok {
			t.Fatalf("%s row %d is not an object", key, i)
		}
	}
	return out
}

func exploreID(t *testing.T, got map[string]any) string {
	t.Helper()
	id, ok := got["id"].(string)
	if !ok {
		t.Fatalf("missing response id: %#v", got)
	}
	if _, err := uuid.Parse(id); err != nil {
		t.Fatal(err)
	}
	return id
}

func exploreEntry(t *testing.T, h *apiHarness, user uuid.UUID, at time.Time, mood int, crisis bool) uuid.UUID {
	t.Helper()
	id := uuid.New()
	exploreSeed(t, h, `INSERT INTO entries(id,user_id,audio_key,status,created_at) VALUES($1,$2,'explore-local','completed',$3)`, id, user, at)
	exploreSeed(t, h, `INSERT INTO entry_analysis(entry_id,summary,mood_score,is_crisis,emotional_tone) VALUES($1,'Explore synthetic summary',$2,$3,'[{"emotion":"joy","intensity":0.8},{"emotion":"noise","intensity":0.1}]')`, id, mood, crisis)
	return id
}

func TestBlackboxExploreChapters(t *testing.T) {
	h := newAPI(t)
	ch := h.request(t, "chapter_create", "POST", "/chapters", `{"title":"A local chapter","description":"Synthetic period","start_date":"2025-01-01","end_date":"2025-01-31","color":"#123456"}`, h.owner, 201)
	id := exploreID(t, ch)
	path := "/chapters/" + id
	got := h.request(t, "chapter_read_persisted", "GET", path, "", h.owner, 200)
	exploreEqual(t, got, "title", "A local chapter")
	exploreEqual(t, got, "user_id", h.ownerID.String())
	rows := exploreRows(t, h.request(t, "chapter_list_contains_created", "GET", "/chapters", "", h.owner, 200), "chapters", 1)
	exploreEqual(t, rows[0], "id", id)
	exploreRows(t, h.request(t, "chapter_other_list_excludes_owner", "GET", "/chapters", "", h.other, 200), "chapters", 0)
	h.request(t, "chapter_other_read_hidden", "GET", path, "", h.other, 404)
	h.request(t, "chapter_other_update_rejected", "PUT", path, `{"title":"Intrusion"}`, h.other, 404)
	h.request(t, "chapter_other_delete_noop", "DELETE", path, "", h.other, 204)
	exploreEqual(t, h.request(t, "chapter_owner_survives_foreign_delete", "GET", path, "", h.owner, 200), "title", "A local chapter")
	exploreEqual(t, h.request(t, "chapter_update", "PUT", path, `{"title":"Revised chapter","description":"Revised description"}`, h.owner, 200), "title", "Revised chapter")
	exploreEqual(t, h.request(t, "chapter_update_persisted", "GET", path, "", h.owner, 200), "description", "Revised description")
	exploreEntry(t, h, h.ownerID, time.Date(2025, 1, 10, 12, 0, 0, 0, time.UTC), 60, false)
	exploreEntry(t, h, h.ownerID, time.Date(2025, 1, 20, 12, 0, 0, 0, time.UTC), 80, false)
	exploreEntry(t, h, h.ownerID, time.Date(2025, 2, 1, 12, 0, 0, 0, time.UTC), 10, false)
	exploreEntry(t, h, h.otherID, time.Date(2025, 1, 10, 12, 0, 0, 0, time.UTC), 10, false)
	detail := h.request(t, "chapter_detail_aggregates_entries", "GET", path+"/detail", "", h.owner, 200)
	exploreEqual(t, detail, "entry_count", float64(2))
	exploreEqual(t, detail, "avg_mood", float64(70))
	h.request(t, "chapter_missing_title", "POST", "/chapters", `{"start_date":"2025-01-01"}`, h.owner, 400)
	h.request(t, "chapter_invalid_id", "GET", "/chapters/not-a-uuid", "", h.owner, 400)
	h.request(t, "chapter_delete", "DELETE", path, "", h.owner, 204)
	h.request(t, "chapter_deleted_read", "GET", path, "", h.owner, 404)
}

func TestBlackboxExploreRelationships(t *testing.T) {
	h := newAPI(t)
	target, source, foreign := uuid.New(), uuid.New(), uuid.New()
	exploreSeed(t, h, `INSERT INTO people(id,user_id,name,role) VALUES($1,$4,'Alex','friend'),($2,$4,'Alexander','other'),($3,$5,'Private person','family')`, target, source, foreign, h.ownerID, h.otherID)
	e1 := exploreEntry(t, h, h.ownerID, time.Now().UTC(), 70, false)
	e2 := exploreEntry(t, h, h.ownerID, time.Now().UTC(), 50, false)
	exploreSeed(t, h, `INSERT INTO person_mentions(person_id,entry_id,user_id,sentiment,context) VALUES($1,$3,$5,'positive','Shared walk'),($2,$3,$5,'neutral','Duplicate mention'),($2,$4,$5,'negative','Difficult call')`, target, source, e1, e2, h.ownerID)
	rows := exploreRows(t, h.request(t, "relationship_map_contains_seed", "GET", "/relationships", "", h.owner, 200), "people", 2)
	seen := map[any]bool{}
	for _, row := range rows {
		seen[row["id"]] = true
	}
	if !seen[target.String()] || !seen[source.String()] {
		t.Fatal("owner map omitted seeded people")
	}
	path := "/relationships/" + target.String()
	mentions := exploreRows(t, h.request(t, "relationship_detail_mentions", "GET", path, "", h.owner, 200), "mentions", 1)
	exploreEqual(t, mentions[0], "context", "Shared walk")
	h.request(t, "relationship_foreign_read_hidden", "GET", path, "", h.other, 404)
	h.request(t, "relationship_foreign_update_rejected", "PATCH", path, `{"name":"Intrusion"}`, h.other, 404)
	edited := h.request(t, "relationship_rename_and_role", "PATCH", path, `{"name":"  Alex Local  ","role":"family"}`, h.owner, 200)
	exploreEqual(t, edited, "name", "Alex Local")
	exploreEqual(t, edited, "role", "family")
	persisted := h.request(t, "relationship_edit_persisted", "GET", path, "", h.owner, 200)
	person, ok := persisted["person"].(map[string]any)
	if !ok {
		t.Fatal("person missing")
	}
	exploreEqual(t, person, "name", "Alex Local")
	h.request(t, "relationship_empty_edit", "PATCH", path, `{}`, h.owner, 400)
	h.request(t, "relationship_blank_name", "PATCH", path, `{"name":"  "}`, h.owner, 400)
	h.request(t, "relationship_invalid_role", "PATCH", path, `{"role":"manager"}`, h.owner, 400)
	h.request(t, "relationship_duplicate_name", "PATCH", path, `{"name":"alexander"}`, h.owner, 409)
	exploreEqual(t, h.request(t, "relationship_hide", "PATCH", path, `{"hidden":true}`, h.owner, 200), "hidden", true)
	hidden := exploreRows(t, h.request(t, "relationship_hidden_map_excludes", "GET", "/relationships", "", h.owner, 200), "people", 1)
	exploreEqual(t, hidden[0], "id", source.String())
	exploreEqual(t, h.request(t, "relationship_unhide", "PATCH", path, `{"hidden":false}`, h.owner, 200), "hidden", false)
	h.request(t, "relationship_self_merge", "POST", path+"/merge", fmt.Sprintf(`{"source_id":%q}`, target), h.owner, 400)
	h.request(t, "relationship_foreign_source_merge", "POST", path+"/merge", fmt.Sprintf(`{"source_id":%q}`, foreign), h.owner, 404)
	merged := h.request(t, "relationship_merge", "POST", path+"/merge", fmt.Sprintf(`{"source_id":%q}`, source), h.owner, 200)
	exploreEqual(t, merged, "mention_count", float64(2))
	exploreEqual(t, merged, "positive_count", float64(1))
	exploreEqual(t, merged, "negative_count", float64(1))
	mergedMentions := exploreRows(t, h.request(t, "relationship_merged_mentions", "GET", path, "", h.owner, 200), "mentions", 2)
	for _, m := range mergedMentions {
		exploreEqual(t, m, "person_id", target.String())
	}
	h.request(t, "relationship_source_removed", "GET", "/relationships/"+source.String(), "", h.owner, 404)
	other := exploreRows(t, h.request(t, "relationship_other_map_isolated", "GET", "/relationships", "", h.other, 200), "people", 1)
	exploreEqual(t, other[0], "id", foreign.String())
	exploreEqual(t, other[0], "name", "Private person")
}

func TestBlackboxExploreJourneys(t *testing.T) {
	h := newAPI(t)
	catalogue := h.request(t, "journey_public_catalogue", "GET", "/journeys", "", "", 200)
	templates, ok := catalogue["journeys"].([]any)
	if !ok || len(templates) == 0 {
		t.Fatal("empty journey catalogue")
	}
	found := false
	for _, v := range templates {
		m := v.(map[string]any)
		if m["id"] == "stress_relief" {
			found = true
			exploreEqual(t, m, "step_count", float64(3))
			if len(m["prompts"].([]any)) != 3 {
				t.Fatal("template prompts missing")
			}
		}
	}
	if !found {
		t.Fatal("stress relief missing")
	}
	h.request(t, "journey_unknown_template", "POST", "/journeys/not-a-template/start", `{}`, h.owner, 404)
	started := h.request(t, "journey_start", "POST", "/journeys/stress_relief/start", `{}`, h.owner, 201)
	path := "/journeys/sessions/" + exploreID(t, started)
	initial := h.request(t, "journey_read_initial_steps", "GET", path, "", h.owner, 200)
	exploreEqual(t, initial, "current_step", float64(0))
	steps := exploreRows(t, initial, "steps", 3)
	for _, s := range steps {
		exploreEqual(t, s, "completed", false)
		exploreEqual(t, s, "entry_id", nil)
	}
	sessions := exploreRows(t, h.request(t, "journey_owner_sessions", "GET", "/journeys/sessions", "", h.owner, 200), "sessions", 1)
	exploreEqual(t, sessions[0], "id", started["id"])
	exploreRows(t, h.request(t, "journey_other_sessions_empty", "GET", "/journeys/sessions", "", h.other, 200), "sessions", 0)
	h.request(t, "journey_foreign_read_hidden", "GET", path, "", h.other, 404)
	h.request(t, "journey_advance_requires_entry", "POST", path+"/advance", `{}`, h.owner, 400)
	e1 := exploreEntry(t, h, h.ownerID, time.Now().UTC(), 70, false)
	e2 := exploreEntry(t, h, h.ownerID, time.Now().UTC(), 60, false)
	e3 := exploreEntry(t, h, h.ownerID, time.Now().UTC(), 80, false)
	first := h.request(t, "journey_advance_first", "POST", path+"/advance", fmt.Sprintf(`{"entry_id":%q}`, e1), h.owner, 200)
	exploreEqual(t, first, "current_step", float64(1))
	exploreEqual(t, exploreRows(t, first, "steps", 3)[0], "entry_id", e1.String())
	second := h.request(t, "journey_advance_second", "POST", path+"/advance", fmt.Sprintf(`{"entry_id":%q}`, e2), h.owner, 200)
	exploreEqual(t, second, "current_step", float64(2))
	done := h.request(t, "journey_advance_complete", "POST", path+"/advance", fmt.Sprintf(`{"entry_id":%q}`, e3), h.owner, 200)
	exploreEqual(t, done, "status", "completed")
	saved := h.request(t, "journey_completed_persisted", "GET", path, "", h.owner, 200)
	exploreEqual(t, saved, "current_step", float64(3))
	for i, s := range exploreRows(t, saved, "steps", 3) {
		exploreEqual(t, s, "completed", true)
		exploreEqual(t, s, "entry_id", []string{e1.String(), e2.String(), e3.String()}[i])
	}
	h.request(t, "journey_completed_rejects_advance", "POST", path+"/advance", fmt.Sprintf(`{"entry_id":%q}`, e1), h.owner, 409)
	foreign := exploreEntry(t, h, h.otherID, time.Now().UTC(), 20, false)
	probe := h.request(t, "journey_foreign_entry_session", "POST", "/journeys/stress_relief/start", `{}`, h.owner, 201)
	probePath := "/journeys/sessions/" + exploreID(t, probe)
	h.request(t, "journey_rejects_foreign_entry", "POST", probePath+"/advance", fmt.Sprintf(`{"entry_id":%q}`, foreign), h.owner, 404)
	unchanged := h.request(t, "journey_foreign_entry_did_not_advance", "GET", probePath, "", h.owner, 200)
	exploreEqual(t, unchanged, "current_step", float64(0))
	exploreEqual(t, exploreRows(t, unchanged, "steps", 3)[0], "entry_id", nil)
}

func TestBlackboxExploreReviews(t *testing.T) {
	h := newAPI(t)
	h.request(t, "weekly_latest_empty", "GET", "/reviews/weekly/latest", "", h.owner, 404)
	h.request(t, "annual_latest_empty", "GET", "/reviews/annual/latest", "", h.owner, 404)
	weekly, annual := uuid.New(), uuid.New()
	exploreSeed(t, h, `INSERT INTO weekly_reviews(id,user_id,week_start,status,narrative,scheduled_at) VALUES($1,$2,'2025-01-05','completed','Owner weekly reflection',NOW()),(gen_random_uuid(),$2,'2025-01-12','pending','Pending private',NOW()),(gen_random_uuid(),$2,'2025-01-19','failed','Failed private',NOW())`, weekly, h.ownerID)
	exploreSeed(t, h, `INSERT INTO annual_reviews(id,user_id,year,status,narrative,scheduled_at) VALUES($1,$2,2024,'completed','Owner annual reflection',NOW()),(gen_random_uuid(),$2,2025,'pending','Pending private',NOW()),(gen_random_uuid(),$2,2026,'failed','Failed private',NOW())`, annual, h.ownerID)
	exploreSeed(t, h, `INSERT INTO weekly_reviews(user_id,week_start,status,narrative,scheduled_at) VALUES($1,'2024-12-29','completed','Older weekly reflection',NOW())`, h.ownerID)
	exploreSeed(t, h, `INSERT INTO annual_reviews(user_id,year,status,narrative,scheduled_at) VALUES($1,2023,'completed','Older annual reflection',NOW())`, h.ownerID)
	wr := exploreRows(t, h.request(t, "weekly_completed_only", "GET", "/reviews/weekly", "", h.owner, 200), "reviews", 2)
	exploreEqual(t, wr[0], "id", weekly.String())
	exploreEqual(t, wr[0], "narrative", "Owner weekly reflection")
	exploreEqual(t, wr[1], "narrative", "Older weekly reflection")
	exploreEqual(t, h.request(t, "weekly_latest_completed", "GET", "/reviews/weekly/latest", "", h.owner, 200), "id", weekly.String())
	ar := exploreRows(t, h.request(t, "annual_completed_only", "GET", "/reviews/annual", "", h.owner, 200), "reviews", 2)
	exploreEqual(t, ar[0], "id", annual.String())
	exploreEqual(t, ar[0], "narrative", "Owner annual reflection")
	exploreEqual(t, ar[1], "narrative", "Older annual reflection")
	exploreEqual(t, h.request(t, "annual_latest_completed", "GET", "/reviews/annual/latest", "", h.owner, 200), "id", annual.String())
	h.request(t, "weekly_free_plan_denied", "GET", "/reviews/weekly", "", h.other, 403)
	h.request(t, "annual_free_plan_denied", "GET", "/reviews/annual", "", h.other, 403)
	exploreSeed(t, h, `UPDATE users SET plan='plus',plan_expires_at=NULL WHERE id=$1`, h.otherID)
	exploreRows(t, h.request(t, "weekly_other_paid_list_isolated", "GET", "/reviews/weekly", "", h.other, 200), "reviews", 0)
	exploreRows(t, h.request(t, "annual_other_paid_list_isolated", "GET", "/reviews/annual", "", h.other, 200), "reviews", 0)
	h.request(t, "weekly_other_paid_latest_missing", "GET", "/reviews/weekly/latest", "", h.other, 404)
	h.request(t, "annual_other_paid_latest_missing", "GET", "/reviews/annual/latest", "", h.other, 404)
}

func TestBlackboxExploreMood(t *testing.T) {
	h := newAPI(t)
	now := time.Now().UTC()
	exploreEntry(t, h, h.ownerID, now.Add(-time.Hour), 60, false)
	exploreEntry(t, h, h.ownerID, now.Add(-time.Hour), 80, false)
	exploreEntry(t, h, h.ownerID, now.Add(-time.Hour), 1, true)
	days := exploreRows(t, h.request(t, "mood_weekly_aggregates_noncrisis", "GET", "/mood/weekly", "", h.owner, 200), "days", 1)
	exploreEqual(t, days[0], "avg_mood", float64(70))
	exploreEqual(t, days[0], "entry_count", float64(2))
	exploreRows(t, h.request(t, "mood_other_weekly_isolated", "GET", "/mood/weekly", "", h.other, 200), "days", 0)
	history := h.request(t, "mood_history_paid_aggregate", "GET", "/mood/history?range=30d", "", h.owner, 200)
	exploreEqual(t, history, "range", "30d")
	exploreEqual(t, history, "avg_mood", float64(70))
	h.request(t, "mood_history_invalid_range", "GET", "/mood/history?range=banana", "", h.owner, 400)
	h.request(t, "mood_history_free_denied", "GET", "/mood/history?range=30d", "", h.other, 403)
	radar := h.request(t, "mood_patterns_exclude_crisis_and_noise", "GET", "/mood/patterns?range=30d", "", h.owner, 200)
	emotions := exploreRows(t, radar, "emotions", 1)
	exploreEqual(t, emotions[0], "emotion", "joy")
	exploreEqual(t, emotions[0], "frequency", float64(2))
	exploreEqual(t, radar, "total_entries", float64(2))
	h.request(t, "mood_patterns_invalid_range", "GET", "/mood/patterns?range=banana", "", h.owner, 400)
	empty := h.request(t, "mood_other_patterns_isolated", "GET", "/mood/patterns", "", h.other, 200)
	exploreRows(t, empty, "emotions", 0)
	exploreEqual(t, empty, "total_entries", float64(0))
	streak := h.request(t, "mood_streak_completed_entries", "GET", "/mood/streak", "", h.owner, 200)
	exploreEqual(t, streak, "current_streak", float64(1))
	exploreEqual(t, streak, "longest_streak", float64(1))
}

func TestBlackboxExploreInsights(t *testing.T) {
	h := newAPI(t)
	initial := h.request(t, "insight_initial_card", "GET", "/insights/card", "", h.owner, 200)
	exploreEqual(t, initial, "share_count", float64(0))
	exploreEqual(t, initial, "entry_count", float64(0))
	shared := h.request(t, "insight_share_explicit_week", "POST", "/insights/share", `{"week_start":"2025-01-06"}`, h.owner, 201)
	exploreEqual(t, shared, "week_start", "2025-01-06")
	exploreEqual(t, shared, "total_shares", float64(1))
	defaultShare := h.request(t, "insight_share_default_week", "POST", "/insights/share", `{}`, h.owner, 201)
	exploreEqual(t, defaultShare, "week_start", initial["week_start"])
	exploreEqual(t, defaultShare, "total_shares", float64(2))
	exploreEqual(t, h.request(t, "insight_card_counts_shares", "GET", "/insights/card", "", h.owner, 200), "share_count", float64(2))
	exploreEqual(t, h.request(t, "insight_other_card_isolated", "GET", "/insights/card", "", h.other, 200), "share_count", float64(0))
	exploreEqual(t, h.request(t, "insight_other_share_own_counter", "POST", "/insights/share", `{}`, h.other, 201), "total_shares", float64(1))
	exploreEqual(t, h.request(t, "insight_owner_counter_unchanged", "GET", "/insights/card", "", h.owner, 200), "share_count", float64(2))
}

func TestBlackboxExploreB2B(t *testing.T) {
	h := newAPI(t)
	company, foreign := uuid.New(), uuid.New()
	slug := "explore-local"
	exploreSeed(t, h, `INSERT INTO companies(id,name,slug,admin_email,seat_limit) VALUES($1,'Explore team',$3,'admin@example.test',2),($2,'Foreign team','explore-foreign','foreign@example.test',1)`, company, foreign, slug)
	exploreSeed(t, h, `INSERT INTO company_members(company_id,user_id,role) VALUES($1,$2,'admin')`, company, h.ownerID)
	exploreSeed(t, h, `INSERT INTO company_members(company_id,user_id,role) VALUES($1,$2,'admin')`, foreign, h.otherID)
	path := "/b2b/companies/" + slug
	h.request(t, "b2b_nonmember_dashboard_denied", "GET", path+"/mood", "", h.other, 403)
	joined := h.request(t, "b2b_join", "POST", path+"/join", `{}`, h.other, 200)
	exploreEqual(t, joined, "company_id", company.String())
	exploreEqual(t, joined, "role", "member")
	h.request(t, "b2b_member_dashboard_denied", "GET", path+"/mood", "", h.other, 403)
	at := time.Now().UTC().AddDate(0, 0, -1)
	// Only entries made after joining count (C3): backdate membership so the
	// fixtures from yesterday are in scope.
	exploreSeed(t, h, `UPDATE company_members SET joined_at = NOW() - INTERVAL '7 days' WHERE company_id = $1`, company)
	exploreEntry(t, h, h.ownerID, at, 60, false)
	exploreEntry(t, h, h.otherID, at, 80, false)
	exploreEntry(t, h, h.ownerID, at, 1, true)
	summary := h.request(t, "b2b_admin_dashboard_aggregate", "GET", path+"/mood?range=30d", "", h.owner, 200)
	exploreEqual(t, summary, "total_members", float64(2))
	// k-anonymity (C3): a 2-person day is below the 5-member floor, so no mood
	// is exposed - per day or in the window aggregates - and no alert fires.
	exploreEqual(t, summary, "avg_mood", nil)
	exploreEqual(t, summary, "is_alerted", false)
	days := exploreRows(t, summary, "days", 1)
	exploreEqual(t, days[0], "active_members", float64(2))
	exploreEqual(t, days[0], "avg_mood", nil)
	exploreEqual(t, days[0], "entry_count", float64(0))
	for _, key := range []string{"user_id", "email", "name", "transcript", "summary"} {
		if _, ok := days[0][key]; ok {
			t.Errorf("anonymized team day leaked %s", key)
		}
	}
	exploreEqual(t, h.request(t, "b2b_admin_dashboard_90d", "GET", path+"/mood?range=90d", "", h.owner, 200), "avg_mood", nil)
	h.request(t, "b2b_join_full_company", "POST", "/b2b/companies/explore-foreign/join", `{}`, h.owner, 409)
	h.request(t, "b2b_unknown_company", "POST", "/b2b/companies/explore-missing/join", `{}`, h.other, 404)
	h.request(t, "b2b_other_company_admin_denied", "GET", "/b2b/companies/explore-foreign/mood", "", h.owner, 403)
}
