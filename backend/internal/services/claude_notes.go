package services

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/dreamlog/backend/internal/models"
)

// ── Vision wire types (OpenAI-compatible multimodal content) ─────────────────

type visionImageURL struct {
	URL string `json:"url"`
}

type visionContentPart struct {
	Type     string          `json:"type"` // "text" | "image_url"
	Text     string          `json:"text,omitempty"`
	ImageURL *visionImageURL `json:"image_url,omitempty"`
}

type visionMessage struct {
	Role    string              `json:"role"`
	Content []visionContentPart `json:"content"`
}

type visionRequest struct {
	Model     string          `json:"model"`
	MaxTokens int             `json:"max_tokens"`
	Messages  []visionMessage `json:"messages"`
}

// ExtractNotesFromImage runs vision OCR on a photographed page of therapist
// notes and returns the raw transcription plus a structured bullet list.
// The configured model must be vision-capable.
func (s *ClaudeService) ExtractNotesFromImage(ctx context.Context, imageData []byte, contentType string) (*models.NotesOCROutput, error) {
	if s.cfg.StubAnalysis {
		return stubNotesOCR(), nil
	}
	if s.cfg.APIKey == "" {
		return nil, fmt.Errorf("AI: API key is not set (set STUB_AI_ANALYSIS=true for local dev)")
	}

	dataURL := fmt.Sprintf("data:%s;base64,%s", contentType, base64.StdEncoding.EncodeToString(imageData))
	body := visionRequest{
		Model:     s.cfg.Model,
		MaxTokens: 2048,
		Messages: []visionMessage{
			{Role: "system", Content: []visionContentPart{{Type: "text", Text: buildNoteOCRSystemPrompt()}}},
			{Role: "user", Content: []visionContentPart{
				{Type: "text", Text: "Transcribe this page of session notes into the JSON schema."},
				{Type: "image_url", ImageURL: &visionImageURL{URL: dataURL}},
			}},
		},
	}

	result, err := s.callVision(ctx, body)
	if err != nil {
		return nil, fmt.Errorf("claude.ExtractNotesFromImage: %w", err)
	}

	cleaned := strings.TrimSpace(result)
	cleaned = strings.TrimPrefix(cleaned, "```json")
	cleaned = strings.TrimPrefix(cleaned, "```")
	cleaned = strings.TrimSuffix(cleaned, "```")
	cleaned = strings.TrimSpace(cleaned)

	var out models.NotesOCROutput
	if err := json.Unmarshal([]byte(cleaned), &out); err != nil {
		// Deliberately no raw-response snippet here (unlike other parsers):
		// the response IS the therapist's note content and must not reach logs.
		return nil, fmt.Errorf("claude.ExtractNotesFromImage: unmarshal: %w", err)
	}
	return &out, nil
}

// GenerateSessionNotesSummary produces a 3-5 sentence professional summary of
// one session's bullet notes. clientLabel is always an anonymous placeholder.
func (s *ClaudeService) GenerateSessionNotesSummary(ctx context.Context, clientLabel, sessionDate string, bullets []string) (string, error) {
	if s.cfg.StubAnalysis {
		return fmt.Sprintf(
			"The session on %s centered on the themes recorded across %d note points, with the client engaging actively throughout. "+
				"The therapist noted observable progress alongside areas that remain unresolved. "+
				"Follow-up items recorded in the notes were carried forward for the next session.",
			sessionDate, len(bullets),
		), nil
	}
	if s.cfg.APIKey == "" {
		return "", fmt.Errorf("AI: API key is not set (set STUB_AI_ANALYSIS=true for local dev)")
	}

	prompt := buildSessionNotesSummaryPrompt(clientLabel, sessionDate, bullets)
	result, err := s.call(ctx, "", []chatMessage{{Role: "user", Content: prompt}}, 512)
	if err != nil {
		return "", fmt.Errorf("claude.GenerateSessionNotesSummary: %w", err)
	}
	return strings.TrimSpace(result), nil
}

// callVision mirrors call() for multimodal requests, with one retry on
// transient errors.
func (s *ClaudeService) callVision(ctx context.Context, body visionRequest) (string, error) {
	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(3 * time.Second):
			}
		}
		result, err := s.doVisionRequest(ctx, body)
		if err == nil {
			return result, nil
		}
		lastErr = err
		if !isRetryableError(err) {
			break
		}
	}
	return "", lastErr
}

func (s *ClaudeService) doVisionRequest(ctx context.Context, body visionRequest) (string, error) {
	b, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("marshal vision request: %w", err)
	}

	url := s.cfg.BaseURL + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+s.cfg.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("http do: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var apiErr chatError
		_ = json.Unmarshal(respBytes, &apiErr)
		return "", &aiAPIError{StatusCode: resp.StatusCode, Message: apiErr.Error.Message}
	}

	var parsed chatResponse
	if err := json.Unmarshal(respBytes, &parsed); err != nil {
		return "", fmt.Errorf("unmarshal response: %w", err)
	}
	if len(parsed.Choices) == 0 {
		return "", fmt.Errorf("empty choices in response")
	}
	return parsed.Choices[0].Message.Content, nil
}

func stubNotesOCR() *models.NotesOCROutput {
	return &models.NotesOCROutput{
		RawText: "Session went well. Client reported better sleep this week.\nDiscussed workplace boundary with manager - client set one boundary successfully.\nAnxiety spikes still present Sunday evenings.\nHomework: continue sleep log, practice the 5-4-3-2-1 grounding exercise.\nNext session: revisit the Sunday-evening pattern.",
		Bullets: []string{
			"Client reported improved sleep this week",
			"Discussed workplace boundary with manager - client set one boundary successfully",
			"Anxiety spikes still present on Sunday evenings",
			"Homework: continue sleep log, practice 5-4-3-2-1 grounding exercise",
			"Next session: revisit the Sunday-evening pattern",
		},
	}
}
