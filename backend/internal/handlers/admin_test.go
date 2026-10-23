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
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type fakeDLQAdmin struct {
	items     []services.DLQItem
	total     int64
	err       error
	lastQueue string
	lastID    string
	lastPage  int
	lastSize  int
	requeues  int
	discards  int
}

func (f *fakeDLQAdmin) List(_ context.Context, q string, page, size int) ([]services.DLQItem, int64, error) {
	f.lastQueue, f.lastPage, f.lastSize = q, page, size
	return f.items, f.total, f.err
}

func (f *fakeDLQAdmin) Requeue(_ context.Context, q, id string) error {
	f.lastQueue, f.lastID = q, id
	f.requeues++
	return f.err
}

func (f *fakeDLQAdmin) Discard(_ context.Context, q, id string) error {
	f.lastQueue, f.lastID = q, id
	f.discards++
	return f.err
}

const adminTestSecret = "admin-test-jwt-secret-32-bytes!!"

// newAdminTestRouter wires the admin routes exactly as router.go does.
func newAdminTestRouter(dlq dlqAdmin, admins []string, user *models.User) *gin.Engine {
	gin.SetMode(gin.TestMode)
	log := zap.NewNop()
	r := gin.New()
	r.Use(middleware.ErrorHandler(log))
	auth := r.Group("/", middleware.AuthMiddleware(adminTestSecret, "", &fakeProvisioner{user: user}, log))
	h := NewAdminHandler(dlq)
	admin := auth.Group("/admin", middleware.RequireAdmin(admins))
	admin.GET("/dlq/:queue", h.ListDLQ)
	admin.POST("/dlq/:queue/:id/requeue", h.RequeueDLQ)
	admin.DELETE("/dlq/:queue/:id", h.DiscardDLQ)
	return r
}

func adminJWT(t *testing.T) string {
	t.Helper()
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"aud": "authenticated", "sub": "admin-sub", "email": "ops@example.test",
		"exp": time.Now().Add(time.Hour).Unix(),
	})
	s, err := tok.SignedString([]byte(adminTestSecret))
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func adminDo(t *testing.T, r *gin.Engine, method, path string, token bool) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, nil)
	if token {
		req.Header.Set("Authorization", "Bearer "+adminJWT(t))
	}
	r.ServeHTTP(w, req)
	return w
}

func opsUser() *models.User {
	return &models.User{ID: uuid.New(), Email: "ops@example.test", EmailVerified: true}
}

func TestAdminDLQ_AccessControl(t *testing.T) {
	unverified := opsUser()
	unverified.EmailVerified = false
	cases := []struct {
		name   string
		admins []string
		user   *models.User
		token  bool
		want   int
	}{
		{"no token", []string{"ops@example.test"}, opsUser(), false, http.StatusUnauthorized},
		{"admin routes disabled", nil, opsUser(), true, http.StatusNotFound},
		{"not an admin", []string{"boss@example.test"}, opsUser(), true, http.StatusNotFound},
		{"unverified admin email", []string{"ops@example.test"}, unverified, true, http.StatusNotFound},
		{"verified admin", []string{"ops@example.test"}, opsUser(), true, http.StatusOK},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dlq := &fakeDLQAdmin{items: []services.DLQItem{}}
			r := newAdminTestRouter(dlq, tc.admins, tc.user)
			for _, call := range []struct{ method, path string }{
				{http.MethodGet, "/admin/dlq/entries"},
				{http.MethodPost, "/admin/dlq/entries/" + uuid.NewString() + "/requeue"},
				{http.MethodDelete, "/admin/dlq/entries/" + uuid.NewString()},
			} {
				w := adminDo(t, r, call.method, call.path, tc.token)
				want := tc.want
				if want == http.StatusOK && call.method == http.MethodDelete {
					want = http.StatusNoContent
				}
				if w.Code != want {
					t.Fatalf("%s %s: want %d, got %d", call.method, call.path, want, w.Code)
				}
			}
			if tc.want != http.StatusOK && (dlq.requeues != 0 || dlq.discards != 0 || dlq.lastQueue != "") {
				t.Fatal("non-admin reached the service")
			}
		})
	}
}

func TestAdminDLQ_List(t *testing.T) {
	entryID := uuid.New()
	dlq := &fakeDLQAdmin{
		items: []services.DLQItem{{ID: entryID.String(), Queue: "entries", EntryID: &entryID, Error: "boom", Attempt: 2}},
		total: 41,
	}
	r := newAdminTestRouter(dlq, []string{"ops@example.test"}, opsUser())

	w := adminDo(t, r, http.MethodGet, "/admin/dlq/notes?page=2&page_size=20", true)
	if w.Code != http.StatusOK {
		t.Fatalf("%d %s", w.Code, w.Body.String())
	}
	if dlq.lastQueue != "notes" || dlq.lastPage != 2 || dlq.lastSize != 20 {
		t.Fatalf("params: %+v", dlq)
	}
	var resp map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["total"] != float64(41) || resp["has_more"] != true || resp["page"] != float64(2) {
		t.Fatalf("resp: %v", resp)
	}
	items := resp["items"].([]any)
	if items[0].(map[string]any)["entry_id"] != entryID.String() {
		t.Fatalf("items: %v", items)
	}

	w = adminDo(t, r, http.MethodGet, "/admin/dlq/entries", true)
	if dlq.lastPage != 1 || dlq.lastSize != 20 {
		t.Fatalf("defaults: page=%d size=%d", dlq.lastPage, dlq.lastSize)
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["has_more"] != true {
		t.Fatal("has_more on page 1 of 41")
	}

	for _, q := range []string{"page=0", "page=x", "page_size=0", "page_size=101", "page_size=x"} {
		if w := adminDo(t, r, http.MethodGet, "/admin/dlq/entries?"+q, true); w.Code != http.StatusBadRequest {
			t.Fatalf("%s: want 400, got %d", q, w.Code)
		}
	}
}

func TestAdminDLQ_RequeueAndDiscard_CanonicalIDs(t *testing.T) {
	dlq := &fakeDLQAdmin{}
	r := newAdminTestRouter(dlq, []string{"ops@example.test"}, opsUser())
	id := uuid.New()

	w := adminDo(t, r, http.MethodPost, "/admin/dlq/entries/"+strings.ToUpper(id.String())+"/requeue", true)
	if w.Code != http.StatusOK || dlq.lastID != id.String() || dlq.lastQueue != "entries" {
		t.Fatalf("requeue: %d id=%s", w.Code, dlq.lastID)
	}
	w = adminDo(t, r, http.MethodDelete, "/admin/dlq/notes/raw-0123456789abcdef", true)
	if w.Code != http.StatusNoContent || dlq.lastID != "raw-0123456789abcdef" || dlq.lastQueue != "notes" {
		t.Fatalf("discard: %d id=%s", w.Code, dlq.lastID)
	}
}

func TestAdminDLQ_ErrorMapping(t *testing.T) {
	cases := []struct {
		err  error
		want int
	}{
		{services.ErrDLQUnknownQueue, http.StatusBadRequest},
		{services.ErrDLQItemNotFound, http.StatusNotFound},
		{services.ErrDLQUnparseable, http.StatusConflict},
		{services.ErrDLQTargetGone, http.StatusConflict},
		{services.ErrDLQCompleted, http.StatusConflict},
		{services.ErrDLQNotFailed, http.StatusConflict},
		{services.ErrDLQSourceGone, http.StatusConflict},
		{services.ErrDLQClaimed, http.StatusConflict},
		{errors.New("redis down"), http.StatusInternalServerError},
	}
	for _, tc := range cases {
		t.Run(tc.err.Error(), func(t *testing.T) {
			r := newAdminTestRouter(&fakeDLQAdmin{err: tc.err}, []string{"ops@example.test"}, opsUser())
			for _, call := range []struct{ method, path string }{
				{http.MethodGet, "/admin/dlq/entries"},
				{http.MethodPost, "/admin/dlq/entries/" + uuid.NewString() + "/requeue"},
				{http.MethodDelete, "/admin/dlq/entries/" + uuid.NewString()},
			} {
				w := adminDo(t, r, call.method, call.path, true)
				if w.Code != tc.want {
					t.Fatalf("%s: want %d, got %d", call.method, tc.want, w.Code)
				}
				if tc.want == http.StatusInternalServerError && strings.Contains(w.Body.String(), "redis down") {
					t.Fatal("internal error detail leaked to the client")
				}
				if tc.want == http.StatusConflict && !strings.Contains(w.Body.String(), "discard") && tc.err != services.ErrDLQNotFailed && tc.err != services.ErrDLQClaimed {
					t.Fatalf("409 must explain what to do: %s", w.Body.String())
				}
			}
		})
	}
}

func TestRouter_AdminRoutesOnlyWithService(t *testing.T) {
	without := NewRouter(Deps{JWTSecret: "local-router-unit-only", Log: zap.NewNop()}).(*gin.Engine)
	for _, rt := range without.Routes() {
		if strings.HasPrefix(rt.Path, "/admin") {
			t.Fatalf("admin route registered without a DLQ service: %s", rt.Path)
		}
	}
	with := NewRouter(Deps{JWTSecret: "local-router-unit-only", Log: zap.NewNop(), DLQAdminSvc: &services.DLQAdminService{}}).(*gin.Engine)
	found := map[string]bool{}
	for _, rt := range with.Routes() {
		found[rt.Method+" "+rt.Path] = true
	}
	for _, want := range []string{"GET /admin/dlq/:queue", "POST /admin/dlq/:queue/:id/requeue", "DELETE /admin/dlq/:queue/:id"} {
		if !found[want] {
			t.Fatalf("missing route %s", want)
		}
	}
	// Unauthenticated callers are stopped by auth before the admin check.
	w := httptest.NewRecorder()
	with.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/admin/dlq/entries", nil))
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", w.Code)
	}
}
