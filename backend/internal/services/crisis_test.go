package services

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	appconfig "github.com/dreamlog/backend/internal/config"
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
	detector := NewCrisisDetector(nil)
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
	detector := NewCrisisDetector(nil)
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
	detector := NewCrisisDetector(nil)
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
	detector := NewCrisisDetector(nil)
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

	detector := NewCrisisDetector(newClaudeWithServer(srv))
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

	detector := NewCrisisDetector(newClaudeWithServer(srv))
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

	detector := NewCrisisDetector(newClaudeWithServer(srv))
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

func TestCrisisStage2_ContextCancelled_DefaultsToCrisis(t *testing.T) {
	// Server that never responds (simulates timeout)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done() // block until client cancels
	}))
	defer srv.Close()

	detector := NewCrisisDetector(newClaudeWithServer(srv))
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

	detector := NewCrisisDetector(newClaudeWithServer(srv))
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
	resp := buildCrisisResponse("US")
	if !strings.Contains(resp, "988") {
		t.Error("crisis response must contain US crisis line 988")
	}
	if !strings.Contains(resp, "iCall") {
		t.Error("crisis response must contain India iCall resource")
	}
	if !strings.Contains(resp, "Vandrevala") {
		t.Error("crisis response must contain India Vandrevala Foundation")
	}
	if !strings.Contains(resp, "findahelpline") {
		t.Error("crisis response must contain international resource link")
	}
}
