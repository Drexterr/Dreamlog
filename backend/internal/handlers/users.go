package handlers

import (
	"net/http"

	"github.com/dreamlog/backend/internal/middleware"
	"github.com/dreamlog/backend/internal/models"
	"github.com/dreamlog/backend/internal/services"
	"github.com/dreamlog/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	svc userProfiler
}

func NewUserHandler(svc *services.UserService) *UserHandler {
	return &UserHandler{svc: svc}
}

// GET /me
func (h *UserHandler) GetMe(c *gin.Context) {
	user := middleware.UserFromCtx(c.Request.Context())
	if user == nil {
		c.Error(apierr.Unauthorized("user not found in context"))
		return
	}
	c.JSON(http.StatusOK, user)
}

// PUT /me
func (h *UserHandler) UpdateMe(c *gin.Context) {
	var input models.UpdateUserInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.Error(apierr.BadRequest("invalid request body", err.Error()))
		return
	}

	if input.Name == nil && input.PreferredName == nil && input.Timezone == nil &&
		input.FCMNudgeHour == nil && input.NudgeEnabled == nil && input.Goal == nil &&
		input.AgeRange == nil && input.Country == nil && input.VoiceLanguage == nil {
		c.Error(apierr.BadRequest("at least one field must be provided"))
		return
	}

	// The supported set lives in models so it stays in sync with the Azure
	// voice map in services/tts.go - too large for a binding oneof tag.
	if input.VoiceLanguage != nil && !models.IsValidVoiceLanguage(*input.VoiceLanguage) {
		c.Error(apierr.BadRequest("unsupported voice_language"))
		return
	}

	userID := middleware.UserIDFromCtx(c.Request.Context())
	user, err := h.svc.UpdateProfile(c.Request.Context(), userID, input)
	if err != nil {
		c.Error(err)
		return
	}
	if user == nil {
		c.Error(apierr.NotFound("user"))
		return
	}
	c.JSON(http.StatusOK, user)
}

// DELETE /me - permanently deletes the authenticated user and all their data.
func (h *UserHandler) DeleteMe(c *gin.Context) {
	userID := middleware.UserIDFromCtx(c.Request.Context())
	if err := h.svc.Delete(c.Request.Context(), userID); err != nil {
		c.Error(apierr.Internal("failed to delete account"))
		return
	}
	c.Status(http.StatusNoContent)
}
