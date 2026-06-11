package services

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	appconfig "github.com/dreamlog/backend/internal/config"
	"github.com/dreamlog/backend/internal/models"
)

// ── fakes ─────────────────────────────────────────────────────────────────────

type fakeTTSStorage struct {
	uploadedKey  string
	uploadedType string
	uploadedBody []byte
	presignedURL string
}

func (f *fakeTTSStorage) Upload(_ context.Context, key, contentType string, body io.Reader) error {
	f.uploadedKey = key
	f.uploadedType = contentType
	f.uploadedBody, _ = io.ReadAll(body)
	return nil
}

func (f *fakeTTSStorage) PresignDownload(_ context.Context, key string, _ time.Duration) (string, error) {
	if f.presignedURL != "" {
		return f.presignedURL, nil
	}
	return "https://storage.example/" + key, nil
}

// ── provider selection ────────────────────────────────────────────────────────

func TestTTS_Disabled_ReturnsEmptyNoError(t *testing.T) {
	svc := NewTTSService(&appconfig.OpenAIConfig{}, &appconfig.AzureTTSConfig{}, &fakeTTSStorage{})

	url, err := svc.Synthesize(context.Background(), "sess", "msg", "hello", models.PersonaComforting, "english")
	if err != nil {
		t.Fatalf("expected nil error when TTS disabled, got %v", err)
	}
	if url != "" {
		t.Fatalf("expected empty URL when TTS disabled, got %q", url)
	}
}

// ── Azure voice selection ─────────────────────────────────────────────────────

func TestTTS_AzureVoiceSelection_EnglishPersonas(t *testing.T) {
	svc := NewTTSService(&appconfig.OpenAIConfig{}, &appconfig.AzureTTSConfig{Key: "k", Region: "eastus"}, &fakeTTSStorage{})

	cases := []struct {
		persona models.TherapyPersona
		voice   string
		style   string
	}{
		{models.PersonaComforting, "en-US-JennyNeural", "empathetic"},
		{models.PersonaRational, "en-US-BrandonNeural", "calm"},
		{models.PersonaCBT, "en-US-AriaNeural", "chat"},
		{models.PersonaMindful, "en-US-SaraNeural", "gentle"},
	}
	for _, tc := range cases {
		voice, style := svc.selectAzureVoice(tc.persona, "english")
		if voice != tc.voice || style != tc.style {
			t.Errorf("persona %s: got (%s, %s), want (%s, %s)", tc.persona, voice, style, tc.voice, tc.style)
		}
	}
}

func TestTTS_AzureVoiceSelection_HindiPersonas(t *testing.T) {
	svc := NewTTSService(&appconfig.OpenAIConfig{}, &appconfig.AzureTTSConfig{Key: "k", Region: "eastus"}, &fakeTTSStorage{})

	cases := []struct {
		persona models.TherapyPersona
		voice   string
	}{
		{models.PersonaComforting, "hi-IN-SwaraNeural"},
		{models.PersonaRational, "hi-IN-MadhurNeural"},
		{models.PersonaCBT, "hi-IN-SwaraNeural"},
		{models.PersonaMindful, "hi-IN-SwaraNeural"},
	}
	for _, lang := range []string{"hindi", "hi", "Hindi"} {
		for _, tc := range cases {
			voice, style := svc.selectAzureVoice(tc.persona, lang)
			if voice != tc.voice {
				t.Errorf("persona %s lang %s: got voice %s, want %s", tc.persona, lang, voice, tc.voice)
			}
			if style != "" {
				t.Errorf("persona %s lang %s: Hindi voices must not get a style, got %q", tc.persona, lang, style)
			}
		}
	}
}

func TestTTS_AzureVoiceSelection_HDModePersonas(t *testing.T) {
	svc := NewTTSService(&appconfig.OpenAIConfig{}, &appconfig.AzureTTSConfig{
		Key: "k", Region: "eastus", UseHD: true,
	}, &fakeTTSStorage{})

	cases := []struct {
		persona models.TherapyPersona
		voice   string
	}{
		{models.PersonaComforting, "en-US-Jenny:DragonHDLatestNeural"},
		{models.PersonaRational, "en-US-Andrew2:DragonHDLatestNeural"},
		{models.PersonaCBT, "en-US-Aria:DragonHDLatestNeural"},
		{models.PersonaMindful, "en-US-Serena:DragonHDLatestNeural"},
	}
	// English / unknown-language sessions use the en-US HD voices
	// (Hindi sessions switch to the Indian HD map - see the next test).
	for _, lang := range []string{"english", ""} {
		for _, tc := range cases {
			voice, style := svc.selectAzureVoice(tc.persona, lang)
			if voice != tc.voice {
				t.Errorf("persona %s lang %q: got voice %s, want %s", tc.persona, lang, voice, tc.voice)
			}
			if style != "" {
				t.Errorf("persona %s lang %q: HD voices must not get a style, got %q", tc.persona, lang, style)
			}
		}
	}
}

func TestTTS_AzureVoiceSelection_HDModeIndianVoices(t *testing.T) {
	svc := NewTTSService(&appconfig.OpenAIConfig{}, &appconfig.AzureTTSConfig{
		Key: "k", Region: "centralindia", UseHD: true,
	}, &fakeTTSStorage{})

	cases := []struct {
		persona models.TherapyPersona
		voice   string
	}{
		{models.PersonaComforting, "en-IN-Diya:DragonHDLatestNeural"},
		{models.PersonaRational, "en-IN-Arjun:DragonHDLatestNeural"},
		{models.PersonaCBT, "en-IN-Diya:DragonHDLatestNeural"},
		{models.PersonaMindful, "en-IN-Diya:DragonHDLatestNeural"},
	}
	// Hindi sessions (detected or user preference "hindi") get the Indian HD voices.
	for _, tc := range cases {
		voice, style := svc.selectAzureVoice(tc.persona, "hindi")
		if voice != tc.voice {
			t.Errorf("persona %s: got voice %s, want %s", tc.persona, voice, tc.voice)
		}
		if style != "" {
			t.Errorf("persona %s: HD voices must not get a style, got %q", tc.persona, style)
		}
	}
	// English sessions stay on the en-US HD voices.
	voice, _ := svc.selectAzureVoice(models.PersonaComforting, "english")
	if voice != "en-US-Jenny:DragonHDLatestNeural" {
		t.Errorf("english HD session: got voice %s, want en-US-Jenny:DragonHDLatestNeural", voice)
	}
}

func TestTTS_AzureVoiceSelection_OverrideBeatsHDMode(t *testing.T) {
	svc := NewTTSService(&appconfig.OpenAIConfig{}, &appconfig.AzureTTSConfig{
		Key: "k", Region: "centralindia", UseHD: true,
		VoiceOverride: "en-IN-Aarti:DragonHDLatestNeural",
	}, &fakeTTSStorage{})

	voice, _ := svc.selectAzureVoice(models.PersonaRational, "english")
	if voice != "en-IN-Aarti:DragonHDLatestNeural" {
		t.Errorf("VoiceOverride must win over UseHD, got %s", voice)
	}
}

func TestTTS_AzureVoiceSelection_OverrideWinsForAllLanguages(t *testing.T) {
	svc := NewTTSService(&appconfig.OpenAIConfig{}, &appconfig.AzureTTSConfig{
		Key: "k", Region: "centralindia",
		VoiceOverride: "en-IN-Aarti:DragonHDLatestNeural",
	}, &fakeTTSStorage{})

	for _, lang := range []string{"english", "hindi", ""} {
		voice, style := svc.selectAzureVoice(models.PersonaComforting, lang)
		if voice != "en-IN-Aarti:DragonHDLatestNeural" {
			t.Errorf("lang %q: got voice %s, want override", lang, voice)
		}
		// DragonHD voices don't support mstts:express-as - style must be dropped.
		if style != "" {
			t.Errorf("lang %q: HD voice must have no style, got %q", lang, style)
		}
	}
}

func TestTTS_IsAzureHDVoice(t *testing.T) {
	cases := map[string]bool{
		"en-IN-Aarti:DragonHDLatestNeural":     true,
		"en-us-ava:DragonHDOmniLatestNeural":   true,
		"zh-CN-Xiaoxiao:DragonHDFlashLatestNeural": true,
		"en-US-JennyNeural":                    false,
		"hi-IN-SwaraNeural":                    false,
	}
	for voice, want := range cases {
		if got := isAzureHDVoice(voice); got != want {
			t.Errorf("isAzureHDVoice(%s) = %v, want %v", voice, got, want)
		}
	}
}

// ── language detection ────────────────────────────────────────────────────────

func TestTTS_DetectTextLanguage(t *testing.T) {
	cases := map[string]string{
		"I had a rough day at work":          "english",
		"आज का दिन बहुत मुश्किल था":            "hindi",
		"aaj ka din mushkil tha yaar":        "english", // romanised Hinglish stays english
		"mixed text with हिंदी words inside": "hindi",
		"":                                   "english",
	}
	for text, want := range cases {
		if got := DetectTextLanguage(text); got != want {
			t.Errorf("DetectTextLanguage(%q) = %s, want %s", text, got, want)
		}
	}
}

// ── SSML building ─────────────────────────────────────────────────────────────

func TestTTS_BuildSSML_StandardVoiceIncludesStyle(t *testing.T) {
	ssml := buildAzureSSML("I hear you.", "en-US-JennyNeural", "empathetic")

	for _, want := range []string{
		`<voice name='en-US-JennyNeural'>`,
		`<mstts:express-as style="empathetic">I hear you.</mstts:express-as>`,
		`xml:lang='en-US'`,
		`xmlns:mstts='https://www.w3.org/2001/mstts'`,
	} {
		if !strings.Contains(ssml, want) {
			t.Errorf("SSML missing %q:\n%s", want, ssml)
		}
	}
}

func TestTTS_BuildSSML_NoStyleOmitsExpressAs(t *testing.T) {
	ssml := buildAzureSSML("Namaste.", "hi-IN-SwaraNeural", "")

	if strings.Contains(ssml, "express-as") {
		t.Errorf("SSML must not contain express-as when style is empty:\n%s", ssml)
	}
	if !strings.Contains(ssml, `xml:lang='hi-IN'`) {
		t.Errorf("SSML should derive hi-IN lang from voice name:\n%s", ssml)
	}
}

func TestTTS_BuildSSML_EscapesXML(t *testing.T) {
	ssml := buildAzureSSML(`Tom & Jerry said "it's <fine>"`, "en-US-JennyNeural", "")

	if strings.Contains(ssml, "<fine>") || strings.Contains(ssml, "& Jerry") {
		t.Errorf("SSML contains unescaped user text:\n%s", ssml)
	}
	if !strings.Contains(ssml, "&amp; Jerry") || !strings.Contains(ssml, "&lt;fine&gt;") {
		t.Errorf("SSML missing escaped entities:\n%s", ssml)
	}
}

// ── Azure HTTP integration (httptest) ─────────────────────────────────────────

func TestTTS_Azure_EndToEnd(t *testing.T) {
	var gotKey, gotContentType, gotFormat, gotBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotKey = r.Header.Get("Ocp-Apim-Subscription-Key")
		gotContentType = r.Header.Get("Content-Type")
		gotFormat = r.Header.Get("X-Microsoft-OutputFormat")
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("fake-mp3-bytes"))
	}))
	defer server.Close()

	storage := &fakeTTSStorage{}
	svc := NewTTSService(&appconfig.OpenAIConfig{}, &appconfig.AzureTTSConfig{
		Key:     "azure-test-key",
		BaseURL: server.URL,
	}, storage)

	url, err := svc.Synthesize(context.Background(), "sess-1", "msg-1", "You are not alone.", models.PersonaComforting, "english")
	if err != nil {
		t.Fatalf("Synthesize: %v", err)
	}

	if gotKey != "azure-test-key" {
		t.Errorf("subscription key header = %q", gotKey)
	}
	if gotContentType != "application/ssml+xml" {
		t.Errorf("content type = %q", gotContentType)
	}
	if gotFormat != "audio-24khz-96kbitrate-mono-mp3" {
		t.Errorf("output format = %q", gotFormat)
	}
	if !strings.Contains(gotBody, "en-US-JennyNeural") || !strings.Contains(gotBody, "You are not alone.") {
		t.Errorf("SSML body missing voice or text:\n%s", gotBody)
	}
	if storage.uploadedKey != "tts/sess-1/msg-1.mp3" {
		t.Errorf("uploaded key = %q", storage.uploadedKey)
	}
	if string(storage.uploadedBody) != "fake-mp3-bytes" {
		t.Errorf("uploaded body = %q", storage.uploadedBody)
	}
	if url != "https://storage.example/tts/sess-1/msg-1.mp3" {
		t.Errorf("presigned url = %q", url)
	}
}

func TestTTS_Azure_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte("invalid key"))
	}))
	defer server.Close()

	svc := NewTTSService(&appconfig.OpenAIConfig{}, &appconfig.AzureTTSConfig{
		Key:     "bad-key",
		BaseURL: server.URL,
	}, &fakeTTSStorage{})

	_, err := svc.Synthesize(context.Background(), "s", "m", "hello", models.PersonaComforting, "english")
	if err == nil {
		t.Fatal("expected error on 401 from Azure, got nil")
	}
}

// ── OpenAI fallback ───────────────────────────────────────────────────────────

func TestTTS_OpenAIFallback_WhenAzureUnset(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		if r.URL.Path != "/audio/speech" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if auth := r.Header.Get("Authorization"); auth != "Bearer openai-key" {
			t.Errorf("auth header = %q", auth)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("openai-mp3"))
	}))
	defer server.Close()

	storage := &fakeTTSStorage{}
	svc := NewTTSService(
		&appconfig.OpenAIConfig{APIKey: "openai-key", BaseURL: server.URL},
		&appconfig.AzureTTSConfig{}, // Azure not configured
		storage,
	)

	url, err := svc.Synthesize(context.Background(), "s", "m", "hello", models.PersonaMindful, "english")
	if err != nil {
		t.Fatalf("Synthesize: %v", err)
	}
	if !called {
		t.Fatal("OpenAI endpoint was not called")
	}
	if url == "" {
		t.Fatal("expected presigned URL")
	}
}
