package services

import (
	"context"
	"strings"
)

// CrisisResult is returned by the crisis screener.
type CrisisResult struct {
	Detected bool
	Response string // pre-written empathetic response with resources
}

// CrisisDetector performs a fast two-stage check before Claude analysis:
//  1. Keyword/phrase pattern matching (< 1ms, no network)
//  2. If ambiguous, a lightweight Claude prompt (only if stage 1 is uncertain)
//
// Stage 1 alone is sufficient for obvious cases. Stage 2 is reserved for
// ambiguous language that might be metaphorical vs. literal.
type CrisisDetector struct {
	claude *ClaudeService
}

func NewCrisisDetector(claude *ClaudeService) *CrisisDetector {
	return &CrisisDetector{claude: claude}
}

// highCertaintyPhrases trigger crisis response immediately — no ambiguity.
var highCertaintyPhrases = []string{
	"kill myself", "killing myself",
	"end my life", "ending my life",
	"want to die", "wanted to die",
	"take my life", "taking my life",
	"commit suicide", "committing suicide",
	"suicide note",
	"don't want to be here anymore",
	"don't want to exist",
	"hurt myself", "hurting myself",
	"cut myself", "cutting myself",
	"harm myself", "harming myself",
	"self-harm", "self harm",
	"overdose on",
	"hurt someone", "hurt them", "hurt him", "hurt her",
	"kill him", "kill her", "kill them",
}

// ambiguousPhrases need Claude to decide — may be metaphorical.
var ambiguousPhrases = []string{
	"not worth living",
	"can't go on",
	"can't do this anymore",
	"what's the point",
	"nobody cares if i",
	"better off without me",
	"no reason to live",
	"don't care what happens to me",
}

// Screen checks a transcript for crisis signals.
// It is synchronous and fast for the common (non-crisis) path.
func (d *CrisisDetector) Screen(ctx context.Context, transcript string) (*CrisisResult, error) {
	lower := strings.ToLower(transcript)

	// Stage 1a: high-certainty phrases → immediate crisis response.
	for _, phrase := range highCertaintyPhrases {
		if strings.Contains(lower, phrase) {
			return &CrisisResult{Detected: true, Response: crisisResponse()}, nil
		}
	}

	// Stage 1b: ambiguous phrases → Claude confirmation prompt.
	for _, phrase := range ambiguousPhrases {
		if strings.Contains(lower, phrase) {
			confirmed, err := d.confirmWithClaude(ctx, transcript)
			if err != nil {
				// Fail safe: treat as crisis if Claude is unavailable.
				return &CrisisResult{Detected: true, Response: crisisResponse()}, nil
			}
			if confirmed {
				return &CrisisResult{Detected: true, Response: crisisResponse()}, nil
			}
			break // Only call Claude once even if multiple ambiguous phrases match.
		}
	}

	return &CrisisResult{Detected: false}, nil
}

// confirmWithClaude uses a minimal prompt to disambiguate.
func (d *CrisisDetector) confirmWithClaude(ctx context.Context, transcript string) (bool, error) {
	system := `You are a safety classifier. Your only job is to determine if a journal entry contains genuine expressions of suicidal ideation, self-harm intent, or intent to harm others.

Answer ONLY with "yes" or "no".
- "yes" = the text contains what appears to be genuine crisis-level distress, not metaphor
- "no" = the language is metaphorical, hyperbolic, or does not indicate genuine risk

Examples:
"I could kill him" → no (common hyperbole)
"I've been thinking about hurting myself" → yes
"I'm so dead at work" → no
"I don't want to be here anymore and I've been thinking about how to do it" → yes`

	prompt := "Journal entry:\n" + truncate(transcript, 1000) + "\n\nIs this a genuine crisis? Answer only yes or no."

	reply, err := d.claude.call(ctx, system, []anthropicMessage{
		{Role: "user", Content: prompt},
	}, 10)
	if err != nil {
		return true, err // fail safe
	}

	answer := strings.ToLower(strings.TrimSpace(reply))
	return strings.HasPrefix(answer, "yes"), nil
}

// crisisResponse returns a plain-text empathetic response stored in the DB.
// The mobile app renders structured crisis UI on top of this; this text is the
// fallback shown in notifications and plain-text contexts.
func crisisResponse() string {
	return `What you're feeling right now is real, and it matters — and so do you.

You don't have to carry this alone. Please reach out to someone who can help right now.

India:
  iCall: 9152987821 (Mon–Sat, 8 AM–10 PM)
  Vandrevala Foundation: 1860-2662-345 (24/7)

United States:
  988 Suicide & Crisis Lifeline: call or text 988 (24/7)
  Crisis Text Line: text HOME to 741741

International:
  findahelpline.com — resources in 200+ countries

If you're in immediate danger, please call your local emergency number (112 in India, 911 in the US).

DreamLog will be here when you're ready.`
}
