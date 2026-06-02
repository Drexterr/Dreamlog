package handlers

import (
	"net/http"
	"strconv"
	"strings"
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
var _ entryQuerier = (*repositories.EntryRepository)(nil)
var _ analysisQuerier = (*repositories.AnalysisRepository)(nil)

// ── Timeline entry (entry + analysis joined) ─────────────────────────────────

type TimelineEntry struct {
	Entry    *models.Entry         `json:"entry"`
	Analysis *models.EntryAnalysis `json:"analysis,omitempty"`
}

type TimelineResponse struct {
	Entries  []TimelineEntry `json:"entries"`
	Total    int             `json:"total"`
	Page     int             `json:"page"`
	PageSize int             `json:"page_size"`
	HasMore  bool            `json:"has_more"`
}

// ── AnalysisHandler ──────────────────────────────────────────────────────────

type AnalysisHandler struct {
	entryRepo    entryQuerier
	analysisRepo analysisQuerier
	convSvc      *services.ConversationService
}

func NewAnalysisHandler(
	entryRepo entryQuerier,
	analysisRepo analysisQuerier,
	convSvc *services.ConversationService,
) *AnalysisHandler {
	return &AnalysisHandler{
		entryRepo:    entryRepo,
		analysisRepo: analysisRepo,
		convSvc:      convSvc,
	}
}

// GET /entries/:id/analysis
func (h *AnalysisHandler) GetAnalysis(c *gin.Context) {
	userID := middleware.UserIDFromCtx(c.Request.Context())
	entryID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.Error(apierr.BadRequest("invalid entry id"))
		return
	}

	// Ownership check via entry fetch.
	entry, err := h.entryRepo.GetByID(c.Request.Context(), entryID, userID)
	if err != nil || entry == nil {
		c.Error(apierr.NotFound("entry"))
		return
	}

	analysis, err := h.analysisRepo.GetByEntryID(c.Request.Context(), entryID)
	if err != nil {
		c.Error(err)
		return
	}
	if analysis == nil {
		c.Error(apierr.NotFound("analysis"))
		return
	}
	c.JSON(http.StatusOK, analysis)
}

// GET /timeline
// Returns paginated entries with their analyses joined.
func (h *AnalysisHandler) GetTimeline(c *gin.Context) {
	userID := middleware.UserIDFromCtx(c.Request.Context())
	page := parseIntQuery(c, "page", 1)
	pageSize := parseIntQuery(c, "page_size", 20)
	if pageSize > 50 {
		pageSize = 50
	}

	entries, total, err := h.entryRepo.List(c.Request.Context(), repositories.ListEntriesOpts{
		UserID:   userID,
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		c.Error(err)
		return
	}

	timeline := make([]TimelineEntry, 0, len(entries))
	for _, e := range entries {
		te := TimelineEntry{Entry: e}
		if e.Status == models.EntryStatusCompleted {
			a, _ := h.analysisRepo.GetByEntryID(c.Request.Context(), e.ID)
			te.Analysis = a
		}
		timeline = append(timeline, te)
	}

	c.JSON(http.StatusOK, TimelineResponse{
		Entries:  timeline,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
		HasMore:  (page * pageSize) < total,
	})
}

// GET /entries/search?q=...
func (h *AnalysisHandler) Search(c *gin.Context) {
	userID := middleware.UserIDFromCtx(c.Request.Context())
	q := strings.TrimSpace(c.Query("q"))
	if q == "" {
		c.Error(apierr.BadRequest("query parameter 'q' is required"))
		return
	}

	limit := 20
	if l := c.Query("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 50 {
			limit = n
		}
	}

	entries, err := h.entryRepo.SearchEntries(c.Request.Context(), userID, q, limit)
	if err != nil {
		c.Error(err)
		return
	}
	if entries == nil {
		entries = []*models.Entry{}
	}
	c.JSON(http.StatusOK, gin.H{"entries": entries, "query": q})
}

// ── MoodHandler ───────────────────────────────────────────────────────────────

type MoodHandler struct {
	analysisRepo analysisQuerier
	nudgeRepo    deviceRegistrar
	freezeRepo   streakFreezer
}

func NewMoodHandler(analysisRepo analysisQuerier, nudgeRepo deviceRegistrar, freezeRepo streakFreezer) *MoodHandler {
	return &MoodHandler{analysisRepo: analysisRepo, nudgeRepo: nudgeRepo, freezeRepo: freezeRepo}
}

// GET /mood/weekly
func (h *MoodHandler) WeeklyMood(c *gin.Context) {
	userID := middleware.UserIDFromCtx(c.Request.Context())
	moods, err := h.analysisRepo.MoodLast7Days(c.Request.Context(), userID)
	if err != nil {
		c.Error(err)
		return
	}
	if moods == nil {
		moods = []*models.DailyMood{}
	}
	c.JSON(http.StatusOK, gin.H{"days": moods})
}

// GET /mood/streak
func (h *MoodHandler) Streak(c *gin.Context) {
	userID := middleware.UserIDFromCtx(c.Request.Context())
	streak, err := h.analysisRepo.StreakInfo(c.Request.Context(), userID)
	if err != nil {
		c.Error(err)
		return
	}
	// Populate freeze count from users table.
	count, err := h.freezeRepo.StreakFreezeCount(c.Request.Context(), userID)
	if err == nil {
		streak.FreezeCount = count
	}
	c.JSON(http.StatusOK, streak)
}

// POST /streak/freeze
func (h *MoodHandler) UseFreeze(c *gin.Context) {
	userID := middleware.UserIDFromCtx(c.Request.Context())

	var input struct {
		FreezeDate string `json:"freeze_date" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.Error(apierr.BadRequest("freeze_date required (YYYY-MM-DD)"))
		return
	}

	t, err := time.Parse("2006-01-02", input.FreezeDate)
	if err != nil {
		c.Error(apierr.BadRequest("freeze_date must be YYYY-MM-DD"))
		return
	}

	if err := h.freezeRepo.UseStreakFreeze(c.Request.Context(), userID, t); err != nil {
		if err.Error() == "no streak freezes remaining" {
			c.Error(apierr.Conflict("no streak freezes remaining"))
			return
		}
		c.Error(apierr.Internal("use streak freeze"))
		return
	}

	count, _ := h.freezeRepo.StreakFreezeCount(c.Request.Context(), userID)
	c.JSON(http.StatusOK, gin.H{"freeze_count": count, "freeze_date": input.FreezeDate})
}

// GET /mood/history?range=30d|90d|365d — requires DreamLog+ or higher.
func (h *MoodHandler) MoodHistory(c *gin.Context) {
	user := middleware.UserFromCtx(c.Request.Context())
	if user == nil {
		c.Error(apierr.Unauthorized("user not found"))
		return
	}
	if !user.Plan.AtLeast(models.PlanPlus) {
		c.Error(apierr.Forbidden("mood history requires DreamLog+ or higher"))
		return
	}
	userID := user.ID

	rangeParam := c.DefaultQuery("range", "30d")
	var days int
	switch rangeParam {
	case "30d":
		days = 30
	case "90d":
		days = 90
	case "365d":
		days = 365
	default:
		c.Error(apierr.BadRequest("range must be one of: 30d, 90d, 365d"))
		return
	}

	history, err := h.analysisRepo.MoodHistory(c.Request.Context(), userID, days)
	if err != nil {
		c.Error(apierr.Internal("mood history"))
		return
	}
	history.Range = rangeParam
	c.JSON(http.StatusOK, history)
}

// GET /mood/patterns?range=30d|90d|365d
// Returns the top-8 emotions with frequency and intensity data for the Pattern Radar visualization.
// Available to all plans (free-tier eligible, unlike MoodHistory).
func (h *MoodHandler) PatternRadar(c *gin.Context) {
	userID := middleware.UserIDFromCtx(c.Request.Context())

	rangeParam := c.DefaultQuery("range", "30d")
	var days int
	switch rangeParam {
	case "30d":
		days = 30
	case "90d":
		days = 90
	case "365d":
		days = 365
	default:
		c.Error(apierr.BadRequest("range must be one of: 30d, 90d, 365d"))
		return
	}

	resp, err := h.analysisRepo.EmotionPatterns(c.Request.Context(), userID, days)
	if err != nil {
		c.Error(apierr.Internal("pattern radar"))
		return
	}
	resp.Range = rangeParam
	if resp.Emotions == nil {
		resp.Emotions = []models.EmotionPattern{}
	}
	c.JSON(http.StatusOK, resp)
}

// POST /devices
func (h *MoodHandler) RegisterDevice(c *gin.Context) {
	userID := middleware.UserIDFromCtx(c.Request.Context())

	var input models.RegisterDeviceInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.Error(apierr.BadRequest("invalid request body", err.Error()))
		return
	}

	if err := h.nudgeRepo.UpsertDevice(c.Request.Context(), userID, input.FCMToken, input.Platform); err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "registered"})
}
