package handlers

import (
	"context"
	"net/http"

	"github.com/dreamlog/backend/internal/middleware"
	"github.com/dreamlog/backend/internal/models"
	"github.com/dreamlog/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// journeyManager is the minimal interface JourneyHandler needs from JourneyService.
type journeyManager interface {
	ListTemplates() []models.JourneyTemplate
	GetTemplate(journeyID string) (*models.JourneyTemplate, bool)
	StartSession(ctx context.Context, userID uuid.UUID, journeyID string) (*models.JourneySession, error)
	GetSession(ctx context.Context, sessionID, userID uuid.UUID) (*models.JourneySession, error)
	ListSessions(ctx context.Context, userID uuid.UUID) ([]*models.JourneySession, error)
	AdvanceSession(ctx context.Context, sessionID, userID uuid.UUID, entryID uuid.UUID) (*models.JourneySession, error)
}

// JourneyHandler handles all /journeys routes.
type JourneyHandler struct {
	svc journeyManager
}

func NewJourneyHandler(svc journeyManager) *JourneyHandler {
	return &JourneyHandler{svc: svc}
}

// GET /journeys - list available journey templates.
func (h *JourneyHandler) ListTemplates(c *gin.Context) {
	templates := h.svc.ListTemplates()
	c.JSON(http.StatusOK, gin.H{"journeys": templates})
}

// POST /journeys/:journeyID/start - start a new session.
func (h *JourneyHandler) StartSession(c *gin.Context) {
	userID := middleware.UserIDFromCtx(c.Request.Context())
	journeyID := c.Param("journeyID")

	if _, ok := h.svc.GetTemplate(journeyID); !ok {
		c.Error(apierr.NotFound("journey"))
		return
	}

	session, err := h.svc.StartSession(c.Request.Context(), userID, journeyID)
	if err != nil {
		c.Error(apierr.Internal("start journey session"))
		return
	}
	c.JSON(http.StatusCreated, session)
}

// GET /journeys/sessions - list the user's sessions (most recent first).
func (h *JourneyHandler) ListSessions(c *gin.Context) {
	userID := middleware.UserIDFromCtx(c.Request.Context())

	sessions, err := h.svc.ListSessions(c.Request.Context(), userID)
	if err != nil {
		c.Error(apierr.Internal("list journey sessions"))
		return
	}
	if sessions == nil {
		sessions = []*models.JourneySession{}
	}
	c.JSON(http.StatusOK, gin.H{"sessions": sessions})
}

// GET /journeys/sessions/:sessionID - get session state with steps.
func (h *JourneyHandler) GetSession(c *gin.Context) {
	userID := middleware.UserIDFromCtx(c.Request.Context())
	sessionID, err := uuid.Parse(c.Param("sessionID"))
	if err != nil {
		c.Error(apierr.BadRequest("invalid session id"))
		return
	}

	session, err := h.svc.GetSession(c.Request.Context(), sessionID, userID)
	if err != nil {
		c.Error(apierr.NotFound("session"))
		return
	}
	c.JSON(http.StatusOK, session)
}

// POST /journeys/sessions/:sessionID/advance - record entry for current step, advance.
func (h *JourneyHandler) AdvanceSession(c *gin.Context) {
	userID := middleware.UserIDFromCtx(c.Request.Context())
	sessionID, err := uuid.Parse(c.Param("sessionID"))
	if err != nil {
		c.Error(apierr.BadRequest("invalid session id"))
		return
	}

	var input models.AdvanceJourneyInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.Error(apierr.BadRequest("entry_id required"))
		return
	}

	session, err := h.svc.AdvanceSession(c.Request.Context(), sessionID, userID, input.EntryID)
	if err != nil {
		if err.Error() == "session already completed" {
			c.Error(apierr.Conflict("journey session already completed"))
			return
		}
		c.Error(apierr.Internal("advance journey session"))
		return
	}
	c.JSON(http.StatusOK, session)
}
