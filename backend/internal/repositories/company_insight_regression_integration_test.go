//go:build integration

package repositories

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestIntegration_CompanyMembershipAndAggregates(t *testing.T) {
	db := regressionDB(t)
	ctx := context.Background()
	owner, other, outsider := regressionUser(t, db), regressionUser(t, db), regressionUser(t, db)
	company := uuid.New()
	if _, err := db.Exec(ctx, `INSERT INTO companies(id,name,slug,admin_email) VALUES($1,'QA company','qa-company','qa@example.test')`, company); err != nil {
		t.Fatal(err)
	}
	r := NewCompanyRepository(db)
	got, err := r.GetBySlug(ctx, "qa-company")
	if err != nil || got == nil || got.ID != company || got.Name != "QA company" || got.Plan != "trial" || got.SeatLimit != 50 {
		t.Fatalf("company: %+v %v", got, err)
	}
	if got, err := r.GetBySlug(ctx, "missing"); err != nil || got != nil {
		t.Fatalf("missing: %+v %v", got, err)
	}
	for _, user := range []uuid.UUID{owner, other, owner} {
		if joined, err := r.JoinCompany(ctx, company, user, 50); err != nil || !joined {
			t.Fatal(err)
		}
	}
	if count, err := r.TotalMembers(ctx, company); err != nil || count != 2 {
		t.Fatalf("idempotent membership: %d %v", count, err)
	}
	// M12: the seat limit is enforced inside the join.
	if joined, err := r.JoinCompany(ctx, company, outsider, 2); err != nil || joined {
		t.Fatalf("join over seat limit: joined=%v err=%v", joined, err)
	}
	// C3: only entries made after joining count - backdate membership so the
	// April fixtures below are in scope.
	if _, err := db.Exec(ctx, `UPDATE company_members SET joined_at = '2026-01-01' WHERE company_id = $1`, company); err != nil {
		t.Fatal(err)
	}
	member, err := r.IsMember(ctx, company, owner)
	if err != nil || member == nil || member.UserID != owner || member.CompanyID != company || member.Role != "member" {
		t.Fatalf("member: %+v %v", member, err)
	}
	if member, err := r.IsMember(ctx, company, outsider); err != nil || member != nil {
		t.Fatalf("nonmember: %+v %v", member, err)
	}
	since := time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC)
	exportRegressionEntry(t, db, owner, since, 40, false, "owner", `[]`, []string{})
	exportRegressionEntry(t, db, other, since.Add(time.Hour), 80, false, "other", `[]`, []string{})
	exportRegressionEntry(t, db, outsider, since, 100, false, "outsider", `[]`, []string{})
	exportRegressionEntry(t, db, owner, since.Add(2*time.Hour), 100, true, "crisis", `[]`, []string{})
	exportRegressionEntry(t, db, owner, since.Add(24*time.Hour), 100, false, "end excluded", `[]`, []string{})
	days, err := r.TeamMoodHistory(ctx, company, since, since.Add(24*time.Hour))
	if err != nil || len(days) != 1 || days[0].Day != "2026-04-10" || days[0].AvgMood == nil || *days[0].AvgMood != 60 || days[0].ActiveMembers != 2 || days[0].EntryCount != 2 {
		t.Fatalf("team aggregate: %+v %v", days, err)
	}
	if days, err := r.TeamMoodHistory(ctx, uuid.New(), since, since.Add(24*time.Hour)); err != nil || len(days) != 0 {
		t.Fatalf("other company: %+v %v", days, err)
	}
	if _, err := r.JoinCompany(ctx, uuid.New(), owner, 50); err == nil {
		t.Fatal("missing company accepted")
	}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := r.GetBySlug(canceled, "qa-company"); !errors.Is(err, context.Canceled) {
		t.Fatalf("slug cancel: %v", err)
	}
	if _, err := r.IsMember(canceled, company, owner); !errors.Is(err, context.Canceled) {
		t.Fatalf("membership cancel: %v", err)
	}
	if _, err := r.TotalMembers(canceled, company); !errors.Is(err, context.Canceled) {
		t.Fatalf("count cancel: %v", err)
	}
	if _, err := r.TeamMoodHistory(canceled, company, since, since.Add(time.Hour)); !errors.Is(err, context.Canceled) {
		t.Fatalf("history cancel: %v", err)
	}
	if _, err := r.JoinCompany(canceled, company, owner, 50); !errors.Is(err, context.Canceled) {
		t.Fatalf("join cancel: %v", err)
	}
}

func TestIntegration_InsightCardDataAndShareCounts(t *testing.T) {
	db := regressionDB(t)
	ctx := context.Background()
	owner, other := regressionUser(t, db), regressionUser(t, db)
	r := NewInsightShareRepository(db)
	week := time.Date(2026, 4, 6, 0, 0, 0, 0, time.UTC)
	empty, err := r.GetCardData(ctx, other, week)
	if err != nil || empty.EntryCount != 0 || empty.ShareCount != 0 || len(empty.MoodArc) != 0 || len(empty.TopEmotions) != 0 || empty.WeekStart != "2026-04-06" {
		t.Fatalf("empty: %+v %v", empty, err)
	}
	for i := 0; i < 2; i++ {
		event, err := r.RecordShare(ctx, owner, "2026-04-06")
		if err != nil || event.UserID != owner || event.WeekStart != "2026-04-06" {
			t.Fatalf("share: %+v %v", event, err)
		}
	}
	if count, err := r.CountByUser(ctx, owner); err != nil || count != 2 {
		t.Fatalf("count: %d %v", count, err)
	}
	if count, err := r.CountByUser(ctx, other); err != nil || count != 0 {
		t.Fatalf("foreign count: %d %v", count, err)
	}
	tone := `[{"emotion":"calm","intensity":0.5},{"emotion":"below","intensity":0.2}]`
	exportRegressionEntry(t, db, owner, week, 40, false, "first", tone, []string{})
	exportRegressionEntry(t, db, owner, week.Add(24*time.Hour), 80, false, "second", tone, []string{})
	exportRegressionEntry(t, db, owner, week.Add(7*24*time.Hour), 100, false, "end", tone, []string{})
	exportRegressionEntry(t, db, other, week, 100, false, "foreign", tone, []string{})
	card, err := r.GetCardData(ctx, owner, week)
	if err != nil || card.EntryCount != 2 || card.ShareCount != 2 || len(card.MoodArc) != 2 || card.MoodArc[0].AvgMood != 40 || card.MoodArc[1].AvgMood != 80 || len(card.TopEmotions) != 1 || card.TopEmotions[0] != "calm" || card.WeekLabel == "" {
		t.Fatalf("card: %+v %v", card, err)
	}
	if _, err := r.RecordShare(ctx, owner, "invalid-date"); err == nil {
		t.Fatal("invalid share date accepted")
	}
	if _, err := r.RecordShare(ctx, uuid.New(), "2026-04-06"); err == nil {
		t.Fatal("missing share owner accepted")
	}
	monday := CurrentWeekStart()
	if monday.Weekday() != time.Monday || monday.Hour() != 0 || monday.Location() != time.UTC || monday.After(time.Now().UTC()) {
		t.Fatalf("week start: %v", monday)
	}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := r.RecordShare(canceled, owner, "2026-04-06"); !errors.Is(err, context.Canceled) {
		t.Fatalf("share cancel: %v", err)
	}
	if _, err := r.CountByUser(canceled, owner); !errors.Is(err, context.Canceled) {
		t.Fatalf("count cancel: %v", err)
	}
	if _, err := r.GetCardData(canceled, owner, week); !errors.Is(err, context.Canceled) {
		t.Fatalf("card cancel: %v", err)
	}
}
