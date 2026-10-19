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

func TestIntegration_WeeklyReviewLifecycle(t *testing.T) {
	db := regressionDB(t)
	ctx := context.Background()
	owner, other := regressionUser(t, db), regressionUser(t, db)
	r := NewWeeklyReviewRepository(db)
	week := time.Date(2026, 4, 6, 0, 0, 0, 0, time.UTC)
	due := time.Now().Add(-time.Hour)
	if got, err := r.GetLatestCompleted(ctx, owner); err != nil || got != nil {
		t.Fatalf("empty: %+v %v", got, err)
	}
	for i := 0; i < 2; i++ {
		if err := r.Schedule(ctx, owner, week, due); err != nil {
			t.Fatal(err)
		}
	}
	if err := r.Schedule(ctx, other, week, time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	pending, err := r.PendingDue(ctx)
	if err != nil || len(pending) != 1 || pending[0].UserID != owner || pending[0].Narrative != "" || pending[0].GeneratedAt != nil || !pending[0].WeekStart.Equal(week) {
		t.Fatalf("pending nullable/idempotent: %+v %v", pending, err)
	}
	arc := []models.MoodArcDay{{Date: "2026-04-06", AvgMood: 60}}
	firstID := pending[0].ID
	if err := r.MarkCompleted(ctx, pending[0].ID, "QA weekly", []string{"calm"}, arc, 2); err != nil {
		t.Fatal(err)
	}
	got, err := r.GetLatestCompleted(ctx, owner)
	if err != nil || got == nil || got.Narrative != "QA weekly" || got.EntryCount != 2 || got.GeneratedAt == nil || !reflect.DeepEqual(got.MoodArc, arc) || !reflect.DeepEqual(got.TopEmotions, []string{"calm"}) {
		t.Fatalf("completed: %+v %v", got, err)
	}
	if got, err := r.GetLatestCompleted(ctx, other); err != nil || got != nil {
		t.Fatalf("foreign latest: %+v %v", got, err)
	}
	later := week.AddDate(0, 0, 7)
	if err := r.Schedule(ctx, owner, later, due); err != nil {
		t.Fatal(err)
	}
	pending, err = r.PendingDue(ctx)
	if err != nil || len(pending) != 1 {
		t.Fatalf("due after complete: %+v %v", pending, err)
	}
	if err := r.MarkCompleted(ctx, pending[0].ID, "newer", []string{}, []models.MoodArcDay{}, 0); err != nil {
		t.Fatal(err)
	}
	list, err := r.ListCompleted(ctx, owner, 1)
	if err != nil || len(list) != 1 || !list[0].WeekStart.Equal(later) {
		t.Fatalf("bounded order: %+v %v", list, err)
	}
	list, err = r.ListCompleted(ctx, owner, 10)
	if err != nil || len(list) != 2 {
		t.Fatalf("list: %+v %v", list, err)
	}
	if list, err := r.ListCompleted(ctx, other, 10); err != nil || len(list) != 0 {
		t.Fatalf("foreign list: %+v %v", list, err)
	}
	if err := r.MarkFailed(ctx, pending[0].ID, "QA failure"); err != nil {
		t.Fatal(err)
	}
	var failure string
	if err := db.QueryRow(ctx, `SELECT error_msg FROM weekly_reviews WHERE id=$1`, pending[0].ID).Scan(&failure); err != nil || failure != "QA failure" {
		t.Fatalf("failed persistence: %s %v", failure, err)
	}
	if got, err := r.GetLatestCompleted(ctx, owner); err != nil || got == nil || got.ID != firstID {
		t.Fatalf("failed excluded: %+v %v", got, err)
	}
	if _, err := r.ListCompleted(ctx, owner, -1); err == nil {
		t.Fatal("negative limit accepted")
	}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if err := r.Schedule(canceled, owner, week, due); !errors.Is(err, context.Canceled) {
		t.Fatalf("schedule cancel: %v", err)
	}
	if _, err := r.PendingDue(canceled); !errors.Is(err, context.Canceled) {
		t.Fatalf("pending cancel: %v", err)
	}
	if err := r.MarkCompleted(canceled, uuid.New(), "", nil, nil, 0); !errors.Is(err, context.Canceled) {
		t.Fatalf("complete cancel: %v", err)
	}
	if err := r.MarkFailed(canceled, uuid.New(), "cancel"); !errors.Is(err, context.Canceled) {
		t.Fatalf("failed cancel: %v", err)
	}
	if _, err := r.GetLatestCompleted(canceled, owner); !errors.Is(err, context.Canceled) {
		t.Fatalf("latest cancel: %v", err)
	}
	if _, err := r.ListCompleted(canceled, owner, 1); !errors.Is(err, context.Canceled) {
		t.Fatalf("list cancel: %v", err)
	}
}

func TestIntegration_AnnualReviewLifecycle(t *testing.T) {
	db := regressionDB(t)
	ctx := context.Background()
	owner, other := regressionUser(t, db), regressionUser(t, db)
	r := NewAnnualReviewRepository(db)
	due := time.Now().Add(-time.Hour)
	if got, err := r.GetLatestCompleted(ctx, owner); err != nil || got != nil {
		t.Fatalf("empty: %+v %v", got, err)
	}
	for i := 0; i < 2; i++ {
		if err := r.Schedule(ctx, owner, 2025, due); err != nil {
			t.Fatal(err)
		}
	}
	if err := r.Schedule(ctx, other, 2025, time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	pending, err := r.PendingDue(ctx)
	if err != nil || len(pending) != 1 || pending[0].UserID != owner || pending[0].Year != 2025 || pending[0].Narrative != "" || pending[0].AvgMood != nil || pending[0].GeneratedAt != nil {
		t.Fatalf("pending nullable/idempotent: %+v %v", pending, err)
	}
	firstID := pending[0].ID
	arc := []models.MonthlyMoodArcDay{{Month: "2025-04", AvgMood: 65, EntryCount: 2}}
	if err := r.MarkCompleted(ctx, firstID, "QA annual", []string{"calm"}, []string{"work"}, arc, 2, 65); err != nil {
		t.Fatal(err)
	}
	got, err := r.GetLatestCompleted(ctx, owner)
	if err != nil || got == nil || got.Narrative != "QA annual" || got.AvgMood == nil || *got.AvgMood != 65 || got.GeneratedAt == nil || !reflect.DeepEqual(got.MoodArc, arc) || !reflect.DeepEqual(got.TopTopics, []string{"work"}) {
		t.Fatalf("completed: %+v %v", got, err)
	}
	if got, err := r.GetLatestCompleted(ctx, other); err != nil || got != nil {
		t.Fatalf("foreign: %+v %v", got, err)
	}
	if err := r.Schedule(ctx, owner, 2026, due); err != nil {
		t.Fatal(err)
	}
	pending, err = r.PendingDue(ctx)
	if err != nil || len(pending) != 1 {
		t.Fatalf("due: %+v %v", pending, err)
	}
	if err := r.MarkCompleted(ctx, pending[0].ID, "newer", []string{}, []string{}, []models.MonthlyMoodArcDay{}, 0, 0); err != nil {
		t.Fatal(err)
	}
	list, err := r.ListCompleted(ctx, owner)
	if err != nil || len(list) != 2 || list[0].Year != 2026 || list[1].Year != 2025 {
		t.Fatalf("order: %+v %v", list, err)
	}
	if list, err := r.ListCompleted(ctx, other); err != nil || len(list) != 0 {
		t.Fatalf("foreign list: %+v %v", list, err)
	}
	if err := r.MarkFailed(ctx, pending[0].ID, "QA failure"); err != nil {
		t.Fatal(err)
	}
	got, err = r.GetLatestCompleted(ctx, owner)
	if err != nil || got == nil || got.ID != firstID {
		t.Fatalf("failed excluded: %+v %v", got, err)
	}
	var failure string
	if err := db.QueryRow(ctx, `SELECT error_msg FROM annual_reviews WHERE id=$1`, pending[0].ID).Scan(&failure); err != nil || failure != "QA failure" {
		t.Fatalf("failed persistence: %s %v", failure, err)
	}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if err := r.Schedule(canceled, owner, 2025, due); !errors.Is(err, context.Canceled) {
		t.Fatalf("schedule cancel: %v", err)
	}
	if _, err := r.PendingDue(canceled); !errors.Is(err, context.Canceled) {
		t.Fatalf("pending cancel: %v", err)
	}
	if err := r.MarkCompleted(canceled, uuid.New(), "", nil, nil, nil, 0, 0); !errors.Is(err, context.Canceled) {
		t.Fatalf("complete cancel: %v", err)
	}
	if err := r.MarkFailed(canceled, uuid.New(), "cancel"); !errors.Is(err, context.Canceled) {
		t.Fatalf("failed cancel: %v", err)
	}
	if _, err := r.GetLatestCompleted(canceled, owner); !errors.Is(err, context.Canceled) {
		t.Fatalf("latest cancel: %v", err)
	}
	if _, err := r.ListCompleted(canceled, owner); !errors.Is(err, context.Canceled) {
		t.Fatalf("list cancel: %v", err)
	}
}
