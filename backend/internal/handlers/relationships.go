package handlers

import (
	"errors"
	"net/http"
	"strings"

	"github.com/dreamlog/backend/internal/middleware"
	"github.com/dreamlog/backend/internal/models"
	"github.com/dreamlog/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
)

// validPersonRoles is the allowed set for re-categorizing a person.
var validPersonRoles = map[string]bool{
	"family": true, "friend": true, "colleague": true, "romantic": true, "other": true,
}

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

// UpdatePerson renames, re-categorizes, or hides/unhides a person.
// PATCH /relationships/:id
func (h *RelationshipHandler) UpdatePerson(c *gin.Context) {
	userID := middleware.UserIDFromCtx(c.Request.Context())

	personID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.Error(apierr.BadRequest("invalid person id"))
		return
	}

	var input models.UpdatePersonInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.Error(apierr.BadRequest("invalid request body"))
		return
	}
	if input.Name == nil && input.Role == nil && input.Hidden == nil {
		c.Error(apierr.BadRequest("at least one of name, role, or hidden is required"))
		return
	}
	if input.Name != nil {
		name := strings.TrimSpace(*input.Name)
		if name == "" {
			c.Error(apierr.BadRequest("name cannot be empty"))
			return
		}
		input.Name = &name
	}
	if input.Role != nil && !validPersonRoles[*input.Role] {
		c.Error(apierr.BadRequest("invalid role"))
		return
	}

	person, err := h.repo.UpdatePerson(c.Request.Context(), personID, userID, input)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			c.Error(apierr.Conflict("a person with that name already exists — merge them instead"))
			return
		}
		c.Error(apierr.Internal("update person"))
		return
	}
	if person == nil {
		c.Error(apierr.NotFound("person"))
		return
	}
	c.JSON(http.StatusOK, person)
}

// MergePerson folds another person (source_id) into this one (:id).
// POST /relationships/:id/merge
func (h *RelationshipHandler) MergePerson(c *gin.Context) {
	userID := middleware.UserIDFromCtx(c.Request.Context())

	targetID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.Error(apierr.BadRequest("invalid person id"))
		return
	}

	var body struct {
		SourceID string `json:"source_id"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.Error(apierr.BadRequest("invalid request body"))
		return
	}
	sourceID, err := uuid.Parse(body.SourceID)
	if err != nil {
		c.Error(apierr.BadRequest("invalid source_id"))
		return
	}
	if sourceID == targetID {
		c.Error(apierr.BadRequest("cannot merge a person into themselves"))
		return
	}

	person, err := h.repo.MergePeople(c.Request.Context(), targetID, sourceID, userID)
	if err != nil {
		c.Error(apierr.Internal("merge people"))
		return
	}
	if person == nil {
		c.Error(apierr.NotFound("person"))
		return
	}
	c.JSON(http.StatusOK, person)
}
