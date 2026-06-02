package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/dreamlog/backend/internal/middleware"
	"github.com/dreamlog/backend/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// ── fake companyRepo ──────────────────────────────────────────────────────────

type fakeCompanyRepo struct {
	getBySlugResp    *models.Company
	getBySlugErr     error
	isMemberResp     *models.CompanyMember
	isMemberErr      error
	totalMembersResp int
	totalMembersErr  error
	teamMoodResp     []*models.TeamDailyMood
	teamMoodErr      error
	joinErr          error
}

func (f *fakeCompanyRepo) GetBySlug(_ context.Context, _ string) (*models.Company, error) {
	return f.getBySlugResp, f.getBySlugErr
}

func (f *fakeCompanyRepo) IsMember(_ context.Context, _, _ uuid.UUID) (*models.CompanyMember, error) {
	return f.isMemberResp, f.isMemberErr
}

func (f *fakeCompanyRepo) TotalMembers(_ context.Context, _ uuid.UUID) (int, error) {
	return f.totalMembersResp, f.totalMembersErr
}

func (f *fakeCompanyRepo) TeamMoodHistory(_ context.Context, _ uuid.UUID, _, _ time.Time) ([]*models.TeamDailyMood, error) {
	return f.teamMoodResp, f.teamMoodErr
}

func (f *fakeCompanyRepo) JoinCompany(_ context.Context, _, _ uuid.UUID) error {
	return f.joinErr
}

// ── router builder ────────────────────────────────────────────────────────────

const b2bTestSecret = "b2b-test-jwt-secret-32-bytes!!!!"

func newB2BTestRouter(t *testing.T, repo *fakeCompanyRepo, testUser *models.User) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	log := zap.NewNop()

	r := gin.New()
	r.Use(middleware.ErrorHandler(log))
	r.Use(middleware.AuthMiddleware(b2bTestSecret, &fakeProvisioner{user: testUser}, log))

	h := NewB2BHandler(repo)
	r.POST("/b2b/companies/:slug/join", h.Join)
	r.GET("/b2b/companies/:slug/mood", h.TeamMood)
	return r
}

func b2bTestJWT(t *testing.T) string {
	t.Helper()
	claims := jwt.MapClaims{
		"sub":   "test-sub-b2b-001",
		"email": "b2b@test.com",
		"exp":   time.Now().Add(time.Hour).Unix(),
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	str, err := tok.SignedString([]byte(b2bTestSecret))
	if err != nil {
		t.Fatal(err)
	}
	return str
}

func b2bTestUser() *models.User {
	return &models.User{ID: uuid.New(), Email: "b2b@test.com", Name: "B2B User"}
}

func defaultCompany() *models.Company {
	return &models.Company{
		ID:        uuid.New(),
		Name:      "Acme Corp",
		Slug:      "acme",
		SeatLimit: 100,
	}
}

// ── Join tests ────────────────────────────────────────────────────────────────

func TestB2BHandler_Join_ValidSlug_Returns200WithCompanyInfo(t *testing.T) {
	company := defaultCompany()
	repo := &fakeCompanyRepo{
		getBySlugResp:    company,
		totalMembersResp: 5, // well under seat limit
	}
	r := newB2BTestRouter(t, repo, b2bTestUser())
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/b2b/companies/acme/join", nil)
	req.Header.Set("Authorization", "Bearer "+b2bTestJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("join: want 200, got %d: %s", w.Code, w.Body.String())
	}
	var body map[string]any
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["company_name"] != "Acme Corp" {
		t.Errorf("company_name: want %q, got %v", "Acme Corp", body["company_name"])
	}
	if body["role"] != "member" {
		t.Errorf("role: want member, got %v", body["role"])
	}
	if body["company_id"] == nil {
		t.Error("response must include company_id")
	}
}

func TestB2BHandler_Join_CompanyNotFound_Returns404(t *testing.T) {
	repo := &fakeCompanyRepo{getBySlugResp: nil}
	r := newB2BTestRouter(t, repo, b2bTestUser())
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/b2b/companies/unknown/join", nil)
	req.Header.Set("Authorization", "Bearer "+b2bTestJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("company not found: want 404, got %d", w.Code)
	}
}

func TestB2BHandler_Join_SeatLimitReached_Returns409(t *testing.T) {
	company := defaultCompany()
	company.SeatLimit = 10
	repo := &fakeCompanyRepo{
		getBySlugResp:    company,
		totalMembersResp: 10, // exactly at seat limit
	}
	r := newB2BTestRouter(t, repo, b2bTestUser())
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/b2b/companies/acme/join", nil)
	req.Header.Set("Authorization", "Bearer "+b2bTestJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("seat limit: want 409, got %d: %s", w.Code, w.Body.String())
	}
}

func TestB2BHandler_Join_SeatLimitExceeded_Returns409(t *testing.T) {
	company := defaultCompany()
	company.SeatLimit = 5
	repo := &fakeCompanyRepo{
		getBySlugResp:    company,
		totalMembersResp: 10, // exceeds limit
	}
	r := newB2BTestRouter(t, repo, b2bTestUser())
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/b2b/companies/acme/join", nil)
	req.Header.Set("Authorization", "Bearer "+b2bTestJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("seat exceeded: want 409, got %d", w.Code)
	}
}

func TestB2BHandler_Join_MissingAuth_Returns401(t *testing.T) {
	r := newB2BTestRouter(t, &fakeCompanyRepo{}, b2bTestUser())
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/b2b/companies/acme/join", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("missing auth: want 401, got %d", w.Code)
	}
}

func TestB2BHandler_Join_JoinRepoError_Returns500(t *testing.T) {
	company := defaultCompany()
	repo := &fakeCompanyRepo{
		getBySlugResp:    company,
		totalMembersResp: 0,
		joinErr:          errors.New("db error"),
	}
	r := newB2BTestRouter(t, repo, b2bTestUser())
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/b2b/companies/acme/join", nil)
	req.Header.Set("Authorization", "Bearer "+b2bTestJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("join db error: want 500, got %d", w.Code)
	}
}

// ── TeamMood tests ────────────────────────────────────────────────────────────

func TestB2BHandler_TeamMood_AdminUser_Returns200WithData(t *testing.T) {
	company := defaultCompany()
	repo := &fakeCompanyRepo{
		getBySlugResp: company,
		isMemberResp: &models.CompanyMember{
			ID:        uuid.New(),
			CompanyID: company.ID,
			Role:      "admin",
		},
		totalMembersResp: 25,
		teamMoodResp: []*models.TeamDailyMood{
			{Day: "2026-05-21", AvgMood: 64, ActiveMembers: 10, EntryCount: 18},
			{Day: "2026-05-22", AvgMood: 70, ActiveMembers: 12, EntryCount: 20},
		},
	}
	r := newB2BTestRouter(t, repo, b2bTestUser())
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/b2b/companies/acme/mood", nil)
	req.Header.Set("Authorization", "Bearer "+b2bTestJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("team mood admin: want 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp models.TeamMoodSummary
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.CompanyName != "Acme Corp" {
		t.Errorf("company_name: want Acme Corp, got %s", resp.CompanyName)
	}
	if resp.TotalMembers != 25 {
		t.Errorf("total_members: want 25, got %d", resp.TotalMembers)
	}
	if len(resp.Days) != 2 {
		t.Errorf("days: want 2, got %d", len(resp.Days))
	}
}

func TestB2BHandler_TeamMood_NonAdmin_Returns403(t *testing.T) {
	company := defaultCompany()
	repo := &fakeCompanyRepo{
		getBySlugResp: company,
		isMemberResp: &models.CompanyMember{
			ID:        uuid.New(),
			CompanyID: company.ID,
			Role:      "member", // not admin
		},
	}
	r := newB2BTestRouter(t, repo, b2bTestUser())
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/b2b/companies/acme/mood", nil)
	req.Header.Set("Authorization", "Bearer "+b2bTestJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("non-admin: want 403, got %d", w.Code)
	}
}

func TestB2BHandler_TeamMood_NotMember_Returns403(t *testing.T) {
	company := defaultCompany()
	repo := &fakeCompanyRepo{
		getBySlugResp: company,
		isMemberResp:  nil, // not a member at all
	}
	r := newB2BTestRouter(t, repo, b2bTestUser())
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/b2b/companies/acme/mood", nil)
	req.Header.Set("Authorization", "Bearer "+b2bTestJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("not member: want 403, got %d", w.Code)
	}
}

func TestB2BHandler_TeamMood_CompanyNotFound_Returns404(t *testing.T) {
	repo := &fakeCompanyRepo{getBySlugResp: nil}
	r := newB2BTestRouter(t, repo, b2bTestUser())
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/b2b/companies/unknown/mood", nil)
	req.Header.Set("Authorization", "Bearer "+b2bTestJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("company not found: want 404, got %d", w.Code)
	}
}

func TestB2BHandler_TeamMood_AlertFiredWhenAvgBelow40(t *testing.T) {
	company := defaultCompany()
	repo := &fakeCompanyRepo{
		getBySlugResp: company,
		isMemberResp:  &models.CompanyMember{Role: "admin"},
		teamMoodResp: []*models.TeamDailyMood{
			{Day: "2026-05-21", AvgMood: 35, ActiveMembers: 8, EntryCount: 10},
		},
	}
	r := newB2BTestRouter(t, repo, b2bTestUser())
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/b2b/companies/acme/mood", nil)
	req.Header.Set("Authorization", "Bearer "+b2bTestJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("alert check: want 200, got %d", w.Code)
	}
	var resp models.TeamMoodSummary
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !resp.IsAlerted {
		t.Error("is_alerted must be true when avg_mood < 40")
	}
}

func TestB2BHandler_TeamMood_NoAlertWhenAvgAbove40(t *testing.T) {
	company := defaultCompany()
	repo := &fakeCompanyRepo{
		getBySlugResp: company,
		isMemberResp:  &models.CompanyMember{Role: "admin"},
		teamMoodResp: []*models.TeamDailyMood{
			{Day: "2026-05-21", AvgMood: 65, ActiveMembers: 8, EntryCount: 10},
		},
	}
	r := newB2BTestRouter(t, repo, b2bTestUser())
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/b2b/companies/acme/mood?range=90d", nil)
	req.Header.Set("Authorization", "Bearer "+b2bTestJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("no alert: want 200, got %d", w.Code)
	}
	var resp models.TeamMoodSummary
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.IsAlerted {
		t.Error("is_alerted must be false when avg_mood >= 40")
	}
}

func TestB2BHandler_TeamMood_NoData_ReturnsNilAvgMood(t *testing.T) {
	company := defaultCompany()
	repo := &fakeCompanyRepo{
		getBySlugResp: company,
		isMemberResp:  &models.CompanyMember{Role: "admin"},
		teamMoodResp:  nil, // no data
	}
	r := newB2BTestRouter(t, repo, b2bTestUser())
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/b2b/companies/acme/mood", nil)
	req.Header.Set("Authorization", "Bearer "+b2bTestJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("no data: want 200, got %d", w.Code)
	}
	var resp models.TeamMoodSummary
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.AvgMood != nil {
		t.Errorf("avg_mood: want nil when no data, got %v", resp.AvgMood)
	}
}
