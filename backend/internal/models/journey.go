package models

import (
	"time"

	"github.com/google/uuid"
)

// JourneyTemplate is a predefined sequence of prompts, defined in code.
type JourneyTemplate struct {
	ID               string   `json:"id"`
	Title            string   `json:"title"`
	Description      string   `json:"description"`
	StepCount        int      `json:"step_count"`
	EstimatedMinutes int      `json:"estimated_minutes"`
	Tags             []string `json:"tags"`
	Prompts          []string `json:"prompts"`
}

type JourneySessionStatus string

const (
	JourneyStatusInProgress JourneySessionStatus = "in_progress"
	JourneyStatusCompleted  JourneySessionStatus = "completed"
)

// JourneySession is one user's run through a JourneyTemplate.
type JourneySession struct {
	ID           uuid.UUID            `json:"id"`
	UserID       uuid.UUID            `json:"user_id"`
	JourneyID    string               `json:"journey_id"`
	JourneyTitle string               `json:"journey_title"`
	CurrentStep  int                  `json:"current_step"`
	TotalSteps   int                  `json:"total_steps"`
	Status       JourneySessionStatus `json:"status"`
	Steps        []JourneyStep        `json:"steps"`
	CreatedAt    time.Time            `json:"created_at"`
	UpdatedAt    time.Time            `json:"updated_at"`
}

// JourneyStep is one recorded step inside a session.
type JourneyStep struct {
	StepIndex int        `json:"step_index"`
	Prompt    string     `json:"prompt"`
	EntryID   *uuid.UUID `json:"entry_id"` // nil until the user records this step
	Completed bool       `json:"completed"`
}

// AdvanceJourneyInput is the body for POST /journeys/sessions/:id/advance.
type AdvanceJourneyInput struct {
	EntryID uuid.UUID `json:"entry_id" binding:"required"`
}
