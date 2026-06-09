package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	appconfig "github.com/dreamlog/backend/internal/config"
	"github.com/dreamlog/backend/internal/models"
)

// ── OpenAI-compatible API wire types ────────────────────────────────────────
// Works with any OpenAI-compatible provider: Groq, OpenRouter, Gemini, etc.

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Keep anthropicMessage as an alias so crisis.go compiles without changes.
type anthropicMessage = chatMessage

type chatRequest struct {
	Model     string        `json:"model"`
	MaxTokens int           `json:"max_tokens"`
	Messages  []chatMessage `json:"messages"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
}

type chatError struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error"`
}

// ── ClaudeService ────────────────────────────────────────────────────────────

type ClaudeService struct {
	cfg    *appconfig.AnthropicConfig
	client *http.Client
}

func NewClaudeService(cfg *appconfig.AnthropicConfig) *ClaudeService {
	return &ClaudeService{
		cfg: cfg,
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// AnalyzeEntryInput carries everything the prompt needs.
type AnalyzeEntryInput struct {
	UserName       string
	PreferredName  string   // if set, used in prompts instead of UserName
	AccountAgeDays int
	Transcript     string
	PastSummaries  []string // last 5, oldest first
	EmotionTrend   string   // e.g. "mostly anxious last week"
	TopicTrend     string   // e.g. "work, relationships"
	UserGoal       string   // stress|anxiety|grief|relationships|career|curious
	Language       string   // Whisper-detected language code (e.g. "hi", "en")
	Mode           string   // entry mode: processing|rant|gratitude|decision
}

// AnalyzeEntry runs the full analysis prompt and returns structured output.
// It does NOT perform crisis detection - that must happen before this call.
func (s *ClaudeService) AnalyzeEntry(ctx context.Context, input AnalyzeEntryInput) (*models.ClaudeAnalysisOutput, error) {
	if s.cfg.StubAnalysis {
		if input.Mode == "dream" {
			return stubDreamAnalysis(input.Transcript), nil
		}
		return stubAnalysis(input.Transcript), nil
	}

	if s.cfg.APIKey == "" {
		return nil, fmt.Errorf("AI: API key is not set (set STUB_AI_ANALYSIS=true for local dev)")
	}

	detectedLang := detectScriptLanguage(input.Language, input.Transcript)
	systemPrompt := buildSystemPromptForModeAndLanguage(input.UserGoal, detectedLang, input.Mode)
	userPrompt := buildUserPrompt(input)

	result, err := s.call(ctx, systemPrompt, []chatMessage{
		{Role: "user", Content: userPrompt},
	}, 1024)
	if err != nil {
		return nil, fmt.Errorf("claude.AnalyzeEntry: %w", err)
	}

	var output models.ClaudeAnalysisOutput
	// Strip any markdown fences the model may add despite instructions.
	cleaned := strings.TrimSpace(result)
	cleaned = strings.TrimPrefix(cleaned, "```json")
	cleaned = strings.TrimPrefix(cleaned, "```")
	cleaned = strings.TrimSuffix(cleaned, "```")
	cleaned = strings.TrimSpace(cleaned)

	if err := json.Unmarshal([]byte(cleaned), &output); err != nil {
		return nil, fmt.Errorf("claude.AnalyzeEntry: unmarshal response: %w (raw: %s)", err, truncate(result, 200))
	}

	// Clamp mood_score to valid range.
	if output.MoodScore < 1 {
		output.MoodScore = 1
	}
	if output.MoodScore > 100 {
		output.MoodScore = 100
	}

	return &output, nil
}

// FollowUpInput carries state for the "Tell me more" conversation.
type FollowUpInput struct {
	OriginalTranscript string
	OriginalReflection string
	OpeningQuestion    string     // last question the AI asked
	History            []chatMessage
	UserMessage        string
}

// GenerateFollowUp produces the AI reply in the follow-up conversation.
func (s *ClaudeService) GenerateFollowUp(ctx context.Context, input FollowUpInput) (string, error) {
	if s.cfg.StubAnalysis {
		return stubFollowUp(input.UserMessage), nil
	}

	systemPrompt := buildFollowUpSystemPrompt(input.OriginalTranscript, input.OriginalReflection)

	messages := make([]chatMessage, 0, len(input.History)+2)
	messages = append(messages, chatMessage{Role: "assistant", Content: input.OpeningQuestion})
	messages = append(messages, input.History...)
	messages = append(messages, chatMessage{Role: "user", Content: input.UserMessage})

	reply, err := s.call(ctx, systemPrompt, messages, 512)
	if err != nil {
		return "", fmt.Errorf("claude.GenerateFollowUp: %w", err)
	}
	return strings.TrimSpace(reply), nil
}

// WeeklyReviewOutput is the structured response from GenerateWeeklyReview.
type WeeklyReviewOutput struct {
	Narrative   string   `json:"narrative"`
	TopEmotions []string `json:"top_emotions"`
}

// GenerateWeeklyReview produces the Sunday narrative and top emotions for a user's week.
func (s *ClaudeService) GenerateWeeklyReview(ctx context.Context, input WeeklyReviewPromptInput) (*WeeklyReviewOutput, error) {
	if s.cfg.StubAnalysis {
		return stubWeeklyReview(input), nil
	}
	if s.cfg.APIKey == "" {
		return nil, fmt.Errorf("AI: API key is not set (set STUB_AI_ANALYSIS=true for local dev)")
	}

	systemPrompt := buildWeeklyReviewSystemPrompt()
	userPrompt := buildWeeklyReviewUserPrompt(input)

	result, err := s.call(ctx, systemPrompt, []chatMessage{
		{Role: "user", Content: userPrompt},
	}, 512)
	if err != nil {
		return nil, fmt.Errorf("claude.GenerateWeeklyReview: %w", err)
	}

	cleaned := strings.TrimSpace(result)
	cleaned = strings.TrimPrefix(cleaned, "```json")
	cleaned = strings.TrimPrefix(cleaned, "```")
	cleaned = strings.TrimSuffix(cleaned, "```")
	cleaned = strings.TrimSpace(cleaned)

	var out WeeklyReviewOutput
	if err := json.Unmarshal([]byte(cleaned), &out); err != nil {
		return nil, fmt.Errorf("claude.GenerateWeeklyReview: unmarshal: %w (raw: %s)", err, truncate(result, 200))
	}
	return &out, nil
}

// YearInReviewOutput is the structured response from GenerateYearInReview.
type YearInReviewOutput struct {
	Narrative   string   `json:"narrative"`
	TopEmotions []string `json:"top_emotions"`
	TopTopics   []string `json:"top_topics"`
}

// GenerateYearInReview produces the annual narrative, top emotions, and top topics.
func (s *ClaudeService) GenerateYearInReview(ctx context.Context, input YearInReviewPromptInput) (*YearInReviewOutput, error) {
	if s.cfg.StubAnalysis {
		return stubYearInReview(input), nil
	}
	if s.cfg.APIKey == "" {
		return nil, fmt.Errorf("AI: API key is not set (set STUB_AI_ANALYSIS=true for local dev)")
	}

	systemPrompt := buildYearInReviewSystemPrompt()
	userPrompt := buildYearInReviewUserPrompt(input)

	result, err := s.call(ctx, systemPrompt, []chatMessage{
		{Role: "user", Content: userPrompt},
	}, 768)
	if err != nil {
		return nil, fmt.Errorf("claude.GenerateYearInReview: %w", err)
	}

	cleaned := strings.TrimSpace(result)
	cleaned = strings.TrimPrefix(cleaned, "```json")
	cleaned = strings.TrimPrefix(cleaned, "```")
	cleaned = strings.TrimSuffix(cleaned, "```")
	cleaned = strings.TrimSpace(cleaned)

	var out YearInReviewOutput
	if err := json.Unmarshal([]byte(cleaned), &out); err != nil {
		return nil, fmt.Errorf("claude.GenerateYearInReview: unmarshal: %w (raw: %s)", err, truncate(result, 200))
	}
	return &out, nil
}

func stubYearInReview(input YearInReviewPromptInput) *YearInReviewOutput {
	name := input.Name
	if name == "" {
		name = "you"
	}
	return &YearInReviewOutput{
		Narrative: fmt.Sprintf(
			"For %s, %d was a year of honest reflection and quiet growth. "+
				"You showed up consistently - %d entries worth of presence with yourself. "+
				"The months had their weight and their lightness, and you moved through both. "+
				"What you carried this year, you carried with more awareness than the year before. "+
				"That counts for something.",
			name, input.Year, input.EntryCount,
		),
		TopEmotions: []string{"reflective", "cautious hope", "quiet determination", "warmth", "uncertainty"},
		TopTopics:   []string{"daily life", "work", "relationships", "self-awareness", "rest"},
	}
}

// GenerateBrief produces a 3-sentence pre-session brief for a therapist.
func (s *ClaudeService) GenerateBrief(ctx context.Context, clientName, recentSummaries, trend string, avg7d *int) (string, error) {
	if s.cfg.StubAnalysis {
		return fmt.Sprintf("This client appears to be experiencing a moderate emotional week with themes of reflection and daily life challenges. There is a notable %s trend in mood scores worth exploring in session. Consider opening with: what has felt most different for you this week?", trend), nil
	}
	if s.cfg.APIKey == "" {
		return "", fmt.Errorf("AI: API key not set (set STUB_AI_ANALYSIS=true for local dev)")
	}
	prompt := BuildTherapistBriefPrompt(clientName, recentSummaries, trend, avg7d)
	result, err := s.call(ctx, "", []chatMessage{{Role: "user", Content: prompt}}, 256)
	if err != nil {
		return "", fmt.Errorf("claude.GenerateBrief: %w", err)
	}
	return strings.TrimSpace(result), nil
}

// ── Person Extraction (Relationship Map) ─────────────────────────────────────

// ExtractPeople identifies real people mentioned in a transcript.
// Non-fatal by design: errors are logged by the caller, not propagated.
func (s *ClaudeService) ExtractPeople(ctx context.Context, transcript string) (*models.PersonExtractionOutput, error) {
	if s.cfg.StubAnalysis {
		return &models.PersonExtractionOutput{People: []models.ExtractedPerson{}}, nil
	}
	if s.cfg.APIKey == "" {
		return nil, fmt.Errorf("AI: API key is not set (set STUB_AI_ANALYSIS=true for local dev)")
	}

	system := buildPersonExtractionSystemPrompt()
	user := BuildPersonExtractionUserPrompt(truncate(transcript, 6000))

	result, err := s.call(ctx, system, []chatMessage{{Role: "user", Content: user}}, 512)
	if err != nil {
		return nil, fmt.Errorf("claude.ExtractPeople: %w", err)
	}

	cleaned := strings.TrimSpace(result)
	cleaned = strings.TrimPrefix(cleaned, "```json")
	cleaned = strings.TrimPrefix(cleaned, "```")
	cleaned = strings.TrimSuffix(cleaned, "```")
	cleaned = strings.TrimSpace(cleaned)

	var out models.PersonExtractionOutput
	if err := json.Unmarshal([]byte(cleaned), &out); err != nil {
		return nil, fmt.Errorf("claude.ExtractPeople: unmarshal: %w (raw: %s)", err, truncate(result, 200))
	}
	return &out, nil
}

// ChapterSummaryOutput is the structured response from GenerateChapterSummary.
type ChapterSummaryOutput struct {
	Summary string `json:"summary"`
}

// GenerateChapterSummary produces a warm narrative summary for a life chapter.
func (s *ClaudeService) GenerateChapterSummary(ctx context.Context, input ChapterSummaryPromptInput) (*ChapterSummaryOutput, error) {
	if s.cfg.StubAnalysis {
		return stubChapterSummary(input), nil
	}
	if s.cfg.APIKey == "" {
		return nil, fmt.Errorf("AI: API key is not set (set STUB_AI_ANALYSIS=true for local dev)")
	}

	systemPrompt := buildChapterSummarySystemPrompt()
	userPrompt := buildChapterSummaryUserPrompt(input)

	result, err := s.call(ctx, systemPrompt, []chatMessage{
		{Role: "user", Content: userPrompt},
	}, 512)
	if err != nil {
		return nil, fmt.Errorf("claude.GenerateChapterSummary: %w", err)
	}

	cleaned := strings.TrimSpace(result)
	cleaned = strings.TrimPrefix(cleaned, "```json")
	cleaned = strings.TrimPrefix(cleaned, "```")
	cleaned = strings.TrimSuffix(cleaned, "```")
	cleaned = strings.TrimSpace(cleaned)

	var out ChapterSummaryOutput
	if err := json.Unmarshal([]byte(cleaned), &out); err != nil {
		return nil, fmt.Errorf("claude.GenerateChapterSummary: unmarshal: %w (raw: %s)", err, truncate(result, 200))
	}
	return &out, nil
}

func stubChapterSummary(input ChapterSummaryPromptInput) *ChapterSummaryOutput {
	title := input.Title
	if title == "" {
		title = "this period"
	}
	name := "you"
	if input.Name != "" {
		name = input.Name
	}
	return &ChapterSummaryOutput{
		Summary: fmt.Sprintf(
			"During %s, %s showed up consistently with %d journal entries that captured the texture of this period. "+
				"There was a recurring thread of %s running through the entries, giving this chapter its particular quality. "+
				"The moods shifted - sometimes heavy, sometimes lighter - but the presence was steady. "+
				"What you lived through during this time shaped something in how you understand yourself.",
			title, name, input.EntryCount,
			func() string {
				if len(input.TopEmotions) > 0 {
					return input.TopEmotions[0]
				}
				return "reflection"
			}(),
		),
	}
}

// call executes a single OpenAI-compatible chat completion request with one retry.
func (s *ClaudeService) call(ctx context.Context, system string, messages []chatMessage, maxTokens int) (string, error) {
	// Prepend system prompt as a system-role message (OpenAI convention).
	all := make([]chatMessage, 0, len(messages)+1)
	all = append(all, chatMessage{Role: "system", Content: system})
	all = append(all, messages...)

	body := chatRequest{
		Model:     s.cfg.Model,
		MaxTokens: maxTokens,
		Messages:  all,
	}

	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(3 * time.Second):
			}
		}

		result, err := s.doRequest(ctx, body)
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

func (s *ClaudeService) doRequest(ctx context.Context, body chatRequest) (string, error) {
	b, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
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
		return "", &aiAPIError{
			StatusCode: resp.StatusCode,
			Message:    apiErr.Error.Message,
		}
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

// ── Error types ──────────────────────────────────────────────────────────────

type aiAPIError struct {
	StatusCode int
	Message    string
}

func (e *aiAPIError) Error() string {
	return fmt.Sprintf("AI API error %d: %s", e.StatusCode, e.Message)
}

func isRetryableError(err error) bool {
	if e, ok := err.(*aiAPIError); ok {
		return e.StatusCode == 429 || e.StatusCode == 500 || e.StatusCode == 503
	}
	return false
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

// ── Stub (dev only) ───────────────────────────────────────────────────────────

func stubAnalysis(transcript string) *models.ClaudeAnalysisOutput {
	preview := truncate(transcript, 80)
	return &models.ClaudeAnalysisOutput{
		EmotionalTone: []models.EmotionalTone{
			{Emotion: "reflective", Intensity: 0.7},
			{Emotion: "calm", Intensity: 0.5},
		},
		Topics:    []string{"daily life", "thoughts"},
		MoodScore: 62,
		KeyQuotes: []string{preview},
		Summary:   fmt.Sprintf("The entry covers: %s", preview),
		Reflection: "There's something worth sitting with in what you shared today. " +
			"The details you chose to mention say something about where your attention is. " +
			"It's interesting how the small moments can carry more weight than they first appear. " +
			"What part of today felt most significant to you?",
		MorningNudge: "Take a moment this morning to notice one small thing you'd like to do differently today.",
	}
}

func stubDreamAnalysis(transcript string) *models.ClaudeAnalysisOutput {
	preview := truncate(transcript, 80)
	return &models.ClaudeAnalysisOutput{
		EmotionalTone: []models.EmotionalTone{
			{Emotion: "unsettled", Intensity: 0.6},
			{Emotion: "curious", Intensity: 0.4},
		},
		Topics:    []string{"dream imagery", "symbolic content"},
		MoodScore: 45,
		KeyQuotes: []string{preview},
		Summary:   fmt.Sprintf("A dream with vivid imagery: %s", preview),
		Reflection: "Dreams often surface what our waking mind sets aside. " +
			"The images you described carry emotional weight worth sitting with. " +
			"Symbols in dreams speak in feelings rather than facts. " +
			"What feeling from this dream is still with you now?",
		MorningNudge: "Notice if the feeling from this dream threads through your day.",
		DreamSymbols:      []string{"unknown space", "movement", "figures"},
		DreamType:         "vivid",
		PsychologicalLens: "The imagery of unknown space and moving figures may reflect the psyche exploring uncharted aspects of itself - a classic Jungian journey into the unconscious. The sense of movement without clear destination often signals a transition phase where something in you is shifting but has not yet found form.",
		VedicLens:         "In the Vedic tradition, dreams of open spaces and movement are considered sattvic in nature when they carry a quality of expansiveness rather than fear. The figures appearing may represent ancestral presences (pitrus) or aspects of the dreamer's own subtle body (sukshma sharira) processing accumulated experience.",
	}
}

func stubWeeklyReview(input WeeklyReviewPromptInput) *WeeklyReviewOutput {
	name := input.Name
	if name == "" {
		name = "you"
	}
	return &WeeklyReviewOutput{
		Narrative: fmt.Sprintf(
			"This was a meaningful week for %s, with %d entries capturing the texture of daily life. "+
				"There was movement in your emotional landscape - moments of difficulty and moments of quiet steadiness. "+
				"What you brought to the page this week says something about where your attention is right now.",
			name, input.EntryCount,
		),
		TopEmotions: func() []string {
			if len(input.TopEmotions) >= 3 {
				return input.TopEmotions[:3]
			}
			return []string{"reflective", "processing", "present"}
		}(),
	}
}

func stubFollowUp(userMessage string) string {
	return fmt.Sprintf(
		"That's a really interesting point - \"%s\". "+
			"It sounds like there's more beneath the surface there. "+
			"What do you think was driving that feeling?",
		truncate(userMessage, 60),
	)
}

// ── Therapy Mode ─────────────────────────────────────────────────────────────

// TherapyTurnInput carries all state for one therapy session turn.
type TherapyTurnInput struct {
	SystemPrompt string        // built once at session start, passed in from context_snapshot
	History      []chatMessage // full message history for this session
	UserMessage  string        // the user's current message (already transcribed if voice)
}

// TherapyTurn sends one user turn in a therapy session and returns the AI reply.
// The system prompt is the full therapy context prompt built at session start.
func (s *ClaudeService) TherapyTurn(ctx context.Context, input TherapyTurnInput) (string, error) {
	if s.cfg.StubAnalysis {
		return stubTherapyTurn(input.UserMessage), nil
	}
	if s.cfg.APIKey == "" {
		return "", fmt.Errorf("AI: API key is not set (set STUB_AI_ANALYSIS=true for local dev)")
	}

	messages := make([]chatMessage, 0, len(input.History)+1)
	messages = append(messages, input.History...)
	messages = append(messages, chatMessage{Role: "user", Content: input.UserMessage})

	reply, err := s.call(ctx, input.SystemPrompt, messages, 512)
	if err != nil {
		return "", fmt.Errorf("claude.TherapyTurn: %w", err)
	}
	return strings.TrimSpace(reply), nil
}

// TherapySummaryInput carries the session transcript for post-session analysis.
type TherapySummaryInput struct {
	Messages []string // alternating user/assistant lines formatted for the prompt
}

// TherapySummary generates structured post-session analysis including mood score,
// emotional tone, topics, key insights, and an 8-12 sentence narrative.
func (s *ClaudeService) TherapySummary(ctx context.Context, input TherapySummaryInput) (*models.TherapySessionAnalysis, error) {
	if s.cfg.StubAnalysis {
		score := 62
		return &models.TherapySessionAnalysis{
			MoodScore: score,
			EmotionalTone: []models.EmotionalTone{
				{Emotion: "cautious hope", Intensity: 0.7},
				{Emotion: "mild anxiety", Intensity: 0.4},
			},
			Topics:      []string{"daily stress", "emotional patterns", "self-awareness"},
			KeyInsights: []string{"recurring theme around sleep and mood connection", "awareness of self-critical thought patterns", "unresolved tension around external expectations"},
			SessionNarrative: "You explored themes of daily stress and the emotional patterns that shape your week. " +
				"A key moment emerged when you connected current feelings to recurring threads from your journal. " +
				"There was a noticeable shift in tone as the session progressed - from heaviness at the start to something closer to relief. " +
				"The conversation surfaced an awareness of self-critical patterns, particularly around productivity. " +
				"You named the sleep-mood connection as something worth paying attention to going forward. " +
				"A thread around balancing external expectations with internal capacity remained open at the close. " +
				"One thing worth carrying forward: the clarity you found around what actually helps versus what you tell yourself should help.",
		}, nil
	}
	if s.cfg.APIKey == "" {
		return nil, fmt.Errorf("AI: API key is not set (set STUB_AI_ANALYSIS=true for local dev)")
	}

	prompt := buildTherapyPostSessionPrompt(input.Messages)
	result, err := s.call(ctx, "", []chatMessage{{Role: "user", Content: prompt}}, 900)
	if err != nil {
		return nil, fmt.Errorf("claude.TherapySummary: %w", err)
	}

	cleaned := strings.TrimSpace(result)
	cleaned = strings.TrimPrefix(cleaned, "```json")
	cleaned = strings.TrimPrefix(cleaned, "```")
	cleaned = strings.TrimSuffix(cleaned, "```")
	cleaned = strings.TrimSpace(cleaned)

	var analysis models.TherapySessionAnalysis
	if err := json.Unmarshal([]byte(cleaned), &analysis); err != nil {
		return nil, fmt.Errorf("claude.TherapySummary: parse JSON: %w (raw: %s)", err, truncate(result, 200))
	}
	if analysis.MoodScore < 1 {
		analysis.MoodScore = 1
	}
	if analysis.MoodScore > 100 {
		analysis.MoodScore = 100
	}
	return &analysis, nil
}

func stubTherapyTurn(userMessage string) string {
	return fmt.Sprintf(
		"Just so we're on the same page - I'm an AI, not a therapist. This conversation is a space for reflection, not clinical care. If you're in crisis, please reach out to a professional.\n\n"+
			"Thank you for sharing that. When you say \"%s\", I'm curious what that felt like in the moment. "+
			"What was going through your mind right then?",
		truncate(userMessage, 60),
	)
}
