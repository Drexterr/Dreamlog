package services

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/dreamlog/backend/pkg/monitoring"
	"go.uber.org/zap"
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
	log    *zap.Logger
}

func NewCrisisDetector(claude *ClaudeService, log *zap.Logger) *CrisisDetector {
	if log == nil {
		log = zap.NewNop()
	}
	return &CrisisDetector{claude: claude, log: log}
}

// highCertaintyPhrases trigger crisis response immediately - no ambiguity.
// Matched against normalizeForCrisis(text): lowercase, typographic quotes
// folded to ASCII, whitespace collapsed.
var highCertaintyPhrases = []string{
	"kill myself", "killing myself",
	"end my life", "ending my life",
	"want to die", "wanted to die",
	"take my life", "taking my life",
	"commit suicide", "committing suicide",
	"suicide note",
	"don't want to be here anymore", "dont want to be here anymore",
	"don't want to exist", "dont want to exist",
	"hurt myself", "hurting myself",
	"cut myself", "cutting myself",
	"harm myself", "harming myself",
	"self-harm", "self harm",
	"overdose on",
	"hurt someone", "hurt them", "hurt him", "hurt her",
	"kill him", "kill her", "kill them",
	"end it all", "ending it all",
}

// ambiguousPhrases need Claude to decide - may be metaphorical. Stage 2 fails
// safe, so anything broad that could be a genuine signal belongs here rather
// than nowhere (e.g. any "suicid…" mention, Hindi/Hinglish phrasing).
var ambiguousPhrases = []string{
	"not worth living",
	"can't go on", "cant go on",
	"can't do this anymore", "cant do this anymore",
	"what's the point", "whats the point",
	"nobody cares if i",
	"better off without me",
	"no reason to live",
	"don't care what happens to me", "dont care what happens to me",
	"suicid", // suicide, suicidal
	// Hindi / Hinglish (romanised)
	"marna chahta", "marna chahti", "mar jana chahta", "mar jana chahti",
	"jeena nahi chahta", "jeena nahi chahti", "jina nahi chahta", "jina nahi chahti",
	"khudkushi", "aatmahatya", "atmahatya",
	"zindagi khatam", "sab khatam kar",
	// Hindi (Devanagari)
	"आत्महत्या", "खुदकुशी", "ख़ुदकुशी",
	"मरना चाहता", "मरना चाहती", "मर जाना चाहता", "मर जाना चाहती",
	"जीना नहीं चाहता", "जीना नहीं चाहती", "ज़िंदगी खत्म", "जिंदगी खत्म",
}

// ambiguousWords are short tokens that only count as whole words.
var ambiguousWords = []string{"kms"}

// crisisQuoteFolder maps typographic apostrophes/quotes (iOS Smart Punctuation,
// some Whisper output) to ASCII so "don’t" matches "don't".
var crisisQuoteFolder = strings.NewReplacer("’", "'", "‘", "'", "ʼ", "'", "′", "'", "`", "'")

func normalizeForCrisis(s string) string {
	return strings.Join(strings.Fields(crisisQuoteFolder.Replace(strings.ToLower(s))), " ")
}

// crisisStage2MaxRunes bounds the transcript excerpt sent to the classifier.
const crisisStage2MaxRunes = 6000

// Screen checks a transcript for crisis signals.
// country is an ISO 3166-1 alpha-2 code (e.g. "IN", "US", "GB"); empty string = international fallback.
// It is synchronous and fast for the common (non-crisis) path.
func (d *CrisisDetector) Screen(ctx context.Context, transcript, country string) (*CrisisResult, error) {
	norm := normalizeForCrisis(transcript)

	// Stage 1a: high-certainty phrases → immediate crisis response.
	for _, phrase := range highCertaintyPhrases {
		if strings.Contains(norm, phrase) {
			return &CrisisResult{Detected: true, Response: buildCrisisResponse(country)}, nil
		}
	}

	// Stage 1b: ambiguous phrases → Claude confirmation prompt (once).
	if idx := ambiguousMatch(norm); idx >= 0 {
		confirmed, err := d.confirmWithClaude(ctx, excerptAround(norm, idx, crisisStage2MaxRunes))
		if err != nil {
			// Fail safe: treat as crisis if Claude is unavailable (ADR-002).
			// This must never be silent - a broad Claude outage would
			// otherwise flood every ambiguous entry as a false-positive
			// crisis with zero operator visibility to explain why.
			d.log.Warn("crisis: stage 2 claude confirmation unavailable, failing safe to crisis (ADR-002)",
				zap.Error(err),
			)
			monitoring.CaptureErr(err, map[string]string{"component": "crisis_stage2_failsafe"})
			return &CrisisResult{Detected: true, Response: buildCrisisResponse(country)}, nil
		}
		if confirmed {
			return &CrisisResult{Detected: true, Response: buildCrisisResponse(country)}, nil
		}
	}

	return &CrisisResult{Detected: false}, nil
}

// ambiguousMatch returns the byte index of the first ambiguous signal in the
// normalized text, or -1.
func ambiguousMatch(norm string) int {
	best := -1
	for _, phrase := range ambiguousPhrases {
		if i := strings.Index(norm, phrase); i >= 0 && (best < 0 || i < best) {
			best = i
		}
	}
	for _, word := range ambiguousWords {
		for _, i := range wordIndexes(norm, word) {
			// "10 kms" is a distance, not a signal.
			if prev := strings.TrimRight(norm[:i], " "); prev != "" && prev[len(prev)-1] >= '0' && prev[len(prev)-1] <= '9' {
				continue
			}
			if best < 0 || i < best {
				best = i
			}
			break
		}
	}
	return best
}

// wordIndexes returns the byte offsets where word appears as a whole word.
func wordIndexes(s, word string) []int {
	var out []int
	for start := 0; ; {
		i := strings.Index(s[start:], word)
		if i < 0 {
			return out
		}
		i += start
		end := i + len(word)
		before := i == 0 || !isWordByte(s[i-1])
		after := end == len(s) || !isWordByte(s[end])
		if before && after {
			out = append(out, i)
		}
		start = i + 1
	}
}

func isWordByte(b byte) bool {
	return b == '_' || (b >= '0' && b <= '9') || (b >= 'a' && b <= 'z')
}

// excerptAround returns the whole text when it fits in maxRunes, otherwise a
// rune-safe window centred on byte offset idx, so the classifier always sees
// the matched signal (never a 1000-byte prefix that cut it off).
func excerptAround(s string, idx, maxRunes int) string {
	runes := []rune(s)
	if len(runes) <= maxRunes {
		return s
	}
	center := utf8.RuneCountInString(s[:idx])
	start := center - maxRunes/2
	if start < 0 {
		start = 0
	}
	end := start + maxRunes
	if end > len(runes) {
		end = len(runes)
		start = end - maxRunes
	}
	return "…" + string(runes[start:end]) + "…"
}

// confirmWithClaude asks the safety classifier to disambiguate.
//
// Inverted, fail-safe decision (ADR-002): only an explicit "no" clears the
// entry. An empty reply, a refusal, a truncated or unexpected answer, or a
// transport error all count as crisis.
func (d *CrisisDetector) confirmWithClaude(ctx context.Context, excerpt string) (bool, error) {
	if d.claude == nil {
		return true, errors.New("crisis: no classifier configured")
	}
	reply, err := d.claude.call(ctx, buildCrisisClassifierSystemPrompt(), []anthropicMessage{
		{Role: "user", Content: buildCrisisClassifierUserPrompt(excerpt)},
	}, 10)
	if err != nil {
		return true, err // fail safe
	}

	switch firstWord(reply) {
	case "no", "nope":
		return false, nil
	case "yes":
		return true, nil
	default:
		d.log.Warn("crisis: stage 2 classifier gave no clear answer, failing safe to crisis (ADR-002)",
			zap.Int("reply_len", len(reply)),
		)
		return true, nil
	}
}

// firstWord returns the first run of ASCII letters in s, lowercased
// ("No." → "no", "**Yes**" → "yes", "" → "").
func firstWord(s string) string {
	s = strings.ToLower(s)
	start := strings.IndexFunc(s, func(r rune) bool { return r >= 'a' && r <= 'z' })
	if start < 0 {
		return ""
	}
	end := strings.IndexFunc(s[start:], func(r rune) bool { return r < 'a' || r > 'z' })
	if end < 0 {
		return s[start:]
	}
	return s[start : start+end]
}

// countryHelplines maps ISO 3166-1 alpha-2 country codes to localised crisis resources.
var countryHelplines = map[string]string{
	"IN": `India:
  iCall: 9152987821 (Mon–Sat, 8 AM–10 PM)
  Vandrevala Foundation: 1860-2662-345 (24/7)
  Tele-MANAS (Govt. of India): 14416 (24/7)
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
  Rozan Counselling: 0304 111 1741

Emergency: 115`,

	"NG": `Nigeria:
  Suicide Research and Prevention Initiative: 0800-SURPIN (0800-78774)
  MENTALLY AWARE Nigeria: mentallyaware.org

Emergency: 112`,

	"ZA": `South Africa:
  SADAG Suicide Crisis Line: 0800 567 567 (24/7)
  Lifeline SA: 0861 322 322

Emergency: 10111`,

	"NL": `Netherlands:
  113 Zelfmoordpreventie: 0800-0113 (free, 24/7) or 113
  Chat: 113.nl

Emergency: 112`,

	"BE": `Belgium:
  Centre de Prévention du Suicide: 0800 32 123 (24/7)
  Zelfmoordlijn 1813: 1813 (24/7)

Emergency: 112`,

	"CH": `Switzerland:
  Die Dargebotene Hand: 143 (24/7)
  Pro Juventute (under 25): 147 (24/7)

Emergency: 112`,

	"AT": `Austria:
  Telefonseelsorge: 142 (free, 24/7)
  Rat auf Draht (youth): 147 (24/7)

Emergency: 112`,

	"ES": `Spain:
  Línea de Atención a la Conducta Suicida: 024 (24/7)
  Teléfono de la Esperanza: 717 003 717 (24/7)

Emergency: 112`,

	"IT": `Italy:
  Telefono Amico Italia: 02 2327 2327
  Samaritans Onlus: 06 77208977

Emergency: 112`,

	"PT": `Portugal:
  SOS Voz Amiga: 213 544 545
  Conversa Amiga: 808 237 327

Emergency: 112`,

	"SE": `Sweden:
  Mind Självmordslinjen: 90101 (24/7)
  Bris (under 18): 116 111

Emergency: 112`,

	"NO": `Norway:
  Mental Helse Hjelpetelefonen: 116 123 (24/7)
  Kirkens SOS: 22 40 00 40 (24/7)

Emergency: 113`,

	"DK": `Denmark:
  Livslinien: 70 201 201

Emergency: 112`,

	"BD": `Bangladesh:
  Kaan Pete Roi: 09612 119 911
  Moner Bondhu: 01776 632 344

Emergency: 999`,

	"BR": `Brazil:
  CVV — Centro de Valorização da Vida: 188 (24/7)
  CVV Chat: cvv.org.br

Emergency: 192`,

	"MX": `Mexico:
  SAPTEL: 55 5259 8121 (24/7)
  Línea de la Vida: 800 911 2000 (24/7)

Emergency: 911`,

	"JP": `Japan:
  TELL Lifeline (English): 03-5774-0992
  Yorisoi Hotline: 0120-279-338 (24/7)

Emergency: 119`,

	"KR": `South Korea:
  Suicide Prevention Hotline: 1393 (24/7)
  LifeLine Korea: 1588-9191 (24/7)

Emergency: 119`,

	"AE": `United Arab Emirates:
  Estijaba (Dept. of Health): 800 1717
  Mental Support Line: 920 033 360

Emergency: 999`,
}

// CrisisResponse returns the empathetic crisis response with country-specific
// helplines. Exported for callers (e.g. the worker pipeline) that must apply
// the ADR-002 fail-safe when the screener itself errors.
func CrisisResponse(country string) string {
	return buildCrisisResponse(country)
}

// buildCrisisResponse returns an empathetic plain-text response with country-specific helplines.
// country should be an ISO 3166-1 alpha-2 code; empty or unknown falls back to international resources.
func buildCrisisResponse(country string) string {
	helplines, ok := countryHelplines[strings.ToUpper(country)]
	if !ok {
		helplines = `International:
  findahelpline.com - resources in 200+ countries
  International Association for Suicide Prevention: iasp.info/resources/Crisis_Centres/

Emergency: call your local emergency number`
	}

	return `What you're feeling right now is real, and it matters - and so do you.

You don't have to carry this alone. Please reach out to someone who can help right now.

` + helplines + `

DreamLog will be here when you're ready.`
}
