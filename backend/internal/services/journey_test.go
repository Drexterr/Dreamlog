package services

import (
	"context"
	"errors"
	"testing"

	"github.com/dreamlog/backend/internal/models"
	"github.com/google/uuid"
)

// ── fake repo ─────────────────────────────────────────────────────────────────

type fakeJourneyRepo struct {
	created  *models.JourneySession
	createErr error
	got      *models.JourneySession
	getErr   error
	listed   []*models.JourneySession
	listErr  error
	advanced bool
	advErr   error
}

func (f *fakeJourneyRepo) CreateSession(_ context.Context, userID uuid.UUID, journeyID string, prompts []string) (*models.JourneySession, error) {
	if f.createErr != nil {
		return nil, f.createErr
	}
	if f.created != nil {
		return f.created, nil
	}
	return &models.JourneySession{
		ID:          uuid.New(),
		UserID:      userID,
		JourneyID:   journeyID,
		TotalSteps:  len(prompts),
		CurrentStep: 0,
		Status:      models.JourneyStatusInProgress,
	}, nil
}

func (f *fakeJourneyRepo) GetSession(_ context.Context, _, _ uuid.UUID) (*models.JourneySession, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	if f.got != nil {
		return f.got, nil
	}
	return &models.JourneySession{
		ID:          uuid.New(),
		JourneyID:   "stress_relief",
		CurrentStep: 0,
		TotalSteps:  3,
		Status:      models.JourneyStatusInProgress,
		Steps: []models.JourneyStep{
			{StepIndex: 0, Prompt: "step 0", Completed: false},
			{StepIndex: 1, Prompt: "step 1", Completed: false},
			{StepIndex: 2, Prompt: "step 2", Completed: false},
		},
	}, nil
}

func (f *fakeJourneyRepo) ListSessions(_ context.Context, _ uuid.UUID) ([]*models.JourneySession, error) {
	return f.listed, f.listErr
}

func (f *fakeJourneyRepo) AdvanceStep(_ context.Context, _ uuid.UUID, _ int, _ uuid.UUID, _ int, _ bool) error {
	f.advanced = true
	return f.advErr
}

// ── ListTemplates ─────────────────────────────────────────────────────────────

func TestJourneyService_ListTemplates_ReturnsAll(t *testing.T) {
	svc := NewJourneyService(&fakeJourneyRepo{})
	templates := svc.ListTemplates()
	if len(templates) == 0 {
		t.Fatal("expected at least one journey template")
	}
	// Verify basic fields are populated.
	for _, tmpl := range templates {
		if tmpl.ID == "" {
			t.Error("template ID must not be empty")
		}
		if tmpl.Title == "" {
			t.Error("template Title must not be empty")
		}
		if len(tmpl.Prompts) == 0 {
			t.Errorf("template %q has no prompts", tmpl.ID)
		}
		if tmpl.StepCount != len(tmpl.Prompts) {
			t.Errorf("template %q step_count mismatch: declared %d, actual %d", tmpl.ID, tmpl.StepCount, len(tmpl.Prompts))
		}
	}
}

func TestJourneyService_GetTemplate_KnownID_ReturnsTemplate(t *testing.T) {
	svc := NewJourneyService(&fakeJourneyRepo{})
	tmpl, ok := svc.GetTemplate("stress_relief")
	if !ok {
		t.Fatal("stress_relief template should exist")
	}
	if tmpl.ID != "stress_relief" {
		t.Errorf("expected stress_relief, got %s", tmpl.ID)
	}
	if len(tmpl.Prompts) != tmpl.StepCount {
		t.Errorf("prompt count mismatch")
	}
}

func TestJourneyService_GetTemplate_UnknownID_ReturnsFalse(t *testing.T) {
	svc := NewJourneyService(&fakeJourneyRepo{})
	_, ok := svc.GetTemplate("nonexistent_journey")
	if ok {
		t.Error("expected ok=false for unknown journey ID")
	}
}

// ── StartSession ──────────────────────────────────────────────────────────────

func TestJourneyService_StartSession_KnownJourney_ReturnSession(t *testing.T) {
	svc := NewJourneyService(&fakeJourneyRepo{})
	userID := uuid.New()

	session, err := svc.StartSession(context.Background(), userID, "stress_relief")
	if err != nil {
		t.Fatalf("StartSession error: %v", err)
	}
	if session.JourneyID != "stress_relief" {
		t.Errorf("expected journey_id=stress_relief, got %s", session.JourneyID)
	}
	if session.JourneyTitle == "" {
		t.Error("JourneyTitle must be injected")
	}
	if session.TotalSteps != 3 {
		t.Errorf("expected total_steps=3, got %d", session.TotalSteps)
	}
	if len(session.Steps) != 3 {
		t.Errorf("expected 3 steps, got %d", len(session.Steps))
	}
	for i, s := range session.Steps {
		if s.StepIndex != i {
			t.Errorf("step %d has wrong index %d", i, s.StepIndex)
		}
		if s.Prompt == "" {
			t.Errorf("step %d prompt must not be empty", i)
		}
		if s.Completed {
			t.Errorf("step %d should not be completed on creation", i)
		}
	}
}

func TestJourneyService_StartSession_UnknownJourney_ReturnsError(t *testing.T) {
	svc := NewJourneyService(&fakeJourneyRepo{})
	_, err := svc.StartSession(context.Background(), uuid.New(), "unknown_journey")
	if err == nil {
		t.Fatal("expected error for unknown journey ID")
	}
}

func TestJourneyService_StartSession_RepoError_Propagates(t *testing.T) {
	repo := &fakeJourneyRepo{createErr: errors.New("db down")}
	svc := NewJourneyService(repo)
	_, err := svc.StartSession(context.Background(), uuid.New(), "gratitude_depth")
	if err == nil {
		t.Fatal("expected error when repo fails")
	}
}

// ── AdvanceSession ────────────────────────────────────────────────────────────

func TestJourneyService_AdvanceSession_FirstStep_Advances(t *testing.T) {
	repo := &fakeJourneyRepo{}
	svc := NewJourneyService(repo)

	_, err := svc.AdvanceSession(context.Background(), uuid.New(), uuid.New(), uuid.New())
	if err != nil {
		t.Fatalf("AdvanceSession error: %v", err)
	}
	if !repo.advanced {
		t.Error("expected AdvanceStep to be called on repo")
	}
}

func TestJourneyService_AdvanceSession_AlreadyCompleted_ReturnsError(t *testing.T) {
	repo := &fakeJourneyRepo{
		got: &models.JourneySession{
			ID:          uuid.New(),
			JourneyID:   "stress_relief",
			Status:      models.JourneyStatusCompleted,
			CurrentStep: 3,
			TotalSteps:  3,
		},
	}
	svc := NewJourneyService(repo)

	_, err := svc.AdvanceSession(context.Background(), uuid.New(), uuid.New(), uuid.New())
	if err == nil {
		t.Fatal("expected error when advancing a completed session")
	}
	if err.Error() != "session already completed" {
		t.Errorf("expected 'session already completed', got %q", err.Error())
	}
}

// ── ListSessions ──────────────────────────────────────────────────────────────

func TestJourneyService_ListSessions_InjectsTitles(t *testing.T) {
	repo := &fakeJourneyRepo{
		listed: []*models.JourneySession{
			{ID: uuid.New(), JourneyID: "stress_relief"},
			{ID: uuid.New(), JourneyID: "gratitude_depth"},
		},
	}
	svc := NewJourneyService(repo)
	sessions, err := svc.ListSessions(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("ListSessions error: %v", err)
	}
	for _, s := range sessions {
		if s.JourneyTitle == "" {
			t.Errorf("session %s missing JourneyTitle", s.JourneyID)
		}
	}
}

func TestJourneyService_ListSessions_RepoError_Propagates(t *testing.T) {
	repo := &fakeJourneyRepo{listErr: errors.New("db error")}
	svc := NewJourneyService(repo)
	_, err := svc.ListSessions(context.Background(), uuid.New())
	if err == nil {
		t.Fatal("expected error from repo")
	}
}

// ── Template integrity check ──────────────────────────────────────────────────

func TestJourneyTemplates_AllHaveUniquIDs(t *testing.T) {
	svc := NewJourneyService(&fakeJourneyRepo{})
	seen := map[string]bool{}
	for _, tmpl := range svc.ListTemplates() {
		if seen[tmpl.ID] {
			t.Errorf("duplicate journey template ID: %s", tmpl.ID)
		}
		seen[tmpl.ID] = true
	}
}

func TestJourneyTemplates_EstimatedMinutesPositive(t *testing.T) {
	svc := NewJourneyService(&fakeJourneyRepo{})
	for _, tmpl := range svc.ListTemplates() {
		if tmpl.EstimatedMinutes <= 0 {
			t.Errorf("template %q: estimated_minutes must be positive, got %d", tmpl.ID, tmpl.EstimatedMinutes)
		}
	}
}
