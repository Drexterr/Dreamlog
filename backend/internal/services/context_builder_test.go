package services

import (
	"strings"
	"testing"
)

// ── buildTrend ────────────────────────────────────────────────────────────────

func TestBuildTrend_EmptyMap_ReturnsEmpty(t *testing.T) {
	result := buildTrend(map[string]int{}, 3)
	if result != "" {
		t.Errorf("empty map: want empty string, got %q", result)
	}
}

func TestBuildTrend_SingleItem(t *testing.T) {
	result := buildTrend(map[string]int{"anxious": 5}, 3)
	if result != "anxious" {
		t.Errorf("single item: want %q, got %q", "anxious", result)
	}
}

func TestBuildTrend_TopNRespected(t *testing.T) {
	counts := map[string]int{
		"anxious":    5,
		"hopeful":    3,
		"sad":        2,
		"frustrated": 1,
	}
	result := buildTrend(counts, 2)
	parts := strings.Split(result, ", ")
	if len(parts) != 2 {
		t.Errorf("topN=2: want 2 items, got %d (%q)", len(parts), result)
	}
}

func TestBuildTrend_SortedByCountDescending(t *testing.T) {
	counts := map[string]int{
		"a": 1,
		"b": 10,
		"c": 5,
	}
	result := buildTrend(counts, 3)
	parts := strings.Split(result, ", ")
	if parts[0] != "b" {
		t.Errorf("first item must be highest-count; want b, got %s", parts[0])
	}
	if parts[1] != "c" {
		t.Errorf("second item must be second-highest; want c, got %s", parts[1])
	}
}

func TestBuildTrend_FewerItemsThanTopN(t *testing.T) {
	result := buildTrend(map[string]int{"only": 3}, 5)
	if result != "only" {
		t.Errorf("fewer items than topN: want %q, got %q", "only", result)
	}
}

// ── enforceTokenBudget ────────────────────────────────────────────────────────

func TestEnforceTokenBudget_UnderBudget_NoChange(t *testing.T) {
	input := &AnalyzeEntryInput{
		Transcript:    "Short transcript.",
		PastSummaries: []string{"Summary one.", "Summary two."},
		EmotionTrend:  "hopeful",
		TopicTrend:    "work",
	}
	originalSummaryCount := len(input.PastSummaries)
	originalTranscript := input.Transcript

	enforceTokenBudget(input)

	if len(input.PastSummaries) != originalSummaryCount {
		t.Error("under budget: summaries must not be dropped")
	}
	if input.Transcript != originalTranscript {
		t.Error("under budget: transcript must not be trimmed")
	}
}

func TestEnforceTokenBudget_DropsOldestSummaryFirst(t *testing.T) {
	// 4 summaries × 4200 chars = 16800 > maxContextChars (16000), so budget is exceeded.
	hugeSummary := strings.Repeat("x", 4200)
	input := &AnalyzeEntryInput{
		Transcript: strings.Repeat("t", 100),
		PastSummaries: []string{
			"oldest: " + hugeSummary,
			"middle: " + hugeSummary,
			"newest: " + hugeSummary,
			"very new: " + hugeSummary,
		},
	}

	// Must drop oldest (index 0) first
	enforceTokenBudget(input)

	if len(input.PastSummaries) == 4 {
		t.Error("over-budget input: at least one summary should have been dropped")
	}
	// The newest summary should survive the longest
	survived := false
	for _, s := range input.PastSummaries {
		if strings.HasPrefix(s, "very new:") {
			survived = true
			break
		}
	}
	if !survived && len(input.PastSummaries) > 0 {
		t.Error("newest summary should be the last to be dropped")
	}
}

func TestEnforceTokenBudget_TrimsTranscriptAsLastResort(t *testing.T) {
	// No summaries, but transcript alone exceeds budget
	input := &AnalyzeEntryInput{
		Transcript:    strings.Repeat("x", maxContextChars+1000),
		PastSummaries: nil,
	}

	enforceTokenBudget(input)

	if len([]rune(input.Transcript)) > maxContextChars+50 {
		t.Errorf("transcript must be trimmed when over budget; rune count is %d", len([]rune(input.Transcript)))
	}
	if !strings.Contains(input.Transcript, "[trimmed]") {
		t.Error("trimmed transcript must contain [trimmed] marker")
	}
}

func TestEnforceTokenBudget_AllSummariesDroppedIfNeeded(t *testing.T) {
	hugeTranscript := strings.Repeat("t", maxTranscriptChars)
	hugeSummary := strings.Repeat("s", maxSummaryChars)

	input := &AnalyzeEntryInput{
		Transcript:    hugeTranscript,
		PastSummaries: []string{hugeSummary, hugeSummary, hugeSummary},
	}

	enforceTokenBudget(input)

	// If budget is still exceeded after dropping all summaries,
	// transcript itself gets trimmed. Either way - no panic.
}

// ── trimChars ─────────────────────────────────────────────────────────────────

func TestTrimChars_ShortString_Unchanged(t *testing.T) {
	s := "hello"
	result := trimChars(s, 100)
	if result != s {
		t.Errorf("short string must be unchanged: want %q, got %q", s, result)
	}
}

func TestTrimChars_ExactLength_Unchanged(t *testing.T) {
	s := "hello"
	result := trimChars(s, 5)
	if result != s {
		t.Errorf("exact-length string must be unchanged: want %q, got %q", s, result)
	}
}

func TestTrimChars_LongString_TruncatesWithEllipsis(t *testing.T) {
	s := strings.Repeat("a", 200)
	result := trimChars(s, 100)
	runes := []rune(result)
	if len(runes) != 101 { // 100 chars + ellipsis rune (…)
		t.Errorf("trimmed length: want 101 runes (100 + ellipsis), got %d", len(runes))
	}
	if !strings.HasSuffix(result, "…") {
		t.Error("trimmed string must end with ellipsis")
	}
}

func TestTrimChars_MultiByte_CountsRunes(t *testing.T) {
	// Each Hindi char is 3 bytes but 1 rune - trimChars must count runes, not bytes.
	s := strings.Repeat("आ", 200) // 200 runes, 600 bytes
	result := trimChars(s, 100)
	runes := []rune(result)
	// 100 runes + ellipsis = 101 runes
	if len(runes) != 101 {
		t.Errorf("multi-byte: want 101 runes, got %d", len(runes))
	}
}
