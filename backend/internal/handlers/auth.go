package handlers

import (
	"errors"
	"net/http"

	"github.com/dreamlog/backend/internal/services"
	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	authSvc *services.AuthService
}

func NewAuthHandler(authSvc *services.AuthService) *AuthHandler {
	return &AuthHandler{authSvc: authSvc}
}

type registerRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Name     string `json:"name" binding:"required,min=1,max=200"`
	Password string `json:"password" binding:"required,min=8"`
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}

	user, token, err := h.authSvc.Register(c.Request.Context(), req.Email, req.Name, req.Password)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, services.ErrEmailTaken) {
			status = http.StatusConflict
		}
		c.JSON(status, gin.H{"message": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"user": user, "token": token})
}

type loginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}

	user, token, err := h.authSvc.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"user": user, "token": token})
}
