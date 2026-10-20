package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/dreamlog/backend/internal/models"
	"github.com/google/uuid"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTextEntryHandlerValidationAndAuth(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		auth       bool
		code       int
	}{
		{"no auth", `{}`, false, 401},
		{"blank", `{"content":"   ","request_id":"` + uuid.NewString() + `"}`, true, 400},
		{"bad id", `{"content":"journal","request_id":"bad"}`, true, 400},
		{"missing token", `{"content":"journal"}`, true, 400},
		{"bad mode", `{"content":"journal","request_id":"` + uuid.NewString() + `","mode":"bad"}`, true, 400},
		{"valid", `{"content":"journal","request_id":"` + uuid.NewString() + `"}`, true, 201},
	} {
		t.Run(tc.name, func(t *testing.T) {
			svc := &fakeEntryServicer{createResp: &models.Entry{ID: uuid.New(), Status: models.EntryStatusPending}}
			r := newEntryTestRouter(t, svc, &fakeStorageUploader{}, entryTestUser())
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/entries/text", bytes.NewBufferString(tc.body))
			req.Header.Set("Content-Type", "application/json")
			if tc.auth {
				req.Header.Set("Authorization", "Bearer "+entryTestJWT(t))
			}
			r.ServeHTTP(w, req)
			if w.Code != tc.code {
				t.Fatalf("status=%d body=%s", w.Code, w.Body)
			}
		})
	}
}

func TestTextEntryHandlerReturnsOriginalOnRetryAndRejectsChangedContent(t *testing.T) {
	content := "journal"
	user := entryTestUser()
	token := uuid.NewString()
	existing := &models.Entry{ID: models.TextEntryID(user.ID, token), UserID: user.ID, Transcript: &content, Mode: models.EntryModeProcessing, Status: models.EntryStatusCompleted}
	for _, text := range []string{content, "changed"} {
		svc := &fakeEntryServicer{getResp: existing}
		r := newEntryTestRouter(t, svc, &fakeStorageUploader{}, user)
		body, _ := json.Marshal(models.CreateTextEntryInput{Content: text, RequestID: token})
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/entries/text", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+entryTestJWT(t))
		r.ServeHTTP(w, req)
		want := 201
		if text != content {
			want = 409
		}
		if w.Code != want {
			t.Fatalf("status=%d body=%s", w.Code, w.Body)
		}
	}
}

func TestTextEntryQuotaBlocksNewEntryButNotRetrievalOfSavedRetry(t *testing.T) {
	content := "complete journal"
	user, token := entryTestUser(), uuid.NewString()
	for _, replay := range []bool{false, true} {
		quota := &fakeEntryQuota{err: errors.New("entry quota reached")}
		svc := &fakeEntryServicer{}
		if replay {
			svc.getResp = &models.Entry{ID: models.TextEntryID(user.ID, token), UserID: user.ID, Transcript: &content, Mode: models.EntryModeProcessing}
		}
		r := newEntryTestRouter(t, svc, &fakeStorageUploader{}, user, quota)
		body, _ := json.Marshal(models.CreateTextEntryInput{Content: content, RequestID: token})
		request := httptest.NewRequest(http.MethodPost, "/entries/text", bytes.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Authorization", "Bearer "+entryTestJWT(t))
		response := httptest.NewRecorder()
		r.ServeHTTP(response, request)
		want := 409
		if replay {
			want = 201
		}
		if response.Code != want || svc.textCalls != 0 {
			t.Fatalf("status=%d, create calls=%d", response.Code, svc.textCalls)
		}
		if replay && quota.calls != 0 {
			t.Fatal("retry consumed quota")
		}
	}
}
