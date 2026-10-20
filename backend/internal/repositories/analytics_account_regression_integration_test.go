//go:build integration

package repositories

import (
	"context"
	"errors"
	"math"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/dreamlog/backend/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func analyticsAccountExec(t *testing.T, db *pgxpool.Pool, sql string, args ...any) {
	t.Helper()
	if _, err := db.Exec(context.Background(), sql, args...); err != nil {
		t.Fatal(err)
	}
}

func analyticsAccountEntry(t *testing.T, db *pgxpool.Pool, user uuid.UUID, at time.Time, status string, mood any, crisis bool, summary, tone string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	analyticsAccountExec(t, db, `INSERT INTO entries(id,user_id,audio_key,status,created_at) VALUES($1,$2,'fixture/analytics-account',$3,$4)`, id, user, status, at)
	analyticsAccountExec(t, db, `INSERT INTO entry_analysis(entry_id,mood_score,is_crisis,summary,emotional_tone,topics) VALUES($1,$2,$3,$4,$5::jsonb,ARRAY['work','sleep'])`, id, mood, crisis, summary, tone)
	return id
}

func analyticsAccountTherapy(t *testing.T, db *pgxpool.Pool, user uuid.UUID, at time.Time, status string, mood any, tone string) {
	t.Helper()
	analyticsAccountExec(t, db, `INSERT INTO therapy_sessions(user_id,status,started_at,expires_at,ended_at,session_mood_score,session_emotional_tone) VALUES($1,$2,$3,$3::timestamptz+INTERVAL '1 hour',$3,$4,$5::jsonb)`, user, status, at, mood, tone)
}

func TestIntegration_AnalyticsAccountMoodAggregation(t *testing.T) {
	db := regressionDB(t)
	ctx := context.Background()
	owner, other := regressionUser(t, db), regressionUser(t, db)
	r := NewAnalysisRepository(db)
	day := time.Now().UTC().Truncate(24 * time.Hour).Add(-48 * time.Hour).Add(12 * time.Hour)
	calm := `[{"emotion":"calm","intensity":0.8},{"emotion":"quiet","intensity":0.49}]`
	analyticsAccountEntry(t, db, owner, day.Add(24*time.Hour), "completed", 90, false, "later", calm)
	analyticsAccountEntry(t, db, owner, day, "completed", 40, false, "first", calm)
	analyticsAccountTherapy(t, db, owner, day, "completed", 60, calm)
	analyticsAccountEntry(t, db, owner, day.Add(-8*24*time.Hour), "completed", 20, false, "prior", `[]`)
	analyticsAccountTherapy(t, db, owner, day.Add(-8*24*time.Hour), "completed", 40, `[]`)
	analyticsAccountEntry(t, db, owner, day, "completed", 1, true, "crisis", `[{"emotion":"excluded","intensity":1}]`)
	analyticsAccountEntry(t, db, other, day, "completed", 100, false, "foreign", `[{"emotion":"excluded","intensity":1}]`)
	analyticsAccountEntry(t, db, owner, day, "completed", nil, false, "no mood", `[]`)
	analyticsAccountTherapy(t, db, owner, day, "active", 100, `[{"emotion":"excluded","intensity":1}]`)
	analyticsAccountTherapy(t, db, owner, day, "crisis_detected", 1, `[]`)
	analyticsAccountTherapy(t, db, other, day, "completed", 100, `[]`)
	analyticsAccountTherapy(t, db, owner, day, "completed", nil, `[]`)
	want := []*models.DailyMood{{Day: day.Format("2006-01-02"), AvgMood: 50, EntryCount: 2}, {Day: day.Add(24 * time.Hour).Format("2006-01-02"), AvgMood: 90, EntryCount: 1}}
	weekly, err := r.MoodLast7Days(ctx, owner)
	if err != nil || !reflect.DeepEqual(weekly, want) {
		t.Fatalf("weekly=%+v want=%+v err=%v", weekly, want, err)
	}
	history, err := r.MoodHistory(ctx, owner, 7)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(history.Days, want) || history.EntryCount != 3 || history.AvgMood == nil || *history.AvgMood != 63 || history.PrevAvgMood == nil || *history.PrevAvgMood != 30 || history.MoodDelta == nil || *history.MoodDelta != 33 || !reflect.DeepEqual(history.TopEmotions, []string{"calm"}) {
		t.Fatalf("weighted history: %+v", history)
	}
	for _, user := range []uuid.UUID{regressionUser(t, db), uuid.New()} {
		empty, err := r.MoodHistory(ctx, user, 7)
		if err != nil || empty == nil || empty.Days == nil || len(empty.Days) != 0 || empty.EntryCount != 0 || empty.AvgMood != nil || empty.PrevAvgMood != nil || empty.MoodDelta != nil || len(empty.TopEmotions) != 0 {
			t.Fatalf("empty: %+v %v", empty, err)
		}
		if got, err := r.MoodLast7Days(ctx, user); err != nil || len(got) != 0 {
			t.Fatalf("empty weekly: %+v %v", got, err)
		}
	}
	priorOnly := regressionUser(t, db)
	analyticsAccountEntry(t, db, priorOnly, day.Add(-8*24*time.Hour), "completed", 35, false, "prior", `[]`)
	prior, err := r.MoodHistory(ctx, priorOnly, 7)
	if err != nil || prior.AvgMood != nil || prior.PrevAvgMood == nil || *prior.PrevAvgMood != 35 || prior.MoodDelta != nil {
		t.Fatalf("prior only: %+v %v", prior, err)
	}
	currentOnly := regressionUser(t, db)
	analyticsAccountEntry(t, db, currentOnly, day, "completed", 70, false, "legacy", `{"legacy":"shape"}`)
	current, err := r.MoodHistory(ctx, currentOnly, 7)
	if err != nil || current.AvgMood == nil || *current.AvgMood != 70 || current.PrevAvgMood != nil || current.MoodDelta != nil || len(current.TopEmotions) != 0 {
		t.Fatalf("current only malformed tone: %+v %v", current, err)
	}
}

func TestIntegration_AnalyticsAccountPatterns(t *testing.T) {
	db := regressionDB(t)
	ctx := context.Background()
	owner, other := regressionUser(t, db), regressionUser(t, db)
	r := NewAnalysisRepository(db)
	at := time.Now().UTC().Add(-time.Hour)
	analyticsAccountEntry(t, db, owner, at, "completed", 70, false, "high boundary", `[{"emotion":"calm","intensity":0.8},{"emotion":"edge","intensity":0.3},{"emotion":"noise","intensity":0.299}]`)
	analyticsAccountEntry(t, db, owner, at, "completed", 40, false, "neutral boundary", `[{"emotion":"calm","intensity":0.6}]`)
	analyticsAccountEntry(t, db, owner, at, "completed", 39, false, "low boundary", `[]`)
	analyticsAccountTherapy(t, db, owner, at, "completed", 69, `[{"emotion":"calm","intensity":1}]`)
	for _, state := range []string{"pending", "failed"} {
		analyticsAccountEntry(t, db, owner, at, state, 100, false, "unfinished", `[{"emotion":"excluded","intensity":1}]`)
	}
	analyticsAccountEntry(t, db, owner, at, "completed", 1, true, "crisis", `[{"emotion":"excluded","intensity":1}]`)
	analyticsAccountEntry(t, db, other, at, "completed", 100, false, "foreign", `[{"emotion":"excluded","intensity":1}]`)
	analyticsAccountEntry(t, db, owner, at.Add(-8*24*time.Hour), "completed", 100, false, "old", `[{"emotion":"excluded","intensity":1}]`)
	analyticsAccountTherapy(t, db, owner, at, "expired", 100, `[{"emotion":"excluded","intensity":1}]`)
	got, err := r.EmotionPatterns(ctx, owner, 7)
	if err != nil {
		t.Fatal(err)
	}
	if got.TotalEntries != 4 || got.MoodDistribution != (models.MoodDistribution{High: 1, Neutral: 2, Low: 1}) || len(got.Emotions) != 2 {
		t.Fatalf("patterns: %+v", got)
	}
	if p := got.Emotions[0]; p.Emotion != "calm" || p.Frequency != 3 || math.Abs(p.AvgIntensity-0.8) > 0.00001 || p.Score != 1 {
		t.Fatalf("calm: %+v", p)
	}
	if p := got.Emotions[1]; p.Emotion != "edge" || p.Frequency != 1 || p.AvgIntensity != 0.3 || p.Score != 0.12 {
		t.Fatalf("threshold/normalization: %+v", p)
	}
	empty, err := r.EmotionPatterns(ctx, uuid.New(), 7)
	if err != nil || empty.Emotions == nil || len(empty.Emotions) != 0 || empty.TotalEntries != 0 || empty.MoodDistribution != (models.MoodDistribution{}) {
		t.Fatalf("empty radar: %+v %v", empty, err)
	}
	bad := regressionUser(t, db)
	analyticsAccountEntry(t, db, bad, at, "completed", 50, false, "bad intensity", `[{"emotion":"bad","intensity":"invalid"}]`)
	if result, err := r.EmotionPatterns(ctx, bad, 7); err == nil || result != nil {
		t.Fatalf("invalid numeric intensity silently succeeded: %+v %v", result, err)
	}
}

func TestIntegration_AnalyticsAccountReviewWindows(t *testing.T) {
	db := regressionDB(t)
	ctx := context.Background()
	owner, other := regressionUser(t, db), regressionUser(t, db)
	r := NewAnalysisRepository(db)
	since := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	until := since.Add(7 * 24 * time.Hour)
	tone := `[{"emotion":"weekly","intensity":0.5},{"emotion":"annual","intensity":0.4},{"emotion":"noise","intensity":0.39}]`
	analyticsAccountEntry(t, db, owner, since.Add(time.Hour), "completed", 60, false, "second", `{"legacy":"shape"}`)
	analyticsAccountEntry(t, db, owner, since, "completed", 50, false, "inclusive start", tone)
	for _, row := range []struct {
		user    uuid.UUID
		at      time.Time
		status  string
		crisis  bool
		summary string
	}{
		{owner, since.Add(-time.Microsecond), "completed", false, "before"}, {owner, until, "completed", false, "exclusive end"},
		{owner, since, "completed", true, "crisis"}, {owner, since, "pending", false, "pending"}, {owner, since, "completed", false, ""}, {other, since, "completed", false, "foreign"},
	} {
		analyticsAccountEntry(t, db, row.user, row.at, row.status, 50, row.crisis, row.summary, tone)
	}
	week, err := r.GetWeekSummaries(ctx, owner, since)
	if err != nil || len(week) != 2 {
		t.Fatalf("week: %+v %v", week, err)
	}
	if week[0].Summary != "inclusive start" || week[0].MoodScore != 50 || !week[0].Date.Equal(since) || !reflect.DeepEqual(week[0].Emotions, []string{"weekly"}) || week[1].Summary != "second" || len(week[1].Emotions) != 0 {
		t.Fatalf("week ordering/threshold: %+v %+v", week[0], week[1])
	}
	year, err := r.GetYearSummaries(ctx, owner, since, until)
	if err != nil || len(year) != 2 {
		t.Fatalf("year: %+v %v", year, err)
	}
	if !year[1].Date.Equal(since.Add(time.Hour)) || year[0].Summary != "inclusive start" || year[0].MoodScore != 50 || !reflect.DeepEqual(year[0].Emotions, []string{"weekly", "annual"}) || !reflect.DeepEqual(year[0].Topics, []string{"work", "sleep"}) || len(year[1].Emotions) != 0 {
		t.Fatalf("annual exact timestamp/threshold: %+v %+v", year[0], year[1])
	}
	if got, err := r.GetWeekSummaries(ctx, uuid.New(), since); err != nil || len(got) != 0 {
		t.Fatalf("unknown week: %+v %v", got, err)
	}
	if got, err := r.GetYearSummaries(ctx, owner, until, since); err != nil || len(got) != 0 {
		t.Fatalf("inverted year: %+v %v", got, err)
	}
}

func TestIntegration_AnalyticsAccountStreaksAndFreezes(t *testing.T) {
	db := regressionDB(t)
	ctx := context.Background()
	r := NewAnalysisRepository(db)
	u := NewUserRepository(db)
	today := time.Now().UTC().Truncate(24 * time.Hour)
	for _, tc := range []struct {
		name                    string
		offsets                 []int
		current, longest, total int
	}{
		{"empty", nil, 0, 0, 0}, {"today with gaps", []int{0, 1, 2, 5, 6, 7, 8}, 3, 4, 7}, {"yesterday", []int{1, 2, 3}, 3, 3, 3}, {"expired", []int{2, 3, 4}, 0, 3, 3},
	} {
		t.Run(tc.name, func(t *testing.T) {
			owner := regressionUser(t, db)
			for _, offset := range tc.offsets {
				analyticsAccountEntry(t, db, owner, today.AddDate(0, 0, -offset).Add(time.Hour), "completed", 50, false, "day", `[]`)
			}
			analyticsAccountEntry(t, db, owner, today.Add(time.Hour), "pending", 50, false, "pending", `[]`)
			got, err := r.StreakInfo(ctx, owner)
			if err != nil || got.CurrentStreak != tc.current || got.LongestStreak != tc.longest || got.TotalDays != tc.total {
				t.Fatalf("streak: %+v %v", got, err)
			}
			if tc.total > 0 && got.NextMilestone != models.NextStreakMilestone(tc.current) {
				t.Fatalf("milestone: %+v", got)
			}
		})
	}
	owner := regressionUser(t, db)
	analyticsAccountExec(t, db, `UPDATE users SET streak_freeze_count=2 WHERE id=$1`, owner)
	analyticsAccountEntry(t, db, owner, today.Add(time.Hour), "completed", 50, false, "today", `[]`)
	analyticsAccountEntry(t, db, owner, today.AddDate(0, 0, -2).Add(time.Hour), "completed", 50, false, "two days", `[]`)
	if err := u.UseStreakFreeze(ctx, owner, today.AddDate(0, 0, -1)); err != nil {
		t.Fatal(err)
	}
	if count, err := u.StreakFreezeCount(ctx, owner); err != nil || count != 1 {
		t.Fatalf("remaining %d %v", count, err)
	}
	if got, err := r.StreakInfo(ctx, owner); err != nil || got.CurrentStreak != 3 || got.LongestStreak != 3 || got.TotalDays != 3 {
		t.Fatalf("freeze bridge: %+v %v", got, err)
	}
	if err := u.UseStreakFreeze(ctx, owner, today.AddDate(0, 0, -3)); err != nil {
		t.Fatal(err)
	}
	if err := u.UseStreakFreeze(ctx, owner, today.AddDate(0, 0, -4)); err == nil || !strings.Contains(err.Error(), "no streak freezes") {
		t.Fatalf("exhaustion: %v", err)
	}
	if count, err := u.StreakFreezeCount(ctx, owner); err != nil || count != 0 {
		t.Fatalf("exhausted count: %d %v", count, err)
	}
	if err := u.UseStreakFreeze(ctx, uuid.New(), today); err == nil {
		t.Fatal("unknown user consumed freeze")
	}
	if count, err := u.StreakFreezeCount(ctx, uuid.New()); err != nil || count != 0 {
		t.Fatalf("unknown count: %d %v", count, err)
	}
	capUser := regressionUser(t, db)
	analyticsAccountExec(t, db, `UPDATE users SET streak_freeze_count=3 WHERE id=$1`, capUser)
	if err := u.GrantWeeklyFreezes(ctx); err != nil {
		t.Fatal(err)
	}
	if err := u.GrantWeeklyFreezes(ctx); err != nil {
		t.Fatal(err)
	}
	if count, err := u.StreakFreezeCount(ctx, owner); err != nil || count != 1 {
		t.Fatalf("weekly idempotence: %d %v", count, err)
	}
	if count, err := u.StreakFreezeCount(ctx, capUser); err != nil || count != 3 {
		t.Fatalf("weekly cap: %d %v", count, err)
	}
	// An actual database rejection must roll the decrement back.
	analyticsAccountExec(t, db, `ALTER TABLE streak_freeze_days ADD CONSTRAINT fixture_reject_date CHECK(frozen_date <> DATE '2000-01-01')`)
	if err := u.UseStreakFreeze(ctx, owner, time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)); err == nil || !strings.Contains(err.Error(), "insert") {
		t.Fatalf("insert rejection: %v", err)
	}
	if count, err := u.StreakFreezeCount(ctx, owner); err != nil || count != 1 {
		t.Fatalf("failed insert lost credit: %d %v", count, err)
	}
}

func TestIntegration_AnalyticsAccountIdentityAndLifecycle(t *testing.T) {
	db := regressionDB(t)
	ctx := context.Background()
	r := NewUserRepository(db)
	user, err := r.CreateLocal(ctx, "local@example.test", "Original", "fixture-hash")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(user.SupabaseID, "local-") || user.Name != "Original" {
		t.Fatalf("local creation: %+v", user)
	}
	if hash, err := r.GetPasswordHash(ctx, user.Email); err != nil || hash != "fixture-hash" {
		t.Fatalf("hash %q %v", hash, err)
	}
	if _, err := r.CreateLocal(ctx, user.Email, "Duplicate", "another-hash"); err == nil {
		t.Fatal("duplicate email accepted")
	}
	for name, lookup := range map[string]func() (*models.User, error){
		"id":                      func() (*models.User, error) { return r.GetByID(ctx, user.ID) },
		"supabase":                func() (*models.User, error) { return r.GetBySupabaseID(ctx, user.SupabaseID) },
		"email":                   func() (*models.User, error) { return r.GetByEmail(ctx, user.Email) },
	} {
		t.Run(name, func(t *testing.T) {
			got, err := lookup()
			if err != nil || !reflect.DeepEqual(got, user) {
				t.Fatalf("lookup: %+v %v", got, err)
			}
		})
	}
	social, err := r.Upsert(ctx, "fixture-social", "social@example.test", "Social")
	if err != nil {
		t.Fatal(err)
	}
	updated, err := r.Upsert(ctx, social.SupabaseID, "social-new@example.test", "Do not replace name")
	if err != nil || updated.ID != social.ID || updated.Email != "social-new@example.test" || updated.Name != "Social" {
		t.Fatalf("upsert identity/name: %+v %v", updated, err)
	}
	if hash, err := r.GetPasswordHash(ctx, updated.Email); err != nil || hash != "" {
		t.Fatalf("null hash: %q %v", hash, err)
	}
	if _, err := r.Upsert(ctx, "another-social", user.Email, "Collision"); err == nil {
		t.Fatal("conflicting email accepted")
	}
	renamed, err := r.UpdateName(ctx, user.ID, "Renamed")
	if err != nil || renamed.Name != "Renamed" || renamed.Email != user.Email {
		t.Fatalf("rename: %+v %v", renamed, err)
	}
	if err := r.Delete(ctx, user.ID); err != nil {
		t.Fatal(err)
	}
	if got, err := r.GetByEmail(ctx, user.Email); err != nil || got != nil {
		t.Fatalf("deleted account is active: %+v %v", got, err)
	}
	// Account deletion is a hard delete: nothing is retained to be revived.
	if deleted, err := r.GetByEmail(ctx, user.Email); err != nil || deleted != nil {
		t.Fatalf("deleted account retained: %+v %v", deleted, err)
	}
	if err := r.Delete(ctx, user.ID); err == nil {
		t.Fatal("duplicate deletion succeeded")
	}
	// Signing back in with the same identity yields a fresh, empty account.
	fresh, err := r.Upsert(ctx, user.SupabaseID, user.Email, "Fresh")
	if err != nil || fresh.ID == user.ID {
		t.Fatalf("fresh account after deletion: %+v %v", fresh, err)
	}
	missing := uuid.New()
	for name, call := range map[string]func() (*models.User, error){
		"id":            func() (*models.User, error) { return r.GetByID(ctx, missing) },
		"supabase":      func() (*models.User, error) { return r.GetBySupabaseID(ctx, missing.String()) },
		"email":         func() (*models.User, error) { return r.GetByEmail(ctx, "absent@example.test") },
		"rename":        func() (*models.User, error) { return r.UpdateName(ctx, missing, "Missing") },
	} {
		t.Run("missing "+name, func(t *testing.T) {
			got, err := call()
			if err != nil || got != nil {
				t.Fatalf("missing: %+v %v", got, err)
			}
		})
	}
	if hash, err := r.GetPasswordHash(ctx, "absent@example.test"); err != nil || hash != "" {
		t.Fatalf("missing hash: %q %v", hash, err)
	}
	if err := r.Delete(ctx, missing); err == nil {
		t.Fatal("unknown deletion succeeded")
	}
}

func TestIntegration_AnalyticsAccountPreferencesPlansAndCounts(t *testing.T) {
	db := regressionDB(t)
	ctx := context.Background()
	r := NewUserRepository(db)
	owner, other := regressionUser(t, db), regressionUser(t, db)
	name, preferred, zone, goal, age, country, language := "Name", "Preferred", "Asia/Kolkata", "curious", "25_34", "IN", "hindi"
	hour, enabled, auto := 0, false, false
	got, err := r.UpdateProfile(ctx, owner, ProfileUpdate{Name: &name, PreferredName: &preferred, Timezone: &zone, FCMNudgeHour: &hour, NudgeEnabled: &enabled, NudgeAutoTime: &auto, Goal: &goal, AgeRange: &age, Country: &country, VoiceLanguage: &language})
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != name || got.PreferredName == nil || *got.PreferredName != preferred || got.Timezone != zone || got.FCMNudgeHour != 0 || got.NudgeEnabled || got.NudgeAutoTime || got.Goal == nil || *got.Goal != goal || got.AgeRange == nil || *got.AgeRange != age || got.Country == nil || *got.Country != country || got.VoiceLanguage != language {
		t.Fatalf("all optional fields: %+v", got)
	}
	persisted, err := r.GetByID(ctx, owner)
	if err != nil || !reflect.DeepEqual(got, persisted) {
		t.Fatalf("persisted profile: %+v %v", persisted, err)
	}
	empty := ""
	partial, err := r.UpdateProfile(ctx, owner, ProfileUpdate{PreferredName: &empty})
	if err != nil || partial.PreferredName == nil || *partial.PreferredName != "" || partial.Name != name || partial.Timezone != zone || partial.VoiceLanguage != language {
		t.Fatalf("partial empty-string update: %+v %v", partial, err)
	}
	if noop, err := r.UpdateProfile(ctx, owner, ProfileUpdate{}); err != nil || !reflect.DeepEqual(noop, partial) {
		t.Fatalf("empty update: %+v %v", noop, err)
	}
	if foreign, err := r.GetByID(ctx, other); err != nil || foreign.Name != "QA" || foreign.PreferredName != nil {
		t.Fatalf("cross-user mutation: %+v %v", foreign, err)
	}
	if pref, err := r.UpdatePreferences(ctx, owner, "UTC", 23); err != nil || pref.Timezone != "UTC" || pref.FCMNudgeHour != 23 || pref.Name != name {
		t.Fatalf("preferences: %+v %v", pref, err)
	}
	for _, invalid := range []ProfileUpdate{{Goal: analyticsAccountPtr("invalid")}, {AgeRange: analyticsAccountPtr("invalid")}, {VoiceLanguage: analyticsAccountPtr("invalid")}} {
		if _, err := r.UpdateProfile(ctx, owner, invalid); err == nil {
			t.Fatalf("invalid profile accepted: %+v", invalid)
		}
	}
	expiry := time.Now().UTC().Add(24 * time.Hour).Truncate(time.Microsecond)
	for _, plan := range []models.Plan{models.PlanPlus, models.PlanPro, models.PlanB2B} {
		got, err := r.UpdatePlan(ctx, owner, plan, &expiry)
		if err != nil || got.Plan != plan || got.PlanExpiresAt == nil || !got.PlanExpiresAt.Equal(expiry) {
			t.Fatalf("plan: %+v %v", got, err)
		}
	}
	if free, err := r.UpdatePlan(ctx, owner, models.PlanFree, nil); err != nil || free.Plan != models.PlanFree || free.PlanExpiresAt != nil {
		t.Fatalf("downgrade cleared expiry: %+v %v", free, err)
	}
	if _, err := r.UpdatePlan(ctx, owner, models.Plan("invalid"), nil); err == nil {
		t.Fatal("invalid plan accepted")
	}
	missing := uuid.New()
	if got, err := r.UpdatePlan(ctx, missing, models.PlanPlus, nil); err != nil || got != nil {
		t.Fatalf("missing plan: %+v %v", got, err)
	}
	if got, err := r.UpdateProfile(ctx, missing, ProfileUpdate{Name: &name}); err != nil || got != nil {
		t.Fatalf("missing profile: %+v %v", got, err)
	}
	if got, err := r.UpdatePreferences(ctx, missing, "UTC", 8); err != nil || got != nil {
		t.Fatalf("missing preferences: %+v %v", got, err)
	}
	var month time.Time
	if err := db.QueryRow(ctx, `SELECT DATE_TRUNC('month',NOW() AT TIME ZONE 'UTC') AT TIME ZONE 'UTC'`).Scan(&month); err != nil {
		t.Fatal(err)
	}
	for _, at := range []time.Time{month, month.Add(time.Hour), month.Add(-time.Microsecond), month.AddDate(0, 1, 0)} {
		analyticsAccountEntry(t, db, owner, at, "pending", 50, false, "count any status", `[]`)
	}
	analyticsAccountEntry(t, db, other, month, "completed", 50, false, "foreign", `[]`)
	if count, err := r.CountMonthlyEntries(ctx, owner); err != nil || count != 2 {
		t.Fatalf("month boundaries: %d %v", count, err)
	}
	if count, err := r.CountMonthlyEntries(ctx, missing); err != nil || count != 0 {
		t.Fatalf("missing month: %d %v", count, err)
	}
}

func analyticsAccountPtr[T any](v T) *T { return &v }

func TestIntegration_AnalyticsAccountAnalysisRoundTrip(t *testing.T) {
	db := regressionDB(t)
	ctx := context.Background()
	owner := regressionUser(t, db)
	id := analyticsAccountEntry(t, db, owner, time.Now().UTC(), "completed", 50, false, "old", `[]`)
	r := NewAnalysisRepository(db)
	mood := 75
	input := &models.EntryAnalysis{MoodScore: mood, EmotionalTone: []models.EmotionalTone{{Emotion: "hope", Intensity: 0.7}}, Topics: []string{"rest"}, KeyQuotes: []string{"A new start"}, Summary: "new", Reflection: "reflection", MorningNudge: "nudge", DreamSymbols: []string{"water"}, DreamType: "lucid", PsychologicalLens: "psychological", VedicLens: "vedic", ConnectionInsight: "connection"}
	got, err := r.Upsert(ctx, id, input)
	if err != nil {
		t.Fatal(err)
	}
	if got.EntryID != id || got.MoodScore != mood || !reflect.DeepEqual(got.EmotionalTone, input.EmotionalTone) || !reflect.DeepEqual(got.DreamSymbols, input.DreamSymbols) || got.DreamType != input.DreamType || got.PsychologicalLens != input.PsychologicalLens || got.VedicLens != input.VedicLens || got.ConnectionInsight != input.ConnectionInsight || !reflect.DeepEqual(got.Topics, input.Topics) || !reflect.DeepEqual(got.KeyQuotes, input.KeyQuotes) || got.Summary != input.Summary || got.Reflection != input.Reflection || got.MorningNudge != input.MorningNudge {
		t.Fatalf("analysis round trip: %+v", got)
	}
	loaded, err := r.GetByEntryID(ctx, id)
	if err != nil || !reflect.DeepEqual(got, loaded) {
		t.Fatalf("loaded analysis: %+v %v", loaded, err)
	}
	input.EmotionalTone[0].Intensity = math.NaN()
	if result, err := r.Upsert(ctx, id, input); err == nil || result != nil || !strings.Contains(err.Error(), "marshal tone") {
		t.Fatalf("non-JSON intensity: %+v %v", result, err)
	}
	input.EmotionalTone[0].Intensity = 0.7
	input.DreamType = "invalid"
	if result, err := r.Upsert(ctx, id, input); err == nil || result != nil {
		t.Fatalf("invalid dream type: %+v %v", result, err)
	}
	unchanged, err := r.GetByEntryID(ctx, id)
	if err != nil || !reflect.DeepEqual(unchanged, loaded) {
		t.Fatalf("failed replacement changed analysis: %+v %v", unchanged, err)
	}
	input.DreamType = ""
	input.PsychologicalLens = ""
	input.VedicLens = ""
	input.ConnectionInsight = ""
	input.DreamSymbols = nil
	cleared, err := r.Upsert(ctx, id, input)
	if err != nil || cleared.ID != loaded.ID || cleared.DreamType != "" || cleared.PsychologicalLens != "" || cleared.VedicLens != "" || cleared.ConnectionInsight != "" || len(cleared.DreamSymbols) != 0 {
		t.Fatalf("clear dream metadata: %+v %v", cleared, err)
	}
	if result, err := r.GetByEntryID(ctx, uuid.New()); err != nil || result != nil {
		t.Fatalf("missing analysis: %+v %v", result, err)
	}
}

func TestIntegration_AnalyticsAccountLoginCounters(t *testing.T) {
	db := regressionDB(t)
	ctx := context.Background()
	r := NewUserRepository(db)
	user, err := r.CreateLocal(ctx, "login@example.test", "Login", "fixture-hash")
	if err != nil {
		t.Fatal(err)
	}
	checkCounter := func(want int) {
		t.Helper()
		var count int
		if err := db.QueryRow(ctx, `SELECT failed_login_attempts FROM users WHERE id=$1`, user.ID).Scan(&count); err != nil || count != want {
			t.Fatalf("counter=%d want=%d err=%v", count, want, err)
		}
	}
	if lock, err := r.GetLoginLockedUntil(ctx, user.Email); err != nil || lock != nil {
		t.Fatalf("initial lock: %v %v", lock, err)
	}
	for attempt := 1; attempt < 3; attempt++ {
		if lock, err := r.RecordFailedLogin(ctx, user.Email, 3, 15*time.Minute); err != nil || lock != nil {
			t.Fatalf("premature lock: %v %v", lock, err)
		}
		checkCounter(attempt)
	}
	before := time.Now().UTC()
	lock, err := r.RecordFailedLogin(ctx, user.Email, 3, 15*time.Minute)
	if err != nil || lock == nil || lock.Before(before.Add(15*time.Minute-time.Second)) || lock.After(time.Now().UTC().Add(15*time.Minute+time.Second)) {
		t.Fatalf("lock duration: %v %v", lock, err)
	}
	checkCounter(0)
	if persisted, err := r.GetLoginLockedUntil(ctx, user.Email); err != nil || persisted == nil || !persisted.Equal(*lock) {
		t.Fatalf("persisted lock: %v %v", persisted, err)
	}
	if same, err := r.RecordFailedLogin(ctx, user.Email, 3, 15*time.Minute); err != nil || same == nil || !same.Equal(*lock) {
		t.Fatalf("below threshold changed existing lock: %v %v", same, err)
	}
	checkCounter(1)
	if err := r.ResetLoginAttempts(ctx, user.Email); err != nil {
		t.Fatal(err)
	}
	checkCounter(0)
	if lock, err := r.GetLoginLockedUntil(ctx, user.Email); err != nil || lock != nil {
		t.Fatalf("reset lock: %v %v", lock, err)
	}
	if lock, err := r.RecordFailedLogin(ctx, "unknown@example.test", 3, time.Minute); err != nil || lock != nil {
		t.Fatalf("unknown lock: %v %v", lock, err)
	}
	if lock, err := r.GetLoginLockedUntil(ctx, "unknown@example.test"); err != nil || lock != nil {
		t.Fatalf("unknown lookup: %v %v", lock, err)
	}
	if err := r.ResetLoginAttempts(ctx, "unknown@example.test"); err != nil {
		t.Fatal(err)
	}
	checkCounter(0)
}

func TestIntegration_AnalyticsAccountCancellation(t *testing.T) {
	db := regressionDB(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	user := regressionUser(t, db)
	r := NewUserRepository(db)
	a := NewAnalysisRepository(db)
	now := time.Now().UTC()
	calls := map[string]func() error{
		"30 day average":   func() error { _, err := a.MoodAvg30Days(ctx, user); return err },
		"recent summaries": func() error { _, err := a.RecentSummaries(ctx, user, 3); return err },
		"top emotions":     func() error { _, err := a.TopEmotions(ctx, user, 3); return err },
		"top topics":       func() error { _, err := a.TopTopics(ctx, user, 3); return err },
		"topic counts": func() error {
			_, err := a.CountTopicOccurrences(ctx, user, uuid.New(), []string{"work"}, now)
			return err
		},
		"flashback":     func() error { _, err := a.Flashback(ctx, user); return err },
		"mood weekly":   func() error { _, err := a.MoodLast7Days(ctx, user); return err },
		"mood history":  func() error { _, err := a.MoodHistory(ctx, user, 7); return err },
		"patterns":      func() error { _, err := a.EmotionPatterns(ctx, user, 7); return err },
		"streak":        func() error { _, err := a.StreakInfo(ctx, user); return err },
		"week":          func() error { _, err := a.GetWeekSummaries(ctx, user, now); return err },
		"year":          func() error { _, err := a.GetYearSummaries(ctx, user, now, now.AddDate(1, 0, 0)); return err },
		"id":            func() error { _, err := r.GetByID(ctx, user); return err },
		"supabase":      func() error { _, err := r.GetBySupabaseID(ctx, user.String()); return err },
		"email":         func() error { _, err := r.GetByEmail(ctx, "x"); return err },
		"upsert":        func() error { _, err := r.Upsert(ctx, "x", "x@example.test", "X"); return err },
		"create":        func() error { _, err := r.CreateLocal(ctx, "x@example.test", "X", "hash"); return err },
		"rename":        func() error { _, err := r.UpdateName(ctx, user, "X"); return err },
		"profile": func() error {
			_, err := r.UpdateProfile(ctx, user, ProfileUpdate{Name: analyticsAccountPtr("X")})
			return err
		},
		"preferences":   func() error { _, err := r.UpdatePreferences(ctx, user, "UTC", 8); return err },
		"plan":          func() error { _, err := r.UpdatePlan(ctx, user, models.PlanPlus, nil); return err },
		"hash":          func() error { _, err := r.GetPasswordHash(ctx, "x"); return err },
		"login lock":    func() error { _, err := r.GetLoginLockedUntil(ctx, "x"); return err },
		"failed login":  func() error { _, err := r.RecordFailedLogin(ctx, "x", 3, time.Minute); return err },
		"reset login":   func() error { return r.ResetLoginAttempts(ctx, "x") },
		"freeze count":  func() error { _, err := r.StreakFreezeCount(ctx, user); return err },
		"use freeze":    func() error { return r.UseStreakFreeze(ctx, user, now) },
		"grant freezes": func() error { return r.GrantWeeklyFreezes(ctx) },
		"delete":        func() error { return r.Delete(ctx, user) },
		"monthly count": func() error { _, err := r.CountMonthlyEntries(ctx, user); return err },
	}
	for name, call := range calls {
		t.Run(name, func(t *testing.T) {
			if err := call(); !errors.Is(err, context.Canceled) {
				t.Fatalf("cancellation lost: %v", err)
			}
		})
	}
	if got, err := r.GetByID(context.Background(), user); err != nil || got.Name != "QA" || got.Plan != models.PlanFree {
		t.Fatalf("canceled writes changed user: %+v %v", got, err)
	}
}

func TestIntegration_AnalyticsAccountContextQueries(t *testing.T) {
	db := regressionDB(t)
	ctx := context.Background()
	owner, other := regressionUser(t, db), regressionUser(t, db)
	r := NewAnalysisRepository(db)
	at := time.Now().UTC().Add(-24 * time.Hour)
	first := analyticsAccountEntry(t, db, owner, at, "completed", 20, false, "oldest", `[{"emotion":"calm","intensity":0.5},{"emotion":"noise","intensity":0.49}]`)
	second := analyticsAccountEntry(t, db, owner, at.Add(time.Hour), "completed", 40, false, "middle", `[{"emotion":"calm","intensity":0.8},{"emotion":"hope","intensity":0.8}]`)
	last := analyticsAccountEntry(t, db, owner, at.Add(2*time.Hour), "completed", 60, false, "newest", `{"legacy":"shape"}`)
	analyticsAccountExec(t, db, `UPDATE entry_analysis SET topics=ARRAY['WORK','work','Sleep'] WHERE entry_id=$1`, first)
	analyticsAccountExec(t, db, `UPDATE entry_analysis SET topics=ARRAY['work','focus'] WHERE entry_id=$1`, second)
	analyticsAccountExec(t, db, `UPDATE entry_analysis SET topics=ARRAY['work'] WHERE entry_id=$1`, last)
	analyticsAccountTherapy(t, db, owner, at, "completed", 80, `[{"emotion":"calm","intensity":0.9},{"emotion":"hope","intensity":0.7},{"emotion":"joy","intensity":0.9}]`)
	analyticsAccountExec(t, db, `UPDATE therapy_sessions SET session_topics=ARRAY['work','focus'] WHERE user_id=$1`, owner)
	analyticsAccountEntry(t, db, owner, at, "pending", 100, false, "pending", `[{"emotion":"excluded","intensity":1}]`)
	analyticsAccountEntry(t, db, owner, at, "completed", 100, true, "crisis", `[{"emotion":"excluded","intensity":1}]`)
	analyticsAccountEntry(t, db, other, at, "completed", 100, false, "foreign", `[{"emotion":"excluded","intensity":1}]`)
	analyticsAccountEntry(t, db, owner, at.Add(-31*24*time.Hour), "completed", 100, false, "outside month", `[{"emotion":"excluded","intensity":1}]`)
	if avg, err := r.MoodAvg30Days(ctx, owner); err != nil || avg == nil || *avg != 50 {
		t.Fatalf("combined month average: %v %v", avg, err)
	}
	if got, err := r.RecentSummaries(ctx, owner, 2); err != nil || !reflect.DeepEqual(got, []string{"middle", "newest"}) {
		t.Fatalf("most recent oldest first: %v %v", got, err)
	}
	if got, err := r.RecentSummaries(ctx, owner, 0); err != nil || len(got) != 0 {
		t.Fatalf("zero limit: %v %v", got, err)
	}
	if got, err := r.RecentSummaries(ctx, owner, -1); err == nil || got != nil {
		t.Fatalf("negative SQL limit: %v %v", got, err)
	}
	if got, err := r.TopEmotions(ctx, owner, 2); err != nil || !reflect.DeepEqual(got, []string{"calm", "hope"}) {
		t.Fatalf("ranked emotions: %v %v", got, err)
	}
	if got, err := r.TopTopics(ctx, owner, 2); err != nil || !reflect.DeepEqual(got, []string{"work", "focus"}) {
		t.Fatalf("ranked topics: %v %v", got, err)
	}
	counts, err := r.CountTopicOccurrences(ctx, owner, last, []string{"work", "sleep", "absent"}, at)
	if err != nil || !reflect.DeepEqual(counts, map[string]int{"work": 2, "sleep": 1}) {
		t.Fatalf("case-insensitive distinct entries/excluded self: %v %v", counts, err)
	}
	if got, err := r.CountTopicOccurrences(ctx, owner, last, nil, at); err != nil || got == nil || len(got) != 0 {
		t.Fatalf("empty topics: %v %v", got, err)
	}
	if got, err := r.CountTopicOccurrences(ctx, owner, last, []string{"work"}, at.Add(time.Hour)); err != nil || !reflect.DeepEqual(got, map[string]int{"work": 1}) {
		t.Fatalf("inclusive since boundary: %v %v", got, err)
	}
	missing := uuid.New()
	if avg, err := r.MoodAvg30Days(ctx, missing); err != nil || avg != nil {
		t.Fatalf("unknown average: %v %v", avg, err)
	}
	if got, err := r.RecentSummaries(ctx, missing, 3); err != nil || len(got) != 0 {
		t.Fatalf("unknown summaries: %v %v", got, err)
	}
	if got, err := r.TopEmotions(ctx, missing, 3); err != nil || len(got) != 0 {
		t.Fatalf("unknown emotions: %v %v", got, err)
	}
	if got, err := r.TopTopics(ctx, missing, 3); err != nil || len(got) != 0 {
		t.Fatalf("unknown topics: %v %v", got, err)
	}
}

func TestIntegration_AnalyticsAccountFlashback(t *testing.T) {
	db := regressionDB(t)
	ctx := context.Background()
	owner, other := regressionUser(t, db), regressionUser(t, db)
	r := NewAnalysisRepository(db)
	now := time.Now().UTC()
	month := analyticsAccountEntry(t, db, owner, now.AddDate(0, -1, 0), "completed", 60, false, "monthly", `[]`)
	analyticsAccountEntry(t, db, other, now.AddDate(-1, 0, 0), "completed", 100, false, "foreign yearly", `[]`)
	analyticsAccountEntry(t, db, owner, now.AddDate(-1, 0, 0), "completed", 1, true, "crisis yearly", `[]`)
	analyticsAccountEntry(t, db, owner, now.AddDate(-1, 0, 0), "pending", 100, false, "pending yearly", `[]`)
	analyticsAccountEntry(t, db, owner, now.AddDate(-1, 0, -8), "completed", 100, false, "outside yearly window", `[]`)
	got, err := r.Flashback(ctx, owner)
	if err != nil || got == nil || got.Label != "one_month_ago" || got.EntryID != month || got.Summary != "monthly" || got.MoodScore != 60 || !reflect.DeepEqual(got.Topics, []string{"work", "sleep"}) {
		t.Fatalf("month fallback: %+v %v", got, err)
	}
	analyticsAccountEntry(t, db, owner, now.AddDate(-1, 0, 3), "completed", 70, false, "farther yearly", `[]`)
	year := analyticsAccountEntry(t, db, owner, now.AddDate(-1, 0, 1), "completed", 80, false, "nearest yearly", `[]`)
	got, err = r.Flashback(ctx, owner)
	if err != nil || got == nil || got.Label != "one_year_ago" || got.EntryID != year || got.Summary != "nearest yearly" || got.MoodScore != 80 {
		t.Fatalf("year priority/proximity: %+v %v", got, err)
	}
	if got, err := r.Flashback(ctx, uuid.New()); err != nil || got != nil {
		t.Fatalf("empty flashback: %+v %v", got, err)
	}
}
