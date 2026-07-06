package handlers

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/dreamlog/backend/internal/middleware"
	"github.com/dreamlog/backend/internal/models"
	"github.com/dreamlog/backend/internal/repositories"
	"github.com/dreamlog/backend/internal/services"
	"github.com/dreamlog/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// compile-time interface satisfaction checks
var _ flashbackQuerier = (*repositories.AnalysisRepository)(nil)
var _ checkinNudger = (*services.NudgeService)(nil)

// flashbackQuerier is the slice of AnalysisRepository HookHandler needs.
type flashbackQuerier interface {
	Flashback(ctx context.Context, userID uuid.UUID) (*models.Flashback, error)
}

// checkinNudger is the slice of NudgeService HookHandler needs.
type checkinNudger interface {
	ScheduleCheckinNudge(ctx context.Context, userID, entryID uuid.UUID, message string) (time.Time, error)
}

// entryGetter is the slice of EntryRepository HookHandler needs.
type entryGetter interface {
	GetByID(ctx context.Context, id, userID uuid.UUID) (*models.Entry, error)
}

// analysisGetter is the slice of AnalysisRepository HookHandler needs for check-ins.
type analysisGetter interface {
	GetByEntryID(ctx context.Context, entryID uuid.UUID) (*models.EntryAnalysis, error)
}

// HookHandler serves the habit-loop endpoints: flashbacks (time capsule) and
// user-requested check-in nudges.
type HookHandler struct {
	entryRepo    entryGetter
	analysisRepo analysisGetter
	flashbacks   flashbackQuerier
	nudges       checkinNudger
}

func NewHookHandler(entryRepo entryGetter, analysisRepo analysisGetter, flashbacks flashbackQuerier, nudges checkinNudger) *HookHandler {
	return &HookHandler{
		entryRepo:    entryRepo,
		analysisRepo: analysisRepo,
		flashbacks:   flashbacks,
		nudges:       nudges,
	}
}

// GET /entries/flashback
// Returns a past entry from ~1 year ago (or ~1 month ago) to resurface as a
// time capsule. 404 when neither window has an entry.
func (h *HookHandler) GetFlashback(c *gin.Context) {
	userID := middleware.UserIDFromCtx(c.Request.Context())

	fb, err := h.flashbacks.Flashback(c.Request.Context(), userID)
	if err != nil {
		c.Error(err)
		return
	}
	if fb == nil {
		c.Error(apierr.NotFound("flashback"))
		return
	}
	if fb.Topics == nil {
		fb.Topics = []string{}
	}
	c.JSON(http.StatusOK, fb)
}

// POST /entries/:id/checkin
// Schedules a "check in on this tomorrow" nudge for the entry - a self-set
// trigger the user opts into right after reading their reflection.
func (h *HookHandler) CreateCheckin(c *gin.Context) {
	userID := middleware.UserIDFromCtx(c.Request.Context())
	entryID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.Error(apierr.BadRequest("invalid entry id"))
		return
	}

	entry, err := h.entryRepo.GetByID(c.Request.Context(), entryID, userID)
	if err != nil || entry == nil {
		c.Error(apierr.NotFound("entry"))
		return
	}
	if entry.Status != models.EntryStatusCompleted {
		c.Error(apierr.Conflict("entry is not completed yet"))
		return
	}

	// Analysis may legitimately be nil (e.g. crisis-skipped fields) - the
	// message composer falls back to a generic caring line.
	analysis, err := h.analysisRepo.GetByEntryID(c.Request.Context(), entryID)
	if err != nil {
		c.Error(err)
		return
	}

	scheduledAt, err := h.nudges.ScheduleCheckinNudge(c.Request.Context(), userID, entryID, services.CheckinMessage(analysis))
	if err != nil {
		switch {
		case errors.Is(err, services.ErrCheckinExists):
			c.Error(apierr.Conflict("a check-in is already scheduled for this entry"))
		case errors.Is(err, services.ErrNudgesDisabled):
			c.Error(apierr.Conflict("nudges are disabled - enable them in settings first"))
		default:
			c.Error(err)
		}
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"entry_id":     entryID,
		"scheduled_at": scheduledAt.UTC(),
	})
}
