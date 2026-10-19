//go:build integration

package repositories

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/dreamlog/backend/internal/models"
	"github.com/google/uuid"
)

func TestIntegration_ChapterCRUDOwnershipAndErrors(t *testing.T) {
	db := regressionDB(t)
	ctx := context.Background()
	owner, other := regressionUser(t, db), regressionUser(t, db)
	r := NewLifeChapterRepository(db)
	ch, err := r.Create(ctx, owner, models.CreateChapterInput{Title: "QA chapter", Description: "original", StartDate: "2026-04-10"})
	if err != nil || ch == nil {
		t.Fatalf("create: %+v %v", ch, err)
	}
	if ch.UserID != owner || ch.StartDate != "2026-04-10" || ch.EndDate != nil || ch.Color != "#7C3AED" || ch.Summary != "" {
		t.Fatalf("defaults: %+v", ch)
	}
	end, title, description, emoji, color := "2026-04-12", "Updated", "updated description", "QA", "#123456"
	updated, err := r.Update(ctx, ch.ID, owner, models.UpdateChapterInput{Title: &title, Description: &description, EndDate: &end, Emoji: &emoji, Color: &color})
	if err != nil || updated == nil || updated.Title != title || updated.Description != description || updated.Emoji != emoji || updated.Color != color || updated.EndDate == nil || *updated.EndDate != end {
		t.Fatalf("update: %+v %v", updated, err)
	}
	if err := r.StoreSummary(ctx, ch.ID, owner, "QA summary"); err != nil {
		t.Fatal(err)
	}
	loaded, err := r.GetByID(ctx, ch.ID, owner)
	if err != nil || loaded.Summary != "QA summary" {
		t.Fatalf("summary: %+v %v", loaded, err)
	}
	if got, err := r.GetByID(ctx, ch.ID, other); err != nil || got != nil {
		t.Fatalf("foreign read: %+v %v", got, err)
	}
	if got, err := r.Update(ctx, ch.ID, other, models.UpdateChapterInput{Title: &title}); err != nil || got != nil {
		t.Fatalf("foreign update: %+v %v", got, err)
	}
	if err := r.StoreSummary(ctx, ch.ID, other, "foreign"); err == nil {
		t.Fatal("foreign summary must fail")
	}
	if err := r.Delete(ctx, ch.ID, other); err != nil {
		t.Fatal(err)
	}
	if got, err := r.GetDetail(ctx, ch.ID, other); err != nil || got != nil {
		t.Fatalf("foreign detail: %+v %v", got, err)
	}
	clear := ""
	updated, err = r.Update(ctx, ch.ID, owner, models.UpdateChapterInput{EndDate: &clear})
	if err != nil || updated.EndDate != nil || updated.Summary != "QA summary" || updated.Title != title {
		t.Fatalf("clear preserves fields: %+v %v", updated, err)
	}
	newer, err := r.Create(ctx, owner, models.CreateChapterInput{Title: "newer", StartDate: "2026-05-01", Color: "#abcdef"})
	if err != nil {
		t.Fatal(err)
	}
	list, err := r.List(ctx, owner)
	if err != nil || len(list) != 2 || list[0].ID != newer.ID || list[1].ID != ch.ID {
		t.Fatalf("order: %+v %v", list, err)
	}
	if list, err := r.List(ctx, other); err != nil || len(list) != 0 {
		t.Fatalf("foreign list: %+v %v", list, err)
	}
	if err := r.Delete(ctx, ch.ID, owner); err != nil {
		t.Fatal(err)
	}
	if got, err := r.GetByID(ctx, ch.ID, owner); err != nil || got != nil {
		t.Fatalf("deleted: %+v %v", got, err)
	}
	if got, err := r.Update(ctx, ch.ID, owner, models.UpdateChapterInput{}); err != nil || got != nil {
		t.Fatalf("missing update: %+v %v", got, err)
	}
	if err := r.Delete(ctx, ch.ID, owner); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Create(ctx, owner, models.CreateChapterInput{Title: "invalid", StartDate: "not-a-date"}); err == nil {
		t.Fatal("invalid date accepted")
	}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := r.List(canceled, owner); !errors.Is(err, context.Canceled) {
		t.Fatalf("list cancel: %v", err)
	}
	if _, err := r.Create(canceled, owner, models.CreateChapterInput{Title: "cancel", StartDate: "2026-01-01"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("create cancel: %v", err)
	}
	if _, err := r.Update(canceled, newer.ID, owner, models.UpdateChapterInput{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("update cancel: %v", err)
	}
	if err := r.StoreSummary(canceled, newer.ID, owner, "cancel"); !errors.Is(err, context.Canceled) {
		t.Fatalf("summary cancel: %v", err)
	}
	if err := r.Delete(canceled, newer.ID, owner); !errors.Is(err, context.Canceled) {
		t.Fatalf("delete cancel: %v", err)
	}
}

func TestIntegration_ChapterStatisticsAndRange(t *testing.T) {
	db := regressionDB(t)
	ctx := context.Background()
	owner, other := regressionUser(t, db), regressionUser(t, db)
	r := NewLifeChapterRepository(db)
	end := "2026-04-11"
	ch, err := r.Create(ctx, owner, models.CreateChapterInput{Title: "QA range", StartDate: "2026-04-10", EndDate: &end})
	if err != nil {
		t.Fatal(err)
	}
	since := time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC)
	tone := `[{"emotion":"calm","intensity":0.8},{"emotion":"low","intensity":0.3}]`
	first := exportRegressionEntry(t, db, owner, since, 40, false, "first", tone, []string{})
	exportRegressionEntry(t, db, owner, since.Add(25*time.Hour), 80, false, "second", tone, []string{})
	exportRegressionEntry(t, db, owner, since.Add(-time.Second), 100, false, "before", tone, []string{})
	exportRegressionEntry(t, db, owner, since.Add(48*time.Hour), 100, false, "after", tone, []string{})
	exportRegressionEntry(t, db, owner, since.Add(time.Hour), 100, true, "crisis", tone, []string{})
	exportRegressionEntry(t, db, other, since.Add(time.Hour), 100, false, "foreign", tone, []string{})
	pending := exportRegressionEntry(t, db, owner, since.Add(2*time.Hour), 100, false, "pending", tone, []string{})
	if _, err := db.Exec(ctx, `UPDATE entries SET status='processing' WHERE id=$1`, pending); err != nil {
		t.Fatal(err)
	}
	detail, err := r.GetDetail(ctx, ch.ID, owner)
	if err != nil || detail == nil || detail.EntryCount != 2 || detail.AvgMood == nil || *detail.AvgMood != 60 || !reflect.DeepEqual(detail.TopEmotions, []string{"calm"}) || len(detail.MoodArc) != 2 || detail.MoodArc[0].Date != "2026-04-10" || detail.MoodArc[0].AvgMood != 40 || detail.MoodArc[1].AvgMood != 80 {
		t.Fatalf("stats: %+v %v", detail, err)
	}
	entries, err := r.GetEntriesInRange(ctx, owner, ch.StartDate, &end)
	if err != nil || len(entries) != 2 || entries[0].Summary != "first" || entries[1].Summary != "second" || !reflect.DeepEqual(entries[0].Emotions, []string{"calm"}) {
		t.Fatalf("range: %+v %v", entries, err)
	}
	if _, err := db.Exec(ctx, `UPDATE entry_analysis SET emotional_tone='{"legacy":true}'::jsonb WHERE entry_id=$1`, first); err != nil {
		t.Fatal(err)
	}
	entries, err = r.GetEntriesInRange(ctx, owner, ch.StartDate, &end)
	if err != nil || len(entries) != 2 || len(entries[0].Emotions) != 0 {
		t.Fatalf("legacy tone: %+v %v", entries, err)
	}
	empty, err := r.Create(ctx, other, models.CreateChapterInput{Title: "empty", StartDate: "2027-01-01"})
	if err != nil {
		t.Fatal(err)
	}
	detail, err = r.GetDetail(ctx, empty.ID, other)
	if err != nil || detail.EntryCount != 0 || detail.AvgMood != nil || detail.MoodArc == nil || detail.TopEmotions == nil {
		t.Fatalf("empty: %+v %v", detail, err)
	}
	ongoing, err := r.GetEntriesInRange(ctx, owner, "2026-04-10", nil)
	if err != nil || len(ongoing) != 3 {
		t.Fatalf("ongoing: %+v %v", ongoing, err)
	}
	if _, err := r.GetEntriesInRange(ctx, owner, "invalid-date", nil); err == nil {
		t.Fatal("invalid date accepted")
	}
	if _, err := db.Exec(ctx, `UPDATE life_chapters SET start_date='2026-04-10'::date WHERE id=$1`, empty.ID); err != nil {
		t.Fatal(err)
	}
	detail, err = r.GetDetail(ctx, empty.ID, other)
	if err != nil || detail.EntryCount != 1 || detail.AvgMood == nil || *detail.AvgMood != 100 {
		t.Fatalf("ongoing detail: %+v %v", detail, err)
	}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := r.GetDetail(canceled, ch.ID, owner); !errors.Is(err, context.Canceled) {
		t.Fatalf("detail cancel: %v", err)
	}
	if _, err := r.GetEntriesInRange(canceled, owner, ch.StartDate, &end); !errors.Is(err, context.Canceled) {
		t.Fatalf("range cancel: %v", err)
	}
	if got, err := r.GetDetail(ctx, uuid.New(), owner); err != nil || got != nil {
		t.Fatalf("missing detail: %+v %v", got, err)
	}
}
