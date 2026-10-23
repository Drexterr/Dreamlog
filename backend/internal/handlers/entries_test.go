package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/dreamlog/backend/internal/middleware"
	"github.com/dreamlog/backend/internal/models"
	"github.com/dreamlog/backend/internal/services"
	"github.com/dreamlog/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// ── fake entryServicer ────────────────────────────────────────────────────────

type fakeEntryServicer struct {
	existing    *models.Entry // FindByAudioKey result
	findErr     error
	createCalls int
	textCalls   int
	presignCalls int
	presignResp *models.PresignResponse
	presignErr  error
	createResp  *models.Entry
	createErr   error
	getResp     *models.Entry
	getErr      error
	listResp    *models.ListEntriesResponse
	listErr     error
}

func (f *fakeEntryServicer) PresignUpload(_ context.Context, _ uuid.UUID) (*models.PresignResponse, error) {
	f.presignCalls++
	return f.presignResp, f.presignErr
}

func (f *fakeEntryServicer) Create(_ context.Context, _ uuid.UUID, _ *models.CreateEntryInput, _ string) (*models.Entry, error) {
	f.createCalls++
	return f.createResp, f.createErr
}

func (f *fakeEntryServicer) FindByAudioKey(_ context.Context, _ uuid.UUID, _ string) (*models.Entry, error) {
	return f.existing, f.findErr
}

func (f *fakeEntryServicer) CreateText(_ context.Context, _ uuid.UUID, _ *models.CreateTextEntryInput, _ string) (*models.Entry, error) {
	f.textCalls++
	return f.createResp, f.createErr
}

func (f *fakeEntryServicer) Get(_ context.Context, _, _ uuid.UUID) (*models.Entry, error) {
	return f.getResp, f.getErr
}

func (f *fakeEntryServicer) List(_ context.Context, _ uuid.UUID, _, _ int) (*models.ListEntriesResponse, error) {
	return f.listResp, f.listErr
}

// ── fake storageUploader ──────────────────────────────────────────────────────

type fakeStorageUploader struct{ uploadErr error }

func (f *fakeStorageUploader) Upload(_ context.Context, _ string, _ io.Reader) error {
	return f.uploadErr
}

// ── fake entryQuotaChecker ────────────────────────────────────────────────────

type fakeEntryQuota struct {
	err   error
	calls int
}

func (f *fakeEntryQuota) CheckEntryQuota(_ context.Context, _ uuid.UUID, _ models.Plan) error {
	f.calls++
	return f.err
}

// ── test router ───────────────────────────────────────────────────────────────

const entryTestSecret = "entry-test-jwt-secret-32-bytes!!"

func newEntryTestRouter(t *testing.T, svc entryServicer, store storageUploader, testUser *models.User, quotas ...entryQuotaChecker) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	log := zap.NewNop()

	r := gin.New()
	r.Use(middleware.ErrorHandler(log))
	r.Use(middleware.AuthMiddleware(entryTestSecret, "", &fakeProvisioner{user: testUser}, log))

	var quota entryQuotaChecker = &fakeEntryQuota{}
	if len(quotas) > 0 {
		quota = quotas[0]
	}
	h := &EntryHandler{svc: svc, storage: store, subscription: quota}
	entries := r.Group("/entries")
	{
		entries.POST("/presign", h.Presign)
		entries.POST("/text", h.CreateText)
		entries.POST("", h.Create)
		entries.GET("", h.List)
		entries.GET("/:id", h.Get)
	}
	return r
}

func entryTestJWT(t *testing.T) string {
	t.Helper()
	claims := jwt.MapClaims{
		"aud":   "authenticated",
		"sub":   "test-sub-entry-001",
		"email": "entry@test.com",
		"exp":   time.Now().Add(time.Hour).Unix(),
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	str, err := tok.SignedString([]byte(entryTestSecret))
	if err != nil {
		t.Fatal(err)
	}
	return str
}

func entryTestUser() *models.User {
	return &models.User{ID: uuid.New(), Email: "entry@test.com", Name: "Entry User"}
}

// ── Presign ───────────────────────────────────────────────────────────────────

func TestEntryHandler_Presign_Returns200WithUploadURLAndKey(t *testing.T) {
	svc := &fakeEntryServicer{
		presignResp: &models.PresignResponse{
			UploadURL: "https://storage.example.com/upload?sig=xxx",
			AudioKey:  "audio/user/entry.aac",
			ExpiresIn: 900,
		},
	}
	r := newEntryTestRouter(t, svc, &fakeStorageUploader{}, entryTestUser())
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/entries/presign", nil)
	req.Header.Set("Authorization", "Bearer "+entryTestJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("presign: want 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp models.PresignResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.UploadURL == "" {
		t.Error("presign response must include upload_url")
	}
	if resp.AudioKey == "" {
		t.Error("presign response must include audio_key")
	}
}

func TestEntryHandler_Presign_MissingAuth_Returns401(t *testing.T) {
	r := newEntryTestRouter(t, &fakeEntryServicer{}, &fakeStorageUploader{}, entryTestUser())
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/entries/presign", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("missing auth: want 401, got %d", w.Code)
	}
}

func TestEntryHandler_Presign_ServiceError_Returns500(t *testing.T) {
	svc := &fakeEntryServicer{presignErr: errors.New("storage unavailable")}
	r := newEntryTestRouter(t, svc, &fakeStorageUploader{}, entryTestUser())
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/entries/presign", nil)
	req.Header.Set("Authorization", "Bearer "+entryTestJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("service error: want 500, got %d", w.Code)
	}
}

// ── Create ────────────────────────────────────────────────────────────────────

func TestEntryHandler_Create_ValidInput_Returns201(t *testing.T) {
	entryID := uuid.New()
	svc := &fakeEntryServicer{
		createResp: &models.Entry{
			ID:     entryID,
			Status: models.EntryStatusPending,
		},
	}
	r := newEntryTestRouter(t, svc, &fakeStorageUploader{}, entryTestUser())
	body, _ := json.Marshal(models.CreateEntryInput{
		AudioKey:       "audio/user/entry.aac",
		AudioSizeBytes: 204800,
		DurationSec:    120.5,
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/entries", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+entryTestJWT(t))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("create entry: want 201, got %d: %s", w.Code, w.Body.String())
	}
	var resp models.Entry
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.ID != entryID {
		t.Errorf("entry id: want %v, got %v", entryID, resp.ID)
	}
	if resp.Status != models.EntryStatusPending {
		t.Errorf("status: want pending, got %s", resp.Status)
	}
}

func TestEntryHandler_Create_MissingAudioKey_Returns400(t *testing.T) {
	r := newEntryTestRouter(t, &fakeEntryServicer{}, &fakeStorageUploader{}, entryTestUser())
	// audio_key is missing (required field)
	body, _ := json.Marshal(map[string]any{"duration_sec": 120.5, "audio_size_bytes": 1024})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/entries", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+entryTestJWT(t))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("missing audio_key: want 400, got %d", w.Code)
	}
}

func TestEntryHandler_Create_EmptyBody_Returns400(t *testing.T) {
	r := newEntryTestRouter(t, &fakeEntryServicer{}, &fakeStorageUploader{}, entryTestUser())
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/entries", nil)
	req.Header.Set("Authorization", "Bearer "+entryTestJWT(t))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("empty body: want 400, got %d", w.Code)
	}
}

func TestEntryHandler_Create_MissingAuth_Returns401(t *testing.T) {
	r := newEntryTestRouter(t, &fakeEntryServicer{}, &fakeStorageUploader{}, entryTestUser())
	body, _ := json.Marshal(models.CreateEntryInput{
		AudioKey:       "audio/user/entry.aac",
		AudioSizeBytes: 1024,
		DurationSec:    10,
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/entries", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("missing auth: want 401, got %d", w.Code)
	}
}

// ── Get ───────────────────────────────────────────────────────────────────────

func TestEntryHandler_Get_ValidID_Returns200(t *testing.T) {
	entryID := uuid.New()
	svc := &fakeEntryServicer{getResp: &models.Entry{
		ID:     entryID,
		Status: models.EntryStatusCompleted,
	}}
	r := newEntryTestRouter(t, svc, &fakeStorageUploader{}, entryTestUser())
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/entries/"+entryID.String(), nil)
	req.Header.Set("Authorization", "Bearer "+entryTestJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("get entry: want 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestEntryHandler_Get_NotFound_Returns404(t *testing.T) {
	// Service returns nil when entry not found or belongs to different user.
	svc := &fakeEntryServicer{getResp: nil}
	r := newEntryTestRouter(t, svc, &fakeStorageUploader{}, entryTestUser())
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/entries/"+uuid.New().String(), nil)
	req.Header.Set("Authorization", "Bearer "+entryTestJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("not found: want 404, got %d", w.Code)
	}
}

func TestEntryHandler_Get_WrongUser_Returns404(t *testing.T) {
	// Ownership check is in the service - nil return means not found or wrong user.
	svc := &fakeEntryServicer{getResp: nil}
	r := newEntryTestRouter(t, svc, &fakeStorageUploader{}, entryTestUser())
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/entries/"+uuid.New().String(), nil)
	req.Header.Set("Authorization", "Bearer "+entryTestJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("wrong user (ownership check): want 404, got %d", w.Code)
	}
}

func TestEntryHandler_Get_InvalidID_Returns400(t *testing.T) {
	r := newEntryTestRouter(t, &fakeEntryServicer{}, &fakeStorageUploader{}, entryTestUser())
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/entries/not-a-uuid", nil)
	req.Header.Set("Authorization", "Bearer "+entryTestJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("invalid id: want 400, got %d", w.Code)
	}
}

// ── List ──────────────────────────────────────────────────────────────────────

func TestEntryHandler_List_Returns200WithPagination(t *testing.T) {
	svc := &fakeEntryServicer{
		listResp: &models.ListEntriesResponse{
			Entries:  []*models.Entry{},
			Total:    0,
			Page:     1,
			PageSize: 20,
			HasMore:  false,
		},
	}
	r := newEntryTestRouter(t, svc, &fakeStorageUploader{}, entryTestUser())
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/entries?page=1&page_size=20", nil)
	req.Header.Set("Authorization", "Bearer "+entryTestJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("list: want 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp models.ListEntriesResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Entries == nil {
		t.Error("entries must not be nil (should be empty slice)")
	}
}

func TestEntryHandler_List_MissingAuth_Returns401(t *testing.T) {
	r := newEntryTestRouter(t, &fakeEntryServicer{}, &fakeStorageUploader{}, entryTestUser())
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/entries", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("missing auth: want 401, got %d", w.Code)
	}
}

// N7: a user at the monthly limit is refused before uploading, so no audio is
// orphaned in storage when registration would fail.
func TestEntryHandler_Presign_QuotaExceeded_Returns409WithoutPresigning(t *testing.T) {
	svc := &fakeEntryServicer{presignResp: &models.PresignResponse{UploadURL: "u", AudioKey: "k"}}
	quota := &fakeEntryQuota{err: services.ErrEntryQuotaExceeded}
	r := newEntryTestRouter(t, svc, &fakeStorageUploader{}, entryTestUser(), quota)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/entries/presign", nil)
	req.Header.Set("Authorization", "Bearer "+entryTestJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("presign at quota: want 409, got %d: %s", w.Code, w.Body.String())
	}
	if quota.calls != 1 || svc.presignCalls != 0 {
		t.Errorf("quota must be checked and no upload URL issued: quota=%d presign=%d", quota.calls, svc.presignCalls)
	}
}

// ── N10: POST /entries error mapping and retry order ─────────────────────────

func postEntry(t *testing.T, r http.Handler) *httptest.ResponseRecorder {
	t.Helper()
	body, _ := json.Marshal(models.CreateEntryInput{AudioKey: "audio/user/entry.aac", AudioSizeBytes: 1, DurationSec: 1})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/entries", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+entryTestJWT(t))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	return w
}

// A lost response for the 10th entry: the retry gets the entry back even
// though the user is now at the monthly limit.
func TestEntryHandler_Create_RetryAtQuotaReturnsExistingEntry(t *testing.T) {
	existing := &models.Entry{ID: uuid.New(), Status: models.EntryStatusPending}
	svc := &fakeEntryServicer{existing: existing}
	quota := &fakeEntryQuota{err: services.ErrEntryQuotaExceeded}
	w := postEntry(t, newEntryTestRouter(t, svc, &fakeStorageUploader{}, entryTestUser(), quota))
	if w.Code != http.StatusCreated || !strings.Contains(w.Body.String(), existing.ID.String()) {
		t.Fatalf("retry must return the existing entry: %d %s", w.Code, w.Body.String())
	}
	if quota.calls != 0 || svc.createCalls != 0 {
		t.Errorf("a retry must not check quota or create: quota=%d create=%d", quota.calls, svc.createCalls)
	}
}

func TestEntryHandler_Create_QuotaReached_Returns409(t *testing.T) {
	quota := &fakeEntryQuota{err: services.ErrEntryQuotaExceeded}
	w := postEntry(t, newEntryTestRouter(t, &fakeEntryServicer{}, &fakeStorageUploader{}, entryTestUser(), quota))
	if w.Code != http.StatusConflict {
		t.Fatalf("want 409, got %d", w.Code)
	}
}

// A failure to count entries is an outage (500), not a quota verdict.
func TestEntryHandler_Create_QuotaCheckError_Returns500(t *testing.T) {
	quota := &fakeEntryQuota{err: errors.New("db: connection refused")}
	w := postEntry(t, newEntryTestRouter(t, &fakeEntryServicer{}, &fakeStorageUploader{}, entryTestUser(), quota))
	if w.Code != http.StatusInternalServerError || strings.Contains(w.Body.String(), "connection refused") {
		t.Fatalf("want 500 without internal text, got %d %s", w.Code, w.Body.String())
	}
}

// Storage / DB failures are retryable 500s without internal detail, while
// typed validation errors keep their status.
func TestEntryHandler_Create_ServiceErrors(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want int
	}{
		{"outage", errors.New("storage: head object: dial tcp: timeout"), http.StatusInternalServerError},
		{"validation", apierr.BadRequest("audio_key does not belong to user"), http.StatusBadRequest},
		{"lookup outage", nil, http.StatusInternalServerError},
	}
	for _, tc := range cases {
		svc := &fakeEntryServicer{createErr: tc.err}
		if tc.err == nil {
			svc.findErr = errors.New("db down")
		}
		w := postEntry(t, newEntryTestRouter(t, svc, &fakeStorageUploader{}, entryTestUser()))
		if w.Code != tc.want {
			t.Errorf("%s: want %d, got %d %s", tc.name, tc.want, w.Code, w.Body.String())
		}
		if tc.want == http.StatusInternalServerError && (strings.Contains(w.Body.String(), "dial tcp") || strings.Contains(w.Body.String(), "db down")) {
			t.Errorf("%s: internal detail leaked: %s", tc.name, w.Body.String())
		}
	}
}
