package handlers

import (
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
	"github.com/dreamlog/backend/internal/services"
	pkgcrypto "github.com/dreamlog/backend/pkg/crypto"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

var notesHandlerExpansionFailure = errors.New("private dependency detail")

type notesHandlerExpansionLookup struct {
	therapist *models.Therapist
	fail      bool
}

func (l *notesHandlerExpansionLookup) GetByUserID(context.Context, uuid.UUID) (*models.Therapist, error) {
	if l.fail {
		return nil, notesHandlerExpansionFailure
	}
	return l.therapist, nil
}

type notesHandlerExpansionConsent struct {
	*fakeConsentStore
	fail        string
	version     string
	therapistID uuid.UUID
}

func (s *notesHandlerExpansionConsent) TherapistClientConsentAccepted(ctx context.Context, id uuid.UUID) (bool, error) {
	if s.fail == "lookup" {
		return false, notesHandlerExpansionFailure
	}
	return s.fakeConsentStore.TherapistClientConsentAccepted(ctx, id)
}
func (s *notesHandlerExpansionConsent) AcceptTherapistClientConsent(ctx context.Context, id uuid.UUID, version string) error {
	if s.fail == "accept-client" {
		return notesHandlerExpansionFailure
	}
	s.therapistID, s.version = id, version
	return s.fakeConsentStore.AcceptTherapistClientConsent(ctx, id, version)
}
func (s *notesHandlerExpansionConsent) AcceptUserTerms(ctx context.Context, id uuid.UUID, version string) error {
	if s.fail == "accept-user" {
		return notesHandlerExpansionFailure
	}
	return s.fakeConsentStore.AcceptUserTerms(ctx, id, version)
}
func (s *notesHandlerExpansionConsent) GetUserTerms(ctx context.Context, id uuid.UUID) (*time.Time, *string, error) {
	if s.fail == "terms" {
		return nil, nil, notesHandlerExpansionFailure
	}
	return s.fakeConsentStore.GetUserTerms(ctx, id)
}

type notesHandlerExpansionRepo struct {
	*hNotesRepo
	sessions        map[uuid.UUID]*models.ClientSessionRow
	fail            string
	includeArchived bool
}

func (r *notesHandlerExpansionRepo) ListExternalClients(ctx context.Context, id uuid.UUID, archived bool) ([]*models.ExternalClientRow, error) {
	r.includeArchived = archived
	if r.fail == "clients" {
		return nil, notesHandlerExpansionFailure
	}
	return r.hNotesRepo.ListExternalClients(ctx, id, archived)
}
func (r *notesHandlerExpansionRepo) UpdateExternalClient(_ context.Context, id, client uuid.UUID, name []byte, role *string, archived *bool) (bool, error) {
	row := r.clients[client]
	if row == nil || row.TherapistID != id {
		return false, nil
	}
	if name != nil {
		row.NameEnc = name
	}
	if role != nil {
		row.Role = *role
	}
	if archived != nil {
		row.Archived = *archived
	}
	return true, nil
}
func (r *notesHandlerExpansionRepo) DeleteExternalClient(_ context.Context, id, client uuid.UUID) (bool, error) {
	row := r.clients[client]
	if row == nil || row.TherapistID != id {
		return false, nil
	}
	delete(r.clients, client)
	return true, nil
}
func (r *notesHandlerExpansionRepo) CreateSession(_ context.Context, row *models.ClientSessionRow) (*models.ClientSessionRow, error) {
	if row.ID == uuid.Nil {
		row.ID = uuid.New()
	}
	r.sessions[row.ID] = row
	return row, nil
}
func (r *notesHandlerExpansionRepo) GetSession(_ context.Context, id, session uuid.UUID) (*models.ClientSessionRow, error) {
	row := r.sessions[session]
	if row == nil || row.TherapistID != id {
		return nil, nil
	}
	return row, nil
}
func (r *notesHandlerExpansionRepo) ListRecentSessions(_ context.Context, id uuid.UUID, _ int) ([]*models.ClientSessionRow, error) {
	if r.fail == "sessions" {
		return nil, notesHandlerExpansionFailure
	}
	var out []*models.ClientSessionRow
	for _, row := range r.sessions {
		if row.TherapistID == id {
			out = append(out, row)
		}
	}
	return out, nil
}
func (r *notesHandlerExpansionRepo) ListSessionsForExternalClient(ctx context.Context, id, client uuid.UUID, limit int) ([]*models.ClientSessionRow, error) {
	rows, err := r.ListRecentSessions(ctx, id, limit)
	var out []*models.ClientSessionRow
	for _, row := range rows {
		if row.ExternalClientID != nil && *row.ExternalClientID == client {
			out = append(out, row)
		}
	}
	return out, err
}
func (r *notesHandlerExpansionRepo) ListSessionsForLinkedClient(ctx context.Context, id, client uuid.UUID, limit int) ([]*models.ClientSessionRow, error) {
	rows, err := r.ListRecentSessions(ctx, id, limit)
	var out []*models.ClientSessionRow
	for _, row := range rows {
		if row.LinkedUserID != nil && *row.LinkedUserID == client {
			out = append(out, row)
		}
	}
	return out, err
}
func (r *notesHandlerExpansionRepo) UpdateSessionBullets(ctx context.Context, id, session uuid.UUID, b []byte) (bool, error) {
	row, _ := r.GetSession(ctx, id, session)
	if row == nil {
		return false, nil
	}
	row.BulletsEnc = b
	return true, nil
}
func (r *notesHandlerExpansionRepo) UpdateSessionSummary(ctx context.Context, id, session uuid.UUID, b []byte) (bool, error) {
	row, _ := r.GetSession(ctx, id, session)
	if row == nil {
		return false, nil
	}
	row.SummaryEnc = b
	return true, nil
}
func (r *notesHandlerExpansionRepo) DeleteSession(ctx context.Context, id, session uuid.UUID) (*string, bool, error) {
	row, _ := r.GetSession(ctx, id, session)
	if row == nil {
		return nil, false, nil
	}
	delete(r.sessions, session)
	return row.ImageKey, true, nil
}
func (r *notesHandlerExpansionRepo) Overview(context.Context, uuid.UUID) (*models.TherapistOverview, error) {
	if r.fail == "overview" {
		return nil, notesHandlerExpansionFailure
	}
	return &models.TherapistOverview{TotalSessions: len(r.sessions)}, nil
}

func newNotesHandlerExpansion(t *testing.T) (*gin.Engine, *notesHandlerExpansionRepo, *notesHandlerExpansionLookup, *notesHandlerExpansionConsent) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	user := notesTestUser()
	lookup := &notesHandlerExpansionLookup{therapist: notesTestTherapist(user.ID)}
	consent := &notesHandlerExpansionConsent{fakeConsentStore: newFakeConsentStore()}
	consent.therapistConsent[lookup.therapist.ID] = true
	repo := &notesHandlerExpansionRepo{hNotesRepo: newHNotesRepo(), sessions: map[uuid.UUID]*models.ClientSessionRow{}}
	master, err := pkgcrypto.NewDEK()
	if err != nil {
		t.Fatal(err)
	}
	svc := services.NewTherapistNotesService(repo, hLinkChecker{}, hNotesQueue{}, hNotesStorage{}, hSummarizer{}, services.NewNotesCipher(master, repo))
	h := NewTherapistNotesHandler(svc, lookup, consent)
	r := gin.New()
	log := zap.NewNop()
	r.Use(middleware.ErrorHandler(log), middleware.AuthMiddleware(notesTestSecret, "", &fakeProvisioner{user: user}, log))
	r.GET("/therapists/me", h.GetMe)
	r.POST("/therapists/consent", h.AcceptClientConsent)
	r.GET("/therapists/overview", h.Overview)
	r.POST("/me/accept-terms", h.AcceptUserTerms)
	r.GET("/me/terms", h.GetUserTerms)
	r.POST("/therapists/external-clients", h.CreateExternalClient)
	r.GET("/therapists/external-clients", h.ListExternalClients)
	r.GET("/therapists/external-clients/:id", h.GetExternalClient)
	r.PATCH("/therapists/external-clients/:id", h.UpdateExternalClient)
	r.DELETE("/therapists/external-clients/:id", h.DeleteExternalClient)
	r.POST("/therapists/sessions/presign", h.PresignNote)
	r.POST("/therapists/sessions", h.CreateSession)
	r.GET("/therapists/sessions", h.ListSessions)
	r.GET("/therapists/sessions/:id", h.GetSession)
	r.PATCH("/therapists/sessions/:id", h.UpdateSessionBullets)
	r.DELETE("/therapists/sessions/:id", h.DeleteSession)
	r.POST("/therapists/sessions/:id/summarize", h.SummarizeSession)
	return r, repo, lookup, consent
}

func notesExpansionStatus(t *testing.T, w *httptest.ResponseRecorder, want int) {
	t.Helper()
	if w.Code != want {
		t.Fatalf("want HTTP %d, got %d: %s", want, w.Code, w.Body.String())
	}
}

func TestNotesExpansionHandlerLifecycle(t *testing.T) {
	r, repo, lookup, _ := newNotesHandlerExpansion(t)
	w := doNotesRequest(t, r, "POST", "/therapists/external-clients", map[string]string{"name": " Private Person "})
	notesExpansionStatus(t, w, 201)
	var client models.ExternalClient
	if err := json.Unmarshal(w.Body.Bytes(), &client); err != nil {
		t.Fatal(err)
	}
	if client.Name != "Private Person" || client.TherapistID != lookup.therapist.ID {
		t.Fatalf("client: %+v", client)
	}
	clientPath := "/therapists/external-clients/" + client.ID.String()
	w = doNotesRequest(t, r, "GET", clientPath, nil)
	notesExpansionStatus(t, w, 200)
	w = doNotesRequest(t, r, "PATCH", clientPath, map[string]any{"name": "Renamed", "archived": true})
	notesExpansionStatus(t, w, 200)
	if !strings.Contains(w.Body.String(), `"name":"Renamed"`) || !strings.Contains(w.Body.String(), `"archived":true`) {
		t.Fatal(w.Body.String())
	}
	w = doNotesRequest(t, r, "GET", "/therapists/external-clients?include_archived=true", nil)
	notesExpansionStatus(t, w, 200)
	if !repo.includeArchived {
		t.Fatal("include_archived not forwarded")
	}
	w = doNotesRequest(t, r, "POST", "/therapists/sessions", map[string]any{"external_client_id": client.ID, "session_date": "2026-10-14", "bullets": []string{" first ", "second"}})
	notesExpansionStatus(t, w, 201)
	var session models.ClientSession
	if err := json.Unmarshal(w.Body.Bytes(), &session); err != nil {
		t.Fatal(err)
	}
	if session.Status != models.ClientSessionCompleted || session.Bullets[0] != "first" || session.SessionDate != "2026-10-14" {
		t.Fatalf("session: %+v", session)
	}
	path := "/therapists/sessions/" + session.ID.String()
	w = doNotesRequest(t, r, "GET", path, nil)
	notesExpansionStatus(t, w, 200)
	w = doNotesRequest(t, r, "PATCH", path, map[string]any{"bullets": []string{"revised"}})
	notesExpansionStatus(t, w, 200)
	if !strings.Contains(w.Body.String(), `"bullets":["revised"]`) {
		t.Fatal(w.Body.String())
	}
	w = doNotesRequest(t, r, "POST", path+"/summarize", nil)
	notesExpansionStatus(t, w, 200)
	if !strings.Contains(w.Body.String(), `"summary":"summary"`) {
		t.Fatal(w.Body.String())
	}
	for _, query := range []string{"", "?external_client_id=" + client.ID.String(), "?linked_client_id=" + uuid.NewString()} {
		w = doNotesRequest(t, r, "GET", "/therapists/sessions"+query, nil)
		notesExpansionStatus(t, w, 200)
		var body struct {
			Sessions []models.ClientSession `json:"sessions"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		want := 1
		if strings.Contains(query, "linked_client_id") {
			want = 0
		}
		if body.Sessions == nil || len(body.Sessions) != want {
			t.Fatalf("list %q: %s", query, w.Body.String())
		}
	}
	w = doNotesRequest(t, r, "GET", "/therapists/overview", nil)
	notesExpansionStatus(t, w, 200)
	if !strings.Contains(w.Body.String(), `"total_sessions":1`) {
		t.Fatal(w.Body.String())
	}
	// Switch the authenticated user's resolved profile to verify ownership is passed through.
	owner := lookup.therapist
	lookup.therapist = notesTestTherapist(owner.UserID)
	for _, tc := range []struct {
		method, path string
		body         any
	}{
		{"GET", clientPath, nil}, {"PATCH", clientPath, map[string]string{"role": "other"}}, {"DELETE", clientPath, nil},
		{"GET", path, nil}, {"PATCH", path, map[string]any{"bullets": []string{"intruder"}}}, {"POST", path + "/summarize", nil}, {"DELETE", path, nil},
	} {
		notesExpansionStatus(t, doNotesRequest(t, r, tc.method, tc.path, tc.body), 404)
	}
	lookup.therapist = owner
	notesExpansionStatus(t, doNotesRequest(t, r, "DELETE", path, nil), 204)
	notesExpansionStatus(t, doNotesRequest(t, r, "GET", path, nil), 404)
	notesExpansionStatus(t, doNotesRequest(t, r, "DELETE", clientPath, nil), 204)
}

func TestNotesExpansionHandlerValidation(t *testing.T) {
	r, _, _, _ := newNotesHandlerExpansion(t)
	id := uuid.NewString()
	for _, tc := range []struct {
		name, method, path string
		body               any
	}{
		{"create-client-body", "POST", "/therapists/external-clients", map[string]int{"name": 1}},
		{"create-client-service", "POST", "/therapists/external-clients", map[string]string{"name": " "}},
		{"get-client-id", "GET", "/therapists/external-clients/bad", nil},
		{"patch-client-id", "PATCH", "/therapists/external-clients/bad", nil},
		{"delete-client-id", "DELETE", "/therapists/external-clients/bad", nil},
		{"patch-client-body", "PATCH", "/therapists/external-clients/" + id, map[string]int{"name": 2}},
		{"patch-client-empty", "PATCH", "/therapists/external-clients/" + id, map[string]string{}},
		{"presign-body", "POST", "/therapists/sessions/presign", map[string]string{}},
		{"session-body", "POST", "/therapists/sessions", map[string]string{"external_client_id": "bad"}},
		{"session-service", "POST", "/therapists/sessions", map[string]string{}},
		{"list-external-id", "GET", "/therapists/sessions?external_client_id=bad", nil},
		{"list-linked-id", "GET", "/therapists/sessions?linked_client_id=bad", nil},
		{"get-session-id", "GET", "/therapists/sessions/bad", nil},
		{"patch-session-id", "PATCH", "/therapists/sessions/bad", nil},
		{"delete-session-id", "DELETE", "/therapists/sessions/bad", nil},
		{"summarize-session-id", "POST", "/therapists/sessions/bad/summarize", nil},
		{"patch-session-body", "PATCH", "/therapists/sessions/" + id, map[string]string{"bullets": "not-array"}},
		{"patch-session-service", "PATCH", "/therapists/sessions/" + id, map[string]any{"bullets": []string{" "}}},
	} {
		t.Run(tc.name, func(t *testing.T) { notesExpansionStatus(t, doNotesRequest(t, r, tc.method, tc.path, tc.body), 400) })
	}
}

func TestNotesExpansionHandlerEveryRouteRequiresTherapist(t *testing.T) {
	id := uuid.NewString()
	for _, tc := range []struct{ method, path string }{
		{"GET", "/therapists/overview"}, {"POST", "/therapists/consent"},
		{"GET", "/therapists/external-clients"}, {"POST", "/therapists/external-clients"},
		{"GET", "/therapists/external-clients/" + id}, {"PATCH", "/therapists/external-clients/" + id}, {"DELETE", "/therapists/external-clients/" + id},
		{"POST", "/therapists/sessions/presign"}, {"POST", "/therapists/sessions"}, {"GET", "/therapists/sessions"},
		{"GET", "/therapists/sessions/" + id}, {"PATCH", "/therapists/sessions/" + id}, {"DELETE", "/therapists/sessions/" + id}, {"POST", "/therapists/sessions/" + id + "/summarize"},
	} {
		t.Run(tc.method+tc.path, func(t *testing.T) {
			r, _, lookup, _ := newNotesHandlerExpansion(t)
			lookup.therapist = nil
			notesExpansionStatus(t, doNotesRequest(t, r, tc.method, tc.path, nil), 403)
			lookup.fail = true
			notesExpansionStatus(t, doNotesRequest(t, r, tc.method, tc.path, nil), 500)
		})
	}
}

func TestNotesExpansionHandlerDependencyFailures(t *testing.T) {
	for _, tc := range []struct {
		name, method, path, repoFail, consentFail string
		lookupFail                                bool
	}{
		{"profile-lookup", "GET", "/therapists/me", "", "", true},
		{"profile-consent", "GET", "/therapists/me", "", "lookup", false},
		{"create-consent", "POST", "/therapists/external-clients", "", "lookup", false},
		{"presign-consent", "POST", "/therapists/sessions/presign", "", "lookup", false},
		{"session-consent", "POST", "/therapists/sessions", "", "lookup", false},
		{"accept-consent", "POST", "/therapists/consent", "", "accept-client", false},
		{"accept-terms", "POST", "/me/accept-terms", "", "accept-user", false},
		{"get-terms", "GET", "/me/terms", "", "terms", false},
		{"overview", "GET", "/therapists/overview", "overview", "", false},
		{"clients", "GET", "/therapists/external-clients", "clients", "", false},
		{"sessions", "GET", "/therapists/sessions", "sessions", "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r, repo, lookup, consent := newNotesHandlerExpansion(t)
			repo.fail = tc.repoFail
			consent.fail = tc.consentFail
			lookup.fail = tc.lookupFail
			w := doNotesRequest(t, r, tc.method, tc.path, nil)
			notesExpansionStatus(t, w, 500)
			if strings.Contains(w.Body.String(), notesHandlerExpansionFailure.Error()) {
				t.Fatal("private dependency error leaked")
			}
		})
	}
}

func TestNotesExpansionHandlerConsentAndTermsVersions(t *testing.T) {
	r, _, lookup, consent := newNotesHandlerExpansion(t)
	consent.therapistConsent[lookup.therapist.ID] = false
	notesExpansionStatus(t, doNotesRequest(t, r, "POST", "/therapists/sessions", nil), 403)
	for _, body := range []any{nil, map[string]string{"version": "custom-version"}} {
		w := doNotesRequest(t, r, "POST", "/therapists/consent", body)
		notesExpansionStatus(t, w, 200)
		want := models.CurrentToSVersion
		if body != nil {
			want = "custom-version"
		}
		if consent.version != want || consent.therapistID != lookup.therapist.ID {
			t.Fatalf("consent version=%q id=%s", consent.version, consent.therapistID)
		}
		w = doNotesRequest(t, r, "POST", "/me/accept-terms", body)
		notesExpansionStatus(t, w, 200)
		if !strings.Contains(w.Body.String(), `"version":"`+want+`"`) {
			t.Fatal(w.Body.String())
		}
	}
	// Malformed JSON follows the existing default-version contract for optional consent bodies.
	req := httptest.NewRequest(http.MethodPost, "/me/accept-terms", strings.NewReader("{"))
	req.Header.Set("Authorization", "Bearer "+notesTestJWT(t))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	notesExpansionStatus(t, w, 200)
	if !strings.Contains(w.Body.String(), `"version":"`+models.CurrentToSVersion+`"`) {
		t.Fatal(w.Body.String())
	}
}
