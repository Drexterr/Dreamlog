package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"

	appconfig "github.com/dreamlog/backend/internal/config"
)

// WhisperResponse matches the verbose_json response format from OpenAI Whisper API.
type WhisperResponse struct {
	Task     string            `json:"task"`
	Language string            `json:"language"`
	Duration float64           `json:"duration"`
	Text     string            `json:"text"`
	Segments []WhisperSegment  `json:"segments"`
}

type WhisperSegment struct {
	ID               int     `json:"id"`
	Start            float64 `json:"start"`
	End              float64 `json:"end"`
	Text             string  `json:"text"`
	AvgLogprob       float64 `json:"avg_logprob"`
	CompressionRatio float64 `json:"compression_ratio"`
	NoSpeechProb     float64 `json:"no_speech_prob"`
}

type TranscriptionService struct {
	cfg    *appconfig.OpenAIConfig
	client *http.Client
}

func NewTranscriptionService(cfg *appconfig.OpenAIConfig) *TranscriptionService {
	return &TranscriptionService{
		cfg: cfg,
		client: &http.Client{
			Timeout: 10 * time.Minute, // long audio files can take time
		},
	}
}

// Transcribe sends audio bytes to the Whisper API and returns the response.
// audioReader is the raw audio content; filename hint must end in a supported extension.
func (s *TranscriptionService) Transcribe(ctx context.Context, audioReader io.Reader, filename string) (*WhisperResponse, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Attach audio file field.
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return nil, fmt.Errorf("whisper: create form file: %w", err)
	}
	if _, err := io.Copy(part, audioReader); err != nil {
		return nil, fmt.Errorf("whisper: copy audio: %w", err)
	}

	// Required fields.
	if err := writer.WriteField("model", "whisper-1"); err != nil {
		return nil, fmt.Errorf("whisper: write model field: %w", err)
	}
	if err := writer.WriteField("response_format", "verbose_json"); err != nil {
		return nil, fmt.Errorf("whisper: write format field: %w", err)
	}
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("whisper: close writer: %w", err)
	}

	url := s.cfg.BaseURL + "/audio/transcriptions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, body)
	if err != nil {
		return nil, fmt.Errorf("whisper: new request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+s.cfg.APIKey)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("whisper: http do: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("whisper: read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("whisper: API returned %d: %s", resp.StatusCode, string(respBody))
	}

	var result WhisperResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("whisper: unmarshal response: %w", err)
	}
	return &result, nil
}
