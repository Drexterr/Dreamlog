package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/dreamlog/backend/internal/middleware"
	"github.com/dreamlog/backend/internal/models"
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

	// Check monthly entry quota (free plan: 10/month).
	if err := h.subscription.CheckEntryQuota(c.Request.Context(), user.ID, user.Plan); err != nil {
		c.Error(apierr.Conflict(err.Error()))
		return
	}

	userCountry := ""
	if user.Country != nil {
		userCountry = *user.Country
	}
	entry, err := h.svc.Create(c.Request.Context(), user.ID, &input, userCountry)
	if err != nil {
		c.Error(apierr.BadRequest(err.Error()))
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
	if key == "" || !strings.HasPrefix(key, "audio/") {
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
