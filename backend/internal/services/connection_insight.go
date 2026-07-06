package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// topicHistorian is the slice of AnalysisRepository the insight builder needs.
type topicHistorian interface {
	CountTopicOccurrences(ctx context.Context, userID, excludeEntryID uuid.UUID, topics []string, since time.Time) (map[string]int, error)
}

// ConnectionInsightService produces the occasional cross-entry pattern insight
// ("this is the 3rd time X has come up this month"). It is deterministic - no
// AI call - and intentionally intermittent: it only fires when a topic from
// the current entry has already appeared in at least 2 prior entries within
// the lookback window, so the reward stays variable rather than routine.
type ConnectionInsightService struct {
	repo           topicHistorian
	lookback       time.Duration
	minPriorCount  int
}

func NewConnectionInsightService(repo topicHistorian) *ConnectionInsightService {
	return &ConnectionInsightService{
		repo:          repo,
		lookback:      30 * 24 * time.Hour,
		minPriorCount: 2, // current entry makes it the 3rd+ occurrence
	}
}

// BuildConnectionInsight returns an insight sentence, or "" when no recurring
// pattern crossed the threshold. Errors are returned for logging but callers
// should treat them as non-fatal.
func (s *ConnectionInsightService) BuildConnectionInsight(ctx context.Context, userID, entryID uuid.UUID, topics []string) (string, error) {
	if len(topics) == 0 {
		return "", nil
	}

	lowered := make([]string, 0, len(topics))
	for _, t := range topics {
		t = strings.TrimSpace(t)
		if t != "" {
			lowered = append(lowered, strings.ToLower(t))
		}
	}
	if len(lowered) == 0 {
		return "", nil
	}

	since := time.Now().Add(-s.lookback)
	counts, err := s.repo.CountTopicOccurrences(ctx, userID, entryID, lowered, since)
	if err != nil {
		return "", fmt.Errorf("connection insight: count topics: %w", err)
	}

	// Pick the most recurrent topic; prefer the entry's own topic ordering on ties.
	bestTopic := ""
	bestCount := 0
	for _, t := range lowered {
		if c := counts[t]; c > bestCount {
			bestTopic = t
			bestCount = c
		}
	}
	if bestCount < s.minPriorCount {
		return "", nil
	}

	total := bestCount + 1 // include the current entry
	return fmt.Sprintf(
		"This is the %s time %s has come up in the last month. It seems to be asking for your attention.",
		ordinal(total), bestTopic,
	), nil
}

// ordinal renders 1 → "1st", 2 → "2nd", 3 → "3rd", 4 → "4th", ...
func ordinal(n int) string {
	if n%100 >= 11 && n%100 <= 13 {
		return fmt.Sprintf("%dth", n)
	}
	switch n % 10 {
	case 1:
		return fmt.Sprintf("%dst", n)
	case 2:
		return fmt.Sprintf("%dnd", n)
	case 3:
		return fmt.Sprintf("%drd", n)
	default:
		return fmt.Sprintf("%dth", n)
	}
}
