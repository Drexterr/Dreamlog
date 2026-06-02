package models

import (
	"time"

	"github.com/google/uuid"
)

// Plan represents a user's subscription tier.
type Plan string

const (
	PlanFree Plan = "free"
	PlanPlus Plan = "plus"
	PlanPro  Plan = "pro"
	PlanB2B  Plan = "b2b"

	// FreeMonthlyEntries is the max entries a free user can create per calendar month.
	FreeMonthlyEntries = 10
	// PlusMonthlyShares is the max therapist share links a Plus user can create per month.
	PlusMonthlyShares = 5
)

// planOrder is used for AtLeast comparisons.
var planOrder = map[Plan]int{
	PlanFree: 0,
	PlanPlus: 1,
	PlanPro:  2,
	PlanB2B:  3,
}

// AtLeast returns true if p is at least as high as required.
func (p Plan) AtLeast(required Plan) bool {
	return planOrder[p] >= planOrder[required]
}

// PlanLimits describes what a plan allows.
type PlanLimits struct {
	Plan              Plan   `json:"plan"`
	MonthlyEntries    int    `json:"monthly_entries"`    // -1 = unlimited
	MonthlyShares     int    `json:"monthly_shares"`     // -1 = unlimited; 0 = not allowed
	HasPDFExport      bool   `json:"has_pdf_export"`
	HasWeeklyReview   bool   `json:"has_weekly_review"`
	HasMoodHistory    bool   `json:"has_mood_history"`
	HasHindi          bool   `json:"has_hindi"`
	HasAllModes       bool   `json:"has_all_modes"`
	HasStreakFreeze    bool   `json:"has_streak_freeze"`
	HasTherapistShare bool   `json:"has_therapist_share"`
	DisplayName       string `json:"display_name"`
	Price             string `json:"price"`
}

// GetPlanLimits returns the canonical limits for a given plan.
func GetPlanLimits(p Plan) *PlanLimits {
	switch p {
	case PlanPlus:
		return &PlanLimits{
			Plan: PlanPlus, MonthlyEntries: -1, MonthlyShares: PlusMonthlyShares,
			HasPDFExport: false, HasWeeklyReview: true, HasMoodHistory: true,
			HasHindi: true, HasAllModes: true, HasStreakFreeze: true, HasTherapistShare: true,
			DisplayName: "DreamLog+", Price: "₹199/month · $7.99/month",
		}
	case PlanPro:
		return &PlanLimits{
			Plan: PlanPro, MonthlyEntries: -1, MonthlyShares: -1,
			HasPDFExport: true, HasWeeklyReview: true, HasMoodHistory: true,
			HasHindi: true, HasAllModes: true, HasStreakFreeze: true, HasTherapistShare: true,
			DisplayName: "DreamLog Pro", Price: "₹499/month · $14.99/month",
		}
	case PlanB2B:
		return &PlanLimits{
			Plan: PlanB2B, MonthlyEntries: -1, MonthlyShares: -1,
			HasPDFExport: true, HasWeeklyReview: true, HasMoodHistory: true,
			HasHindi: true, HasAllModes: true, HasStreakFreeze: true, HasTherapistShare: true,
			DisplayName: "B2B Wellness", Price: "₹199/employee/month",
		}
	default: // PlanFree
		return &PlanLimits{
			Plan: PlanFree, MonthlyEntries: FreeMonthlyEntries, MonthlyShares: 0,
			HasPDFExport: false, HasWeeklyReview: false, HasMoodHistory: false,
			HasHindi: false, HasAllModes: false, HasStreakFreeze: false, HasTherapistShare: false,
			DisplayName: "Free", Price: "₹0 · $0",
		}
	}
}

type User struct {
	ID                uuid.UUID  `json:"id"`
	SupabaseID        string     `json:"supabase_id"`
	Email             string     `json:"email"`
	Name              string     `json:"name"`
	PreferredName     *string    `json:"preferred_name,omitempty"`
	Timezone          string     `json:"timezone"`
	FCMNudgeHour      int        `json:"fcm_nudge_hour"` // 0-23
	NudgeEnabled      bool       `json:"nudge_enabled"`
	Goal              *string    `json:"goal,omitempty"`
	AgeRange          *string    `json:"age_range,omitempty"`
	StreakFreezeCount int        `json:"streak_freeze_count"`
	Plan              Plan       `json:"plan"`
	PlanExpiresAt     *time.Time `json:"plan_expires_at,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

type UpdateUserInput struct {
	Name          *string `json:"name" binding:"omitempty,min=1,max=200"`
	PreferredName *string `json:"preferred_name" binding:"omitempty,max=100"`
	Timezone      *string `json:"timezone" binding:"omitempty"`
	FCMNudgeHour  *int    `json:"fcm_nudge_hour" binding:"omitempty,min=0,max=23"`
	NudgeEnabled  *bool   `json:"nudge_enabled"`
	Goal          *string `json:"goal" binding:"omitempty,oneof=stress anxiety grief depression trauma relationships career curious"`
	AgeRange      *string `json:"age_range" binding:"omitempty,oneof=under_18 18_24 25_34 35_44 45_plus"`
}
