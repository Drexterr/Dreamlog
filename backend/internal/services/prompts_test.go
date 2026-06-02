package services

import (
	"strings"
	"testing"
)

// ── buildSystemPrompt ─────────────────────────────────────────────────────────

func TestBuildSystemPrompt_NoGoal_ContainsNoCoreText(t *testing.T) {
	prompt := buildSystemPrompt("")
	if !strings.Contains(prompt, "DreamLog's reflection companion") {
		t.Error("system prompt must contain the core identity description")
	}
	if strings.Contains(prompt, "JOURNALING GOAL CONTEXT") {
		t.Error("empty goal must not produce a JOURNALING GOAL CONTEXT section")
	}
}

func TestBuildSystemPrompt_UnknownGoal_NoGoalSection(t *testing.T) {
	prompt := buildSystemPrompt("notarealgoal")
	if strings.Contains(prompt, "JOURNALING GOAL CONTEXT") {
		t.Error("unknown goal must not produce a JOURNALING GOAL CONTEXT section")
	}
}

func TestBuildSystemPrompt_AllValidGoals_HaveGuidanceSection(t *testing.T) {
	validGoals := []string{"stress", "anxiety", "grief", "relationships", "career", "curious", "depression", "trauma"}
	for _, goal := range validGoals {
		prompt := buildSystemPrompt(goal)
		if !strings.Contains(prompt, "JOURNALING GOAL CONTEXT") {
			t.Errorf("goal %q: prompt must contain JOURNALING GOAL CONTEXT section", goal)
		}
		if !strings.Contains(prompt, goalGuidance[goal]) {
			t.Errorf("goal %q: prompt must contain the specific guidance text", goal)
		}
	}
}

func TestBuildSystemPrompt_StressGoal_ContainsOverwhelmGuidance(t *testing.T) {
	prompt := buildSystemPrompt("stress")
	if !strings.Contains(prompt, "stress") {
		t.Error("stress prompt must reference stress")
	}
	if !strings.Contains(prompt, "control") {
		t.Error("stress guidance must mention what the person can control")
	}
}

func TestBuildSystemPrompt_GriefGoal_HonorsLossAndPresence(t *testing.T) {
	prompt := buildSystemPrompt("grief")
	if !strings.Contains(prompt, "loss") && !strings.Contains(prompt, "Loss") {
		t.Error("grief guidance must reference loss")
	}
	// Guidance must emphasize presence over resolution.
	if !strings.Contains(prompt, "presence") {
		t.Error("grief guidance must emphasize presence over comfort or resolution")
	}
}

func TestBuildSystemPrompt_GoalSectionDoesNotBreakOutputSchema(t *testing.T) {
	// Goal section must be injected before the output schema, not inside it.
	for _, goal := range []string{"stress", "anxiety", "grief", "relationships", "career", "curious", "depression", "trauma"} {
		prompt := buildSystemPrompt(goal)
		goalIdx := strings.Index(prompt, "JOURNALING GOAL CONTEXT")
		schemaIdx := strings.Index(prompt, "OUTPUT FORMAT")
		if goalIdx == -1 || schemaIdx == -1 {
			t.Fatalf("goal %q: expected both JOURNALING GOAL CONTEXT and OUTPUT FORMAT in prompt", goal)
		}
		if goalIdx > schemaIdx {
			t.Errorf("goal %q: JOURNALING GOAL CONTEXT must appear before OUTPUT FORMAT", goal)
		}
	}
}

func TestBuildSystemPrompt_ContainsOutputSchema(t *testing.T) {
	for _, goal := range []string{"", "stress", "curious"} {
		prompt := buildSystemPrompt(goal)
		for _, required := range []string{
			"emotional_tone", "mood_score", "key_quotes",
			"summary", "reflection", "morning_nudge", "topics",
		} {
			if !strings.Contains(prompt, required) {
				t.Errorf("goal %q: system prompt must contain output field %q", goal, required)
			}
		}
	}
}

func TestBuildSystemPrompt_ContainsSafetyOverride(t *testing.T) {
	// The SAFETY OVERRIDE block must survive goal injection.
	for _, goal := range []string{"", "anxiety", "grief"} {
		prompt := buildSystemPrompt(goal)
		if !strings.Contains(prompt, "SAFETY OVERRIDE") {
			t.Errorf("goal %q: safety override block must be present", goal)
		}
		if !strings.Contains(prompt, `{"crisis": true}`) {
			t.Errorf("goal %q: crisis JSON response must be present in safety override", goal)
		}
	}
}

// ── buildUserPrompt ───────────────────────────────────────────────────────────

func TestBuildUserPrompt_UsesPreferredNameOverName(t *testing.T) {
	input := AnalyzeEntryInput{
		UserName:      "Bharat",
		PreferredName: "B",
		Transcript:    "Had a good day.",
	}
	prompt := buildUserPrompt(input)
	if !strings.Contains(prompt, "Name: B") {
		t.Error("preferred_name must be used when set, got prompt without 'Name: B'")
	}
	if strings.Contains(prompt, "Name: Bharat") {
		t.Error("UserName must not appear when PreferredName is set")
	}
}

func TestBuildUserPrompt_FallsBackToNameWhenNoPreferredName(t *testing.T) {
	input := AnalyzeEntryInput{
		UserName:   "Bharat",
		Transcript: "Had a good day.",
	}
	prompt := buildUserPrompt(input)
	if !strings.Contains(prompt, "Name: Bharat") {
		t.Error("UserName must be used when PreferredName is empty")
	}
}

func TestBuildUserPrompt_NoNameNoPreferredName_NoNameLine(t *testing.T) {
	input := AnalyzeEntryInput{Transcript: "Had a good day."}
	prompt := buildUserPrompt(input)
	if strings.Contains(prompt, "Name:") {
		t.Error("no name should appear in prompt when both UserName and PreferredName are empty")
	}
}

func TestBuildUserPrompt_ContainsTranscript(t *testing.T) {
	input := AnalyzeEntryInput{Transcript: "I felt really hopeful today about the project."}
	prompt := buildUserPrompt(input)
	if !strings.Contains(prompt, input.Transcript) {
		t.Error("user prompt must contain the transcript verbatim")
	}
}

func TestBuildUserPrompt_ContainsPastSummaries(t *testing.T) {
	input := AnalyzeEntryInput{
		Transcript:    "Today's entry.",
		PastSummaries: []string{"First past summary.", "Second past summary."},
	}
	prompt := buildUserPrompt(input)
	for _, s := range input.PastSummaries {
		if !strings.Contains(prompt, s) {
			t.Errorf("user prompt must contain past summary: %q", s)
		}
	}
}

func TestBuildUserPrompt_ContainsEmotionAndTopicTrends(t *testing.T) {
	input := AnalyzeEntryInput{
		Transcript:   "Entry text.",
		EmotionTrend: "anxious, hopeful",
		TopicTrend:   "work, family",
	}
	prompt := buildUserPrompt(input)
	if !strings.Contains(prompt, "anxious, hopeful") {
		t.Error("user prompt must contain emotion trend")
	}
	if !strings.Contains(prompt, "work, family") {
		t.Error("user prompt must contain topic trend")
	}
}

func TestBuildUserPrompt_AccountAgeIncluded(t *testing.T) {
	input := AnalyzeEntryInput{
		Transcript:     "Entry text.",
		AccountAgeDays: 42,
	}
	prompt := buildUserPrompt(input)
	if !strings.Contains(prompt, "42") {
		t.Error("user prompt must include account age in days")
	}
}

// ── detectScriptLanguage ──────────────────────────────────────────────────────

func TestDetectScriptLanguage_NonHindi_ReturnsEN(t *testing.T) {
	cases := []struct{ lang, transcript string }{
		{"en", "I had a great day at work."},
		{"", "Some text here."},
		{"fr", "Bonjour le monde."},
		{"ja", "今日はいい天気でした。"},
	}
	for _, c := range cases {
		got := detectScriptLanguage(c.lang, c.transcript)
		if got != "en" {
			t.Errorf("lang=%q: expected 'en', got %q", c.lang, got)
		}
	}
}

func TestDetectScriptLanguage_HindiDevanagari_ReturnsHI(t *testing.T) {
	// Mostly Devanagari → pure Hindi
	transcript := "आज मैं बहुत खुश था। काम पर सब ठीक रहा। शाम को घर आकर अच्छा लगा।"
	got := detectScriptLanguage("hi", transcript)
	if got != "hi" {
		t.Errorf("expected 'hi' for Devanagari-heavy transcript, got %q", got)
	}
}

func TestDetectScriptLanguage_HinglishRomanized_ReturnsHinglish(t *testing.T) {
	// Whisper reports "hi" but text is mostly Latin (romanized Hinglish)
	transcript := "Aaj ka din bahut acha tha. Main kaam pe gaya aur sab theek raha. Shaam ko ghar aake kaafi relief feel ki."
	got := detectScriptLanguage("hi", transcript)
	if got != "hinglish" {
		t.Errorf("expected 'hinglish' for romanized Hinglish, got %q", got)
	}
}

func TestDetectScriptLanguage_EmptyTranscript_ReturnsHIForHindi(t *testing.T) {
	// No characters to analyze → no Latin fraction → returns "hi"
	got := detectScriptLanguage("hi", "")
	if got != "hi" {
		t.Errorf("expected 'hi' for empty transcript with Hindi language code, got %q", got)
	}
}

// ── buildSystemPromptForLanguage ──────────────────────────────────────────────

func TestBuildSystemPromptForLanguage_ENUsesEnglishPrompt(t *testing.T) {
	prompt := buildSystemPromptForLanguage("stress", "en")
	if !strings.Contains(prompt, "DreamLog's reflection companion") {
		t.Error("English prompt must contain core identity")
	}
	if !strings.Contains(prompt, "JOURNALING GOAL CONTEXT") {
		t.Error("English prompt must contain goal context for 'stress'")
	}
}

func TestBuildSystemPromptForLanguage_HIUsesHindiPrompt(t *testing.T) {
	prompt := buildSystemPromptForLanguage("", "hi")
	if !strings.Contains(prompt, "DreamLog") {
		t.Error("Hindi prompt must contain DreamLog")
	}
	// Hindi prompt must output JSON schema in Hindi context
	if !strings.Contains(prompt, "emotional_tone") {
		t.Error("Hindi prompt must include emotional_tone field in output schema")
	}
	if !strings.Contains(prompt, `{"crisis": true}`) {
		t.Error("Hindi prompt must include crisis safety override")
	}
}

func TestBuildSystemPromptForLanguage_HinglishUsesHinglishPrompt(t *testing.T) {
	prompt := buildSystemPromptForLanguage("", "hinglish")
	if !strings.Contains(prompt, "DreamLog") {
		t.Error("Hinglish prompt must contain DreamLog")
	}
	if !strings.Contains(prompt, "emotional_tone") {
		t.Error("Hinglish prompt must include emotional_tone field in output schema")
	}
	if !strings.Contains(prompt, `{"crisis": true}`) {
		t.Error("Hinglish prompt must include crisis safety override")
	}
}

func TestBuildSystemPromptForLanguage_HindiGoalInjected(t *testing.T) {
	prompt := buildSystemPromptForLanguage("stress", "hi")
	if !strings.Contains(prompt, goalGuidanceHindi["stress"]) {
		t.Error("Hindi prompt must inject Hindi stress goal guidance")
	}
}

func TestBuildSystemPromptForLanguage_HinglishGoalInjected(t *testing.T) {
	prompt := buildSystemPromptForLanguage("anxiety", "hinglish")
	if !strings.Contains(prompt, goalGuidanceHinglish["anxiety"]) {
		t.Error("Hinglish prompt must inject Hinglish anxiety goal guidance")
	}
}

func TestBuildSystemPromptForLanguage_UnknownLangFallsBackToEnglish(t *testing.T) {
	prompt := buildSystemPromptForLanguage("", "zh")
	if !strings.Contains(prompt, "DreamLog's reflection companion") {
		t.Error("Unknown language must fall back to English prompt")
	}
}

// ── buildSystemPromptForModeAndLanguage ───────────────────────────────────────

func TestBuildSystemPromptForMode_ProcessingUsesLanguageAwarePrompt(t *testing.T) {
	enPrompt := buildSystemPromptForModeAndLanguage("", "en", "processing")
	if !strings.Contains(enPrompt, "DreamLog's reflection companion") {
		t.Error("processing mode (en) must use English prompt")
	}
	hiPrompt := buildSystemPromptForModeAndLanguage("", "hi", "processing")
	if !strings.Contains(hiPrompt, "DreamLog") {
		t.Error("processing mode (hi) must use Hindi prompt")
	}
}

func TestBuildSystemPromptForMode_EmptyModeFallsToProcessing(t *testing.T) {
	prompt := buildSystemPromptForModeAndLanguage("", "en", "")
	if !strings.Contains(prompt, "DreamLog's reflection companion") {
		t.Error("empty mode must fall back to processing (English) prompt")
	}
}

func TestBuildSystemPromptForMode_RantMode(t *testing.T) {
	prompt := buildSystemPromptForModeAndLanguage("stress", "en", "rant")
	if !strings.Contains(prompt, "Rant Mode") {
		t.Error("rant mode prompt must identify itself as Rant Mode")
	}
	if !strings.Contains(prompt, `{"crisis": true}`) {
		t.Error("rant mode prompt must include crisis safety override")
	}
	if !strings.Contains(prompt, "emotional_tone") {
		t.Error("rant mode prompt must include JSON output schema")
	}
	// Rant mode should not deep-analyze — no mention of patterns/insights
	if strings.Contains(prompt, "JOURNALING GOAL CONTEXT") {
		t.Error("rant mode must not inject goal context")
	}
}

func TestBuildSystemPromptForMode_GratitudeMode(t *testing.T) {
	prompt := buildSystemPromptForModeAndLanguage("", "en", "gratitude")
	if !strings.Contains(prompt, "Gratitude Mode") {
		t.Error("gratitude mode prompt must identify itself as Gratitude Mode")
	}
	if !strings.Contains(prompt, `{"crisis": true}`) {
		t.Error("gratitude mode prompt must include crisis safety override")
	}
	if !strings.Contains(prompt, "emotional_tone") {
		t.Error("gratitude mode prompt must include JSON output schema")
	}
}

func TestBuildSystemPromptForMode_DecisionMode(t *testing.T) {
	prompt := buildSystemPromptForModeAndLanguage("", "en", "decision")
	if !strings.Contains(prompt, "Decision Mode") {
		t.Error("decision mode prompt must identify itself as Decision Mode")
	}
	if !strings.Contains(prompt, `{"crisis": true}`) {
		t.Error("decision mode prompt must include crisis safety override")
	}
	if !strings.Contains(prompt, "emotional_tone") {
		t.Error("decision mode prompt must include JSON output schema")
	}
	// Decision mode should be Socratic
	if !strings.Contains(prompt, "Socratic") {
		t.Error("decision mode prompt must mention Socratic approach")
	}
}

func TestBuildSystemPromptForMode_NonProcessingModesIgnoreLanguage(t *testing.T) {
	// Rant/gratitude/decision ignore the language parameter — always English for now
	rantEN := buildSystemPromptForModeAndLanguage("", "en", "rant")
	rantHI := buildSystemPromptForModeAndLanguage("", "hi", "rant")
	if rantEN != rantHI {
		t.Error("rant mode prompt must be identical regardless of language")
	}
}

// ── BuildTherapistBriefPrompt ─────────────────────────────────────────────────

func TestBuildTherapistBriefPrompt_ContainsClientName(t *testing.T) {
	prompt := BuildTherapistBriefPrompt("Dr Alice's Client", "some summaries", "improving", nil)
	if !strings.Contains(prompt, "Dr Alice's Client") {
		t.Error("prompt must contain the client name")
	}
}

func TestBuildTherapistBriefPrompt_ContainsTrend(t *testing.T) {
	for _, trend := range []string{"improving", "declining", "stable", "insufficient_data"} {
		prompt := BuildTherapistBriefPrompt("Alice", "summaries", trend, nil)
		if !strings.Contains(prompt, trend) {
			t.Errorf("prompt must contain trend %q", trend)
		}
	}
}

func TestBuildTherapistBriefPrompt_WithAvgMood_ContainsMoodScore(t *testing.T) {
	avg := 72
	prompt := BuildTherapistBriefPrompt("Alice", "summaries", "improving", &avg)
	if !strings.Contains(prompt, "72/100") {
		t.Errorf("prompt must contain avg mood as '72/100', got prompt without it")
	}
}

func TestBuildTherapistBriefPrompt_NilAvgMood_ShowsNoData(t *testing.T) {
	prompt := BuildTherapistBriefPrompt("Alice", "summaries", "stable", nil)
	if !strings.Contains(prompt, "no data") {
		t.Error("nil avg_mood must produce 'no data' in prompt")
	}
}

func TestBuildTherapistBriefPrompt_ContainsRecentSummaries(t *testing.T) {
	summaries := "2026-05-21 | mood 72 | Had a good week at work."
	prompt := BuildTherapistBriefPrompt("Alice", summaries, "stable", nil)
	if !strings.Contains(prompt, summaries) {
		t.Error("prompt must contain the recent summaries text verbatim")
	}
}

func TestBuildTherapistBriefPrompt_ContainsRequirements(t *testing.T) {
	prompt := BuildTherapistBriefPrompt("Alice", "summaries", "improving", nil)
	for _, required := range []string{
		"Exactly 3 sentences",
		"clinical",
		"neutral",
		"factual",
	} {
		if !strings.Contains(strings.ToLower(prompt), strings.ToLower(required)) {
			t.Errorf("prompt must contain requirement %q", required)
		}
	}
}

// ── buildPersonExtractionSystemPrompt ─────────────────────────────────────────

func TestBuildPersonExtractionSystemPrompt_ContainsOutputSchema(t *testing.T) {
	prompt := buildPersonExtractionSystemPrompt()
	for _, field := range []string{"name", "role", "sentiment", "context"} {
		if !strings.Contains(prompt, field) {
			t.Errorf("person extraction prompt must contain field %q", field)
		}
	}
}

func TestBuildPersonExtractionSystemPrompt_ContainsRoleValues(t *testing.T) {
	prompt := buildPersonExtractionSystemPrompt()
	for _, role := range []string{"family", "friend", "colleague", "romantic", "other"} {
		if !strings.Contains(prompt, role) {
			t.Errorf("person extraction prompt must list role %q", role)
		}
	}
}

func TestBuildPersonExtractionSystemPrompt_ContainsSentimentValues(t *testing.T) {
	prompt := buildPersonExtractionSystemPrompt()
	for _, s := range []string{"positive", "neutral", "negative"} {
		if !strings.Contains(prompt, s) {
			t.Errorf("person extraction prompt must list sentiment %q", s)
		}
	}
}

func TestBuildPersonExtractionSystemPrompt_EmptyPeopleInstruction(t *testing.T) {
	prompt := buildPersonExtractionSystemPrompt()
	// Must instruct Claude to return empty array when no people found.
	if !strings.Contains(prompt, `{"people": []}`) {
		t.Error("prompt must show the empty-array response when no people found")
	}
}

func TestBuildPersonExtractionUserPrompt_ContainsTranscript(t *testing.T) {
	transcript := "Today I had lunch with Priya and talked about her new job."
	prompt := BuildPersonExtractionUserPrompt(transcript)
	if !strings.Contains(prompt, transcript) {
		t.Error("user prompt must include the transcript verbatim")
	}
}

func TestBuildTherapistBriefPrompt_NoJSONInstructions(t *testing.T) {
	prompt := BuildTherapistBriefPrompt("Alice", "summaries", "improving", nil)
	// Brief prompt must not ask for JSON — response is plain text.
	if strings.Contains(prompt, "OUTPUT FORMAT") {
		t.Error("therapist brief prompt must not include OUTPUT FORMAT (plain text, not JSON)")
	}
	if strings.Contains(prompt, "emotional_tone") {
		t.Error("therapist brief prompt must not include analysis JSON schema fields")
	}
}

// ── buildDreamSystemPrompt ────────────────────────────────────────────────────

func TestBuildDreamSystemPrompt_ContainsDreamMode(t *testing.T) {
	prompt := buildDreamSystemPrompt()
	if !strings.Contains(prompt, "dream") && !strings.Contains(prompt, "Dream") {
		t.Error("dream prompt must reference dreams")
	}
}

func TestBuildDreamSystemPrompt_ContainsDreamSpecificOutputFields(t *testing.T) {
	prompt := buildDreamSystemPrompt()
	for _, field := range []string{"dream_symbols", "dream_type"} {
		if !strings.Contains(prompt, field) {
			t.Errorf("dream prompt must contain output field %q", field)
		}
	}
}

func TestBuildDreamSystemPrompt_ContainsStandardOutputFields(t *testing.T) {
	prompt := buildDreamSystemPrompt()
	for _, field := range []string{"emotional_tone", "mood_score", "topics", "key_quotes", "summary", "reflection", "morning_nudge"} {
		if !strings.Contains(prompt, field) {
			t.Errorf("dream prompt must still contain standard field %q", field)
		}
	}
}

func TestBuildDreamSystemPrompt_ContainsDreamTypes(t *testing.T) {
	prompt := buildDreamSystemPrompt()
	for _, dt := range []string{"nightmare", "lucid", "recurring", "vivid", "surreal", "mundane"} {
		if !strings.Contains(prompt, dt) {
			t.Errorf("dream prompt must document dream_type value %q", dt)
		}
	}
}

func TestBuildDreamSystemPrompt_ContainsSafetyOverride(t *testing.T) {
	prompt := buildDreamSystemPrompt()
	if !strings.Contains(prompt, "SAFETY OVERRIDE") {
		t.Error("dream prompt must contain SAFETY OVERRIDE block")
	}
	if !strings.Contains(prompt, `{"crisis": true}`) {
		t.Error("dream safety override must include crisis JSON sentinel")
	}
}

func TestBuildSystemPromptForMode_DreamMode(t *testing.T) {
	prompt := buildSystemPromptForModeAndLanguage("", "en", "dream")
	if !strings.Contains(prompt, "dream_symbols") {
		t.Error("dream mode must use the dream system prompt (missing dream_symbols)")
	}
	if !strings.Contains(prompt, "dream_type") {
		t.Error("dream mode must use the dream system prompt (missing dream_type)")
	}
}

func TestBuildSystemPromptForMode_DreamIgnoresLanguage(t *testing.T) {
	promptEN := buildSystemPromptForModeAndLanguage("", "en", "dream")
	promptHI := buildSystemPromptForModeAndLanguage("", "hi", "dream")
	if promptEN != promptHI {
		t.Error("dream mode prompt must be identical regardless of language")
	}
}
