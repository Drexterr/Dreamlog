package services

import (
	"context"
	"fmt"

	"github.com/dreamlog/backend/internal/models"
	"github.com/google/uuid"
)

// journeySessionRepo is the minimal DB interface JourneyService needs.
type journeySessionRepo interface {
	CreateSession(ctx context.Context, userID uuid.UUID, journeyID string, prompts []string) (*models.JourneySession, error)
	GetSession(ctx context.Context, sessionID, userID uuid.UUID) (*models.JourneySession, error)
	ListSessions(ctx context.Context, userID uuid.UUID) ([]*models.JourneySession, error)
	AdvanceStep(ctx context.Context, sessionID uuid.UUID, stepIndex int, entryID uuid.UUID, nextStep int, done bool) error
}

// Catalogue of all journey templates. Defined here so they are auditable in one place.
var journeyTemplates = []models.JourneyTemplate{
	{
		ID:               "stress_relief",
		Title:            "Stress Relief",
		Description:      "Explore what's weighing on you and find a small path forward.",
		StepCount:        3,
		EstimatedMinutes: 8,
		Tags:             []string{"stress", "anxiety"},
		Prompts: []string{
			"What's weighing on you most right now? Don't filter — just speak.",
			"Where do you feel this in your body, and what does it feel like physically?",
			"What is one small thing — however tiny — that could ease this today?",
		},
	},
	{
		ID:               "gratitude_depth",
		Title:            "Gratitude Depth",
		Description:      "Move beyond surface gratitude into what truly matters.",
		StepCount:        3,
		EstimatedMinutes: 7,
		Tags:             []string{"gratitude", "positive"},
		Prompts: []string{
			"What are you genuinely grateful for today? Take your time.",
			"Pick one thing from what you just said. Why does it actually matter to you?",
			"How can you honour or carry this feeling into the rest of your day?",
		},
	},
	{
		ID:               "decision_clarity",
		Title:            "Decision Clarity",
		Description:      "Think through a real decision using structured reflection.",
		StepCount:        4,
		EstimatedMinutes: 12,
		Tags:             []string{"decisions", "clarity"},
		Prompts: []string{
			"What decision are you facing right now? Describe the situation fully.",
			"What's pulling you toward each option? Be honest about both logic and emotion.",
			"If you imagine yourself one year from now — what would you regret not choosing?",
			"Setting everything aside, what does your gut actually say?",
		},
	},
	{
		ID:               "weekly_intention",
		Title:            "Weekly Intention",
		Description:      "Begin the week with clarity about what matters and what might get in the way.",
		StepCount:        3,
		EstimatedMinutes: 8,
		Tags:             []string{"planning", "focus"},
		Prompts: []string{
			"Looking at this week ahead, what matters most to you — what would make it meaningful?",
			"What might get in the way of that? Be specific.",
			"What is one commitment you can make to yourself this week?",
		},
	},
	{
		ID:               "letting_go",
		Title:            "Letting Go",
		Description:      "Release what's no longer serving you.",
		StepCount:        3,
		EstimatedMinutes: 8,
		Tags:             []string{"grief", "closure"},
		Prompts: []string{
			"What is something you're holding onto — a feeling, a situation, a version of yourself — that it's time to release?",
			"What has holding onto this cost you? What has it protected you from?",
			"What would it feel like to genuinely let this go? Describe that feeling.",
		},
	},
	{
		ID:               "self_compassion",
		Title:            "Self Compassion",
		Description:      "Speak to yourself the way you would speak to someone you love.",
		StepCount:        3,
		EstimatedMinutes: 7,
		Tags:             []string{"self-care", "anxiety"},
		Prompts: []string{
			"What have you been hard on yourself about lately? Say it out loud.",
			"If a close friend told you exactly what you just said about themselves, what would you say to them?",
			"Now say that to yourself. What do you actually need to hear right now?",
		},
	},
}

// templateByID returns a template by its ID.
func templateByID(id string) (*models.JourneyTemplate, bool) {
	for i := range journeyTemplates {
		if journeyTemplates[i].ID == id {
			return &journeyTemplates[i], true
		}
	}
	return nil, false
}

// JourneyService manages journey templates and user sessions.
type JourneyService struct {
	repo journeySessionRepo
}

func NewJourneyService(repo journeySessionRepo) *JourneyService {
	return &JourneyService{repo: repo}
}

func (s *JourneyService) ListTemplates() []models.JourneyTemplate {
	return journeyTemplates
}

func (s *JourneyService) GetTemplate(journeyID string) (*models.JourneyTemplate, bool) {
	return templateByID(journeyID)
}

// StartSession creates a new journey session for the user.
func (s *JourneyService) StartSession(ctx context.Context, userID uuid.UUID, journeyID string) (*models.JourneySession, error) {
	tmpl, ok := templateByID(journeyID)
	if !ok {
		return nil, fmt.Errorf("journey %q not found", journeyID)
	}

	session, err := s.repo.CreateSession(ctx, userID, journeyID, tmpl.Prompts)
	if err != nil {
		return nil, err
	}

	// Populate steps from the template prompts for the response.
	session.JourneyTitle = tmpl.Title
	session.Steps = make([]models.JourneyStep, len(tmpl.Prompts))
	for i, p := range tmpl.Prompts {
		session.Steps[i] = models.JourneyStep{StepIndex: i, Prompt: p, Completed: false}
	}
	return session, nil
}

// GetSession returns a session with steps and the journey title injected.
func (s *JourneyService) GetSession(ctx context.Context, sessionID, userID uuid.UUID) (*models.JourneySession, error) {
	session, err := s.repo.GetSession(ctx, sessionID, userID)
	if err != nil {
		return nil, err
	}
	if tmpl, ok := templateByID(session.JourneyID); ok {
		session.JourneyTitle = tmpl.Title
	}
	return session, nil
}

// ListSessions returns all sessions for the user with titles injected.
func (s *JourneyService) ListSessions(ctx context.Context, userID uuid.UUID) ([]*models.JourneySession, error) {
	sessions, err := s.repo.ListSessions(ctx, userID)
	if err != nil {
		return nil, err
	}
	for _, s2 := range sessions {
		if tmpl, ok := templateByID(s2.JourneyID); ok {
			s2.JourneyTitle = tmpl.Title
		}
	}
	return sessions, nil
}

// AdvanceSession records the entry for the current step and advances the cursor.
// Returns the updated session (with steps populated).
func (s *JourneyService) AdvanceSession(ctx context.Context, sessionID, userID uuid.UUID, entryID uuid.UUID) (*models.JourneySession, error) {
	session, err := s.repo.GetSession(ctx, sessionID, userID)
	if err != nil {
		return nil, err
	}
	if session.Status == models.JourneyStatusCompleted {
		return nil, fmt.Errorf("session already completed")
	}

	stepIndex := session.CurrentStep
	nextStep := stepIndex + 1
	done := nextStep >= session.TotalSteps

	if err := s.repo.AdvanceStep(ctx, sessionID, stepIndex, entryID, nextStep, done); err != nil {
		return nil, err
	}

	// Re-fetch to return the updated session.
	return s.GetSession(ctx, sessionID, userID)
}
