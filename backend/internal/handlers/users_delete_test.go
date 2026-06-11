package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dreamlog/backend/internal/middleware"
	"github.com/dreamlog/backend/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type fakeUserDeleter struct {
	deleted   []uuid.UUID
	deleteErr error
}

func (f *fakeUserDeleter) UpdateProfile(_ context.Context, _ uuid.UUID, _ models.UpdateUserInput) (*models.User, error) {
	return nil, nil
}

func (f *fakeUserDeleter) Delete(_ context.Context, id uuid.UUID) error {
	if f.deleteErr != nil {
		return f.deleteErr
	}
	f.deleted = append(f.deleted, id)
	return nil
}

func newDeleteTestRouter(t *testing.T, svc userProfiler) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	log := zap.NewNop()
	r := gin.New()
	r.Use(middleware.ErrorHandler(log))
	r.Use(middleware.AuthMiddleware(userTestSecret, "", &fakeProvisioner{user: userTestUser()}, log))
	h := &UserHandler{svc: svc}
	r.DELETE("/me", h.DeleteMe)
	return r
}

func TestDeleteMe_Success_Returns204(t *testing.T) {
	svc := &fakeUserDeleter{}
	r := newDeleteTestRouter(t, svc)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodDelete, "/me", nil)
	req.Header.Set("Authorization", "Bearer "+userTestJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
	}
	if len(svc.deleted) != 1 {
		t.Fatalf("expected exactly one delete call, got %d", len(svc.deleted))
	}
}

func TestDeleteMe_ServiceError_Returns500(t *testing.T) {
	svc := &fakeUserDeleter{deleteErr: errors.New("db down")}
	r := newDeleteTestRouter(t, svc)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodDelete, "/me", nil)
	req.Header.Set("Authorization", "Bearer "+userTestJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestDeleteMe_MissingAuth_Returns401(t *testing.T) {
	r := newDeleteTestRouter(t, &fakeUserDeleter{})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodDelete, "/me", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}
