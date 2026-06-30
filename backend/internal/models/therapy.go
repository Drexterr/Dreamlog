package models

import (
	"time"

	"github.com/google/uuid"
)

type TherapySessionStatus string

const (
	TherapyStatusActive         TherapySessionStatus = "active"
	TherapyStatusCompleted      TherapySessionStatus = "completed"
	TherapyStatusExpired        TherapySessionStatus = "expired"
	TherapyStatusCrisisDetected TherapySessionStatus = "crisis_detected"

	// TherapySessionDuration is the server-enforced hard cap per session.
	TherapySessionDuration = 60 * time.Minute

	// TherapySessionPricePaise is the standalone pay-per-use price in Indian paise (₹499).
	TherapySessionPricePaise = 49900

	// TherapyMemberSessionPricePaise is the discounted extra-session price for Pro
	// members who have used their included monthly session (₹299).
	TherapyMemberSessionPricePaise = 29900

	// TherapyProMonthlyAllowance is the number of sessions included in the Pro plan per month.
	TherapyProMonthlyAllowance = 1
)

// TherapyPersona defines the AI's conversational style for a session.
type TherapyPersona string

const (
	PersonaComforting TherapyPersona = "comforting"
	PersonaRational   TherapyPersona = "rational"
	PersonaCBT        TherapyPersona = "cbt"
	PersonaMindful    TherapyPersona = "mindful"
)

// ValidPersona returns true if p is one of the four allowed values.
func ValidPersona(p string) bool {
	switch TherapyPersona(p) {
	case PersonaComforting, PersonaRational, PersonaCBT, PersonaMindful:
		return true
	}
	return false
}

// TherapyContextSnapshot is stored as JSONB at session start; never re-fetched mid-session.
type TherapyContextSnapshot struct {
	MoodAvg30d           *float64 `json:"mood_avg_30d"`            // null if no data
	TopEmotions          []string `json:"top_emotions"`
	TopTopics            []string `json:"top_topics"`
	RecentSummaries      []string `json:"recent_summaries"`        // last 5 entry summaries, oldest first
	PastSessionSummaries []string `json:"past_session_summaries"`  // last 3 completed session summaries, oldest first
	TopPeople            []string `json:"top_people,omitempty"`    // most-mentioned people + sentiment lean, e.g. "Mom — mostly warm"
	Country              string   `json:"country"`                 // user's country at session start (ISO 3166-1 alpha-2)
	VoiceLanguage        string   `json:"voice_language,omitempty"` // user's TTS preference at session start: auto | english | hindi
}

// TherapySessionAnalysis is the structured output Claude generates at session end.
// It mirrors the shape of EntryAnalysis so therapy data feeds into the same mood/emotion pipelines.
type TherapySessionAnalysis struct {
	MoodScore        int             `json:"mood_score"`
	EmotionalTone    []EmotionalTone `json:"emotional_tone"`
	Topics           []string        `json:"topics"`
	KeyInsights      []string        `json:"key_insights"`
	SessionNarrative string          `json:"session_narrative"`
}

type TherapySession struct {
	ID                   uuid.UUID              `json:"id"`
	UserID               uuid.UUID              `json:"user_id"`
	Status               TherapySessionStatus   `json:"status"`
	Persona              TherapyPersona         `json:"persona"`
	StartedAt            time.Time              `json:"started_at"`
	ExpiresAt            time.Time              `json:"expires_at"`
	EndedAt              *time.Time             `json:"ended_at,omitempty"`
	DurationSec          *int                   `json:"duration_sec,omitempty"`
	TurnCount            int                    `json:"turn_count"`
	CrisisWarnings       int                    `json:"crisis_warnings"`
	ContextSnapshot      TherapyContextSnapshot `json:"context_snapshot"`
	PostSessionSummary   *string                `json:"post_session_summary,omitempty"`
	SessionMoodScore     *int                   `json:"session_mood_score,omitempty"`
	SessionEmotionalTone []EmotionalTone        `json:"session_emotional_tone,omitempty"`
	SessionTopics        []string               `json:"session_topics,omitempty"`
	SessionKeyInsights   []string               `json:"session_key_insights,omitempty"`
	BillingAmountPaise   int                    `json:"billing_amount_paise"`
	CreatedAt            time.Time              `json:"created_at"`

	// Computed fields (not stored)
	TimeRemainingSec int                     `json:"time_remaining_sec"`
	Messages         []TherapySessionMessage `json:"messages,omitempty"`
}

type TherapySessionMessage struct {
	ID        uuid.UUID `json:"id"`
	SessionID uuid.UUID `json:"session_id"`
	Role      string    `json:"role"`      // "user" | "assistant"
	Content   string    `json:"content"`
	InputMode string    `json:"input_mode"` // "voice" | "text" | "system"
	CreatedAt time.Time `json:"created_at"`

	// TTSUrl is a short-lived presigned GET URL to the AI voice audio.
	// Populated only on assistant messages in SendTherapyMessageResponse; not stored in DB.
	TTSUrl *string `json:"tts_url,omitempty"`
}

// TherapySessionState is the session_state block returned on each message.
type TherapySessionState struct {
	Status           TherapySessionStatus `json:"status"`
	TurnCount        int                  `json:"turn_count"`
	TimeRemainingSec int                  `json:"time_remaining_sec"`
	IsCrisis         bool                 `json:"is_crisis"`
	CrisisWarnings   int                  `json:"crisis_warnings"`
}

// ── Request / Response shapes ────────────────────────────────────────────────

type StartSessionResponse struct {
	ID                 uuid.UUID      `json:"id"`
	Status             string         `json:"status"`
	Persona            TherapyPersona `json:"persona"`
	StartedAt          time.Time      `json:"started_at"`
	ExpiresAt          time.Time      `json:"expires_at"`
	ContextLoaded      bool           `json:"context_loaded"`
	HasSessionHistory  bool           `json:"has_session_history"`
	BillingAmountPaise int            `json:"billing_amount_paise"`
}

type StartSessionRequest struct {
	Persona string `json:"persona"` // optional; defaults to "comforting"
}

type TherapyPresignRequest struct {
	Filename    string `json:"filename" binding:"required"`
	ContentType string `json:"content_type" binding:"required"`
}

type SendTherapyMessageRequest struct {
	AudioKey  string `json:"audio_key"`
	Content   string `json:"content"`
	InputMode string `json:"input_mode" binding:"required,oneof=voice text"`
}

type SendTherapyMessageResponse struct {
	UserMessage      TherapySessionMessage `json:"user_message"`
	AssistantMessage TherapySessionMessage `json:"assistant_message"`
	SessionState     TherapySessionState   `json:"session_state"`
}

type EndSessionResponse struct {
	SessionID          uuid.UUID `json:"session_id"`
	Status             string    `json:"status"`
	DurationSec        int       `json:"duration_sec"`
	TurnCount          int       `json:"turn_count"`
	PostSessionSummary string    `json:"post_session_summary"`
}

type ListTherapySessionsResponse struct {
	Sessions []TherapySessionSummary `json:"sessions"`
}

type TherapySessionSummary struct {
	ID                 uuid.UUID            `json:"id"`
	Status             TherapySessionStatus `json:"status"`
	StartedAt          time.Time            `json:"started_at"`
	EndedAt            *time.Time           `json:"ended_at,omitempty"`
	DurationSec        *int                 `json:"duration_sec,omitempty"`
	TurnCount          int                  `json:"turn_count"`
	PostSessionSummary *string              `json:"post_session_summary,omitempty"`
}
