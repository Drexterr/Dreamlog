package handlers

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/dreamlog/backend/internal/services"
	"github.com/dreamlog/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// dlqAdmin is what AdminHandler needs from DLQAdminService.
type dlqAdmin interface {
	List(ctx context.Context, queueName string, page, pageSize int) ([]services.DLQItem, int64, error)
	Requeue(ctx context.Context, queueName, id string) error
	Discard(ctx context.Context, queueName, id string) error
}

// AdminHandler serves operator endpoints. Every route is behind
// middleware.RequireAdmin (ADMIN_EMAILS + verified email).
type AdminHandler struct {
	dlq dlqAdmin
}

func NewAdminHandler(dlq dlqAdmin) *AdminHandler {
	return &AdminHandler{dlq: dlq}
}

const (
	dlqDefaultPageSize = 20
	dlqMaxPageSize     = 100
)

// GET /admin/dlq/:queue?page=1&page_size=20 - dead letters, newest first.
func (h *AdminHandler) ListDLQ(c *gin.Context) {
	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		_ = c.Error(apierr.BadRequest("page must be a positive integer"))
		return
	}
	size, err := strconv.Atoi(c.DefaultQuery("page_size", strconv.Itoa(dlqDefaultPageSize)))
	if err != nil || size < 1 || size > dlqMaxPageSize {
		_ = c.Error(apierr.BadRequest("page_size must be between 1 and 100"))
		return
	}
	queueName := c.Param("queue")
	items, total, err := h.dlq.List(c.Request.Context(), queueName, page, size)
	if err != nil {
		_ = c.Error(dlqError(err))
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"queue":     queueName,
		"items":     items,
		"total":     total,
		"page":      page,
		"page_size": size,
		"has_more":  int64(page*size) < total,
	})
}

// POST /admin/dlq/:queue/:id/requeue - reset the entry/session and retry it.
func (h *AdminHandler) RequeueDLQ(c *gin.Context) {
	queueName, id := c.Param("queue"), dlqItemID(c.Param("id"))
	if err := h.dlq.Requeue(c.Request.Context(), queueName, id); err != nil {
		_ = c.Error(dlqError(err))
		return
	}
	c.JSON(http.StatusOK, gin.H{"queue": queueName, "id": id, "status": "requeued"})
}

// DELETE /admin/dlq/:queue/:id - drop a dead letter without retrying it.
func (h *AdminHandler) DiscardDLQ(c *gin.Context) {
	queueName, id := c.Param("queue"), dlqItemID(c.Param("id"))
	if err := h.dlq.Discard(c.Request.Context(), queueName, id); err != nil {
		_ = c.Error(dlqError(err))
		return
	}
	c.Status(http.StatusNoContent)
}

// dlqItemID canonicalizes UUID ids (any case) to the form items are listed
// with; "raw-..." ids pass through.
func dlqItemID(raw string) string {
	if id, err := uuid.Parse(raw); err == nil {
		return id.String()
	}
	return raw
}

func dlqError(err error) error {
	switch {
	case errors.Is(err, services.ErrDLQUnknownQueue):
		return apierr.BadRequest("queue must be entries or notes")
	case errors.Is(err, services.ErrDLQItemNotFound):
		return apierr.NotFound("dead-letter item")
	case errors.Is(err, services.ErrDLQUnparseable),
		errors.Is(err, services.ErrDLQTargetGone),
		errors.Is(err, services.ErrDLQCompleted),
		errors.Is(err, services.ErrDLQNotFailed),
		errors.Is(err, services.ErrDLQSourceGone),
		errors.Is(err, services.ErrDLQClaimed):
		return apierr.Conflict(err.Error())
	default:
		return apierr.InternalErr("dead-letter operation failed", err)
	}
}
