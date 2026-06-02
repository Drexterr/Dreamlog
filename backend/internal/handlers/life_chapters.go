package handlers

import (
	"context"
	"net/http"

	"github.com/dreamlog/backend/internal/middleware"
	"github.com/dreamlog/backend/internal/models"
	"github.com/dreamlog/backend/internal/services"
	"github.com/dreamlog/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// chapterSummarizer generates a Claude summary for a life chapter.
// Satisfied by *services.ClaudeService in production.
type chapterSummarizer interface {
	GenerateChapterSummary(ctx context.Context, input services.ChapterSummaryPromptInput) (*services.ChapterSummaryOutput, error)
}

type LifeChapterHandler struct {
	repo   lifeChapterRepo
	claude chapterSummarizer
}

func NewLifeChapterHandler(repo lifeChapterRepo, claude chapterSummarizer) *LifeChapterHandler {
	return &LifeChapterHandler{repo: repo, claude: claude}
}

// List returns all life chapters for the authenticated user.
// GET /chapters
func (h *LifeChapterHandler) List(c *gin.Context) {
	user := middleware.UserFromCtx(c.Request.Context())
	if user == nil {
		c.Error(apierr.Unauthorized("user not found"))
		return
	}
	chapters, err := h.repo.List(c.Request.Context(), user.ID)
	if err != nil {
		c.Error(apierr.Internal("list chapters"))
		return
	}
	if chapters == nil {
		chapters = []*models.LifeChapter{}
	}
	c.JSON(http.StatusOK, gin.H{"chapters": chapters})
}

// Create adds a new life chapter.
// POST /chapters
func (h *LifeChapterHandler) Create(c *gin.Context) {
	user := middleware.UserFromCtx(c.Request.Context())
	if user == nil {
		c.Error(apierr.Unauthorized("user not found"))
		return
	}
	var input models.CreateChapterInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.Error(apierr.BadRequest(err.Error()))
		return
	}
	ch, err := h.repo.Create(c.Request.Context(), user.ID, input)
	if err != nil {
		c.Error(apierr.Internal("create chapter"))
		return
	}
	c.JSON(http.StatusCreated, ch)
}

// GetByID returns a single chapter.
// GET /chapters/:id
func (h *LifeChapterHandler) GetByID(c *gin.Context) {
	user := middleware.UserFromCtx(c.Request.Context())
	if user == nil {
		c.Error(apierr.Unauthorized("user not found"))
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.Error(apierr.BadRequest("invalid chapter id"))
		return
	}
	ch, err := h.repo.GetByID(c.Request.Context(), id, user.ID)
	if err != nil {
		c.Error(apierr.Internal("get chapter"))
		return
	}
	if ch == nil {
		c.Error(apierr.NotFound("chapter"))
		return
	}
	c.JSON(http.StatusOK, ch)
}

// Update patches a chapter's mutable fields.
// PUT /chapters/:id
func (h *LifeChapterHandler) Update(c *gin.Context) {
	user := middleware.UserFromCtx(c.Request.Context())
	if user == nil {
		c.Error(apierr.Unauthorized("user not found"))
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.Error(apierr.BadRequest("invalid chapter id"))
		return
	}
	var input models.UpdateChapterInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.Error(apierr.BadRequest(err.Error()))
		return
	}
	ch, err := h.repo.Update(c.Request.Context(), id, user.ID, input)
	if err != nil {
		c.Error(apierr.Internal("update chapter"))
		return
	}
	if ch == nil {
		c.Error(apierr.NotFound("chapter"))
		return
	}
	c.JSON(http.StatusOK, ch)
}

// Delete removes a chapter.
// DELETE /chapters/:id
func (h *LifeChapterHandler) Delete(c *gin.Context) {
	user := middleware.UserFromCtx(c.Request.Context())
	if user == nil {
		c.Error(apierr.Unauthorized("user not found"))
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.Error(apierr.BadRequest("invalid chapter id"))
		return
	}
	if err := h.repo.Delete(c.Request.Context(), id, user.ID); err != nil {
		c.Error(apierr.Internal("delete chapter"))
		return
	}
	c.Status(http.StatusNoContent)
}

// GetDetail returns a chapter enriched with entry stats.
// GET /chapters/:id/detail
func (h *LifeChapterHandler) GetDetail(c *gin.Context) {
	user := middleware.UserFromCtx(c.Request.Context())
	if user == nil {
		c.Error(apierr.Unauthorized("user not found"))
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.Error(apierr.BadRequest("invalid chapter id"))
		return
	}
	detail, err := h.repo.GetDetail(c.Request.Context(), id, user.ID)
	if err != nil {
		c.Error(apierr.Internal("get chapter detail"))
		return
	}
	if detail == nil {
		c.Error(apierr.NotFound("chapter"))
		return
	}
	c.JSON(http.StatusOK, detail)
}

// Summarize generates or returns a Claude summary for a chapter.
// POST /chapters/:id/summarize
func (h *LifeChapterHandler) Summarize(c *gin.Context) {
	user := middleware.UserFromCtx(c.Request.Context())
	if user == nil {
		c.Error(apierr.Unauthorized("user not found"))
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.Error(apierr.BadRequest("invalid chapter id"))
		return
	}

	detail, err := h.repo.GetDetail(c.Request.Context(), id, user.ID)
	if err != nil {
		c.Error(apierr.Internal("get chapter detail"))
		return
	}
	if detail == nil {
		c.Error(apierr.NotFound("chapter"))
		return
	}

	entries, err := h.repo.GetEntriesInRange(c.Request.Context(), user.ID, detail.StartDate, detail.EndDate)
	if err != nil {
		c.Error(apierr.Internal("get chapter entries"))
		return
	}

	summaries := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.Summary != "" {
			summaries = append(summaries, e.Summary)
		}
	}

	avgMood := 0
	if detail.AvgMood != nil {
		avgMood = *detail.AvgMood
	}

	endDate := ""
	if detail.EndDate != nil {
		endDate = *detail.EndDate
	}

	displayName := user.Name
	if user.PreferredName != nil && *user.PreferredName != "" {
		displayName = *user.PreferredName
	}

	input := services.ChapterSummaryPromptInput{
		Name:        displayName,
		Title:       detail.Title,
		Description: detail.Description,
		StartDate:   detail.StartDate,
		EndDate:     endDate,
		EntryCount:  detail.EntryCount,
		AvgMood:     avgMood,
		TopEmotions: detail.TopEmotions,
		Summaries:   summaries,
	}

	out, err := h.claude.GenerateChapterSummary(c.Request.Context(), input)
	if err != nil {
		c.Error(apierr.Internal("generate chapter summary"))
		return
	}

	if err := h.repo.StoreSummary(c.Request.Context(), id, user.ID, out.Summary); err != nil {
		c.Error(apierr.Internal("store chapter summary"))
		return
	}

	c.JSON(http.StatusOK, gin.H{"summary": out.Summary})
}
