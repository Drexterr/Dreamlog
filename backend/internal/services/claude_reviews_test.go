package services

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	appconfig "github.com/dreamlog/backend/internal/config"
)

// chatStub serves an OpenAI-compatible chat completion whose message content is body.
func chatStub(t *testing.T, body string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"choices": []map[string]interface{}{
				{"message": map[string]string{"content": body}, "finish_reason": "stop"},
			},
		})
	}))
}

func stubModeClaude() *ClaudeService {
	return NewClaudeService(&appconfig.AnthropicConfig{StubAnalysis: true, Model: "stub"})
}

// ── GenerateWeeklyReview ──────────────────────────────────────────────────────

func TestGenerateWeeklyReview_StubMode_ReturnsValidOutput(t *testing.T) {
	out, err := stubModeClaude().GenerateWeeklyReview(context.Background(), WeeklyReviewPromptInput{
		Name: "Asha", WeekLabel: "Jun 1 – Jun 7", EntryCount: 3,
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.Narrative == "" {
		t.Error("stub weekly review must include a narrative")
	}
}

func TestGenerateWeeklyReview_ParsesJSON(t *testing.T) {
	srv := chatStub(t, `{"narrative":"A steady week.","top_emotions":["calm","hopeful"]}`)
	defer srv.Close()

	out, err := newClaudeWithServer(srv).GenerateWeeklyReview(context.Background(), WeeklyReviewPromptInput{
		Name: "Asha", WeekLabel: "Jun 1 – Jun 7", EntryCount: 3,
		DailyMoods: []string{"Mon Jun 1: mood 65"}, Summaries: []string{"A good day."}, TopEmotions: []string{"calm"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.Narrative != "A steady week." {
		t.Errorf("narrative: got %q", out.Narrative)
	}
	if len(out.TopEmotions) != 2 {
		t.Errorf("top_emotions: got %v", out.TopEmotions)
	}
}

func TestGenerateWeeklyReview_MalformedJSON_ReturnsError(t *testing.T) {
	srv := chatStub(t, `not json at all`)
	defer srv.Close()

	if _, err := newClaudeWithServer(srv).GenerateWeeklyReview(context.Background(), WeeklyReviewPromptInput{}); err == nil {
		t.Fatal("malformed JSON must return an error, not panic")
	}
}

func TestGenerateWeeklyReview_NoAPIKey_ReturnsError(t *testing.T) {
	svc := NewClaudeService(&appconfig.AnthropicConfig{StubAnalysis: false, APIKey: ""})
	if _, err := svc.GenerateWeeklyReview(context.Background(), WeeklyReviewPromptInput{}); err == nil {
		t.Fatal("missing API key must return an error")
	}
}

// ── GenerateYearInReview ──────────────────────────────────────────────────────

func TestGenerateYearInReview_StubMode_ReturnsValidOutput(t *testing.T) {
	out, err := stubModeClaude().GenerateYearInReview(context.Background(), YearInReviewPromptInput{
		Name: "Asha", Year: 2025, EntryCount: 80, AvgMood: 67,
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.Narrative == "" {
		t.Error("stub year in review must include a narrative")
	}
}

func TestGenerateYearInReview_ParsesJSON(t *testing.T) {
	srv := chatStub(t, `{"narrative":"A year of growth.","top_emotions":["hopeful"],"top_topics":["work","family"]}`)
	defer srv.Close()

	out, err := newClaudeWithServer(srv).GenerateYearInReview(context.Background(), YearInReviewPromptInput{
		Name: "Asha", Year: 2025, EntryCount: 80, AvgMood: 67,
		MonthlyArc: []string{"Jan 2025: mood 65 (4 entries)"}, Summaries: []string{"s1"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.Narrative != "A year of growth." || len(out.TopTopics) != 2 {
		t.Errorf("unexpected output: %+v", out)
	}
}

func TestGenerateYearInReview_MalformedJSON_ReturnsError(t *testing.T) {
	srv := chatStub(t, `{{{`)
	defer srv.Close()
	if _, err := newClaudeWithServer(srv).GenerateYearInReview(context.Background(), YearInReviewPromptInput{}); err == nil {
		t.Fatal("malformed JSON must return an error")
	}
}

// ── GenerateChapterSummary ────────────────────────────────────────────────────

func TestGenerateChapterSummary_StubMode_ReturnsValidOutput(t *testing.T) {
	out, err := stubModeClaude().GenerateChapterSummary(context.Background(), ChapterSummaryPromptInput{
		Name: "Asha", Title: "Bangalore Chapter", StartDate: "2024-01-01", EntryCount: 10, AvgMood: 60,
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.Summary == "" {
		t.Error("stub chapter summary must include a summary")
	}
}

func TestGenerateChapterSummary_ParsesJSON(t *testing.T) {
	srv := chatStub(t, `{"summary":"This chapter was about new beginnings."}`)
	defer srv.Close()

	out, err := newClaudeWithServer(srv).GenerateChapterSummary(context.Background(), ChapterSummaryPromptInput{
		Name: "Asha", Title: "Bangalore Chapter", StartDate: "2024-01-01", EndDate: "2024-12-31",
		EntryCount: 10, AvgMood: 60, TopEmotions: []string{"curious"}, Summaries: []string{"s1"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.Summary, "new beginnings") {
		t.Errorf("summary: got %q", out.Summary)
	}
}

// ── Prompt builders (ADR-003: all prompts in prompts.go) ─────────────────────

func TestWeeklyReviewPrompts_ContainInputFields(t *testing.T) {
	system := buildWeeklyReviewSystemPrompt()
	if system == "" {
		t.Fatal("weekly review system prompt must not be empty")
	}

	user := buildWeeklyReviewUserPrompt(WeeklyReviewPromptInput{
		Name: "Asha", WeekLabel: "Jun 1 – Jun 7", EntryCount: 3,
		DailyMoods:  []string{"Mon Jun 1: mood 65"},
		Summaries:   []string{"Felt grounded after a long walk."},
		TopEmotions: []string{"calm", "hopeful"},
	})
	for _, want := range []string{"Asha", "Jun 1", "mood 65", "long walk", "calm"} {
		if !strings.Contains(user, want) {
			t.Errorf("weekly review user prompt must contain %q", want)
		}
	}
}

func TestYearInReviewPrompts_ContainInputFields(t *testing.T) {
	if buildYearInReviewSystemPrompt() == "" {
		t.Fatal("year in review system prompt must not be empty")
	}
	user := buildYearInReviewUserPrompt(YearInReviewPromptInput{
		Name: "Asha", Year: 2025, EntryCount: 80, AvgMood: 67,
		MonthlyArc:  []string{"Jan 2025: mood 65 (4 entries)"},
		TopEmotions: []string{"hopeful"},
		TopTopics:   []string{"work"},
		Summaries:   []string{"A representative summary."},
	})
	for _, want := range []string{"Asha", "2025", "Jan 2025", "hopeful", "work", "representative"} {
		if !strings.Contains(user, want) {
			t.Errorf("year in review user prompt must contain %q", want)
		}
	}
}

func TestChapterSummaryPrompts_ContainInputFields(t *testing.T) {
	if buildChapterSummarySystemPrompt() == "" {
		t.Fatal("chapter summary system prompt must not be empty")
	}
	user := buildChapterSummaryUserPrompt(ChapterSummaryPromptInput{
		Name: "Asha", Title: "Bangalore Chapter", Description: "Moving to a new city",
		StartDate: "2024-01-01", EndDate: "2024-12-31",
		EntryCount: 10, AvgMood: 60,
		TopEmotions: []string{"curious"}, Summaries: []string{"First day in the new flat."},
	})
	for _, want := range []string{"Bangalore Chapter", "new city", "2024-01-01", "curious", "new flat"} {
		if !strings.Contains(user, want) {
			t.Errorf("chapter summary user prompt must contain %q", want)
		}
	}
}

func TestTherapyPostSessionPrompt_ContainsHistoryAndSchema(t *testing.T) {
	prompt := buildTherapyPostSessionPrompt([]string{"[user]: I felt anxious today.", "[assistant]: That sounds heavy."})
	for _, want := range []string{"I felt anxious today", "mood_score", "emotional_tone"} {
		if !strings.Contains(prompt, want) {
			t.Errorf("therapy post-session prompt must contain %q", want)
		}
	}
}

// ── TherapySummary ────────────────────────────────────────────────────────────

func TestTherapySummary_ParsesAllFields(t *testing.T) {
	srv := chatStub(t, `{
		"mood_score": 70,
		"emotional_tone": [{"emotion":"relief","intensity":0.6}],
		"topics": ["work stress"],
		"key_insights": ["sleep affects mood"],
		"session_narrative": "A session about work stress."
	}`)
	defer srv.Close()

	out, err := newClaudeWithServer(srv).TherapySummary(context.Background(), TherapySummaryInput{
		Messages: []string{"[user]: stressful week", "[assistant]: tell me more"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.MoodScore != 70 || len(out.EmotionalTone) != 1 || out.SessionNarrative == "" {
		t.Errorf("unexpected analysis: %+v", out)
	}
}

func TestTherapySummary_MoodScoreClampedToBounds(t *testing.T) {
	for _, tc := range []struct{ raw, want int }{{0, 1}, {-5, 1}, {150, 100}} {
		srv := chatStub(t, `{"mood_score": `+strconv.Itoa(tc.raw)+`, "session_narrative": "x"}`)
		out, err := newClaudeWithServer(srv).TherapySummary(context.Background(), TherapySummaryInput{Messages: []string{"[user]: hi"}})
		srv.Close()
		if err != nil {
			t.Fatal(err)
		}
		if out.MoodScore != tc.want {
			t.Errorf("mood_score %d must clamp to %d, got %d", tc.raw, tc.want, out.MoodScore)
		}
	}
}

func TestTherapySummary_MalformedJSON_ReturnsError(t *testing.T) {
	srv := chatStub(t, `narrative only, no json`)
	defer srv.Close()
	if _, err := newClaudeWithServer(srv).TherapySummary(context.Background(), TherapySummaryInput{Messages: []string{"[user]: hi"}}); err == nil {
		t.Fatal("malformed JSON must return an error")
	}
}

func TestTherapySummary_StubMode_ReturnsValidAnalysis(t *testing.T) {
	out, err := stubModeClaude().TherapySummary(context.Background(), TherapySummaryInput{Messages: []string{"[user]: hi"}})
	if err != nil {
		t.Fatal(err)
	}
	if out.MoodScore < 1 || out.MoodScore > 100 || out.SessionNarrative == "" {
		t.Errorf("stub analysis must be valid: %+v", out)
	}
}

// ── CrisisResponse exported wrapper ──────────────────────────────────────────

func TestCrisisResponse_ExportedWrapper_MatchesInternal(t *testing.T) {
	for _, country := range []string{"IN", "US", ""} {
		if CrisisResponse(country) != buildCrisisResponse(country) {
			t.Errorf("CrisisResponse(%q) must match buildCrisisResponse", country)
		}
	}
}
