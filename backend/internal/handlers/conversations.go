package handlers

import (
	"errors"
	"net/http"

	"github.com/dreamlog/backend/internal/middleware"
	"github.com/dreamlog/backend/internal/models"
	"github.com/dreamlog/backend/internal/services"
	"github.com/dreamlog/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ConversationHandler struct {
	svc *services.ConversationService
}

func NewConversationHandler(svc *services.ConversationService) *ConversationHandler {
	return &ConversationHandler{svc: svc}
}

// POST /entries/:id/conversation
// Opens or resumes the follow-up conversation for an entry.
func (h *ConversationHandler) GetOrCreate(c *gin.Context) {
	userID := middleware.UserIDFromCtx(c.Request.Context())
	entryID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.Error(apierr.BadRequest("invalid entry id"))
		return
	}

	conv, err := h.svc.GetOrCreate(c.Request.Context(), entryID, userID)
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, conv)
}

// POST /conversations/:id/messages
// Sends one user message; returns the updated conversation with assistant reply.
func (h *ConversationHandler) SendMessage(c *gin.Context) {
	userID := middleware.UserIDFromCtx(c.Request.Context())
	convID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.Error(apierr.BadRequest("invalid conversation id"))
		return
	}

	var input models.SendMessageInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.Error(apierr.BadRequest("invalid request body", err.Error()))
		return
	}

	country := ""
	if u := middleware.UserFromCtx(c.Request.Context()); u != nil && u.Country != nil {
		country = *u.Country
	}

	conv, err := h.svc.SendMessage(c.Request.Context(), convID, userID, input.Content, country)
	if err != nil {
		if errors.Is(err, services.ErrConversationClosed) {
			c.Error(apierr.Conflict("conversation is closed"))
			return
		}
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, conv)
}
