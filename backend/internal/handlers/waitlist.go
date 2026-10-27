package handlers

import (
	"context"
	"github.com/dreamlog/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
	"net/http"
	"time"
)

type waitlistJoiner interface {
	Join(context.Context, string, string) error
}
type WaitlistHandler struct{ svc waitlistJoiner }

func NewWaitlistHandler(svc waitlistJoiner) *WaitlistHandler { return &WaitlistHandler{svc: svc} }

func (h *WaitlistHandler) Join(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 2048)
	var req struct {
		Email   string `json:"email"`
		Website string `json:"website"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(apierr.BadRequest("Enter a valid email address."))
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	if err := h.svc.Join(ctx, req.Email, req.Website); err != nil {
		_ = c.Error(err)
		return
	}
	// Identical on duplicate submissions: do not reveal who has signed up.
	c.JSON(http.StatusOK, gin.H{"message": "You're on the list. We'll email you when Ode launches."})
}
