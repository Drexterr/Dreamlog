package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/dreamlog/backend/internal/middleware"
	"github.com/dreamlog/backend/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

// ── fake shareLinkRepo ────────────────────────────────────────────────────────

type fakeShareRepo struct {
	createResp     *models.ShareLink
	createErr      error
	getByTokenResp *models.ShareLink
	getByTokenErr  error
	listResp       []*models.ShareLink
	listErr        error
	revokeErr      error
	viewResp       *models.ShareLinkView
	viewErr        error
}

func (f *fakeShareRepo) Create(_ context.Context, in models.CreateShareLinkInput) (*models.ShareLink, error) {
	if f.createErr != nil {
		return nil, f.createErr
	}
	if f.createResp != nil {
		return f.createResp, nil
	}
	return &models.ShareLink{
		ID:           uuid.New(),
		UserID:       in.UserID,
		Token:        in.Token,
		PasscodeHash: in.PasscodeHash,
		ExpiresAt:    in.ExpiresAt,
	}, nil
}

func (f *fakeShareRepo) GetByToken(_ context.Context, _ string) (*models.ShareLink, error) {
	return f.getByTokenResp, f.getByTokenErr
}

func (f *fakeShareRepo) ListByUser(_ context.Context, _ uuid.UUID) ([]*models.ShareLink, error) {
	return f.listResp, f.listErr
}

func (f *fakeShareRepo) Revoke(_ context.Context, _, _ uuid.UUID) error {
	return f.revokeErr
}

func (f *fakeShareRepo) ShareView(_ context.Context, _ uuid.UUID) (*models.ShareLinkView, error) {
	if f.viewErr != nil {
		return nil, f.viewErr
	}
	if f.viewResp != nil {
		return f.viewResp, nil
	}
	return &models.ShareLinkView{
		UserName:    "Test User",
		Period:      "30d",
		MoodArc:     []models.MoodArcDay{{Date: "2026-05-21", AvgMood: 65}},
		TopEmotions: []string{"hopeful"},
		Summaries:   []models.EntrySummary{},
	}, nil
}

// ── fake share quota checker ──────────────────────────────────────────────────

type fakeShareQuota struct{ err error }

func (f *fakeShareQuota) CheckShareQuota(_ context.Context, _ uuid.UUID, _ models.Plan) error {
	return f.err
}

// ── router builder ────────────────────────────────────────────────────────────

const shareTestSecret = "share-test-jwt-secret-32-bytes!!"

func newShareTestRouter(t *testing.T, repo *fakeShareRepo, testUser *models.User) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	log := zap.NewNop()

	r := gin.New()
	r.Use(middleware.ErrorHandler(log))

	// Default: quota is OK (Pro plan user in tests).
	h := NewShareHandler(repo, &fakeShareQuota{}, "https://dreamlog.app")

	// Auth-required group.
	auth := r.Group("/")
	auth.Use(middleware.AuthMiddleware(shareTestSecret, &fakeProvisioner{user: testUser}, log))
	auth.POST("/share", h.Create)
	auth.GET("/share", h.List)
	auth.DELETE("/share/:id", h.Revoke)

	// Public endpoint — no auth.
	r.GET("/view/:token", h.View)
	return r
}

func shareTestJWT(t *testing.T) string {
	t.Helper()
	claims := jwt.MapClaims{
		"sub":   "test-sub-share-001",
		"email": "share@test.com",
		"exp":   time.Now().Add(time.Hour).Unix(),
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	str, err := tok.SignedString([]byte(shareTestSecret))
	if err != nil {
		t.Fatal(err)
	}
	return str
}

func shareTestUser() *models.User {
	return &models.User{ID: uuid.New(), Email: "share@test.com", Name: "Share User", Plan: models.PlanPro}
}

// bcryptHash returns a bcrypt hash at minimum cost — fast for unit tests.
func bcryptHash(t *testing.T, plain string) string {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	return string(hash)
}

// ── Create tests ──────────────────────────────────────────────────────────────

func TestShareHandler_Create_Returns201WithTokenAndPasscode(t *testing.T) {
	r := newShareTestRouter(t, &fakeShareRepo{}, shareTestUser())
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/share", bytes.NewReader([]byte("{}")))
	req.Header.Set("Authorization", "Bearer "+shareTestJWT(t))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("create share link: want 201, got %d: %s", w.Code, w.Body.String())
	}
	var resp models.CreateShareLinkResult
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Token == "" {
		t.Error("response must include a token")
	}
	if resp.Passcode == "" {
		t.Error("response must include a passcode")
	}
	if len(resp.Passcode) != 4 {
		t.Errorf("passcode must be 4 digits, got %q", resp.Passcode)
	}
	if resp.URL == "" {
		t.Error("response must include a url")
	}
	if !strings.Contains(resp.URL, resp.Token) {
		t.Errorf("url %q must contain token %q", resp.URL, resp.Token)
	}
	// Expiry must be ~72 h from now.
	want := time.Now().Add(72 * time.Hour)
	if resp.ExpiresAt.Before(want.Add(-5*time.Minute)) || resp.ExpiresAt.After(want.Add(5*time.Minute)) {
		t.Errorf("expires_at should be ~72h from now, got %v", resp.ExpiresAt)
	}
}

func TestShareHandler_Create_MissingAuth_Returns401(t *testing.T) {
	r := newShareTestRouter(t, &fakeShareRepo{}, shareTestUser())
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/share", bytes.NewReader([]byte("{}")))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("missing auth: want 401, got %d", w.Code)
	}
}

// ── List tests ────────────────────────────────────────────────────────────────

func TestShareHandler_List_Returns200WithLinksArray(t *testing.T) {
	repo := &fakeShareRepo{
		listResp: []*models.ShareLink{
			{ID: uuid.New(), Token: "tok1", ExpiresAt: time.Now().Add(24 * time.Hour)},
			{ID: uuid.New(), Token: "tok2", ExpiresAt: time.Now().Add(48 * time.Hour)},
		},
	}
	r := newShareTestRouter(t, repo, shareTestUser())
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/share", nil)
	req.Header.Set("Authorization", "Bearer "+shareTestJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("list: want 200, got %d: %s", w.Code, w.Body.String())
	}
	var body map[string]any
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	links, ok := body["links"].([]any)
	if !ok {
		t.Fatal("response must have links array")
	}
	if len(links) != 2 {
		t.Errorf("want 2 links, got %d", len(links))
	}
}

func TestShareHandler_List_EmptyList_Returns200WithEmptyArray(t *testing.T) {
	r := newShareTestRouter(t, &fakeShareRepo{listResp: nil}, shareTestUser())
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/share", nil)
	req.Header.Set("Authorization", "Bearer "+shareTestJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("empty list: want 200, got %d", w.Code)
	}
	var body map[string]any
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	links, ok := body["links"].([]any)
	if !ok {
		t.Fatal("response must have links field")
	}
	if len(links) != 0 {
		t.Errorf("want empty links array, got %d items", len(links))
	}
}

// ── Revoke tests ──────────────────────────────────────────────────────────────

func TestShareHandler_Revoke_ValidID_Returns204(t *testing.T) {
	r := newShareTestRouter(t, &fakeShareRepo{revokeErr: nil}, shareTestUser())
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodDelete, "/share/"+uuid.New().String(), nil)
	req.Header.Set("Authorization", "Bearer "+shareTestJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("revoke: want 204, got %d: %s", w.Code, w.Body.String())
	}
}

func TestShareHandler_Revoke_InvalidID_Returns400(t *testing.T) {
	r := newShareTestRouter(t, &fakeShareRepo{}, shareTestUser())
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodDelete, "/share/not-a-uuid", nil)
	req.Header.Set("Authorization", "Bearer "+shareTestJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("invalid id: want 400, got %d", w.Code)
	}
}

func TestShareHandler_Revoke_NotFound_Returns404(t *testing.T) {
	r := newShareTestRouter(t, &fakeShareRepo{revokeErr: errors.New("not found")}, shareTestUser())
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodDelete, "/share/"+uuid.New().String(), nil)
	req.Header.Set("Authorization", "Bearer "+shareTestJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("not found: want 404, got %d", w.Code)
	}
}

func TestShareHandler_Revoke_MissingAuth_Returns401(t *testing.T) {
	r := newShareTestRouter(t, &fakeShareRepo{}, shareTestUser())
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodDelete, "/share/"+uuid.New().String(), nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("missing auth: want 401, got %d", w.Code)
	}
}

// ── View tests (public endpoint) ──────────────────────────────────────────────

func TestShareHandler_View_CorrectPasscode_Returns200WithView(t *testing.T) {
	hash := bcryptHash(t, "1234")
	repo := &fakeShareRepo{
		getByTokenResp: &models.ShareLink{
			ID:           uuid.New(),
			UserID:       uuid.New(),
			Token:        "validtoken",
			PasscodeHash: hash,
			ExpiresAt:    time.Now().Add(24 * time.Hour),
		},
	}
	r := newShareTestRouter(t, repo, shareTestUser())
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/view/validtoken?p=1234", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("correct passcode: want 200, got %d: %s", w.Code, w.Body.String())
	}
	var view models.ShareLinkView
	if err := json.NewDecoder(w.Body).Decode(&view); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if view.UserName == "" {
		t.Error("view must include user_name")
	}
}

func TestShareHandler_View_WrongPasscode_Returns401(t *testing.T) {
	hash := bcryptHash(t, "1234")
	repo := &fakeShareRepo{
		getByTokenResp: &models.ShareLink{
			ID:           uuid.New(),
			UserID:       uuid.New(),
			Token:        "validtoken",
			PasscodeHash: hash,
			ExpiresAt:    time.Now().Add(24 * time.Hour),
		},
	}
	r := newShareTestRouter(t, repo, shareTestUser())
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/view/validtoken?p=9999", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("wrong passcode: want 401, got %d", w.Code)
	}
}

func TestShareHandler_View_MissingPasscode_Returns401(t *testing.T) {
	r := newShareTestRouter(t, &fakeShareRepo{}, shareTestUser())
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/view/sometoken", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("missing passcode: want 401, got %d", w.Code)
	}
}

func TestShareHandler_View_TokenNotFound_Returns404(t *testing.T) {
	repo := &fakeShareRepo{getByTokenResp: nil}
	r := newShareTestRouter(t, repo, shareTestUser())
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/view/unknowntoken?p=1234", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("token not found: want 404, got %d", w.Code)
	}
}

func TestShareHandler_View_ExpiredLink_Returns404(t *testing.T) {
	hash := bcryptHash(t, "1234")
	repo := &fakeShareRepo{
		getByTokenResp: &models.ShareLink{
			ID:           uuid.New(),
			UserID:       uuid.New(),
			Token:        "expiredtoken",
			PasscodeHash: hash,
			ExpiresAt:    time.Now().Add(-1 * time.Hour), // expired 1 hour ago
		},
	}
	r := newShareTestRouter(t, repo, shareTestUser())
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/view/expiredtoken?p=1234", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expired link: want 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestShareHandler_View_GetByTokenError_Returns500(t *testing.T) {
	repo := &fakeShareRepo{getByTokenErr: errors.New("db error")}
	r := newShareTestRouter(t, repo, shareTestUser())
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/view/sometoken?p=1234", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("get token error: want 500, got %d", w.Code)
	}
}

// ── Plan-gating tests ─────────────────────────────────────────────────────────

func newShareTestRouterWithQuota(t *testing.T, repo *fakeShareRepo, quota shareQuotaChecker, testUser *models.User) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	log := zap.NewNop()

	r := gin.New()
	r.Use(middleware.ErrorHandler(log))

	h := NewShareHandler(repo, quota, "https://dreamlog.app")

	auth := r.Group("/")
	auth.Use(middleware.AuthMiddleware(shareTestSecret, &fakeProvisioner{user: testUser}, log))
	auth.POST("/share", h.Create)
	auth.GET("/share", h.List)
	auth.DELETE("/share/:id", h.Revoke)
	r.GET("/view/:token", h.View)
	return r
}

func TestShareHandler_Create_FreePlanReturns403(t *testing.T) {
	freeUser := &models.User{ID: uuid.New(), Email: "free@test.com", Plan: models.PlanFree}
	quota := &fakeShareQuota{err: errors.New("therapist share links require DreamLog+ or higher")}
	r := newShareTestRouterWithQuota(t, &fakeShareRepo{}, quota, freeUser)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/share", bytes.NewReader([]byte("{}")))
	req.Header.Set("Authorization", "Bearer "+shareTestJWT(t))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("free plan share: want 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestShareHandler_Create_PlusQuotaExceededReturns403(t *testing.T) {
	plusUser := &models.User{ID: uuid.New(), Email: "plus@test.com", Plan: models.PlanPlus}
	quota := &fakeShareQuota{err: errors.New("monthly share link limit reached")}
	r := newShareTestRouterWithQuota(t, &fakeShareRepo{}, quota, plusUser)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/share", bytes.NewReader([]byte("{}")))
	req.Header.Set("Authorization", "Bearer "+shareTestJWT(t))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("plus quota exceeded: want 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestShareHandler_Create_RepoError_Returns500(t *testing.T) {
	repo := &fakeShareRepo{createErr: errors.New("db write failed")}
	r := newShareTestRouter(t, repo, shareTestUser())

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/share", bytes.NewReader([]byte("{}")))
	req.Header.Set("Authorization", "Bearer "+shareTestJWT(t))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("repo error: want 500, got %d: %s", w.Code, w.Body.String())
	}
}

func TestShareHandler_List_MissingAuth_Returns401(t *testing.T) {
	r := newShareTestRouter(t, &fakeShareRepo{}, shareTestUser())
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/share", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("missing auth: want 401, got %d", w.Code)
	}
}
