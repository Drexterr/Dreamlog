package handlers

import (
	"net/http"

	"github.com/dreamlog/backend/internal/middleware"
	"github.com/dreamlog/backend/internal/models"
	"github.com/dreamlog/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

type AnnualReviewHandler struct {
	repo annualReviewListRepo
}

func NewAnnualReviewHandler(repo annualReviewListRepo) *AnnualReviewHandler {
	return &AnnualReviewHandler{repo: repo}
}

// GetLatest returns the most recent completed annual review for the authenticated user.
// GET /reviews/annual/latest — requires DreamLog+ or higher.
func (h *AnnualReviewHandler) GetLatest(c *gin.Context) {
	user := middleware.UserFromCtx(c.Request.Context())
	if user == nil {
		c.Error(apierr.Unauthorized("user not found"))
		return
	}
	if !user.Plan.AtLeast(models.PlanPlus) {
		c.Error(apierr.Forbidden("annual reviews require DreamLog+ or higher"))
		return
	}
	rv, err := h.repo.GetLatestCompleted(c.Request.Context(), user.ID)
	if err != nil {
		c.Error(apierr.Internal("fetch annual review"))
		return
	}
	if rv == nil {
		c.Error(apierr.NotFound("annual review"))
		return
	}
	c.JSON(http.StatusOK, rv)
}

// List returns all completed annual reviews for the authenticated user.
// GET /reviews/annual — requires DreamLog+ or higher.
func (h *AnnualReviewHandler) List(c *gin.Context) {
	user := middleware.UserFromCtx(c.Request.Context())
	if user == nil {
		c.Error(apierr.Unauthorized("user not found"))
		return
	}
	if !user.Plan.AtLeast(models.PlanPlus) {
		c.Error(apierr.Forbidden("annual reviews require DreamLog+ or higher"))
		return
	}
	rv, err := h.repo.ListCompleted(c.Request.Context(), user.ID)
	if err != nil {
		c.Error(apierr.Internal("list annual reviews"))
		return
	}
	if rv == nil {
		rv = []*models.AnnualReview{}
	}
	c.JSON(http.StatusOK, gin.H{"reviews": rv})
}
