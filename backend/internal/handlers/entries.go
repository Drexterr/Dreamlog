package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/dreamlog/backend/internal/middleware"
	"github.com/dreamlog/backend/internal/models"
	"github.com/dreamlog/backend/internal/services"
	"github.com/dreamlog/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type EntryHandler struct {
	svc          entryServicer
	storage      storageUploader
	subscription entryQuotaChecker
}

func NewEntryHandler(svc entryServicer, storage storageUploader, subscription entryQuotaChecker) *EntryHandler {
	return &EntryHandler{svc: svc, storage: storage, subscription: subscription}
}

// POST /entries/presign
// Returns a pre-signed PUT URL so the client can upload directly to storage.
func (h *EntryHandler) Presign(c *gin.Context) {
	userID := middleware.UserIDFromCtx(c.Request.Context())

	// Refuse the upload up front when the monthly quota is used up: checking
	// only at registration lets the audio upload and then orphans it (N7).
	if user := middleware.UserFromCtx(c.Request.Context()); user != nil {
		if err := h.subscription.CheckEntryQuota(c.Request.Context(), user.ID, user.EffectivePlan()); err != nil {
			c.Error(quotaError(err))
			return
		}
	}

	resp, err := h.svc.PresignUpload(c.Request.Context(), userID)
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

// POST /entries
// Called after the client finishes uploading audio to storage.
// Validates the file exists, creates DB row, queues transcription job.
func (h *EntryHandler) Create(c *gin.Context) {
	var input models.CreateEntryInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.Error(apierr.BadRequest("invalid request body", err.Error()))
		return
	}

	user := middleware.UserFromCtx(c.Request.Context())
	if user == nil {
		c.Error(apierr.Unauthorized("user not found"))
		return
	}

	// Resolve retries before the quota check (N10): a free user's 10th
	// registration whose response was lost must get its entry back, not 409.
	existing, err := h.svc.FindByAudioKey(c.Request.Context(), user.ID, input.AudioKey)
	if err != nil {
		c.Error(serviceError(err))
		return
	}
	if existing != nil {
		c.JSON(http.StatusCreated, existing)
		return
	}

	// Check monthly entry quota (free plan: 10/month).
	if err := h.subscription.CheckEntryQuota(c.Request.Context(), user.ID, user.EffectivePlan()); err != nil {
		c.Error(quotaError(err))
		return
	}

	userCountry := ""
	if user.Country != nil {
		userCountry = *user.Country
	}
	entry, err := h.svc.Create(c.Request.Context(), user.ID, &input, userCountry)
	if err != nil {
		c.Error(serviceError(err))
		return
	}
	c.JSON(http.StatusCreated, entry)
}

// serviceError passes typed API errors (validation 400s, conflicts) through
// unchanged and turns anything else - a DB or storage outage - into a 500,
// which clients retry, instead of a permanent-looking 400 carrying internal
// text (N10).
func serviceError(err error) error {
	if _, ok := apierr.As(err); ok {
		return err
	}
	return apierr.InternalErr("could not register the recording", err)
}

// quotaError maps the monthly-limit error to 409; a failure to count is a 500.
func quotaError(err error) error {
	if services.IsEntryQuotaExceeded(err) {
		return apierr.Conflict(err.Error())
	}
	return apierr.InternalErr("could not check the monthly entry limit", err)
}

// POST /entries/text
// CreateText accepts one complete written journal with a stable retry token.
func (h *EntryHandler) CreateText(c *gin.Context) {
	var input models.CreateTextEntryInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.Error(apierr.BadRequest("invalid typed journal request"))
		return
	}
	if err := input.Validate(); err != nil {
		c.Error(err)
		return
	}
	user := middleware.UserFromCtx(c.Request.Context())
	if user == nil {
		c.Error(apierr.Unauthorized("user not found"))
		return
	}
	// Resolve retries before quota checking: a lost response at the quota boundary
	// must not prevent retrieval of the journal that was already saved.
	existing, err := h.svc.Get(c.Request.Context(), models.TextEntryID(user.ID, input.RequestID), user.ID)
	if err != nil {
		c.Error(apierr.InternalErr("could not check journal submission", err))
		return
	}
	if existing != nil {
		if !input.Matches(existing) {
			c.Error(apierr.Conflict("request_id was already used for a different journal"))
			return
		}
		c.JSON(http.StatusCreated, existing)
		return
	}
	if err := h.subscription.CheckEntryQuota(c.Request.Context(), user.ID, user.EffectivePlan()); err != nil {
		c.Error(quotaError(err))
		return
	}
	country := ""
	if user.Country != nil {
		country = *user.Country
	}
	entry, err := h.svc.CreateText(c.Request.Context(), user.ID, &input, country)
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusCreated, entry)
}

// GET /entries
// Returns paginated entries for the authenticated user.
func (h *EntryHandler) List(c *gin.Context) {
	userID := middleware.UserIDFromCtx(c.Request.Context())

	page := parseIntQuery(c, "page", 1)
	pageSize := parseIntQuery(c, "page_size", 20)

	resp, err := h.svc.List(c.Request.Context(), userID, page, pageSize)
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

// GET /entries/:id
func (h *EntryHandler) Get(c *gin.Context) {
	userID := middleware.UserIDFromCtx(c.Request.Context())

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.Error(apierr.BadRequest("invalid entry id"))
		return
	}

	entry, err := h.svc.Get(c.Request.Context(), id, userID)
	if err != nil {
		c.Error(err)
		return
	}
	if entry == nil {
		c.Error(apierr.NotFound("entry"))
		return
	}
	c.JSON(http.StatusOK, entry)
}

// PUT /upload?key=audio/...
// Dev-only upload proxy: receives audio from the mobile client and streams it
// directly to MinIO. Used when MinIO isn't reachable from the device (e.g. Windows dev).
func (h *EntryHandler) UploadProxy(c *gin.Context) {
	key := c.Query("key")
	// Dev-only proxy (refused in production): accepts the same prefixes our
	// presigns generate, so voice therapy works in this setup too.
	if key == "" || (!strings.HasPrefix(key, "audio/") && !strings.HasPrefix(key, "therapy/")) || strings.Contains(key, "..") {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid key"})
		return
	}

	if err := h.storage.Upload(c.Request.Context(), key, c.Request.Body); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "upload failed: " + err.Error()})
		return
	}
	c.Status(http.StatusOK)
}

func parseIntQuery(c *gin.Context, key string, fallback int) int {
	if v := c.Query(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return fallback
}
