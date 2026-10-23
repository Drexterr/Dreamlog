package handlers

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/dreamlog/backend/internal/middleware"
	"github.com/dreamlog/backend/internal/models"
	"github.com/dreamlog/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type companyRepo interface {
	GetBySlug(ctx context.Context, slug string) (*models.Company, error)
	IsMember(ctx context.Context, companyID, userID uuid.UUID) (*models.CompanyMember, error)
	TotalMembers(ctx context.Context, companyID uuid.UUID) (int, error)
	TeamMoodHistory(ctx context.Context, companyID uuid.UUID, since, until time.Time) ([]*models.TeamDailyMood, error)
	JoinCompany(ctx context.Context, companyID, userID uuid.UUID, seatLimit int) (bool, error)
}

type B2BHandler struct {
	repo companyRepo
}

func NewB2BHandler(repo companyRepo) *B2BHandler {
	return &B2BHandler{repo: repo}
}

// POST /b2b/companies/:slug/join
// Adds the authenticated user to the company.
func (h *B2BHandler) Join(c *gin.Context) {
	userID := middleware.UserIDFromCtx(c.Request.Context())
	slug := c.Param("slug")

	company, err := h.repo.GetBySlug(c.Request.Context(), slug)
	if err != nil {
		_ = c.Error(apierr.Internal("lookup failed"))
		return
	}
	if company == nil {
		_ = c.Error(apierr.NotFound("company not found"))
		return
	}

	// Knowing a slug is not enough to enrol: only employees whose email is on
	// the company's domain may self-join (stops seat burning and mood injection).
	// The address must be verified: anyone can register any email locally (N6).
	user := middleware.UserFromCtx(c.Request.Context())
	if user != nil && !user.EmailVerified {
		_ = c.Error(apierr.Forbidden("verify your email address before joining a company"))
		return
	}
	if user == nil || !sameEmailDomain(user.Email, company.AdminEmail) {
		_ = c.Error(apierr.Forbidden("your email address is not on this company's domain"))
		return
	}

	joined, err := h.repo.JoinCompany(c.Request.Context(), company.ID, userID, company.SeatLimit)
	if err != nil {
		_ = c.Error(apierr.Internal("failed to join company"))
		return
	}
	if !joined {
		_ = c.Error(apierr.Conflict("company has reached its seat limit"))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"company_id":   company.ID,
		"company_name": company.Name,
		"role":         "member",
	})
}

// GET /b2b/companies/:slug/mood?range=30d|90d
// Returns anonymised team mood summary. Only accessible to company admins.
func (h *B2BHandler) TeamMood(c *gin.Context) {
	userID := middleware.UserIDFromCtx(c.Request.Context())
	slug := c.Param("slug")

	company, err := h.repo.GetBySlug(c.Request.Context(), slug)
	if err != nil {
		_ = c.Error(apierr.Internal("lookup failed"))
		return
	}
	if company == nil {
		_ = c.Error(apierr.NotFound("company not found"))
		return
	}

	member, err := h.repo.IsMember(c.Request.Context(), company.ID, userID)
	if err != nil {
		_ = c.Error(apierr.Internal("membership check failed"))
		return
	}
	if member == nil || member.Role != "admin" {
		_ = c.Error(apierr.Forbidden("admin access required"))
		return
	}

	rangeStr := c.DefaultQuery("range", "30d")
	var days int
	switch rangeStr {
	case "90d":
		days = 90
	default:
		days = 30
	}

	until := time.Now().UTC()
	since := until.AddDate(0, 0, -days)
	prevSince := since.AddDate(0, 0, -days)

	current, err := h.repo.TeamMoodHistory(c.Request.Context(), company.ID, since, until)
	if err != nil {
		_ = c.Error(apierr.Internal("failed to load team mood"))
		return
	}
	prior, err := h.repo.TeamMoodHistory(c.Request.Context(), company.ID, prevSince, since)
	if err != nil {
		_ = c.Error(apierr.Internal("failed to load prior mood"))
		return
	}

	total, _ := h.repo.TotalMembers(c.Request.Context(), company.ID)

	// k-anonymity: hide every day below the group-size floor, and only
	// aggregate days that pass it, so no number traces back to one person.
	current = suppressSmallGroups(current)
	prior = suppressSmallGroups(prior)

	avgMood, _ := weightedAvg(current)
	prevAvg, _ := weightedAvg(prior)

	var moodDelta *int
	if avgMood != nil && prevAvg != nil {
		d := *avgMood - *prevAvg
		moodDelta = &d
	}

	const alertThreshold = 40 // configurable per-company in future
	isAlerted := avgMood != nil && *avgMood < alertThreshold

	if current == nil {
		current = []*models.TeamDailyMood{}
	}

	c.JSON(http.StatusOK, models.TeamMoodSummary{
		CompanyID:      company.ID,
		CompanyName:    company.Name,
		TotalMembers:   total,
		Days:           current,
		AvgMood:        avgMood,
		PrevAvgMood:    prevAvg,
		MoodDelta:      moodDelta,
		AlertThreshold: alertThreshold,
		IsAlerted:      isAlerted,
	})
}

// weightedAvg computes entry-count–weighted average mood across daily rows.
func weightedAvg(days []*models.TeamDailyMood) (*int, int) {
	var totalMood, totalEntries int
	for _, d := range days {
		if d.AvgMood == nil {
			continue // suppressed day - never feeds an aggregate
		}
		totalMood += *d.AvgMood * d.EntryCount
		totalEntries += d.EntryCount
	}
	if totalEntries == 0 {
		return nil, 0
	}
	v := totalMood / totalEntries
	return &v, totalEntries
}

// suppressSmallGroups nulls the mood (and entry count, which could be used to
// narrow down who journalled) of any day with fewer than TeamMinGroupSize
// distinct members.
func suppressSmallGroups(days []*models.TeamDailyMood) []*models.TeamDailyMood {
	for _, d := range days {
		if d.ActiveMembers < models.TeamMinGroupSize {
			d.AvgMood = nil
			d.EntryCount = 0
		}
	}
	return days
}

// sameEmailDomain reports whether two addresses share a (case-insensitive) domain.
func sameEmailDomain(a, b string) bool {
	da, db := emailDomain(a), emailDomain(b)
	return da != "" && da == db
}

func emailDomain(email string) string {
	at := strings.LastIndex(email, "@")
	if at < 0 || at == len(email)-1 {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(email[at+1:]))
}
