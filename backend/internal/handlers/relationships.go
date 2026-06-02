package handlers

import (
	"net/http"

	"github.com/dreamlog/backend/internal/middleware"
	"github.com/dreamlog/backend/internal/models"
	"github.com/dreamlog/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type RelationshipHandler struct {
	repo relationshipMapRepo
}

func NewRelationshipHandler(repo relationshipMapRepo) *RelationshipHandler {
	return &RelationshipHandler{repo: repo}
}

// GetMap returns the full relationship map for the authenticated user.
// GET /relationships
func (h *RelationshipHandler) GetMap(c *gin.Context) {
	userID := middleware.UserIDFromCtx(c.Request.Context())

	people, err := h.repo.GetMap(c.Request.Context(), userID)
	if err != nil {
		c.Error(apierr.Internal("get relationship map"))
		return
	}
	if people == nil {
		people = []*models.Person{}
	}
	c.JSON(http.StatusOK, gin.H{"people": people})
}

// GetPersonDetail returns a single person with their recent mentions.
// GET /relationships/:id
func (h *RelationshipHandler) GetPersonDetail(c *gin.Context) {
	userID := middleware.UserIDFromCtx(c.Request.Context())

	personID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.Error(apierr.BadRequest("invalid person id"))
		return
	}

	detail, err := h.repo.GetDetail(c.Request.Context(), personID, userID)
	if err != nil {
		c.Error(apierr.Internal("get person detail"))
		return
	}
	if detail == nil {
		c.Error(apierr.NotFound("person"))
		return
	}
	c.JSON(http.StatusOK, detail)
}
