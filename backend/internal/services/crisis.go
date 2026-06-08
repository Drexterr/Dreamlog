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
// country is an ISO 3166-1 alpha-2 code (e.g. "IN", "US", "GB"); empty string = international fallback.
// It is synchronous and fast for the common (non-crisis) path.
func (d *CrisisDetector) Screen(ctx context.Context, transcript, country string) (*CrisisResult, error) {
	lower := strings.ToLower(transcript)

	// Stage 1a: high-certainty phrases → immediate crisis response.
	for _, phrase := range highCertaintyPhrases {
		if strings.Contains(lower, phrase) {
			return &CrisisResult{Detected: true, Response: buildCrisisResponse(country)}, nil
		}
	}

	// Stage 1b: ambiguous phrases → Claude confirmation prompt.
	for _, phrase := range ambiguousPhrases {
		if strings.Contains(lower, phrase) {
			confirmed, err := d.confirmWithClaude(ctx, transcript)
			if err != nil {
				// Fail safe: treat as crisis if Claude is unavailable.
				return &CrisisResult{Detected: true, Response: buildCrisisResponse(country)}, nil
			}
			if confirmed {
				return &CrisisResult{Detected: true, Response: buildCrisisResponse(country)}, nil
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

// countryHelplines maps ISO 3166-1 alpha-2 country codes to localised crisis resources.
var countryHelplines = map[string]string{
	"IN": `India:
  iCall: 9152987821 (Mon–Sat, 8 AM–10 PM)
  Vandrevala Foundation: 1860-2662-345 (24/7)
  iCall Chat: icallhelpline.org

Emergency: 112`,

	"US": `United States:
  988 Suicide & Crisis Lifeline: call or text 988 (24/7)
  Crisis Text Line: text HOME to 741741
  Veterans Crisis Line: 988, then press 1

Emergency: 911`,

	"GB": `United Kingdom:
  Samaritans: 116 123 (free, 24/7)
  Crisis Text Line: text SHOUT to 85258 (24/7)
  PAPYRUS (under 35): 0800 068 4141

Emergency: 999`,

	"AU": `Australia:
  Lifeline: 13 11 14 (24/7)
  Beyond Blue: 1300 22 4636 (24/7)
  Crisis Text Line: text HELLO to 741741

Emergency: 000`,

	"CA": `Canada:
  Crisis Services Canada: 1-833-456-4566 (24/7)
  Kids Help Phone: 1-800-668-6868 (24/7)
  Crisis Text Line: text HOME to 686868

Emergency: 911`,

	"NZ": `New Zealand:
  Lifeline: 0800 543 354 (24/7)
  Suicide Crisis Helpline: 0508 828 865 (24/7)
  1737 (Need to Talk?): call or text 1737

Emergency: 111`,

	"IE": `Ireland:
  Samaritans: 116 123 (free, 24/7)
  Pieta House: 1800 247 247 (24/7)
  Crisis Text Line: text HELLO to 50808

Emergency: 999 or 112`,

	"DE": `Germany:
  Telefonseelsorge: 0800 111 0 111 or 0800 111 0 222 (free, 24/7)
  Online counselling: online.telefonseelsorge.de

Emergency: 112`,

	"FR": `France:
  Numéro National de Prévention du Suicide: 3114 (24/7)
  SOS Amitié: 09 72 39 40 50

Emergency: 15 (SAMU) or 112`,

	"SG": `Singapore:
  SOS (Samaritans of Singapore): 1-767 (24/7)
  IMH Crisis Helpline: 6389 2222 (24/7)

Emergency: 995`,

	"PK": `Pakistan:
  Umang: 0317 4288665
  Rozan Counselling: 051 2890505

Emergency: 115`,

	"NG": `Nigeria:
  Suicide Research and Prevention Initiative: 0800-SURPIN (0800-78774)
  MENTALLY AWARE Nigeria: mentallyaware.org

Emergency: 112`,

	"ZA": `South Africa:
  SADAG: 0800 456 789 (24/7)
  Lifeline SA: 0861 322 322

Emergency: 10111`,
}

// buildCrisisResponse returns an empathetic plain-text response with country-specific helplines.
// country should be an ISO 3166-1 alpha-2 code; empty or unknown falls back to international resources.
func buildCrisisResponse(country string) string {
	helplines, ok := countryHelplines[strings.ToUpper(country)]
	if !ok {
		helplines = `International:
  findahelpline.com — resources in 200+ countries
  International Association for Suicide Prevention: iasp.info/resources/Crisis_Centres/

Emergency: call your local emergency number`
	}

	return `What you're feeling right now is real, and it matters — and so do you.

You don't have to carry this alone. Please reach out to someone who can help right now.

` + helplines + `

DreamLog will be here when you're ready.`
}
