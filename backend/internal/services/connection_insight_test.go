package services

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

type fakeTopicHistorian struct {
	counts    map[string]int
	err       error
	gotTopics []string
	gotSince  time.Time
}

func (f *fakeTopicHistorian) CountTopicOccurrences(_ context.Context, _, _ uuid.UUID, topics []string, since time.Time) (map[string]int, error) {
	f.gotTopics = topics
	f.gotSince = since
	if f.err != nil {
		return nil, f.err
	}
	return f.counts, nil
}

func newInsightFixture(counts map[string]int) (*ConnectionInsightService, *fakeTopicHistorian) {
	repo := &fakeTopicHistorian{counts: counts}
	return NewConnectionInsightService(repo), repo
}

func TestConnectionInsight_NoTopics_ReturnsEmpty(t *testing.T) {
	svc, repo := newInsightFixture(map[string]int{})

	got, err := svc.BuildConnectionInsight(context.Background(), uuid.New(), uuid.New(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "" {
		t.Errorf("no topics: want empty insight, got %q", got)
	}
	if repo.gotTopics != nil {
		t.Error("repo must not be queried when there are no topics")
	}
}

func TestConnectionInsight_BelowThreshold_ReturnsEmpty(t *testing.T) {
	// Only 1 prior occurrence - not yet a pattern (needs 2+ prior).
	svc, _ := newInsightFixture(map[string]int{"work stress": 1})

	got, err := svc.BuildConnectionInsight(context.Background(), uuid.New(), uuid.New(), []string{"work stress"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "" {
		t.Errorf("below threshold: want empty insight, got %q", got)
	}
}

func TestConnectionInsight_AtThreshold_ProducesThirdTimeInsight(t *testing.T) {
	svc, _ := newInsightFixture(map[string]int{"work stress": 2})

	got, err := svc.BuildConnectionInsight(context.Background(), uuid.New(), uuid.New(), []string{"Work Stress"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(got, "3rd time") {
		t.Errorf("want insight mentioning '3rd time', got %q", got)
	}
	if !strings.Contains(got, "work stress") {
		t.Errorf("want insight mentioning the topic, got %q", got)
	}
}

func TestConnectionInsight_PicksMostRecurrentTopic(t *testing.T) {
	svc, _ := newInsightFixture(map[string]int{"sleep": 2, "family": 5})

	got, err := svc.BuildConnectionInsight(context.Background(), uuid.New(), uuid.New(), []string{"sleep", "family"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(got, "family") {
		t.Errorf("want the most recurrent topic (family), got %q", got)
	}
	if !strings.Contains(got, "6th time") {
		t.Errorf("want '6th time' (5 prior + current), got %q", got)
	}
}

func TestConnectionInsight_LowercasesAndTrimsTopics(t *testing.T) {
	svc, repo := newInsightFixture(map[string]int{})

	_, err := svc.BuildConnectionInsight(context.Background(), uuid.New(), uuid.New(), []string{"  Work Stress ", ""})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(repo.gotTopics) != 1 || repo.gotTopics[0] != "work stress" {
		t.Errorf("want lowercased trimmed topics [work stress], got %v", repo.gotTopics)
	}
}

func TestConnectionInsight_RepoError_Propagates(t *testing.T) {
	svc, repo := newInsightFixture(nil)
	repo.err = errors.New("db down")

	got, err := svc.BuildConnectionInsight(context.Background(), uuid.New(), uuid.New(), []string{"work"})
	if err == nil {
		t.Fatal("want error when repo fails")
	}
	if got != "" {
		t.Errorf("on error insight must be empty, got %q", got)
	}
}

func TestOrdinal(t *testing.T) {
	cases := map[int]string{
		1: "1st", 2: "2nd", 3: "3rd", 4: "4th",
		11: "11th", 12: "12th", 13: "13th",
		21: "21st", 22: "22nd", 23: "23rd", 111: "111th",
	}
	for n, want := range cases {
		if got := ordinal(n); got != want {
			t.Errorf("ordinal(%d): want %s, got %s", n, want, got)
		}
	}
}
