package handlers

import (
	"net/http"

	"github.com/dreamlog/backend/internal/middleware"
	"github.com/dreamlog/backend/internal/models"
	"github.com/dreamlog/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

type WeeklyReviewHandler struct {
	repo weeklyReviewListRepo
}

func NewWeeklyReviewHandler(repo weeklyReviewListRepo) *WeeklyReviewHandler {
	return &WeeklyReviewHandler{repo: repo}
}

// GetLatest returns the most recent completed weekly review for the authenticated user.
// GET /reviews/weekly/latest — requires DreamLog+ or higher.
func (h *WeeklyReviewHandler) GetLatest(c *gin.Context) {
	user := middleware.UserFromCtx(c.Request.Context())
	if user == nil {
		c.Error(apierr.Unauthorized("user not found"))
		return
	}
	if !user.Plan.AtLeast(models.PlanPlus) {
		c.Error(apierr.Forbidden("weekly reviews require DreamLog+ or higher"))
		return
	}
	rv, err := h.repo.GetLatestCompleted(c.Request.Context(), user.ID)
	if err != nil {
		c.Error(apierr.Internal("fetch weekly review"))
		return
	}
	if rv == nil {
		c.Error(apierr.NotFound("weekly review"))
		return
	}
	c.JSON(http.StatusOK, rv)
}

// List returns the most recent completed weekly reviews for the authenticated user.
// GET /reviews/weekly — requires DreamLog+ or higher.
func (h *WeeklyReviewHandler) List(c *gin.Context) {
	user := middleware.UserFromCtx(c.Request.Context())
	if user == nil {
		c.Error(apierr.Unauthorized("user not found"))
		return
	}
	if !user.Plan.AtLeast(models.PlanPlus) {
		c.Error(apierr.Forbidden("weekly reviews require DreamLog+ or higher"))
		return
	}
	rv, err := h.repo.ListCompleted(c.Request.Context(), user.ID, 10)
	if err != nil {
		c.Error(apierr.Internal("list weekly reviews"))
		return
	}
	if rv == nil {
		rv = []*models.WeeklyReview{}
	}
	c.JSON(http.StatusOK, gin.H{"reviews": rv})
}
