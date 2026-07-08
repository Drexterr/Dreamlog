package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/dreamlog/backend/internal/middleware"
	"github.com/dreamlog/backend/internal/models"
	"github.com/dreamlog/backend/internal/services"
	pkgcrypto "github.com/dreamlog/backend/pkg/crypto"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// ── fakes ─────────────────────────────────────────────────────────────────────

type fakeTherapistLookup struct {
	therapist *models.Therapist // nil = caller is not a therapist
}

func (f *fakeTherapistLookup) GetByUserID(_ context.Context, _ uuid.UUID) (*models.Therapist, error) {
	return f.therapist, nil
}

type fakeConsentStore struct {
	mu               sync.Mutex
	userAccepted     map[uuid.UUID]string
	therapistConsent map[uuid.UUID]bool
}

func newFakeConsentStore() *fakeConsentStore {
	return &fakeConsentStore{
		userAccepted:     map[uuid.UUID]string{},
		therapistConsent: map[uuid.UUID]bool{},
	}
}

func (f *fakeConsentStore) AcceptUserTerms(_ context.Context, userID uuid.UUID, version string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.userAccepted[userID] = version
	return nil
}

func (f *fakeConsentStore) GetUserTerms(_ context.Context, userID uuid.UUID) (*time.Time, *string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if v, ok := f.userAccepted[userID]; ok {
		now := time.Now()
		return &now, &v, nil
	}
	return nil, nil, nil
}

func (f *fakeConsentStore) AcceptTherapistClientConsent(_ context.Context, therapistID uuid.UUID, _ string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.therapistConsent[therapistID] = true
	return nil
}

func (f *fakeConsentStore) TherapistClientConsentAccepted(_ context.Context, therapistID uuid.UUID) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.therapistConsent[therapistID], nil
}

// hNotesRepo is a minimal in-memory notes repo for handler-level tests.
type hNotesRepo struct {
	mu      sync.Mutex
	keys    map[uuid.UUID][]byte
	clients map[uuid.UUID]*models.ExternalClientRow
}

func newHNotesRepo() *hNotesRepo {
	return &hNotesRepo{keys: map[uuid.UUID][]byte{}, clients: map[uuid.UUID]*models.ExternalClientRow{}}
}

func (r *hNotesRepo) GetWrappedDEK(_ context.Context, id uuid.UUID) ([]byte, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.keys[id], nil
}

func (r *hNotesRepo) InsertWrappedDEK(_ context.Context, id uuid.UUID, wrapped []byte) ([]byte, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if existing, ok := r.keys[id]; ok {
		return existing, nil
	}
	r.keys[id] = wrapped
	return wrapped, nil
}

func (r *hNotesRepo) CreateExternalClient(_ context.Context, therapistID uuid.UUID, nameEnc []byte, role string) (*models.ExternalClientRow, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	row := &models.ExternalClientRow{ID: uuid.New(), TherapistID: therapistID, NameEnc: nameEnc, Role: role, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	r.clients[row.ID] = row
	return row, nil
}

func (r *hNotesRepo) ListExternalClients(_ context.Context, therapistID uuid.UUID, _ bool) ([]*models.ExternalClientRow, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []*models.ExternalClientRow
	for _, c := range r.clients {
		if c.TherapistID == therapistID {
			out = append(out, c)
		}
	}
	return out, nil
}

func (r *hNotesRepo) GetExternalClient(_ context.Context, therapistID, clientID uuid.UUID) (*models.ExternalClientRow, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	c, ok := r.clients[clientID]
	if !ok || c.TherapistID != therapistID {
		return nil, nil
	}
	return c, nil
}

func (r *hNotesRepo) UpdateExternalClient(_ context.Context, _, _ uuid.UUID, _ []byte, _ *string, _ *bool) (bool, error) {
	return false, nil
}
func (r *hNotesRepo) DeleteExternalClient(_ context.Context, _, _ uuid.UUID) (bool, error) {
	return false, nil
}
func (r *hNotesRepo) CreateSession(_ context.Context, s *models.ClientSessionRow) (*models.ClientSessionRow, error) {
	s.ID = uuid.New()
	return s, nil
}
func (r *hNotesRepo) GetSession(_ context.Context, _, _ uuid.UUID) (*models.ClientSessionRow, error) {
	return nil, nil
}
func (r *hNotesRepo) ListSessionsForExternalClient(_ context.Context, _, _ uuid.UUID, _ int) ([]*models.ClientSessionRow, error) {
	return nil, nil
}
func (r *hNotesRepo) ListSessionsForLinkedClient(_ context.Context, _, _ uuid.UUID, _ int) ([]*models.ClientSessionRow, error) {
	return nil, nil
}
func (r *hNotesRepo) ListRecentSessions(_ context.Context, _ uuid.UUID, _ int) ([]*models.ClientSessionRow, error) {
	return nil, nil
}
func (r *hNotesRepo) UpdateSessionBullets(_ context.Context, _, _ uuid.UUID, _ []byte) (bool, error) {
	return false, nil
}
func (r *hNotesRepo) UpdateSessionSummary(_ context.Context, _, _ uuid.UUID, _ []byte) (bool, error) {
	return false, nil
}
func (r *hNotesRepo) DeleteSession(_ context.Context, _, _ uuid.UUID) (*string, bool, error) {
	return nil, false, nil
}
func (r *hNotesRepo) Overview(_ context.Context, _ uuid.UUID) (*models.TherapistOverview, error) {
	return &models.TherapistOverview{}, nil
}

type hLinkChecker struct{}

func (hLinkChecker) GetClientLink(_ context.Context, _, _ uuid.UUID) (*models.ClientTherapistLink, error) {
	return nil, nil
}

type hNotesStorage struct{}

func (hNotesStorage) PresignPutKey(_ context.Context, _ string) (string, error) {
	return "https://storage.example/put", nil
}
func (hNotesStorage) Delete(_ context.Context, _ string) error { return nil }

type hNotesQueue struct{}

func (hNotesQueue) Enqueue(_ context.Context, _ any) error { return nil }

type hSummarizer struct{}

func (hSummarizer) GenerateSessionNotesSummary(_ context.Context, _, _ string, _ []string) (string, error) {
	return "summary", nil
}

// ── router builder ────────────────────────────────────────────────────────────

const notesTestSecret = "notes-test-jwt-secret-32-bytes!!"

func newNotesTestRouter(t *testing.T, lookup *fakeTherapistLookup, consent *fakeConsentStore, testUser *models.User) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)

	repo := newHNotesRepo()
	master, err := pkgcrypto.NewDEK()
	if err != nil {
		t.Fatal(err)
	}
	cipher := services.NewNotesCipher(master, repo)
	svc := services.NewTherapistNotesService(repo, hLinkChecker{}, hNotesQueue{}, hNotesStorage{}, hSummarizer{}, cipher)
	h := NewTherapistNotesHandler(svc, lookup, consent)

	log := zap.NewNop()
	r := gin.New()
	r.Use(middleware.ErrorHandler(log))
	r.Use(middleware.AuthMiddleware(notesTestSecret, "", &fakeProvisioner{user: testUser}, log))
	r.GET("/therapists/me", h.GetMe)
	r.POST("/therapists/consent", h.AcceptClientConsent)
	r.GET("/therapists/overview", h.Overview)
	r.POST("/therapists/external-clients", h.CreateExternalClient)
	r.GET("/therapists/external-clients", h.ListExternalClients)
	r.POST("/therapists/sessions/presign", h.PresignNote)
	r.POST("/me/accept-terms", h.AcceptUserTerms)
	r.GET("/me/terms", h.GetUserTerms)
	return r
}

func notesTestJWT(t *testing.T) string {
	t.Helper()
	claims := jwt.MapClaims{
		"sub":   "notes-test-sub",
		"email": "notes@test.com",
		"exp":   time.Now().Add(time.Hour).Unix(),
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	str, err := tok.SignedString([]byte(notesTestSecret))
	if err != nil {
		t.Fatal(err)
	}
	return str
}

func doNotesRequest(t *testing.T, r *gin.Engine, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var reader *bytes.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		reader = bytes.NewReader(b)
	} else {
		reader = bytes.NewReader(nil)
	}
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(method, path, reader)
	req.Header.Set("Authorization", "Bearer "+notesTestJWT(t))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	return w
}

func notesTestUser() *models.User {
	return &models.User{ID: uuid.New(), Email: "notes@test.com", Name: "Notes Tester"}
}

func notesTestTherapist(userID uuid.UUID) *models.Therapist {
	return &models.Therapist{ID: uuid.New(), UserID: userID, Name: "Dr. Rao", Email: "rao@clinic.example", Plan: "trial"}
}

// ── tests ────────────────────────────────────────────────────────────────────

func TestNotesHandler_GetMe_NotATherapist404(t *testing.T) {
	r := newNotesTestRouter(t, &fakeTherapistLookup{therapist: nil}, newFakeConsentStore(), notesTestUser())
	w := doNotesRequest(t, r, http.MethodGet, "/therapists/me", nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("want 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestNotesHandler_GetMe_Therapist200WithConsentFlag(t *testing.T) {
	user := notesTestUser()
	therapist := notesTestTherapist(user.ID)
	consent := newFakeConsentStore()
	r := newNotesTestRouter(t, &fakeTherapistLookup{therapist: therapist}, consent, user)

	w := doNotesRequest(t, r, http.MethodGet, "/therapists/me", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp struct {
		Therapist             models.Therapist `json:"therapist"`
		ClientConsentAccepted bool             `json:"client_consent_accepted"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Therapist.ID != therapist.ID || resp.ClientConsentAccepted {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestNotesHandler_ExternalClients_RequireTherapist(t *testing.T) {
	r := newNotesTestRouter(t, &fakeTherapistLookup{therapist: nil}, newFakeConsentStore(), notesTestUser())

	if w := doNotesRequest(t, r, http.MethodPost, "/therapists/external-clients", map[string]string{"name": "Asha"}); w.Code != http.StatusForbidden {
		t.Fatalf("create: want 403, got %d", w.Code)
	}
	if w := doNotesRequest(t, r, http.MethodGet, "/therapists/external-clients", nil); w.Code != http.StatusForbidden {
		t.Fatalf("list: want 403, got %d", w.Code)
	}
	if w := doNotesRequest(t, r, http.MethodGet, "/therapists/overview", nil); w.Code != http.StatusForbidden {
		t.Fatalf("overview: want 403, got %d", w.Code)
	}
}

func TestNotesHandler_CreateClient_ConsentGate(t *testing.T) {
	user := notesTestUser()
	therapist := notesTestTherapist(user.ID)
	consent := newFakeConsentStore()
	r := newNotesTestRouter(t, &fakeTherapistLookup{therapist: therapist}, consent, user)

	// Before consent: 403.
	w := doNotesRequest(t, r, http.MethodPost, "/therapists/external-clients", map[string]string{"name": "Asha K"})
	if w.Code != http.StatusForbidden {
		t.Fatalf("pre-consent create: want 403, got %d: %s", w.Code, w.Body.String())
	}
	// Presign also gated.
	w = doNotesRequest(t, r, http.MethodPost, "/therapists/sessions/presign", map[string]string{"content_type": "image/jpeg"})
	if w.Code != http.StatusForbidden {
		t.Fatalf("pre-consent presign: want 403, got %d", w.Code)
	}

	// Accept consent.
	w = doNotesRequest(t, r, http.MethodPost, "/therapists/consent", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("consent: want 200, got %d: %s", w.Code, w.Body.String())
	}

	// After consent: create succeeds and returns the decrypted name.
	w = doNotesRequest(t, r, http.MethodPost, "/therapists/external-clients", map[string]string{"name": "Asha K"})
	if w.Code != http.StatusCreated {
		t.Fatalf("post-consent create: want 201, got %d: %s", w.Code, w.Body.String())
	}
	var client models.ExternalClient
	if err := json.Unmarshal(w.Body.Bytes(), &client); err != nil {
		t.Fatal(err)
	}
	if client.Name != "Asha K" || client.TherapistID != therapist.ID {
		t.Fatalf("unexpected client: %+v", client)
	}
}

func TestNotesHandler_Presign_RejectsBadContentType(t *testing.T) {
	user := notesTestUser()
	therapist := notesTestTherapist(user.ID)
	consent := newFakeConsentStore()
	consent.therapistConsent[therapist.ID] = true
	r := newNotesTestRouter(t, &fakeTherapistLookup{therapist: therapist}, consent, user)

	w := doNotesRequest(t, r, http.MethodPost, "/therapists/sessions/presign", map[string]string{"content_type": "application/pdf"})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400 for pdf, got %d", w.Code)
	}

	w = doNotesRequest(t, r, http.MethodPost, "/therapists/sessions/presign", map[string]string{"content_type": "image/jpeg"})
	if w.Code != http.StatusOK {
		t.Fatalf("want 200 for jpeg, got %d: %s", w.Code, w.Body.String())
	}
	var resp struct {
		UploadURL string `json:"upload_url"`
		ImageKey  string `json:"image_key"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	// Key must be namespaced under this therapist.
	wantPrefix := "notes/" + therapist.ID.String() + "/"
	if len(resp.ImageKey) < len(wantPrefix) || resp.ImageKey[:len(wantPrefix)] != wantPrefix {
		t.Fatalf("image key not therapist-scoped: %q", resp.ImageKey)
	}
}

func TestNotesHandler_UserTerms_AcceptAndRead(t *testing.T) {
	user := notesTestUser()
	r := newNotesTestRouter(t, &fakeTherapistLookup{therapist: nil}, newFakeConsentStore(), user)

	// Initially unaccepted.
	w := doNotesRequest(t, r, http.MethodGet, "/me/terms", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
	var state struct {
		ToSAcceptedAt  *time.Time `json:"tos_accepted_at"`
		CurrentVersion string     `json:"current_version"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &state)
	if state.ToSAcceptedAt != nil {
		t.Fatal("expected no acceptance yet")
	}
	if state.CurrentVersion != models.CurrentToSVersion {
		t.Fatalf("current version missing: %+v", state)
	}

	// Accept (works for non-therapists too).
	w = doNotesRequest(t, r, http.MethodPost, "/me/accept-terms", map[string]string{"version": "1.0"})
	if w.Code != http.StatusOK {
		t.Fatalf("accept: want 200, got %d: %s", w.Code, w.Body.String())
	}

	// Now recorded.
	w = doNotesRequest(t, r, http.MethodGet, "/me/terms", nil)
	_ = json.Unmarshal(w.Body.Bytes(), &state)
	if state.ToSAcceptedAt == nil {
		t.Fatal("acceptance not recorded")
	}
}
