package services

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/dreamlog/backend/internal/repositories"
	"github.com/google/uuid"
)

const (
	// maxContextChars is a safe proxy for ~4000 tokens (avg ~4 chars/token).
	maxContextChars = 16_000
	// maxTranscriptChars is the hard limit for the current entry transcript.
	maxTranscriptChars = 8_000
	// maxSummaryChars per past summary.
	maxSummaryChars = 600
)

// ContextBuilder assembles the AnalyzeEntryInput for Claude from DB state.
type ContextBuilder struct {
	entryRepo    *repositories.EntryRepository
	userRepo     *repositories.UserRepository
	analysisRepo *repositories.AnalysisRepository
}

func NewContextBuilder(
	entryRepo *repositories.EntryRepository,
	userRepo *repositories.UserRepository,
	analysisRepo *repositories.AnalysisRepository,
) *ContextBuilder {
	return &ContextBuilder{
		entryRepo:    entryRepo,
		userRepo:     userRepo,
		analysisRepo: analysisRepo,
	}
}

// Build constructs the full AnalyzeEntryInput for the given entry.
func (b *ContextBuilder) Build(ctx context.Context, entryID, userID uuid.UUID) (*AnalyzeEntryInput, error) {
	// Fetch user.
	user, err := b.userRepo.GetByID(ctx, userID)
	if err != nil || user == nil {
		return nil, fmt.Errorf("context: fetch user: %w", err)
	}

	// Fetch current entry.
	entry, err := b.entryRepo.GetByIDInternal(ctx, entryID)
	if err != nil || entry == nil {
		return nil, fmt.Errorf("context: fetch entry: %w", err)
	}

	transcript := ""
	if entry.Transcript != nil {
		transcript = *entry.Transcript
	}
	transcript = trimChars(transcript, maxTranscriptChars)

	// Last 5 completed entries (excluding current).
	past, err := b.entryRepo.ListCompletedBefore(ctx, userID, entry.CreatedAt, 5)
	if err != nil {
		return nil, fmt.Errorf("context: fetch past entries: %w", err)
	}

	// Fetch analyses for past entries and pull summaries.
	pastSummaries := make([]string, 0, len(past))
	emotionCounts := map[string]int{}
	topicCounts := map[string]int{}

	for _, pe := range past {
		analysis, err := b.analysisRepo.GetByEntryID(ctx, pe.ID)
		if err != nil || analysis == nil {
			continue
		}
		if analysis.Summary != "" {
			pastSummaries = append(pastSummaries, trimChars(analysis.Summary, maxSummaryChars))
		}
		for _, et := range analysis.EmotionalTone {
			if et.Intensity >= 0.5 {
				emotionCounts[et.Emotion]++
			}
		}
		for _, t := range analysis.Topics {
			topicCounts[t]++
		}
	}

	emotionTrend := buildTrend(emotionCounts, 3)
	topicTrend := buildTrend(topicCounts, 4)
	accountAgeDays := int(time.Since(user.CreatedAt).Hours() / 24)

	preferredName := ""
	if user.PreferredName != nil {
		preferredName = *user.PreferredName
	}
	userGoal := ""
	if user.Goal != nil {
		userGoal = *user.Goal
	}

	entryLang := ""
	if entry.Language != nil {
		entryLang = *entry.Language
	}

	input := &AnalyzeEntryInput{
		UserName:       user.Name,
		PreferredName:  preferredName,
		AccountAgeDays: accountAgeDays,
		Transcript:     transcript,
		PastSummaries:  pastSummaries,
		EmotionTrend:   emotionTrend,
		TopicTrend:     topicTrend,
		UserGoal:       userGoal,
		Language:       entryLang,
		Mode:           string(entry.Mode),
	}

	// Final token-budget enforcement.
	enforceTokenBudget(input)

	return input, nil
}

// enforceTokenBudget trims context if total chars exceed maxContextChars.
// Priority: current transcript > past summaries (newest first, so we drop oldest).
func enforceTokenBudget(input *AnalyzeEntryInput) {
	totalChars := utf8.RuneCountInString(input.Transcript) +
		utf8.RuneCountInString(input.EmotionTrend) +
		utf8.RuneCountInString(input.TopicTrend)

	for _, s := range input.PastSummaries {
		totalChars += utf8.RuneCountInString(s)
	}

	// Drop oldest summaries until we're under budget.
	for totalChars > maxContextChars && len(input.PastSummaries) > 0 {
		dropped := input.PastSummaries[0]
		input.PastSummaries = input.PastSummaries[1:]
		totalChars -= utf8.RuneCountInString(dropped)
	}

	// Last resort: trim the transcript itself.
	if totalChars > maxContextChars {
		excess := totalChars - maxContextChars
		runes := []rune(input.Transcript)
		if excess < len(runes) {
			input.Transcript = string(runes[:len(runes)-excess]) + "… [trimmed]"
		}
	}
}

func buildTrend(counts map[string]int, topN int) string {
	if len(counts) == 0 {
		return ""
	}
	// Simple top-N by count.
	type kv struct{ k string; v int }
	sorted := make([]kv, 0, len(counts))
	for k, v := range counts {
		sorted = append(sorted, kv{k, v})
	}
	// Insertion sort for small N.
	for i := 1; i < len(sorted); i++ {
		for j := i; j > 0 && sorted[j].v > sorted[j-1].v; j-- {
			sorted[j], sorted[j-1] = sorted[j-1], sorted[j]
		}
	}
	if len(sorted) > topN {
		sorted = sorted[:topN]
	}
	parts := make([]string, len(sorted))
	for i, kv := range sorted {
		parts[i] = kv.k
	}
	return strings.Join(parts, ", ")
}

func trimChars(s string, maxChars int) string {
	runes := []rune(s)
	if len(runes) <= maxChars {
		return s
	}
	return string(runes[:maxChars]) + "…"
}
