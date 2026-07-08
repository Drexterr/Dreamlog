package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/dreamlog/backend/internal/middleware"
	"github.com/dreamlog/backend/internal/models"
	"github.com/dreamlog/backend/internal/services"
	"github.com/dreamlog/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// therapistLookup resolves the therapist profile for the authenticated user.
type therapistLookup interface {
	GetByUserID(ctx context.Context, userID uuid.UUID) (*models.Therapist, error)
}

// consentStore records ToS / client-data-consent acceptances.
type consentStore interface {
	AcceptUserTerms(ctx context.Context, userID uuid.UUID, version string) error
	GetUserTerms(ctx context.Context, userID uuid.UUID) (*time.Time, *string, error)
	AcceptTherapistClientConsent(ctx context.Context, therapistID uuid.UUID, version string) error
	TherapistClientConsentAccepted(ctx context.Context, therapistID uuid.UUID) (bool, error)
}

// TherapistNotesHandler serves the in-app therapist workspace: external
// clients, session notes (photo OCR or typed), AI summaries, and overview
// metrics. All routes require an authenticated user with a therapist profile.
type TherapistNotesHandler struct {
	svc        *services.TherapistNotesService
	therapists therapistLookup
	consent    consentStore
}

func NewTherapistNotesHandler(svc *services.TherapistNotesService, therapists therapistLookup, consent consentStore) *TherapistNotesHandler {
	return &TherapistNotesHandler{svc: svc, therapists: therapists, consent: consent}
}

// requireTherapist resolves the caller's therapist profile or writes a 403.
func (h *TherapistNotesHandler) requireTherapist(c *gin.Context) *models.Therapist {
	userID := middleware.UserIDFromCtx(c.Request.Context())
	t, err := h.therapists.GetByUserID(c.Request.Context(), userID)
	if err != nil {
		_ = c.Error(apierr.Internal("therapist lookup failed"))
		return nil
	}
	if t == nil {
		_ = c.Error(apierr.Forbidden("therapist account required"))
		return nil
	}
	return t
}

// requireConsented additionally requires the client-data consent acceptance.
func (h *TherapistNotesHandler) requireConsented(c *gin.Context) *models.Therapist {
	t := h.requireTherapist(c)
	if t == nil {
		return nil
	}
	ok, err := h.consent.TherapistClientConsentAccepted(c.Request.Context(), t.ID)
	if err != nil {
		_ = c.Error(apierr.Internal("consent lookup failed"))
		return nil
	}
	if !ok {
		_ = c.Error(apierr.Forbidden("client-data consent must be accepted first (POST /therapists/consent)"))
		return nil
	}
	return t
}

// ── Profile & consent ────────────────────────────────────────────────────────

// GET /therapists/me
// Returns the caller's therapist profile, or 404 when they have none - the
// mobile login pill uses this to route to registration vs dashboard.
func (h *TherapistNotesHandler) GetMe(c *gin.Context) {
	userID := middleware.UserIDFromCtx(c.Request.Context())
	t, err := h.therapists.GetByUserID(c.Request.Context(), userID)
	if err != nil {
		_ = c.Error(apierr.Internal("therapist lookup failed"))
		return
	}
	if t == nil {
		_ = c.Error(apierr.NotFound("therapist profile"))
		return
	}
	consented, err := h.consent.TherapistClientConsentAccepted(c.Request.Context(), t.ID)
	if err != nil {
		_ = c.Error(apierr.Internal("consent lookup failed"))
		return
	}
	c.JSON(http.StatusOK, gin.H{"therapist": t, "client_consent_accepted": consented})
}

// POST /therapists/consent
// Records the therapist's acceptance of the client-data responsibility terms.
func (h *TherapistNotesHandler) AcceptClientConsent(c *gin.Context) {
	t := h.requireTherapist(c)
	if t == nil {
		return
	}
	var body struct {
		Version string `json:"version"`
	}
	_ = c.ShouldBindJSON(&body)
	if body.Version == "" {
		body.Version = models.CurrentToSVersion
	}
	if err := h.consent.AcceptTherapistClientConsent(c.Request.Context(), t.ID, body.Version); err != nil {
		_ = c.Error(apierr.Internal("failed to record consent"))
		return
	}
	c.JSON(http.StatusOK, gin.H{"client_consent_accepted": true, "version": body.Version})
}

// POST /me/accept-terms
// Records the authenticated USER's ToS/privacy acceptance (any account).
func (h *TherapistNotesHandler) AcceptUserTerms(c *gin.Context) {
	userID := middleware.UserIDFromCtx(c.Request.Context())
	var body struct {
		Version string `json:"version"`
	}
	_ = c.ShouldBindJSON(&body)
	if body.Version == "" {
		body.Version = models.CurrentToSVersion
	}
	if err := h.consent.AcceptUserTerms(c.Request.Context(), userID, body.Version); err != nil {
		_ = c.Error(apierr.Internal("failed to record acceptance"))
		return
	}
	c.JSON(http.StatusOK, gin.H{"tos_accepted": true, "version": body.Version})
}

// GET /me/terms
// Returns the user's current acceptance state plus the version the app should
// require - the client re-prompts when they differ.
func (h *TherapistNotesHandler) GetUserTerms(c *gin.Context) {
	userID := middleware.UserIDFromCtx(c.Request.Context())
	acceptedAt, version, err := h.consent.GetUserTerms(c.Request.Context(), userID)
	if err != nil {
		_ = c.Error(apierr.Internal("failed to load acceptance"))
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"tos_accepted_at": acceptedAt,
		"tos_version":     version,
		"current_version": models.CurrentToSVersion,
	})
}

// ── Overview ─────────────────────────────────────────────────────────────────

// GET /therapists/overview
func (h *TherapistNotesHandler) Overview(c *gin.Context) {
	t := h.requireTherapist(c)
	if t == nil {
		return
	}
	overview, err := h.svc.Overview(c.Request.Context(), t.ID)
	if err != nil {
		_ = c.Error(apierr.Internal("failed to load overview"))
		return
	}
	c.JSON(http.StatusOK, overview)
}

// ── External clients ─────────────────────────────────────────────────────────

// POST /therapists/external-clients
func (h *TherapistNotesHandler) CreateExternalClient(c *gin.Context) {
	t := h.requireConsented(c)
	if t == nil {
		return
	}
	var body struct {
		Name string `json:"name" binding:"required"`
		Role string `json:"role"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		_ = c.Error(apierr.BadRequest("name is required"))
		return
	}
	client, err := h.svc.CreateExternalClient(c.Request.Context(), t.ID, body.Name, body.Role)
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusCreated, client)
}

// GET /therapists/external-clients?include_archived=true
func (h *TherapistNotesHandler) ListExternalClients(c *gin.Context) {
	t := h.requireTherapist(c)
	if t == nil {
		return
	}
	includeArchived := c.Query("include_archived") == "true"
	clients, err := h.svc.ListExternalClients(c.Request.Context(), t.ID, includeArchived)
	if err != nil {
		_ = c.Error(apierr.Internal("failed to load clients"))
		return
	}
	if clients == nil {
		clients = []*models.ExternalClient{}
	}
	c.JSON(http.StatusOK, gin.H{"clients": clients})
}

// GET /therapists/external-clients/:id
func (h *TherapistNotesHandler) GetExternalClient(c *gin.Context) {
	t := h.requireTherapist(c)
	if t == nil {
		return
	}
	clientID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		_ = c.Error(apierr.BadRequest("invalid client id"))
		return
	}
	client, err := h.svc.GetExternalClient(c.Request.Context(), t.ID, clientID)
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, client)
}

// PATCH /therapists/external-clients/:id
func (h *TherapistNotesHandler) UpdateExternalClient(c *gin.Context) {
	t := h.requireTherapist(c)
	if t == nil {
		return
	}
	clientID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		_ = c.Error(apierr.BadRequest("invalid client id"))
		return
	}
	var input services.UpdateExternalClientInput
	if err := c.ShouldBindJSON(&input); err != nil {
		_ = c.Error(apierr.BadRequest("invalid request body"))
		return
	}
	client, err := h.svc.UpdateExternalClient(c.Request.Context(), t.ID, clientID, input)
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, client)
}

// DELETE /therapists/external-clients/:id
func (h *TherapistNotesHandler) DeleteExternalClient(c *gin.Context) {
	t := h.requireTherapist(c)
	if t == nil {
		return
	}
	clientID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		_ = c.Error(apierr.BadRequest("invalid client id"))
		return
	}
	if err := h.svc.DeleteExternalClient(c.Request.Context(), t.ID, clientID); err != nil {
		_ = c.Error(err)
		return
	}
	c.Status(http.StatusNoContent)
}

// ── Sessions ─────────────────────────────────────────────────────────────────

// POST /therapists/sessions/presign
func (h *TherapistNotesHandler) PresignNote(c *gin.Context) {
	t := h.requireConsented(c)
	if t == nil {
		return
	}
	var body struct {
		Filename    string `json:"filename"`
		ContentType string `json:"content_type" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		_ = c.Error(apierr.BadRequest("content_type is required"))
		return
	}
	uploadURL, key, err := h.svc.PresignNoteUpload(c.Request.Context(), t.ID, body.Filename, body.ContentType)
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"upload_url": uploadURL, "image_key": key})
}

// POST /therapists/sessions
func (h *TherapistNotesHandler) CreateSession(c *gin.Context) {
	t := h.requireConsented(c)
	if t == nil {
		return
	}
	var input services.CreateSessionInput
	if err := c.ShouldBindJSON(&input); err != nil {
		_ = c.Error(apierr.BadRequest("invalid request body"))
		return
	}
	session, err := h.svc.CreateSession(c.Request.Context(), t.ID, input)
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusCreated, session)
}

// GET /therapists/sessions?external_client_id=&linked_client_id=&limit=
func (h *TherapistNotesHandler) ListSessions(c *gin.Context) {
	t := h.requireTherapist(c)
	if t == nil {
		return
	}
	var externalID, linkedID *uuid.UUID
	if v := c.Query("external_client_id"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			_ = c.Error(apierr.BadRequest("invalid external_client_id"))
			return
		}
		externalID = &id
	}
	if v := c.Query("linked_client_id"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			_ = c.Error(apierr.BadRequest("invalid linked_client_id"))
			return
		}
		linkedID = &id
	}
	limit := 50
	sessions, err := h.svc.ListSessions(c.Request.Context(), t.ID, externalID, linkedID, limit)
	if err != nil {
		_ = c.Error(apierr.Internal("failed to load sessions"))
		return
	}
	if sessions == nil {
		sessions = []*models.ClientSession{}
	}
	c.JSON(http.StatusOK, gin.H{"sessions": sessions})
}

// GET /therapists/sessions/:id
func (h *TherapistNotesHandler) GetSession(c *gin.Context) {
	t := h.requireTherapist(c)
	if t == nil {
		return
	}
	sessionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		_ = c.Error(apierr.BadRequest("invalid session id"))
		return
	}
	session, err := h.svc.GetSession(c.Request.Context(), t.ID, sessionID)
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, session)
}

// PATCH /therapists/sessions/:id  { "bullets": [...] }
func (h *TherapistNotesHandler) UpdateSessionBullets(c *gin.Context) {
	t := h.requireTherapist(c)
	if t == nil {
		return
	}
	sessionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		_ = c.Error(apierr.BadRequest("invalid session id"))
		return
	}
	var body struct {
		Bullets []string `json:"bullets" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		_ = c.Error(apierr.BadRequest("bullets array is required"))
		return
	}
	session, err := h.svc.UpdateBullets(c.Request.Context(), t.ID, sessionID, body.Bullets)
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, session)
}

// POST /therapists/sessions/:id/summarize
func (h *TherapistNotesHandler) SummarizeSession(c *gin.Context) {
	t := h.requireTherapist(c)
	if t == nil {
		return
	}
	sessionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		_ = c.Error(apierr.BadRequest("invalid session id"))
		return
	}
	session, err := h.svc.Summarize(c.Request.Context(), t.ID, sessionID)
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, session)
}

// DELETE /therapists/sessions/:id
func (h *TherapistNotesHandler) DeleteSession(c *gin.Context) {
	t := h.requireTherapist(c)
	if t == nil {
		return
	}
	sessionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		_ = c.Error(apierr.BadRequest("invalid session id"))
		return
	}
	if err := h.svc.DeleteSession(c.Request.Context(), t.ID, sessionID); err != nil {
		_ = c.Error(err)
		return
	}
	c.Status(http.StatusNoContent)
}
