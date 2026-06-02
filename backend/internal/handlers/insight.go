package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/dreamlog/backend/internal/middleware"
	"github.com/dreamlog/backend/internal/models"
	"github.com/dreamlog/backend/internal/repositories"
	"github.com/dreamlog/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// insightRepo is the minimal interface InsightHandler needs.
type insightRepo interface {
	GetCardData(ctx context.Context, userID uuid.UUID, weekStart time.Time) (*models.InsightCardData, error)
	RecordShare(ctx context.Context, userID uuid.UUID, weekStart string) (*models.InsightShare, error)
	CountByUser(ctx context.Context, userID uuid.UUID) (int, error)
}

// insightStreakQuerier gets streak info for the card.
type insightStreakQuerier interface {
	StreakInfo(ctx context.Context, userID uuid.UUID) (*models.StreakInfo, error)
}

// compile-time check
var _ insightRepo = (*repositories.InsightShareRepository)(nil)

type InsightHandler struct {
	repo       insightRepo
	streakRepo insightStreakQuerier
}

func NewInsightHandler(repo insightRepo, streakRepo insightStreakQuerier) *InsightHandler {
	return &InsightHandler{repo: repo, streakRepo: streakRepo}
}

// GET /insights/card — returns all data needed to render the week's shareable insight card.
// Available to all plans (not gated) to maximise viral sharing.
func (h *InsightHandler) GetCard(c *gin.Context) {
	userID := middleware.UserIDFromCtx(c.Request.Context())

	weekStart := repositories.CurrentWeekStart()

	data, err := h.repo.GetCardData(c.Request.Context(), userID, weekStart)
	if err != nil {
		_ = c.Error(apierr.Internal("failed to load insight card data"))
		return
	}

	// Attach streak from the analysis repository.
	if info, err := h.streakRepo.StreakInfo(c.Request.Context(), userID); err == nil {
		data.Streak = info.CurrentStreak
	}

	c.JSON(http.StatusOK, data)
}

// POST /insights/share — records that the user shared their insight card.
// Returns updated share count. Available to all plans.
func (h *InsightHandler) RecordShare(c *gin.Context) {
	userID := middleware.UserIDFromCtx(c.Request.Context())

	var req struct {
		WeekStart string `json:"week_start"` // YYYY-MM-DD; defaults to current week if omitted
	}
	_ = c.ShouldBindJSON(&req) // optional body

	weekStart := repositories.CurrentWeekStart().Format("2006-01-02")
	if req.WeekStart != "" {
		weekStart = req.WeekStart
	}

	_, err := h.repo.RecordShare(c.Request.Context(), userID, weekStart)
	if err != nil {
		_ = c.Error(apierr.Internal("failed to record share"))
		return
	}

	total, err := h.repo.CountByUser(c.Request.Context(), userID)
	if err != nil {
		total = 0 // non-critical; return what we have
	}

	c.JSON(http.StatusCreated, models.InsightShareResult{
		TotalShares: total,
		WeekStart:   weekStart,
	})
}
