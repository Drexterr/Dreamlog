package handlers

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math/big"
	"net/http"
	"time"

	"github.com/dreamlog/backend/internal/middleware"
	"github.com/dreamlog/backend/internal/models"
	"github.com/dreamlog/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// shareLinkRepo is the minimal interface ShareHandler needs.
type shareLinkRepo interface {
	Create(ctx context.Context, in models.CreateShareLinkInput) (*models.ShareLink, error)
	GetByToken(ctx context.Context, token string) (*models.ShareLink, error)
	ListByUser(ctx context.Context, userID uuid.UUID) ([]*models.ShareLink, error)
	Revoke(ctx context.Context, id, userID uuid.UUID) error
	ShareView(ctx context.Context, userID uuid.UUID) (*models.ShareLinkView, error)
}

type ShareHandler struct {
	repo         shareLinkRepo
	subscription shareQuotaChecker
	appURL       string // e.g. "https://dreamlog.app" - used to build the share URL
}

func NewShareHandler(repo shareLinkRepo, subscription shareQuotaChecker, appURL string) *ShareHandler {
	return &ShareHandler{repo: repo, subscription: subscription, appURL: appURL}
}

// POST /share - create a 72-hour passcode-protected share link.
func (h *ShareHandler) Create(c *gin.Context) {
	user := middleware.UserFromCtx(c.Request.Context())
	if user == nil {
		_ = c.Error(apierr.Unauthorized("user not found"))
		return
	}
	userID := user.ID

	// Check share quota for the user's plan.
	if err := h.subscription.CheckShareQuota(c.Request.Context(), userID, user.Plan); err != nil {
		_ = c.Error(apierr.Forbidden(err.Error()))
		return
	}

	// Generate a 32-byte random URL-safe token.
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		_ = c.Error(apierr.Internal("failed to generate token"))
		return
	}
	token := hex.EncodeToString(tokenBytes)

	// Generate a 4-digit numeric passcode.
	passcode, err := randomPasscode()
	if err != nil {
		_ = c.Error(apierr.Internal("failed to generate passcode"))
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(passcode), bcrypt.DefaultCost)
	if err != nil {
		_ = c.Error(apierr.Internal("failed to hash passcode"))
		return
	}

	expiresAt := time.Now().Add(72 * time.Hour)
	sl, err := h.repo.Create(c.Request.Context(), models.CreateShareLinkInput{
		UserID:       userID,
		Token:        token,
		PasscodeHash: string(hash),
		ExpiresAt:    expiresAt,
	})
	if err != nil {
		_ = c.Error(apierr.Internal("failed to create share link"))
		return
	}

	shareURL := fmt.Sprintf("%s/share/%s", h.appURL, sl.Token)
	c.JSON(http.StatusCreated, models.CreateShareLinkResult{
		Token:     sl.Token,
		Passcode:  passcode,
		URL:       shareURL,
		ExpiresAt: sl.ExpiresAt,
	})
}

// GET /share - list all active share links for the authenticated user.
func (h *ShareHandler) List(c *gin.Context) {
	userID := middleware.UserIDFromCtx(c.Request.Context())
	links, err := h.repo.ListByUser(c.Request.Context(), userID)
	if err != nil {
		_ = c.Error(apierr.Internal("failed to list share links"))
		return
	}
	type item struct {
		ID        uuid.UUID `json:"id"`
		Token     string    `json:"token"`
		URL       string    `json:"url"`
		ExpiresAt time.Time `json:"expires_at"`
	}
	out := make([]item, 0, len(links))
	for _, l := range links {
		out = append(out, item{
			ID:        l.ID,
			Token:     l.Token,
			URL:       fmt.Sprintf("%s/share/%s", h.appURL, l.Token),
			ExpiresAt: l.ExpiresAt,
		})
	}
	c.JSON(http.StatusOK, gin.H{"links": out})
}

// DELETE /share/:id - revoke a share link.
func (h *ShareHandler) Revoke(c *gin.Context) {
	userID := middleware.UserIDFromCtx(c.Request.Context())
	linkID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		_ = c.Error(apierr.BadRequest("invalid link id"))
		return
	}
	if err := h.repo.Revoke(c.Request.Context(), linkID, userID); err != nil {
		_ = c.Error(apierr.NotFound("share link not found"))
		return
	}
	c.Status(http.StatusNoContent)
}

// GET /share/:token - public endpoint; validates passcode and returns shared data.
// Passcode is passed as query param ?p=1234.
func (h *ShareHandler) View(c *gin.Context) {
	token := c.Param("token")
	passcode := c.Query("p")
	if passcode == "" {
		_ = c.Error(apierr.Unauthorized("passcode required"))
		return
	}

	sl, err := h.repo.GetByToken(c.Request.Context(), token)
	if err != nil {
		_ = c.Error(apierr.Internal("lookup failed"))
		return
	}
	if sl == nil {
		_ = c.Error(apierr.NotFound("link not found or expired"))
		return
	}
	if time.Now().After(sl.ExpiresAt) {
		_ = c.Error(apierr.NotFound("link not found or expired"))
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(sl.PasscodeHash), []byte(passcode)); err != nil {
		_ = c.Error(apierr.Unauthorized("incorrect passcode"))
		return
	}

	view, err := h.repo.ShareView(c.Request.Context(), sl.UserID)
	if err != nil {
		_ = c.Error(apierr.Internal("failed to build share view"))
		return
	}
	view.ExpiresAt = sl.ExpiresAt
	c.JSON(http.StatusOK, view)
}

// randomPasscode returns a cryptographically random 4-digit string.
func randomPasscode() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(10000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%04d", n.Int64()), nil
}
