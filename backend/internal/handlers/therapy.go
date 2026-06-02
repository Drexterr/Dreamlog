package handlers

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/dreamlog/backend/internal/middleware"
	"github.com/dreamlog/backend/internal/models"
	"github.com/dreamlog/backend/internal/services"
	"github.com/dreamlog/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// therapyServicer is the minimal interface TherapyHandler needs.
type therapyServicer interface {
	StartSession(ctx context.Context, userID uuid.UUID, userPlan models.Plan, persona models.TherapyPersona) (*models.TherapySession, error)
	SendMessage(ctx context.Context, sessionID, userID uuid.UUID, req models.SendTherapyMessageRequest) (*models.SendTherapyMessageResponse, error)
	EndSession(ctx context.Context, sessionID, userID uuid.UUID) (*models.EndSessionResponse, error)
	GetSession(ctx context.Context, sessionID, userID uuid.UUID) (*models.TherapySession, error)
	ListSessions(ctx context.Context, userID uuid.UUID) (*models.ListTherapySessionsResponse, error)
}

// therapyPresigner generates pre-signed PUT URLs for voice turns.
// Satisfied by *services.StorageService in production.
type therapyPresigner interface {
	PresignPut(ctx context.Context, key string, contentType string, expiry time.Duration) (string, string, error)
}

type TherapyHandler struct {
	svc      therapyServicer
	storage  therapyPresigner
	userRepo therapyUserGetter
}

// therapyUserGetter fetches the user record to read their plan.
type therapyUserGetter interface {
	GetByID(ctx context.Context, id uuid.UUID) (*models.User, error)
}

func NewTherapyHandler(svc therapyServicer, storage therapyPresigner, userRepo therapyUserGetter) *TherapyHandler {
	return &TherapyHandler{svc: svc, storage: storage, userRepo: userRepo}
}

// POST /therapy/sessions
func (h *TherapyHandler) StartSession(c *gin.Context) {
	userID := middleware.UserIDFromCtx(c.Request.Context())

	var req models.StartSessionRequest
	// Body is optional — ignore binding error; persona defaults to "comforting"
	_ = c.ShouldBindJSON(&req)

	persona := models.TherapyPersona(req.Persona)
	if persona == "" {
		persona = models.PersonaComforting
	}
	if !models.ValidPersona(string(persona)) {
		c.Error(apierr.BadRequest("invalid persona; must be one of: comforting, rational, cbt, mindful"))
		return
	}

	user, err := h.userRepo.GetByID(c.Request.Context(), userID)
	if err != nil || user == nil {
		c.Error(apierr.Internal("could not load user"))
		return
	}

	session, err := h.svc.StartSession(c.Request.Context(), userID, user.Plan, persona)
	if err != nil {
		if isPaymentRequired(err) {
			c.JSON(http.StatusPaymentRequired, gin.H{
				"error": "no therapy session credits remaining",
				"code":  "PAYMENT_REQUIRED",
			})
			return
		}
		c.Error(err)
		return
	}

	snap := session.ContextSnapshot
	c.JSON(http.StatusCreated, models.StartSessionResponse{
		ID:                 session.ID,
		Status:             string(session.Status),
		Persona:            session.Persona,
		StartedAt:          session.StartedAt,
		ExpiresAt:          session.ExpiresAt,
		ContextLoaded:      snap.MoodAvg30d != nil || len(snap.RecentSummaries) > 0,
		HasSessionHistory:  len(snap.PastSessionSummaries) > 0,
		BillingAmountPaise: session.BillingAmountPaise,
	})
}

// POST /therapy/sessions/:id/presign
func (h *TherapyHandler) PresignAudio(c *gin.Context) {
	userID := middleware.UserIDFromCtx(c.Request.Context())
	sessionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.Error(apierr.BadRequest("invalid session id"))
		return
	}

	var req models.TherapyPresignRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apierr.BadRequest("invalid request body", err.Error()))
		return
	}

	session, err := h.svc.GetSession(c.Request.Context(), sessionID, userID)
	if err != nil || session == nil {
		c.Error(apierr.NotFound("session not found"))
		return
	}
	if session.Status != models.TherapyStatusActive {
		c.Error(apierr.Conflict("session is not active"))
		return
	}

	uploadURL, audioKey, err := h.storage.PresignPut(c.Request.Context(), req.Filename, req.ContentType, 15*time.Minute)
	if err != nil {
		c.Error(apierr.Internal("could not generate upload URL"))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"upload_url": uploadURL,
		"audio_key":  audioKey,
	})
}

// POST /therapy/sessions/:id/messages
func (h *TherapyHandler) SendMessage(c *gin.Context) {
	userID := middleware.UserIDFromCtx(c.Request.Context())
	sessionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.Error(apierr.BadRequest("invalid session id"))
		return
	}

	var req models.SendTherapyMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apierr.BadRequest("invalid request body", err.Error()))
		return
	}

	if req.InputMode == "text" && req.Content == "" {
		c.Error(apierr.BadRequest("content is required for text input"))
		return
	}
	if req.InputMode == "voice" && req.AudioKey == "" {
		c.Error(apierr.BadRequest("audio_key is required for voice input"))
		return
	}

	resp, err := h.svc.SendMessage(c.Request.Context(), sessionID, userID, req)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrTherapyNotFound):
			c.Error(apierr.NotFound("session not found"))
		case errors.Is(err, services.ErrTherapyNotActive):
			c.Error(apierr.Conflict("session is not active"))
		case errors.Is(err, services.ErrTherapyExpired):
			c.JSON(http.StatusGone, gin.H{"error": "session expired", "code": "SESSION_EXPIRED"})
		default:
			c.Error(err)
		}
		return
	}

	c.JSON(http.StatusCreated, resp)
}

// POST /therapy/sessions/:id/end
func (h *TherapyHandler) EndSession(c *gin.Context) {
	userID := middleware.UserIDFromCtx(c.Request.Context())
	sessionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.Error(apierr.BadRequest("invalid session id"))
		return
	}

	resp, err := h.svc.EndSession(c.Request.Context(), sessionID, userID)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrTherapyNotFound):
			c.Error(apierr.NotFound("session not found"))
		case errors.Is(err, services.ErrTherapyAlreadyEnded):
			c.Error(apierr.Conflict("session already ended"))
		default:
			c.Error(err)
		}
		return
	}

	c.JSON(http.StatusOK, resp)
}

// GET /therapy/sessions/:id
func (h *TherapyHandler) GetSession(c *gin.Context) {
	userID := middleware.UserIDFromCtx(c.Request.Context())
	sessionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.Error(apierr.BadRequest("invalid session id"))
		return
	}

	session, err := h.svc.GetSession(c.Request.Context(), sessionID, userID)
	if err != nil {
		if errors.Is(err, services.ErrTherapyNotFound) {
			c.Error(apierr.NotFound("session not found"))
			return
		}
		c.Error(err)
		return
	}
	if session == nil {
		c.Error(apierr.NotFound("session not found"))
		return
	}

	c.JSON(http.StatusOK, session)
}

// GET /therapy/sessions
func (h *TherapyHandler) ListSessions(c *gin.Context) {
	userID := middleware.UserIDFromCtx(c.Request.Context())

	resp, err := h.svc.ListSessions(c.Request.Context(), userID)
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func isPaymentRequired(err error) bool {
	return err != nil && err.Error() == "therapy session requires payment"
}
