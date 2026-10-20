//go:build integration

package repositories

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/dreamlog/backend/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func regressionDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	return regressionDBUpTo(t, "")
}

// regressionDBUpTo migrates a fresh schema through the migration whose file
// name starts with last ("" = all), so a test can seed pre-migration state
// and then apply the next migration itself.
func regressionDBUpTo(t *testing.T, last string) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("ODE_TEST_DATABASE_URL")
	if dsn == "" {
		t.Fatal("ODE_TEST_DATABASE_URL required: use the isolated regression compose services")
	}
	u, err := url.Parse(dsn)
	if err != nil || !strings.HasSuffix(u.Path, "_test") || (u.Hostname() != "127.0.0.1" && u.Hostname() != "localhost") {
		t.Fatal("integration tests require a loopback URL with a database name ending in _test")
	}
	ctx := context.Background()
	base, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	schema := "qa_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	quoted := pgx.Identifier{schema}.Sanitize()
	if _, err = base.Exec(ctx, "CREATE EXTENSION IF NOT EXISTS pgcrypto WITH SCHEMA public"); err != nil {
		base.Close()
		t.Fatal(err)
	}
	if _, err = base.Exec(ctx, "CREATE SCHEMA "+quoted); err != nil {
		base.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, cleanupErr := base.Exec(ctx, "DROP SCHEMA "+quoted+" CASCADE"); cleanupErr != nil {
			t.Error(cleanupErr)
		}
		base.Close()
	})
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schema + ",public"
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	files, err := filepath.Glob("../../migrations/*.up.sql")
	if err != nil || len(files) == 0 {
		t.Fatal("migration files missing")
	}
	sort.Strings(files)
	for _, file := range files {
		if last != "" && filepath.Base(file) > last+"~" {
			break
		}
		sql, readErr := os.ReadFile(file)
		if readErr != nil {
			t.Fatal(readErr)
		}
		if _, err = pool.Exec(ctx, string(sql)); err != nil {
			t.Fatalf("migration %s: %v", filepath.Base(file), err)
		}
	}
	return pool
}

func regressionUser(t *testing.T, db *pgxpool.Pool) uuid.UUID {
	t.Helper()
	id := uuid.New()
	_, err := db.Exec(context.Background(), `INSERT INTO users(id,supabase_id,email,name) VALUES($1,$2,$3,'QA')`, id, id.String(), id.String()+"@example.test")
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func TestIntegration_EntryLifecycleAndOwnership(t *testing.T) {
	db := regressionDB(t)
	ctx := context.Background()
	user := regressionUser(t, db)
	other := regressionUser(t, db)
	r := NewEntryRepository(db)
	entry, err := r.Create(ctx, user, "owned/clip", 100, 3.5, models.EntryModeDream)
	if err != nil {
		t.Fatal(err)
	}
	if entry.Status != models.EntryStatusPending || entry.Mode != models.EntryModeDream {
		t.Fatalf("created %+v", entry)
	}
	if foreign, err := r.GetByID(ctx, entry.ID, other); err != nil || foreign != nil {
		t.Fatalf("ownership leak: %+v %v", foreign, err)
	}
	if absent, err := r.GetByID(ctx, uuid.New(), user); err != nil || absent != nil {
		t.Fatalf("missing entry: %+v %v", absent, err)
	}
	if ok, err := r.SetProcessing(ctx, entry.ID); err != nil || !ok {
		t.Fatalf("transition: %v %v", ok, err)
	}
	if ok, err := r.SetProcessing(ctx, entry.ID); err != nil || ok {
		t.Fatalf("duplicate transition: %v %v", ok, err)
	}
	if err := r.SetFailed(ctx, entry.ID, "provider unavailable"); err != nil {
		t.Fatal(err)
	}
	failed, err := r.GetByID(ctx, entry.ID, user)
	if err != nil || failed.RetryCount != 1 || failed.ErrorMsg == nil {
		t.Fatalf("failed entry %+v %v", failed, err)
	}
	if ok, err := r.SetProcessing(ctx, entry.ID); err != nil || !ok {
		t.Fatalf("retry: %v %v", ok, err)
	}
	if err := r.SetTranscript(ctx, entry.ID, "A calm day", "english"); err != nil {
		t.Fatal(err)
	}
	transcribed, err := r.GetByIDInternal(ctx, entry.ID)
	if err != nil || transcribed.Status != models.EntryStatusProcessing || transcribed.Transcript == nil || *transcribed.Transcript != "A calm day" {
		t.Fatalf("transcript persistence advertised premature completion: %+v %v", transcribed, err)
	}
	if err := r.SetCompleted(ctx, entry.ID, "A calm day", "english"); err != nil {
		t.Fatal(err)
	}
	completed, err := r.GetByIDInternal(ctx, entry.ID)
	if err != nil || completed.Status != models.EntryStatusCompleted || completed.Transcript == nil || *completed.Transcript != "A calm day" || completed.ErrorMsg != nil {
		t.Fatalf("completed %+v %v", completed, err)
	}
	if ok, err := r.SetProcessing(ctx, entry.ID); err != nil || ok {
		t.Fatalf("completed entry was reprocessed: %v %v", ok, err)
	}
}

func TestIntegration_ConcurrentWorkerClaims(t *testing.T) {
	db := regressionDB(t)
	ctx := context.Background()
	user := regressionUser(t, db)
	r := NewEntryRepository(db)
	entry, err := r.Create(ctx, user, "owned/clip", 100, 3, models.EntryModeProcessing)
	if err != nil {
		t.Fatal(err)
	}
	results := make(chan bool, 8)
	failures := make(chan error, 8)
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); ok, err := r.SetProcessing(ctx, entry.ID); results <- ok; failures <- err }()
	}
	wg.Wait()
	close(results)
	close(failures)
	count := 0
	for ok := range results {
		if ok {
			count++
		}
	}
	for err := range failures {
		if err != nil {
			t.Fatal(err)
		}
	}
	if count != 1 {
		t.Fatalf("expected one worker claim; got %d", count)
	}
}

func TestIntegration_EntryPaginationAndForeignKeys(t *testing.T) {
	db := regressionDB(t)
	ctx := context.Background()
	user := regressionUser(t, db)
	other := regressionUser(t, db)
	r := NewEntryRepository(db)
	for i := 0; i < 3; i++ {
		if _, err := r.Create(ctx, user, fmt.Sprintf("owned/%d", i), 100, 3, models.EntryModeProcessing); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := r.Create(ctx, other, "other/clip", 100, 3, models.EntryModeProcessing); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ page, size, want int }{{1, 2, 2}, {2, 2, 1}, {3, 2, 0}, {0, 0, 3}, {1, 101, 3}} {
		t.Run(fmt.Sprintf("page_%d_size_%d", tc.page, tc.size), func(t *testing.T) {
			entries, total, err := r.List(ctx, ListEntriesOpts{UserID: user, Page: tc.page, PageSize: tc.size})
			if err != nil || total != 3 || len(entries) != tc.want {
				t.Fatalf("rows=%d total=%d err=%v", len(entries), total, err)
			}
			for _, entry := range entries {
				if entry.UserID != user {
					t.Fatal("foreign entry in results")
				}
			}
		})
	}
	if _, err := r.Create(ctx, uuid.New(), "missing/clip", 100, 3, models.EntryModeProcessing); err == nil {
		t.Fatal("foreign key allowed nonexistent user")
	}
	entry, err := r.Create(ctx, user, "owned/default", 100, 3, models.EntryMode("invalid"))
	if err != nil || entry.Mode != models.EntryModeProcessing {
		t.Fatalf("invalid-mode fallback %+v %v", entry, err)
	}
}

func TestIntegration_UserProfilePersistenceAndConstraints(t *testing.T) {
	db := regressionDB(t)
	ctx := context.Background()
	user := regressionUser(t, db)
	r := NewUserRepository(db)
	name, goal, age, country, language := "QA updated", "stress", "25_34", "IN", "hindi"
	updated, err := r.UpdateProfile(ctx, user, ProfileUpdate{Name: &name, Goal: &goal, AgeRange: &age, Country: &country, VoiceLanguage: &language})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Name != name || updated.Goal == nil || *updated.Goal != goal || updated.AgeRange == nil || *updated.AgeRange != age {
		t.Fatalf("profile not persisted %+v", updated)
	}
	badAge := "invalid"
	if _, err := r.UpdateProfile(ctx, user, ProfileUpdate{AgeRange: &badAge}); err == nil {
		t.Fatal("DB accepted invalid age range")
	}
	if _, err := r.UpdateProfile(ctx, user, ProfileUpdate{}); err != nil {
		t.Fatal(err)
	}
}

func exportRegressionEntry(t *testing.T, db *pgxpool.Pool, user uuid.UUID, at time.Time, mood int, crisis bool, summary, tones string, quotes []string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	ctx := context.Background()
	if _, err := db.Exec(ctx, `INSERT INTO entries(id,user_id,audio_key,status,created_at) VALUES($1,$2,'fixture/not-uploaded','completed',$3)`, id, user, at); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(ctx, `INSERT INTO entry_analysis(entry_id,mood_score,is_crisis,summary,emotional_tone,topics,key_quotes) VALUES($1,$2,$3,$4,$5::jsonb,ARRAY['fixture-topic'],$6)`, id, mood, crisis, summary, tones, quotes); err != nil {
		t.Fatal(err)
	}
	return id
}

func TestIntegration_ExportWindowOwnershipAndAggregates(t *testing.T) {
	db := regressionDB(t)
	ctx := context.Background()
	owner, other := regressionUser(t, db), regressionUser(t, db)
	since := time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC)
	until := since.Add(48 * time.Hour)
	prevSince := since.Add(-48 * time.Hour)
	emptyQuotes := []string{}
	seed := func(user uuid.UUID, at time.Time, mood int, crisis bool, summary, tones string, quotes []string) {
		exportRegressionEntry(t, db, user, at, mood, crisis, summary, tones, quotes)
	}
	// Insert out of chronological order so sorting is tested, not fixture order.
	seed(owner, since.Add(24*time.Hour), 90, false, "second day", `[{"emotion":"calm","intensity":0.9}]`, emptyQuotes)
	seed(owner, since, 40, false, "window start", `[{"emotion":"calm","intensity":0.5},{"emotion":"hopeful","intensity":0.8},{"emotion":"below-threshold","intensity":0.49}]`, []string{"first quote", "not selected"})
	seed(owner, since.Add(time.Hour), 60, false, "later first day", `[{"emotion":"calm","intensity":0.7},{"emotion":"hopeful","intensity":0.8}]`, emptyQuotes)
	seed(owner, prevSince, 20, false, "prior start", `[]`, emptyQuotes)
	seed(owner, since.Add(-time.Microsecond), 40, false, "prior end", `[]`, emptyQuotes)
	seed(owner, prevSince.Add(-time.Microsecond), 100, false, "too old", `[{"emotion":"excluded","intensity":1}]`, emptyQuotes)
	seed(owner, until, 100, false, "exclusive end", `[{"emotion":"excluded","intensity":1}]`, emptyQuotes)
	seed(owner, until.Add(time.Hour), 100, false, "future", `[{"emotion":"excluded","intensity":1}]`, emptyQuotes)
	seed(owner, since.Add(2*time.Hour), 1, true, "private crisis", `[{"emotion":"excluded","intensity":1}]`, emptyQuotes)
	seed(owner, prevSince.Add(time.Hour), 100, true, "prior crisis", `[]`, emptyQuotes)
	seed(other, since.Add(time.Hour), 100, false, "foreign current", `[{"emotion":"excluded","intensity":1}]`, emptyQuotes)
	seed(other, prevSince.Add(time.Hour), 100, false, "foreign prior", `[]`, emptyQuotes)
	if _, err := db.Exec(ctx, `INSERT INTO entries(user_id,audio_key,status,created_at) VALUES($1,'fixture/no-analysis','pending',$2)`, owner, since.Add(3*time.Hour)); err != nil {
		t.Fatal(err)
	}

	got, err := NewAnalysisRepository(db).ExportData(ctx, owner, since, until)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Since.Equal(since) || !got.Until.Equal(until) || got.EntryCount != 3 {
		t.Fatalf("window/count: %+v", got)
	}
	// Two entries at 50 on day one and one at 90 on day two: weighted mean 63, not 70.
	if got.AvgMood == nil || *got.AvgMood != 63 || got.PrevAvgMood == nil || *got.PrevAvgMood != 30 || got.MoodDelta == nil || *got.MoodDelta != 33 {
		t.Fatalf("averages/delta: %+v", got)
	}
	wantDays := []*models.DailyMood{{Day: "2026-04-10", AvgMood: 50, EntryCount: 2}, {Day: "2026-04-11", AvgMood: 90, EntryCount: 1}}
	if !reflect.DeepEqual(got.DailyMoods, wantDays) || !reflect.DeepEqual(got.TopEmotions, []string{"calm", "hopeful"}) {
		t.Fatalf("days=%+v emotions=%v", got.DailyMoods, got.TopEmotions)
	}
	wantEntries := []*models.ExportEntrySummary{
		{Date: since, Summary: "window start", MoodScore: 40, Topics: []string{"fixture-topic"}, KeyQuote: "first quote"},
		{Date: since.Add(time.Hour), Summary: "later first day", MoodScore: 60, Topics: []string{"fixture-topic"}},
		{Date: since.Add(24 * time.Hour), Summary: "second day", MoodScore: 90, Topics: []string{"fixture-topic"}},
	}
	if len(got.Entries) != len(wantEntries) {
		t.Fatalf("summaries count=%d, want %d", len(got.Entries), len(wantEntries))
	}
	for i, want := range wantEntries {
		entry := got.Entries[i]
		if !entry.Date.Equal(want.Date) || entry.Summary != want.Summary || entry.MoodScore != want.MoodScore || entry.KeyQuote != want.KeyQuote || !reflect.DeepEqual(entry.Topics, want.Topics) {
			t.Errorf("entry %d: got %+v, want %+v", i, entry, want)
		}
	}
}

func TestIntegration_ExportEmptyAndMissingPriorPeriod(t *testing.T) {
	db := regressionDB(t)
	ctx := context.Background()
	owner := regressionUser(t, db)
	r := NewAnalysisRepository(db)
	since := time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC)
	until := since.Add(24 * time.Hour)
	checkEmpty := func(t *testing.T, user uuid.UUID, wantPrior *int) {
		t.Helper()
		got, err := r.ExportData(ctx, user, since, until)
		if err != nil {
			t.Fatal(err)
		}
		if got.EntryCount != 0 || got.AvgMood != nil || got.MoodDelta != nil || !reflect.DeepEqual(got.PrevAvgMood, wantPrior) || len(got.Entries) != 0 || len(got.DailyMoods) != 0 || len(got.TopEmotions) != 0 {
			t.Fatalf("empty current period produced data: %+v", got)
		}
	}
	t.Run("empty user", func(t *testing.T) { checkEmpty(t, owner, nil) })
	exportRegressionEntry(t, db, owner, since.Add(-time.Hour), 40, false, "prior only", `[]`, []string{})
	t.Run("prior only has no delta", func(t *testing.T) { prior := 40; checkEmpty(t, owner, &prior) })
	t.Run("unknown user cannot see owner data", func(t *testing.T) { checkEmpty(t, uuid.New(), nil) })
	newUser := regressionUser(t, db)
	exportRegressionEntry(t, db, newUser, since, 70, false, "first entry", `[]`, []string{})
	t.Run("current only has no prior or delta", func(t *testing.T) {
		got, err := r.ExportData(ctx, newUser, since, until)
		if err != nil {
			t.Fatal(err)
		}
		if got.EntryCount != 1 || got.AvgMood == nil || *got.AvgMood != 70 || got.PrevAvgMood != nil || got.MoodDelta != nil || len(got.Entries) != 1 || got.Entries[0].Summary != "first entry" {
			t.Fatalf("first-period export: %+v", got)
		}
	})
}

func TestIntegration_ExportMalformedEmotionAndCanceledQuery(t *testing.T) {
	db := regressionDB(t)
	ctx := context.Background()
	owner := regressionUser(t, db)
	since := time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC)
	until := since.Add(24 * time.Hour)
	exportRegressionEntry(t, db, owner, since, 50, false, "legacy emotion shape", `{"unexpected":"shape"}`, []string{})
	r := NewAnalysisRepository(db)
	got, err := r.ExportData(ctx, owner, since, until)
	if err != nil || got == nil {
		t.Fatalf("malformed emotion export: %+v %v", got, err)
	}
	if got.EntryCount != 1 || got.AvgMood == nil || *got.AvgMood != 50 || len(got.TopEmotions) != 0 || len(got.Entries) != 1 || got.Entries[0].Summary != "legacy emotion shape" {
		t.Fatalf("malformed tone discarded otherwise valid entry: %+v", got)
	}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if data, err := r.ExportData(canceled, owner, since, until); !errors.Is(err, context.Canceled) || data != nil || !strings.Contains(err.Error(), "ExportData daily:") {
		t.Fatalf("canceled query returned partial success: %+v %v", data, err)
	}
}
