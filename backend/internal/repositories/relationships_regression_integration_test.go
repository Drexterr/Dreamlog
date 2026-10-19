//go:build integration

package repositories

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/dreamlog/backend/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestIntegration_RelationshipConcurrentRetries(t *testing.T) {
	db := regressionDB(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	owner := regressionUser(t, db)
	entry := uuid.New()
	if _, err := db.Exec(ctx, `INSERT INTO entries(id,user_id,audio_key,status) VALUES($1,$2,'fixture/concurrent-retry','completed')`, entry, owner); err != nil {
		t.Fatal(err)
	}
	r := NewRelationshipRepository(db)
	var group sync.WaitGroup
	failures := make(chan error, 6)
	for i := 0; i < 6; i++ {
		group.Add(1)
		go func() {
			defer group.Done()
			failures <- r.UpsertPersonMentions(ctx, owner, entry, []models.ExtractedPerson{{Name: "QA concurrent", Sentiment: "positive"}})
		}()
	}
	group.Wait()
	close(failures)
	for err := range failures {
		if err != nil {
			t.Fatal(err)
		}
	}
	people, err := r.GetMap(ctx, owner)
	if err != nil || len(people) != 1 || people[0].MentionCount != 1 || people[0].PositiveCount != 1 {
		t.Fatalf("concurrent counters: %+v %v", people, err)
	}
	detail, err := r.GetDetail(ctx, people[0].ID, owner)
	if err != nil || detail == nil || len(detail.Mentions) != 1 {
		t.Fatalf("concurrent mentions: %+v %v", detail, err)
	}
}

func TestIntegration_RelationshipRetryAndManagement(t *testing.T) {
	db := regressionDB(t)
	ctx := context.Background()
	owner, other := regressionUser(t, db), regressionUser(t, db)
	entry := uuid.New()
	if _, err := db.Exec(ctx, `INSERT INTO entries(id,user_id,audio_key,status) VALUES($1,$2,'fixture/relationship','completed')`, entry, owner); err != nil {
		t.Fatal(err)
	}
	r := NewRelationshipRepository(db)
	mentions := []models.ExtractedPerson{{Name: "Alice", Role: "friend", Sentiment: "positive", Context: "fixture"}, {Name: ""}}
	for i := 0; i < 2; i++ {
		if err := r.UpsertPersonMentions(ctx, owner, entry, mentions); err != nil {
			t.Fatal(err)
		}
	}
	people, err := r.GetMap(ctx, owner)
	if err != nil || len(people) != 1 {
		t.Fatalf("map: %+v %v", people, err)
	}
	person := people[0]
	if person.MentionCount != 1 || person.PositiveCount != 1 || person.NegativeCount != 0 {
		t.Fatalf("retry inflated counters: %+v", person)
	}
	detail, err := r.GetDetail(ctx, person.ID, owner)
	if err != nil || detail == nil || len(detail.Mentions) != 1 || detail.Mentions[0].EntryID != entry || detail.Mentions[0].Context != "fixture" {
		t.Fatalf("detail: %+v %v", detail, err)
	}
	if got, err := r.GetDetail(ctx, person.ID, other); err != nil || got != nil {
		t.Fatalf("foreign detail: %+v %v", got, err)
	}
	name, role, hidden := "Renamed Alice", "family", true
	updated, err := r.UpdatePerson(ctx, person.ID, owner, models.UpdatePersonInput{Name: &name, Role: &role, Hidden: &hidden})
	if err != nil || updated == nil || updated.Name != name || string(updated.Role) != role || !updated.Hidden || updated.MentionCount != 1 {
		t.Fatalf("update: %+v %v", updated, err)
	}
	if got, err := r.GetMap(ctx, owner); err != nil || len(got) != 0 {
		t.Fatalf("hidden map: %+v %v", got, err)
	}
	if got, err := r.UpdatePerson(ctx, person.ID, other, models.UpdatePersonInput{Name: &name}); err != nil || got != nil {
		t.Fatalf("foreign edit: %+v %v", got, err)
	}
	hidden = false
	if got, err := r.UpdatePerson(ctx, person.ID, owner, models.UpdatePersonInput{Hidden: &hidden}); err != nil || got == nil || got.Name != name || got.Hidden {
		t.Fatalf("restore: %+v %v", got, err)
	}
	if got, err := r.GetMap(ctx, other); err != nil || len(got) != 0 {
		t.Fatalf("foreign map: %+v %v", got, err)
	}
}

func TestIntegration_RelationshipMergeAndRollback(t *testing.T) {
	db := regressionDB(t)
	ctx := context.Background()
	owner, other := regressionUser(t, db), regressionUser(t, db)
	r := NewRelationshipRepository(db)
	entries := []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}
	for _, id := range entries {
		if _, err := db.Exec(ctx, `INSERT INTO entries(id,user_id,audio_key,status) VALUES($1,$2,'fixture/merge','completed')`, id, owner); err != nil {
			t.Fatal(err)
		}
	}
	add := func(entry uuid.UUID, people ...models.ExtractedPerson) {
		t.Helper()
		if err := r.UpsertPersonMentions(ctx, owner, entry, people); err != nil {
			t.Fatal(err)
		}
	}
	add(entries[0], models.ExtractedPerson{Name: "Target", Sentiment: "positive"}, models.ExtractedPerson{Name: "Source", Sentiment: "negative"})
	add(entries[1], models.ExtractedPerson{Name: "target", Sentiment: "positive"})
	add(entries[2], models.ExtractedPerson{Name: "Source", Sentiment: "negative"})
	people, err := r.GetMap(ctx, owner)
	if err != nil || len(people) != 2 {
		t.Fatalf("map: %+v %v", people, err)
	}
	var target, source uuid.UUID
	for _, p := range people {
		if strings.EqualFold(p.Name, "target") {
			target = p.ID
		} else {
			source = p.ID
		}
	}
	if got, err := r.MergePeople(ctx, target, source, other); err != nil || got != nil {
		t.Fatalf("foreign merge: %+v %v", got, err)
	}
	if got, err := r.MergePeople(ctx, target, target, owner); err == nil || got != nil {
		t.Fatalf("self merge: %+v %v", got, err)
	}
	collision := "TARGET"
	if _, err := r.UpdatePerson(ctx, source, owner, models.UpdatePersonInput{Name: &collision}); err == nil {
		t.Fatal("expected case-insensitive name collision")
	} else {
		var pgerr *pgconn.PgError
		if !errors.As(err, &pgerr) || pgerr.Code != "23505" {
			t.Fatalf("collision: %v", err)
		}
	}
	merged, err := r.MergePeople(ctx, target, source, owner)
	if err != nil || merged == nil || merged.MentionCount != 3 || merged.PositiveCount != 2 || merged.NegativeCount != 1 {
		t.Fatalf("merge: %+v %v", merged, err)
	}
	if got, err := r.GetDetail(ctx, source, owner); err != nil || got != nil {
		t.Fatalf("source retained: %+v %v", got, err)
	}
	detail, err := r.GetDetail(ctx, target, owner)
	if err != nil || len(detail.Mentions) != 3 {
		t.Fatalf("merged mentions: %+v %v", detail, err)
	}
	top, err := r.TopPeople(ctx, owner, 1)
	if err != nil || len(top) != 1 || !strings.Contains(top[0], "mixed feelings") {
		t.Fatalf("top: %v %v", top, err)
	}
	if err := r.UpsertPersonMentions(ctx, owner, uuid.New(), []models.ExtractedPerson{{Name: "Rollback"}}); err == nil {
		t.Fatal("missing entry must fail")
	}
	people, err = r.GetMap(ctx, owner)
	if err != nil || len(people) != 1 {
		t.Fatalf("transaction leaked person: %+v %v", people, err)
	}
	if got, err := r.MergePeople(ctx, target, uuid.New(), owner); err != nil || got != nil {
		t.Fatalf("missing merge: %+v %v", got, err)
	}
	if got, err := r.GetDetail(ctx, uuid.New(), owner); err != nil || got != nil {
		t.Fatalf("missing detail: %+v %v", got, err)
	}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if err := r.UpsertPersonMentions(canceled, owner, entries[0], nil); err != nil {
		t.Fatal(err)
	}
	if err := r.UpsertPersonMentions(canceled, owner, entries[0], []models.ExtractedPerson{{Name: "Canceled"}}); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled insert: %v", err)
	}
	if _, err := r.GetMap(canceled, owner); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled map: %v", err)
	}
	if _, err := r.GetDetail(canceled, target, owner); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled detail: %v", err)
	}
	if _, err := r.UpdatePerson(canceled, target, owner, models.UpdatePersonInput{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled edit: %v", err)
	}
	if _, err := r.MergePeople(canceled, target, source, owner); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled merge: %v", err)
	}
	if _, err := r.TopPeople(canceled, owner, 1); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled top: %v", err)
	}
}
