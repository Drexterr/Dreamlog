package services

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	appconfig "github.com/dreamlog/backend/internal/config"
)

func TestTranscriptionClient_MultipartAndResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/audio/transcriptions" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-only" {
			t.Error("missing provider authorization")
		}
		if err := r.ParseMultipartForm(1024); err != nil {
			t.Error(err)
			return
		}
		defer r.MultipartForm.RemoveAll()
		if r.FormValue("model") != "whisper-1" || r.FormValue("response_format") != "verbose_json" {
			t.Error("wrong transcription settings")
		}
		file, header, err := r.FormFile("file")
		if err != nil {
			t.Error(err)
			return
		}
		defer file.Close()
		body, _ := io.ReadAll(file)
		if header.Filename != "clip.m4a" || string(body) != "fixture-audio" {
			t.Error("audio changed in transit")
		}
		_, _ = io.WriteString(w, `{"text":"A calm day.","language":"english","duration":3.5,"segments":[{"id":0,"text":"A calm day."}]}`)
	}))
	defer server.Close()
	svc := NewTranscriptionService(&appconfig.OpenAIConfig{BaseURL: server.URL, APIKey: "test-only"})
	result, err := svc.Transcribe(context.Background(), strings.NewReader("fixture-audio"), "clip.m4a")
	if err != nil {
		t.Fatal(err)
	}
	if result.Text != "A calm day." || result.Language != "english" || result.Duration != 3.5 || len(result.Segments) != 1 {
		t.Fatalf("incomplete response: %+v", result)
	}
}

func TestTranscriptionClient_ProviderFailures(t *testing.T) {
	for _, test := range []struct {
		name       string
		status     int
		body, want string
	}{
		{"unauthorized", 401, "denied", "401"},
		{"rate_limited", 429, "rate limited", "429"},
		{"server_failure", 500, "unavailable", "500"},
		{"malformed_json", 200, "{broken", "unmarshal"},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(test.status)
				_, _ = io.WriteString(w, test.body)
			}))
			defer server.Close()
			svc := NewTranscriptionService(&appconfig.OpenAIConfig{BaseURL: server.URL})
			result, err := svc.Transcribe(context.Background(), strings.NewReader("audio"), "clip.m4a")
			if result != nil || err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("result=%+v err=%v", result, err)
			}
		})
	}
}

type brokenAudioReader struct{}

func (brokenAudioReader) Read([]byte) (int, error) { return 0, errors.New("audio read failed") }

func TestTranscriptionClient_UnreadableAudio(t *testing.T) {
	svc := NewTranscriptionService(&appconfig.OpenAIConfig{BaseURL: "http://127.0.0.1:1"})
	_, err := svc.Transcribe(context.Background(), brokenAudioReader{}, "clip.m4a")
	if err == nil || !strings.Contains(err.Error(), "copy audio") {
		t.Fatalf("expected local read failure: %v", err)
	}
}

func TestTranscriptionClient_CancelledRequest(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	svc := NewTranscriptionService(&appconfig.OpenAIConfig{BaseURL: "http://127.0.0.1:1"})
	_, err := svc.Transcribe(ctx, strings.NewReader("audio"), "clip.m4a")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context cancellation: %v", err)
	}
}

func TestTranscriptionClient_InvalidProviderURL(t *testing.T) {
	svc := NewTranscriptionService(&appconfig.OpenAIConfig{BaseURL: "://invalid"})
	_, err := svc.Transcribe(context.Background(), strings.NewReader("audio"), "clip.m4a")
	if err == nil || !strings.Contains(err.Error(), "new request") {
		t.Fatalf("expected URL validation error: %v", err)
	}
}
