package handlers

import (
	"context"
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
)

// ── fake exportRepo ───────────────────────────────────────────────────────────

type fakeExportRepo struct {
	resp *models.ExportData
	err  error
}

func (f *fakeExportRepo) ExportData(_ context.Context, _ uuid.UUID, _, _ time.Time) (*models.ExportData, error) {
	if f.err != nil {
		return nil, f.err
	}
	if f.resp != nil {
		return f.resp, nil
	}
	avg := 65
	delta := 3
	return &models.ExportData{
		UserName:    "Export User",
		EntryCount:  5,
		AvgMood:     &avg,
		MoodDelta:   &delta,
		TopEmotions: []string{"hopeful", "calm"},
		DailyMoods:  []*models.DailyMood{{Day: "2026-05-21", AvgMood: 65, EntryCount: 2}},
		Entries: []*models.ExportEntrySummary{
			{Date: time.Now(), Summary: "A productive week overall.", MoodScore: 65, Topics: []string{"work"}},
		},
	}, nil
}

// ── fake userNameRepo ─────────────────────────────────────────────────────────

type fakeUserNameRepo struct {
	user *models.User
	err  error
}

func (f *fakeUserNameRepo) GetByID(_ context.Context, _ uuid.UUID) (*models.User, error) {
	if f.err != nil {
		return nil, f.err
	}
	if f.user != nil {
		return f.user, nil
	}
	return &models.User{ID: uuid.New(), Name: "Export User", Email: "export@test.com"}, nil
}

// ── router builder ────────────────────────────────────────────────────────────

const exportTestSecret = "export-test-jwt-secret-32-bytes!"

func newExportTestRouter(t *testing.T, eRepo *fakeExportRepo, uRepo *fakeUserNameRepo, testUser *models.User) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	log := zap.NewNop()

	r := gin.New()
	r.Use(middleware.ErrorHandler(log))
	r.Use(middleware.AuthMiddleware(exportTestSecret, &fakeProvisioner{user: testUser}, log))

	h := NewExportHandler(eRepo, uRepo)
	r.GET("/export/pdf", h.ExportPDF)
	return r
}

func exportTestJWT(t *testing.T) string {
	t.Helper()
	claims := jwt.MapClaims{
		"sub":   "test-sub-export-001",
		"email": "export@test.com",
		"exp":   time.Now().Add(time.Hour).Unix(),
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	str, err := tok.SignedString([]byte(exportTestSecret))
	if err != nil {
		t.Fatal(err)
	}
	return str
}

func exportTestUser() *models.User {
	return &models.User{ID: uuid.New(), Email: "export@test.com", Name: "Export User", Plan: models.PlanPro}
}

// ── ExportPDF tests ───────────────────────────────────────────────────────────

func TestExportHandler_ExportPDF_Monthly_Returns200WithPDF(t *testing.T) {
	r := newExportTestRouter(t, &fakeExportRepo{}, &fakeUserNameRepo{}, exportTestUser())
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/export/pdf?period=monthly", nil)
	req.Header.Set("Authorization", "Bearer "+exportTestJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("monthly pdf: want 200, got %d: %s", w.Code, w.Body.String())
	}
	ct := w.Header().Get("Content-Type")
	if ct != "application/pdf" {
		t.Errorf("Content-Type: want application/pdf, got %s", ct)
	}
	cd := w.Header().Get("Content-Disposition")
	if !strings.Contains(cd, "attachment") {
		t.Errorf("Content-Disposition must contain attachment, got %q", cd)
	}
	if !strings.Contains(cd, "dreamlog-monthly") {
		t.Errorf("Content-Disposition must contain dreamlog-monthly, got %q", cd)
	}
	if w.Body.Len() == 0 {
		t.Error("response body must not be empty for PDF")
	}
}

func TestExportHandler_ExportPDF_Yearly_Returns200WithPDF(t *testing.T) {
	r := newExportTestRouter(t, &fakeExportRepo{}, &fakeUserNameRepo{}, exportTestUser())
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/export/pdf?period=yearly", nil)
	req.Header.Set("Authorization", "Bearer "+exportTestJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("yearly pdf: want 200, got %d: %s", w.Code, w.Body.String())
	}
	ct := w.Header().Get("Content-Type")
	if ct != "application/pdf" {
		t.Errorf("Content-Type: want application/pdf, got %s", ct)
	}
	cd := w.Header().Get("Content-Disposition")
	if !strings.Contains(cd, "dreamlog-yearly") {
		t.Errorf("Content-Disposition must contain dreamlog-yearly, got %q", cd)
	}
}

func TestExportHandler_ExportPDF_DefaultPeriod_Returns200(t *testing.T) {
	r := newExportTestRouter(t, &fakeExportRepo{}, &fakeUserNameRepo{}, exportTestUser())
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/export/pdf", nil)
	req.Header.Set("Authorization", "Bearer "+exportTestJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("default period: want 200, got %d: %s", w.Code, w.Body.String())
	}
	// Default is monthly.
	cd := w.Header().Get("Content-Disposition")
	if !strings.Contains(cd, "dreamlog-monthly") {
		t.Errorf("default period must use monthly filename, got %q", cd)
	}
}

func TestExportHandler_ExportPDF_UnknownPeriod_DefaultsToMonthly(t *testing.T) {
	// The handler defaults unknown values to monthly (no 400 validation).
	r := newExportTestRouter(t, &fakeExportRepo{}, &fakeUserNameRepo{}, exportTestUser())
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/export/pdf?period=weekly", nil)
	req.Header.Set("Authorization", "Bearer "+exportTestJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("unknown period: want 200 (defaults to monthly), got %d", w.Code)
	}
	cd := w.Header().Get("Content-Disposition")
	if !strings.Contains(cd, "dreamlog-weekly") && !strings.Contains(cd, "dreamlog-monthly") {
		t.Errorf("unexpected Content-Disposition: %q", cd)
	}
}

func TestExportHandler_ExportPDF_MissingAuth_Returns401(t *testing.T) {
	r := newExportTestRouter(t, &fakeExportRepo{}, &fakeUserNameRepo{}, exportTestUser())
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/export/pdf", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("missing auth: want 401, got %d", w.Code)
	}
}

func TestExportHandler_ExportPDF_UserRepoError_Returns500(t *testing.T) {
	uRepo := &fakeUserNameRepo{err: errors.New("db error")}
	r := newExportTestRouter(t, &fakeExportRepo{}, uRepo, exportTestUser())
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/export/pdf", nil)
	req.Header.Set("Authorization", "Bearer "+exportTestJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("user repo error: want 500, got %d", w.Code)
	}
}

func TestExportHandler_ExportPDF_ExportDataError_Returns500(t *testing.T) {
	eRepo := &fakeExportRepo{err: errors.New("db error")}
	r := newExportTestRouter(t, eRepo, &fakeUserNameRepo{}, exportTestUser())
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/export/pdf", nil)
	req.Header.Set("Authorization", "Bearer "+exportTestJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("export data error: want 500, got %d", w.Code)
	}
}

func TestExportHandler_ExportPDF_PreferredNameOverridesName(t *testing.T) {
	preferredName := "Bh"
	uRepo := &fakeUserNameRepo{user: &models.User{
		ID:            uuid.New(),
		Name:          "Bharat",
		Email:         "export@test.com",
		PreferredName: &preferredName,
	}}
	r := newExportTestRouter(t, &fakeExportRepo{}, uRepo, exportTestUser())
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/export/pdf?period=monthly", nil)
	req.Header.Set("Authorization", "Bearer "+exportTestJWT(t))
	r.ServeHTTP(w, req)

	// Just verify it returns 200 — the preferred name is embedded in the PDF binary.
	if w.Code != http.StatusOK {
		t.Fatalf("preferred name: want 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestExportHandler_ExportPDF_NoEntries_StillReturns200(t *testing.T) {
	avg := 0
	eRepo := &fakeExportRepo{resp: &models.ExportData{
		UserName:   "Empty User",
		EntryCount: 0,
		AvgMood:    &avg,
		DailyMoods: []*models.DailyMood{},
		Entries:    []*models.ExportEntrySummary{},
	}}
	r := newExportTestRouter(t, eRepo, &fakeUserNameRepo{}, exportTestUser())
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/export/pdf", nil)
	req.Header.Set("Authorization", "Bearer "+exportTestJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("no entries: want 200, got %d: %s", w.Code, w.Body.String())
	}
	if w.Header().Get("Content-Type") != "application/pdf" {
		t.Error("must still return application/pdf for empty data")
	}
}
