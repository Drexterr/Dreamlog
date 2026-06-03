package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	appconfig "github.com/dreamlog/backend/internal/config"
	"github.com/dreamlog/backend/internal/models"
)

// personaVoice maps each therapy persona to its OpenAI TTS voice (ROADMAP 8b).
var personaVoice = map[models.TherapyPersona]string{
	models.PersonaComforting: "nova",
	models.PersonaRational:   "onyx",
	models.PersonaCBT:        "alloy",
	models.PersonaMindful:    "shimmer",
}

type ttsStorageClient interface {
	Upload(ctx context.Context, key, contentType string, body io.Reader) error
	PresignDownload(ctx context.Context, key string, expiry time.Duration) (string, error)
}

// TTSService generates AI voice audio via OpenAI TTS and stores it for presigned access.
// When OPENAI_API_KEY is not set, all calls are no-ops (returns empty string, nil error).
type TTSService struct {
	cfg     *appconfig.OpenAIConfig
	storage ttsStorageClient
	client  *http.Client
}

func NewTTSService(cfg *appconfig.OpenAIConfig, storage ttsStorageClient) *TTSService {
	return &TTSService{
		cfg:     cfg,
		storage: storage,
		client:  &http.Client{Timeout: 30 * time.Second},
	}
}

// Synthesize converts text to speech for the given persona, uploads it to storage,
// and returns a short-lived presigned GET URL. Returns ("", nil) when TTS is disabled.
func (s *TTSService) Synthesize(ctx context.Context, sessionID, messageID string, text string, persona models.TherapyPersona) (string, error) {
	if s.cfg.APIKey == "" {
		return "", nil
	}

	voice := personaVoice[persona]
	if voice == "" {
		voice = "nova"
	}

	audioBytes, err := s.callTTSAPI(ctx, text, voice)
	if err != nil {
		return "", fmt.Errorf("tts: synthesize: %w", err)
	}

	key := fmt.Sprintf("tts/%s/%s.mp3", sessionID, messageID)
	if err := s.storage.Upload(ctx, key, "audio/mpeg", bytes.NewReader(audioBytes)); err != nil {
		return "", fmt.Errorf("tts: upload: %w", err)
	}

	url, err := s.storage.PresignDownload(ctx, key, 5*time.Minute)
	if err != nil {
		return "", fmt.Errorf("tts: presign: %w", err)
	}

	return url, nil
}

func (s *TTSService) callTTSAPI(ctx context.Context, text, voice string) ([]byte, error) {
	payload, err := json.Marshal(map[string]string{
		"model": "tts-1",
		"input": text,
		"voice": voice,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal: %w", err)
	}

	url := s.cfg.BaseURL + "/audio/speech"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+s.cfg.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http do: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned %d: %s", resp.StatusCode, string(body))
	}

	return body, nil
}
