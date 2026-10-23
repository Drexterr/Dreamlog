//go:build integration

package repositories

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// followupEntry seeds a completed, analysed entry. The tone, topic and summary
// carry a marker so a test can tell which entry fed an aggregate.
func followupEntry(t *testing.T, db *pgxpool.Pool, user uuid.UUID, at time.Time, mood int, marker string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	ctx := context.Background()
	if _, err := db.Exec(ctx, `INSERT INTO entries(id,user_id,audio_key,status,created_at,transcript) VALUES($1,$2,'fixture/followup/' || gen_random_uuid()::text,'completed',$3,'transcript')`, id, user, at); err != nil {
		t.Fatal(err)
	}
	tone := `[{"emotion":"` + marker + `-emotion","intensity":0.9}]`
	if _, err := db.Exec(ctx, `INSERT INTO entry_analysis(entry_id,mood_score,is_crisis,summary,reflection,emotional_tone,topics,key_quotes) VALUES($1,$2,false,$3,$4,$5::jsonb,ARRAY[$6],ARRAY['quote'])`,
		id, mood, marker+"-summary", marker+"-reflection", tone, marker+"-topic"); err != nil {
		t.Fatal(err)
	}
	return id
}

// closeFollowupInCrisis runs the production path: a follow-up conversation on
// the entry is closed by crisis screening.
func closeFollowupInCrisis(t *testing.T, db *pgxpool.Pool, entry, user uuid.UUID) {
	t.Helper()
	ctx := context.Background()
	r := NewConversationRepository(db)
	conv, err := r.GetOrCreate(ctx, entry, user)
	if err != nil {
		t.Fatal(err)
	}
	closed, err := r.Close(ctx, conv.ID)
	if err != nil || !closed.IsCrisis {
		t.Fatalf("close in crisis: %+v %v", closed, err)
	}
}

// Option C: an entry whose follow-up ended in crisis keeps its reflection but
// leaves every aggregate - mood stats, reviews, insight cards, chapters,
// share links, therapist briefs, connection insights and future context.
func TestIntegration_FollowupCrisis_ExcludedFromInsightsReflectionKept(t *testing.T) {
	db := regressionDB(t)
	ctx := context.Background()
	user := regressionUser(t, db)
	now := time.Now().UTC()
	at := now.Add(-2 * time.Hour)
	safe := followupEntry(t, db, user, at, 80, "safe")
	flagged := followupEntry(t, db, user, at.Add(time.Minute), 10, "flagged")
	closeFollowupInCrisis(t, db, flagged, user)

	var followup, crisis bool
	if err := db.QueryRow(ctx, `SELECT followup_crisis, is_crisis FROM entry_analysis WHERE entry_id=$1`, flagged).Scan(&followup, &crisis); err != nil || !followup || crisis {
		t.Fatalf("Close must set followup_crisis and leave is_crisis false: followup=%v crisis=%v %v", followup, crisis, err)
	}

	a := NewAnalysisRepository(db)
	// The reflection itself is untouched and still served.
	got, err := a.GetByEntryID(ctx, flagged)
	if err != nil || got == nil || got.Reflection != "flagged-reflection" || got.IsCrisis {
		t.Fatalf("reflection must stay visible: %+v %v", got, err)
	}

	noFlagged := func(name string, values []string) {
		t.Helper()
		for _, v := range values {
			if strings.Contains(v, "flagged") {
				t.Errorf("%s includes the follow-up crisis entry: %v", name, values)
			}
		}
	}

	days, err := a.MoodLast7Days(ctx, user)
	therapistDataOK(t, err)
	count := 0
	for _, d := range days {
		count += d.EntryCount
		if d.EntryCount > 0 && d.AvgMood != 80 {
			t.Errorf("weekly mood must only average the safe entry, got %d", d.AvgMood)
		}
	}
	if count != 1 {
		t.Errorf("weekly mood entry count: want 1, got %d", count)
	}

	if avg, err := a.MoodAvg30Days(ctx, user); err != nil || avg == nil || *avg != 80 {
		t.Errorf("30-day mood average: %v %v", avg, err)
	}
	if h, err := a.MoodHistory(ctx, user, 30); err != nil || h.EntryCount != 1 || h.AvgMood == nil || *h.AvgMood != 80 {
		t.Errorf("mood history: %+v %v", h, err)
	} else {
		noFlagged("mood history emotions", h.TopEmotions)
	}
	if p, err := a.EmotionPatterns(ctx, user, 30); err != nil || p.TotalEntries != 1 {
		t.Errorf("pattern radar: %+v %v", p, err)
	} else {
		for _, e := range p.Emotions {
			noFlagged("pattern radar", []string{e.Emotion})
		}
	}

	sums, err := a.RecentSummaries(ctx, user, 10)
	therapistDataOK(t, err)
	noFlagged("therapy context summaries", sums)
	emotions, err := a.TopEmotions(ctx, user, 10)
	therapistDataOK(t, err)
	noFlagged("therapy context emotions", emotions)
	topics, err := a.TopTopics(ctx, user, 10)
	therapistDataOK(t, err)
	noFlagged("therapy context topics", topics)

	week, err := a.GetWeekSummaries(ctx, user, now.Add(-7*24*time.Hour))
	therapistDataOK(t, err)
	if len(week) != 1 || week[0].Summary != "safe-summary" {
		t.Errorf("weekly review input: %+v", week)
	}
	year, err := a.GetYearSummaries(ctx, user, now.Add(-24*time.Hour), now.Add(time.Hour))
	therapistDataOK(t, err)
	if len(year) != 1 || year[0].Summary != "safe-summary" {
		t.Errorf("annual review input: %+v", year)
	}

	occ, err := a.CountTopicOccurrences(ctx, user, uuid.New(), []string{"flagged-topic", "safe-topic"}, now.Add(-30*24*time.Hour))
	therapistDataOK(t, err)
	if occ["flagged-topic"] != 0 || occ["safe-topic"] != 1 {
		t.Errorf("connection insight counts: %v", occ)
	}

	// PDF export: stats exclude the entry, the user's own entry list keeps it.
	exp, err := a.ExportData(ctx, user, now.Add(-24*time.Hour), now.Add(time.Hour))
	therapistDataOK(t, err)
	if exp.AvgMood == nil || *exp.AvgMood != 80 || exp.EntryCount != 1 {
		t.Errorf("export stats must exclude the entry: avg=%v count=%d", exp.AvgMood, exp.EntryCount)
	}
	noFlagged("export top emotions", exp.TopEmotions)
	listed := map[string]bool{}
	for _, e := range exp.Entries {
		listed[e.Summary] = true
	}
	if !listed["safe-summary"] || !listed["flagged-summary"] {
		t.Errorf("export entry list must keep both entries: %v", listed)
	}

	// Context for the next reflection never includes it (ADR-009).
	past, err := NewEntryRepository(db).ListCompletedBefore(ctx, user, now, 10)
	therapistDataOK(t, err)
	if len(past) != 1 || past[0].ID != safe {
		t.Errorf("reflection context must skip the entry: %+v", past)
	}

	card, err := NewInsightShareRepository(db).GetCardData(ctx, user, startOfWeekUTC(at))
	therapistDataOK(t, err)
	if card.EntryCount != 1 {
		t.Errorf("insight card entry count: %d", card.EntryCount)
	}
	noFlagged("insight card emotions", card.TopEmotions)

	view, err := NewShareRepository(db).ShareView(ctx, user)
	therapistDataOK(t, err)
	for _, s := range view.Summaries {
		noFlagged("therapist share link", []string{s.Summary})
	}
	if view.AvgMood == nil || *view.AvgMood != 80 {
		t.Errorf("share link average: %v", view.AvgMood)
	}

	// Linked therapist brief.
	therapistUser := regressionUser(t, db)
	tr := NewTherapistRepository(db)
	th, err := tr.Register(ctx, therapistUser, "QA", therapistUser.String()+"@example.test", "QA")
	therapistDataOK(t, err)
	if _, err := db.Exec(ctx, `INSERT INTO client_therapist_links(therapist_id,client_id,status) VALUES($1,$2,'active')`, th.ID, user); err != nil {
		t.Fatal(err)
	}
	if n, err := tr.ClientEntryCount(ctx, user); err != nil || n != 1 {
		t.Errorf("therapist entry count: %d %v", n, err)
	}
	recent, err := tr.ClientRecentEntries(ctx, user)
	therapistDataOK(t, err)
	for _, e := range recent {
		noFlagged("therapist recent entries", []string{e.Summary})
	}
	text, err := tr.ClientRecentSummariesText(ctx, user, now.Add(-24*time.Hour))
	therapistDataOK(t, err)
	noFlagged("therapist brief text", []string{text})
}

// Option C: a follow-up crisis entry is never resurfaced as a flashback.
func TestIntegration_FollowupCrisis_NotAFlashback(t *testing.T) {
	db := regressionDB(t)
	ctx := context.Background()
	user := regressionUser(t, db)
	entry := followupEntry(t, db, user, time.Now().UTC().Add(-30*24*time.Hour), 40, "flagged")
	a := NewAnalysisRepository(db)
	if f, err := a.Flashback(ctx, user); err != nil || f == nil || f.EntryID != entry {
		t.Fatalf("precondition: the month-old entry is a flashback: %+v %v", f, err)
	}
	closeFollowupInCrisis(t, db, entry, user)
	if f, err := a.Flashback(ctx, user); err != nil || f != nil {
		t.Fatalf("a follow-up crisis entry must not be resurfaced: %+v %v", f, err)
	}
}

// Option C: the team dashboard view drops follow-up crisis entries.
func TestIntegration_FollowupCrisis_ExcludedFromTeamMood(t *testing.T) {
	db := regressionDB(t)
	ctx := context.Background()
	company := uuid.New()
	if _, err := db.Exec(ctx, `INSERT INTO companies(id,name,slug,admin_email,seat_limit) VALUES($1,'QA',$2,'admin@example.test',50)`, company, "qa-"+company.String()[:8]); err != nil {
		t.Fatal(err)
	}
	var flagged []uuid.UUID
	for i := 0; i < 2; i++ {
		u := regressionUser(t, db)
		if _, err := db.Exec(ctx, `INSERT INTO company_members(company_id,user_id,role,joined_at) VALUES($1,$2,'member',NOW()-INTERVAL '1 day')`, company, u); err != nil {
			t.Fatal(err)
		}
		followupEntry(t, db, u, time.Now().UTC().Add(-time.Hour), 70, "safe")
		e := followupEntry(t, db, u, time.Now().UTC().Add(-time.Hour), 5, "flagged")
		closeFollowupInCrisis(t, db, e, u)
		flagged = append(flagged, e)
	}
	var entries int
	var avg int
	if err := db.QueryRow(ctx, `SELECT COALESCE(SUM(entry_count),0)::INT, COALESCE(MAX(avg_mood),0) FROM v_team_daily_mood WHERE company_id=$1`, company).Scan(&entries, &avg); err != nil {
		t.Fatal(err)
	}
	if entries != 2 || avg != 70 {
		t.Fatalf("team mood must count only safe entries: entries=%d avg=%d (flagged %v)", entries, avg, flagged)
	}
}

// Migration 000048 flags entries whose follow-up already ended in crisis and
// its down migration removes the column again.
func TestIntegration_Migration048_BackfillsAndReverses(t *testing.T) {
	db := regressionDBUpTo(t, "000047")
	ctx := context.Background()
	user := uuid.New()
	exec := func(sql string, args ...any) {
		t.Helper()
		if _, err := db.Exec(ctx, sql, args...); err != nil {
			t.Fatal(err)
		}
	}
	exec(`INSERT INTO users(id,supabase_id,email,name) VALUES($1,$2,$3,'QA')`, user, user.String(), user.String()+"@example.test")
	crisisEntry, calmEntry := uuid.New(), uuid.New()
	for _, e := range []uuid.UUID{crisisEntry, calmEntry} {
		exec(`INSERT INTO entries(id,user_id,audio_key,status) VALUES($1,$2,'fixture/m048/' || gen_random_uuid()::text,'completed')`, e, user)
		exec(`INSERT INTO entry_analysis(entry_id,mood_score,summary) VALUES($1,50,'s')`, e)
	}
	exec(`INSERT INTO conversations(entry_id,user_id,is_closed,is_crisis) VALUES($1,$3,true,true),($2,$3,false,false)`, crisisEntry, calmEntry, user)

	up, err := os.ReadFile("../../migrations/000048_followup_crisis.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	exec(string(up))
	for e, want := range map[uuid.UUID]bool{crisisEntry: true, calmEntry: false} {
		var got bool
		if err := db.QueryRow(ctx, `SELECT followup_crisis FROM entry_analysis WHERE entry_id=$1`, e).Scan(&got); err != nil || got != want {
			t.Fatalf("backfill %s: %v %v, want %v", e, got, err, want)
		}
	}
	var dailyEntries int
	if err := db.QueryRow(ctx, `SELECT COALESCE(SUM(entry_count),0)::INT FROM v_daily_mood WHERE user_id=$1`, user).Scan(&dailyEntries); err != nil || dailyEntries != 1 {
		t.Fatalf("v_daily_mood must exclude the flagged entry: %d %v", dailyEntries, err)
	}

	down, err := os.ReadFile("../../migrations/000048_followup_crisis.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	exec(string(down))
	var cols int
	if err := db.QueryRow(ctx, `SELECT COUNT(*) FROM information_schema.columns WHERE table_name='entry_analysis' AND column_name='followup_crisis'`).Scan(&cols); err != nil || cols != 0 {
		t.Fatalf("down migration must drop the column: %d %v", cols, err)
	}
	exec(string(up)) // and up again
}

func startOfWeekUTC(t time.Time) time.Time {
	t = t.UTC()
	offset := (int(t.Weekday()) + 6) % 7 // Monday-based weeks
	d := t.AddDate(0, 0, -offset)
	return time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, time.UTC)
}
