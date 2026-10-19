package handlers

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math/big"
	"net/http"
	"strings"
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
	RecordFailedPasscode(ctx context.Context, id uuid.UUID, maxAttempts int, lockDuration time.Duration) (*time.Time, error)
	ResetPasscodeAttempts(ctx context.Context, id uuid.UUID) error
}

// Passcode lockout policy: after this many consecutive wrong passcodes a link
// is temporarily locked, defeating brute-force of the 4-digit space even across
// API replicas (the per-IP rate limiter only bounds a single instance).
const (
	sharePasscodeMaxAttempts = 5
	sharePasscodeLockWindow  = 15 * time.Minute
)

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
	if err := h.subscription.CheckShareQuota(c.Request.Context(), userID, user.EffectivePlan()); err != nil {
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

	// Generate an 8-character alphanumeric passcode.
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
// The passcode is read from the X-Share-Passcode header (preferred: headers
// don't land in proxy, CDN or browser-history logs) or, for existing
// viewers, the ?p= query param. Matching is case-insensitive.
func (h *ShareHandler) View(c *gin.Context) {
	token := c.Param("token")
	passcode := c.GetHeader("X-Share-Passcode")
	if passcode == "" {
		passcode = c.Query("p")
	}
	passcode = strings.ToUpper(strings.TrimSpace(passcode))
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

	// Reject while the link is under a passcode lockout.
	if sl.LockedUntil != nil && time.Now().Before(*sl.LockedUntil) {
		c.Header("Retry-After", "900")
		_ = c.Error(apierr.New(http.StatusTooManyRequests, "too many incorrect passcode attempts; try again later"))
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(sl.PasscodeHash), []byte(passcode)); err != nil {
		lockedUntil, recErr := h.repo.RecordFailedPasscode(c.Request.Context(), sl.ID, sharePasscodeMaxAttempts, sharePasscodeLockWindow)
		if recErr == nil && lockedUntil != nil && time.Now().Before(*lockedUntil) {
			c.Header("Retry-After", "900")
			_ = c.Error(apierr.New(http.StatusTooManyRequests, "too many incorrect passcode attempts; try again later"))
			return
		}
		_ = c.Error(apierr.Unauthorized("incorrect passcode"))
		return
	}

	// Correct passcode - clear any accumulated failed attempts.
	_ = h.repo.ResetPasscodeAttempts(c.Request.Context(), sl.ID)

	view, err := h.repo.ShareView(c.Request.Context(), sl.UserID)
	if err != nil {
		_ = c.Error(apierr.Internal("failed to build share view"))
		return
	}
	view.ExpiresAt = sl.ExpiresAt
	c.JSON(http.StatusOK, view)
}

// passcodeAlphabet omits look-alikes (0/O, 1/I/L) so the code can be read
// out or retyped. 31 symbols ^ 8 ≈ 8.5e11 combinations - with the per-link
// lifetime cap of 20 wrong guesses, brute force is not a realistic path.
const passcodeAlphabet = "23456789ABCDEFGHJKMNPQRSTUVWXYZ"

const passcodeLength = 8

// randomPasscode returns a cryptographically random 8-character code.
func randomPasscode() (string, error) {
	out := make([]byte, passcodeLength)
	for i := range out {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(passcodeAlphabet))))
		if err != nil {
			return "", err
		}
		out[i] = passcodeAlphabet[n.Int64()]
	}
	return string(out), nil
}
