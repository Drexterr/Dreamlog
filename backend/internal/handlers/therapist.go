package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/dreamlog/backend/internal/middleware"
	"github.com/dreamlog/backend/internal/models"
	"github.com/dreamlog/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type therapistRepo interface {
	GetByUserID(ctx context.Context, userID uuid.UUID) (*models.Therapist, error)
	Register(ctx context.Context, userID uuid.UUID, name, email, credentials string) (*models.Therapist, error)
	LinkClient(ctx context.Context, therapistID, clientID uuid.UUID) error
	UnlinkClient(ctx context.Context, therapistID, clientID uuid.UUID) error
	ListClients(ctx context.Context, therapistID uuid.UUID) ([]*models.ClientSummary, error)
	GetClientLink(ctx context.Context, therapistID, clientID uuid.UUID) (*models.ClientTherapistLink, error)
	ClientRecentEntries(ctx context.Context, clientID uuid.UUID) ([]*models.ExportEntrySummary, error)
	ClientMoodStats(ctx context.Context, clientID uuid.UUID) (avg7d *int, topEmotions []string, trend string, err error)
	ClientEntryCount(ctx context.Context, clientID uuid.UUID) (int, error)
	ClientDisplayName(ctx context.Context, clientID uuid.UUID) (string, error)
	ClientRecentSummariesText(ctx context.Context, clientID uuid.UUID, since time.Time) (string, error)
}

type therapistAnalysisRepo interface {
	ExportData(ctx context.Context, userID uuid.UUID, since, until time.Time) (*models.ExportData, error)
}

type briefGenerator interface {
	GenerateBrief(ctx context.Context, clientName, recentSummaries, trend string, avg7d *int) (string, error)
}

type TherapistHandler struct {
	repo         therapistRepo
	analysisRepo therapistAnalysisRepo
	claude       briefGenerator
}

func NewTherapistHandler(repo therapistRepo, analysisRepo therapistAnalysisRepo, claude briefGenerator) *TherapistHandler {
	return &TherapistHandler{repo: repo, analysisRepo: analysisRepo, claude: claude}
}

// POST /therapists/register
func (h *TherapistHandler) Register(c *gin.Context) {
	userID := middleware.UserIDFromCtx(c.Request.Context())

	var body struct {
		Name        string `json:"name"        binding:"required"`
		Email       string `json:"email"       binding:"required,email"`
		Credentials string `json:"credentials"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		_ = c.Error(apierr.BadRequest("name and email are required"))
		return
	}

	t, err := h.repo.Register(c.Request.Context(), userID, body.Name, body.Email, body.Credentials)
	if err != nil {
		_ = c.Error(apierr.Internal("registration failed"))
		return
	}
	c.JSON(http.StatusCreated, t)
}

// POST /therapists/clients/link
// Body: { "client_id": "uuid" }  - client must have shared their UUID out-of-band.
func (h *TherapistHandler) LinkClient(c *gin.Context) {
	userID := middleware.UserIDFromCtx(c.Request.Context())

	therapist, err := h.repo.GetByUserID(c.Request.Context(), userID)
	if err != nil || therapist == nil {
		_ = c.Error(apierr.Forbidden("therapist account required"))
		return
	}

	var body struct {
		ClientID string `json:"client_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		_ = c.Error(apierr.BadRequest("client_id required"))
		return
	}
	clientID, err := uuid.Parse(body.ClientID)
	if err != nil {
		_ = c.Error(apierr.BadRequest("invalid client_id"))
		return
	}

	if err := h.repo.LinkClient(c.Request.Context(), therapist.ID, clientID); err != nil {
		_ = c.Error(apierr.Internal("failed to link client"))
		return
	}
	c.JSON(http.StatusOK, gin.H{"therapist_id": therapist.ID, "client_id": clientID, "status": "active"})
}

// DELETE /therapists/clients/:clientID
func (h *TherapistHandler) UnlinkClient(c *gin.Context) {
	userID := middleware.UserIDFromCtx(c.Request.Context())

	therapist, err := h.repo.GetByUserID(c.Request.Context(), userID)
	if err != nil || therapist == nil {
		_ = c.Error(apierr.Forbidden("therapist account required"))
		return
	}

	clientID, err := uuid.Parse(c.Param("clientID"))
	if err != nil {
		_ = c.Error(apierr.BadRequest("invalid client id"))
		return
	}

	if err := h.repo.UnlinkClient(c.Request.Context(), therapist.ID, clientID); err != nil {
		_ = c.Error(apierr.Internal("failed to unlink client"))
		return
	}
	c.Status(http.StatusNoContent)
}

// GET /therapists/clients
func (h *TherapistHandler) ListClients(c *gin.Context) {
	userID := middleware.UserIDFromCtx(c.Request.Context())

	therapist, err := h.repo.GetByUserID(c.Request.Context(), userID)
	if err != nil || therapist == nil {
		_ = c.Error(apierr.Forbidden("therapist account required"))
		return
	}

	clients, err := h.repo.ListClients(c.Request.Context(), therapist.ID)
	if err != nil {
		_ = c.Error(apierr.Internal("failed to load clients"))
		return
	}
	if clients == nil {
		clients = []*models.ClientSummary{}
	}
	c.JSON(http.StatusOK, gin.H{"clients": clients})
}

// GET /therapists/clients/:clientID/brief
// Generates a Claude pre-session brief for the specified client.
func (h *TherapistHandler) ClientBrief(c *gin.Context) {
	userID := middleware.UserIDFromCtx(c.Request.Context())
	ctx := c.Request.Context()

	therapist, err := h.repo.GetByUserID(ctx, userID)
	if err != nil || therapist == nil {
		_ = c.Error(apierr.Forbidden("therapist account required"))
		return
	}

	clientID, err := uuid.Parse(c.Param("clientID"))
	if err != nil {
		_ = c.Error(apierr.BadRequest("invalid client id"))
		return
	}

	// Verify the link exists.
	link, err := h.repo.GetClientLink(ctx, therapist.ID, clientID)
	if err != nil {
		_ = c.Error(apierr.Internal("link check failed"))
		return
	}
	if link == nil {
		_ = c.Error(apierr.NotFound("client"))
		return
	}

	// Gather data concurrently via sequential calls (acceptable for low-traffic endpoint).
	displayName, _ := h.repo.ClientDisplayName(ctx, clientID)
	avg7d, topEmotions, trend, _ := h.repo.ClientMoodStats(ctx, clientID)
	entryCount, _ := h.repo.ClientEntryCount(ctx, clientID)
	recentEntries, _ := h.repo.ClientRecentEntries(ctx, clientID)

	since := time.Now().UTC().AddDate(0, 0, -7)
	summariesText, _ := h.repo.ClientRecentSummariesText(ctx, clientID, since)

	brief, err := h.claude.GenerateBrief(ctx, displayName, summariesText, trend, avg7d)
	if err != nil {
		_ = c.Error(apierr.Internal("failed to generate brief"))
		return
	}

	c.JSON(http.StatusOK, models.ClientBrief{
		ClientID:      clientID,
		ClientName:    displayName,
		GeneratedAt:   time.Now().UTC(),
		Brief:         brief,
		TopEmotions:   topEmotions,
		MoodTrend:     trend,
		AvgMood7d:     avg7d,
		EntryCount:    entryCount,
		RecentEntries: recentEntries,
	})
}
