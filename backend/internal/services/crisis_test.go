package services

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"unicode/utf8"

	appconfig "github.com/dreamlog/backend/internal/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

// claudeYesServer returns a chat completion that says "yes".
func claudeYesServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(chatResponse{
			Choices: []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
				FinishReason string `json:"finish_reason"`
			}{
				{Message: struct {
					Content string `json:"content"`
				}{Content: "yes"}},
			},
		})
	}))
}

// claudeNoServer returns a chat completion that says "no".
func claudeNoServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(chatResponse{
			Choices: []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
				FinishReason string `json:"finish_reason"`
			}{
				{Message: struct {
					Content string `json:"content"`
				}{Content: "no"}},
			},
		})
	}))
}

// claudeErrorServer returns a 500 error on every request.
func claudeErrorServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":{"message":"internal error","type":"server_error"}}`))
	}))
}

func newClaudeWithServer(srv *httptest.Server) *ClaudeService {
	return NewClaudeService(&appconfig.AnthropicConfig{
		BaseURL:      srv.URL,
		APIKey:       "test-key",
		Model:        "test-model",
		StubAnalysis: false,
	})
}

// ── Stage 1: high-certainty keyword tests ────────────────────────────────────

func TestCrisisStage1_AllHighCertaintyPhrases(t *testing.T) {
	// No Claude server needed - Stage 1 never calls the API.
	detector := NewCrisisDetector(nil, nil)
	ctx := context.Background()

	for _, phrase := range highCertaintyPhrases {
		transcript := "Today I thought about how I want to " + phrase + " because life is hard."
		result, err := detector.Screen(ctx, transcript, "IN")
		if err != nil {
			t.Errorf("phrase %q: unexpected error: %v", phrase, err)
			continue
		}
		if !result.Detected {
			t.Errorf("phrase %q: expected crisis detection, got none", phrase)
		}
		if result.Response == "" {
			t.Errorf("phrase %q: crisis response must not be empty", phrase)
		}
	}
}

func TestCrisisStage1_CaseInsensitive(t *testing.T) {
	detector := NewCrisisDetector(nil, nil)
	ctx := context.Background()

	phrases := []string{
		"KILL MYSELF",
		"Kill Myself",
		"WANT TO DIE",
		"Hurt Myself",
	}
	for _, phrase := range phrases {
		result, err := detector.Screen(ctx, phrase, "IN")
		if err != nil {
			t.Fatalf("unexpected error for %q: %v", phrase, err)
		}
		if !result.Detected {
			t.Errorf("uppercase phrase %q should trigger crisis", phrase)
		}
	}
}

func TestCrisisStage1_NormalText_NoMatch(t *testing.T) {
	detector := NewCrisisDetector(nil, nil)
	ctx := context.Background()

	benign := []string{
		"Today was a really tough day at work.",
		"I feel exhausted and stressed about the project deadline.",
		"My relationship has been difficult lately.",
		"I'm feeling a bit down but I'll be okay.",
		"The meeting was killing me with boredom.",
		"I could kill for a coffee right now.",
		"I'm dying of laughter.",
	}

	for _, transcript := range benign {
		result, err := detector.Screen(ctx, transcript, "IN")
		if err != nil {
			t.Fatalf("unexpected error for %q: %v", transcript, err)
		}
		if result.Detected {
			t.Errorf("benign text %q should not trigger crisis", transcript)
		}
	}
}

func TestCrisisStage1_EmbeddedPhrase(t *testing.T) {
	detector := NewCrisisDetector(nil, nil)
	ctx := context.Background()

	// Phrase buried in a longer transcript
	transcript := strings.Repeat("I talked to my friends and family. ", 20) +
		"I just want to kill myself sometimes when everything feels overwhelming." +
		strings.Repeat(" I don't know what to do.", 10)

	result, err := detector.Screen(ctx, transcript, "IN")
	if err != nil {
		t.Fatal(err)
	}
	if !result.Detected {
		t.Error("phrase embedded in long transcript must still be detected")
	}
}

// ── Stage 2: Claude confirmation tests ───────────────────────────────────────

func TestCrisisStage2_ClaudeConfirmsYes(t *testing.T) {
	srv := claudeYesServer(t)
	defer srv.Close()

	detector := NewCrisisDetector(newClaudeWithServer(srv), nil)
	ctx := context.Background()

	// ambiguous phrase triggers Stage 2
	transcript := "I feel like it's not worth living anymore."
	result, err := detector.Screen(ctx, transcript, "IN")
	if err != nil {
		t.Fatal(err)
	}
	if !result.Detected {
		t.Error("Claude confirmed yes - should be crisis")
	}
	if result.Response == "" {
		t.Error("crisis response must not be empty")
	}
}

func TestCrisisStage2_ClaudeConfirmsNo(t *testing.T) {
	srv := claudeNoServer(t)
	defer srv.Close()

	detector := NewCrisisDetector(newClaudeWithServer(srv), nil)
	ctx := context.Background()

	// ambiguous phrase, but Claude says no
	transcript := "I feel like I can't go on with this project anymore, it's too much."
	result, err := detector.Screen(ctx, transcript, "IN")
	if err != nil {
		t.Fatal(err)
	}
	if result.Detected {
		t.Error("Claude confirmed no - should not be crisis")
	}
}

func TestCrisisStage2_ClaudeUnreachable_DefaultsToCrisis(t *testing.T) {
	srv := claudeErrorServer(t)
	defer srv.Close()

	detector := NewCrisisDetector(newClaudeWithServer(srv), nil)
	ctx := context.Background()

	transcript := "I feel like nobody cares if I live or die."
	result, err := detector.Screen(ctx, transcript, "IN")
	// Screen itself must NOT return an error - it absorbs the Claude error and fails safe.
	if err != nil {
		t.Fatalf("Screen must not surface Claude errors: %v", err)
	}
	if !result.Detected {
		t.Error("Claude unreachable must default to crisis (fail-safe)")
	}
}

func TestCrisisStage2_ClaudeUnreachable_FailSafeIsLogged(t *testing.T) {
	srv := claudeErrorServer(t)
	defer srv.Close()

	core, logs := observer.New(zap.WarnLevel)
	detector := NewCrisisDetector(newClaudeWithServer(srv), zap.New(core))
	ctx := context.Background()

	result, err := detector.Screen(ctx, "I feel like nobody cares if I live or die.", "IN")
	if err != nil {
		t.Fatalf("Screen must not surface Claude errors: %v", err)
	}
	if !result.Detected {
		t.Fatal("must still fail safe to crisis regardless of logging")
	}
	if logs.Len() != 1 {
		t.Fatalf("the fail-safe branch must be logged exactly once so an outage is operator-visible, got %d entries", logs.Len())
	}
	if !strings.Contains(logs.All()[0].Message, "failing safe to crisis") {
		t.Errorf("log message must explain the fail-safe, got %q", logs.All()[0].Message)
	}
}

func TestCrisisStage2_ContextCancelled_DefaultsToCrisis(t *testing.T) {
	// Server that never responds (simulates timeout)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done() // block until client cancels
	}))
	defer srv.Close()

	detector := NewCrisisDetector(newClaudeWithServer(srv), nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	transcript := "I feel like I can't go on."
	result, err := detector.Screen(ctx, transcript, "IN")
	if err != nil {
		t.Fatalf("Screen must not surface context errors: %v", err)
	}
	if !result.Detected {
		t.Error("cancelled context must default to crisis (fail-safe)")
	}
}

func TestCrisisStage2_OnlyCallsClaudeOnce(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(chatResponse{
			Choices: []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
				FinishReason string `json:"finish_reason"`
			}{
				{Message: struct {
					Content string `json:"content"`
				}{Content: "no"}},
			},
		})
	}))
	defer srv.Close()

	detector := NewCrisisDetector(newClaudeWithServer(srv), nil)
	ctx := context.Background()

	// Multiple ambiguous phrases - Claude should only be called once.
	transcript := "I feel like I can't go on and it's not worth living and better off without me."
	_, err := detector.Screen(ctx, transcript, "IN")
	if err != nil {
		t.Fatal(err)
	}
	if callCount != 1 {
		t.Errorf("Claude must be called exactly once even with multiple ambiguous phrases; got %d calls", callCount)
	}
}

// ── Crisis response content test ─────────────────────────────────────────────

func TestCrisisResponse_ContainsCrisisResources(t *testing.T) {
	usResp := buildCrisisResponse("US")
	if !strings.Contains(usResp, "988") {
		t.Error("US crisis response must contain crisis line 988")
	}

	inResp := buildCrisisResponse("IN")
	if !strings.Contains(inResp, "iCall") {
		t.Error("India crisis response must contain iCall resource")
	}
	if !strings.Contains(inResp, "Vandrevala") {
		t.Error("India crisis response must contain Vandrevala Foundation")
	}

	// Unknown or empty country must fall back to international resources.
	for _, country := range []string{"", "ZZ"} {
		resp := buildCrisisResponse(country)
		if !strings.Contains(resp, "findahelpline") {
			t.Errorf("crisis response for country %q must contain international resource link", country)
		}
	}

	// Every localised response must include an emergency number line.
	for country := range countryHelplines {
		if !strings.Contains(buildCrisisResponse(country), "Emergency:") {
			t.Errorf("crisis response for %s must include an emergency number", country)
		}
	}
}

// claudeReplyServer returns a chat completion with the given content and
// records the user prompt it received.
func claudeReplyServer(t *testing.T, reply string, gotPrompt *string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Messages []struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"messages"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		if gotPrompt != nil && len(req.Messages) > 0 {
			*gotPrompt = req.Messages[len(req.Messages)-1].Content
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(chatResponse{
			Choices: []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
				FinishReason string `json:"finish_reason"`
			}{
				{Message: struct {
					Content string `json:"content"`
				}{Content: reply}},
			},
		})
	}))
}

// H6: iOS Smart Punctuation / Whisper curly quotes must still match Stage 1.
func TestCrisisStage1_TypographicApostrophes(t *testing.T) {
	detector := NewCrisisDetector(nil, nil)
	for _, transcript := range []string{
		"I don’t want to be here anymore",
		"I DON‘T   WANT TO EXIST",
		"honestly I just want to end it all",
	} {
		result, err := detector.Screen(context.Background(), transcript, "IN")
		if err != nil {
			t.Fatal(err)
		}
		if !result.Detected {
			t.Errorf("%q must be detected", transcript)
		}
	}
}

// H6: suicide mentions, Hindi/Hinglish and "kms" reach Stage 2 (fail-safe on error).
func TestCrisisStage1b_NewSignalsReachStage2(t *testing.T) {
	srv := claudeErrorServer(t)
	defer srv.Close()
	detector := NewCrisisDetector(newClaudeWithServer(srv), nil)
	for _, transcript := range []string{
		"I've been feeling suicidal lately",
		"main ab jeena nahi chahti",
		"मैं मरना चाहता हूँ",
		"honestly i might just kms",
	} {
		result, err := detector.Screen(context.Background(), transcript, "IN")
		if err != nil {
			t.Fatal(err)
		}
		if !result.Detected {
			t.Errorf("%q must reach Stage 2 and fail safe to crisis", transcript)
		}
	}
	// Whole-word only: distances are not a crisis signal.
	if res, _ := NewCrisisDetector(nil, nil).Screen(context.Background(), "ran 10 kms today", "IN"); res.Detected {
		t.Error(`"10 kms" must not trigger the kms token`)
	}
}

// H5: only an explicit "no" clears an ambiguous entry.
func TestCrisisStage2_UnclearReplies_FailSafe(t *testing.T) {
	for _, reply := range []string{"", "I can't help with that.", "Not sure", "maybe"} {
		srv := claudeReplyServer(t, reply, nil)
		detector := NewCrisisDetector(newClaudeWithServer(srv), nil)
		result, err := detector.Screen(context.Background(), "I can't go on like this", "IN")
		srv.Close()
		if err != nil {
			t.Fatal(err)
		}
		if !result.Detected {
			t.Errorf("reply %q must be treated as crisis", reply)
		}
	}
	for _, reply := range []string{"no", "No.", "**No**"} {
		srv := claudeReplyServer(t, reply, nil)
		detector := NewCrisisDetector(newClaudeWithServer(srv), nil)
		result, _ := detector.Screen(context.Background(), "I can't go on with this project", "IN")
		srv.Close()
		if result.Detected {
			t.Errorf("explicit %q must clear the entry", reply)
		}
	}
}

// H5: a signal deep in a long entry must be inside what the classifier sees.
func TestCrisisStage2_LatePhraseIsSentToClassifier(t *testing.T) {
	var got string
	srv := claudeReplyServer(t, "no", &got)
	defer srv.Close()
	detector := NewCrisisDetector(newClaudeWithServer(srv), nil)

	transcript := strings.Repeat("एक लंबा दिन था और मैं थक गया। ", 400) + "sometimes I feel like there's no reason to live."
	if _, err := detector.Screen(context.Background(), transcript, "IN"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "no reason to live") {
		t.Error("the matched phrase must be included in the Stage 2 excerpt")
	}
	if !utf8.ValidString(got) {
		t.Error("Stage 2 excerpt must be valid UTF-8 (rune-safe)")
	}
}

// M11: the classifier prompt carries the injection guard and labels the data.
func TestCrisisClassifierPrompt_HasInjectionGuard(t *testing.T) {
	if !strings.Contains(buildCrisisClassifierSystemPrompt(), promptInjectionGuard) {
		t.Error("classifier system prompt must include promptInjectionGuard")
	}
	if !strings.Contains(buildCrisisClassifierUserPrompt("x"), "USER-SUPPLIED DATA") {
		t.Error("classifier user prompt must label the entry as USER-SUPPLIED DATA")
	}
	if !strings.Contains(buildDeEscalationPrompt(), promptInjectionGuard) {
		t.Error("de-escalation prompt must include promptInjectionGuard")
	}
}
