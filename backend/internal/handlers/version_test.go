package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func newVersionTestRouter(minimumVersion, androidURL, iosURL string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewVersionHandler(minimumVersion, androidURL, iosURL)
	r.GET("/version", h.Get)
	return r
}

func TestVersion_ReturnsConfiguredValues(t *testing.T) {
	r := newVersionTestRouter(
		"1.2.0",
		"https://play.google.com/store/apps/details?id=com.dreamlog.app",
		"https://apps.apple.com/app/id1234567890",
	)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/version", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var body struct {
		MinimumVersion  string `json:"minimum_version"`
		AndroidStoreURL string `json:"android_store_url"`
		IOSStoreURL     string `json:"ios_store_url"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON response: %v", err)
	}
	if body.MinimumVersion != "1.2.0" {
		t.Errorf("minimum_version = %q, want %q", body.MinimumVersion, "1.2.0")
	}
	if body.AndroidStoreURL != "https://play.google.com/store/apps/details?id=com.dreamlog.app" {
		t.Errorf("android_store_url = %q", body.AndroidStoreURL)
	}
	if body.IOSStoreURL != "https://apps.apple.com/app/id1234567890" {
		t.Errorf("ios_store_url = %q", body.IOSStoreURL)
	}
}

func TestVersion_NoAuthRequired(t *testing.T) {
	r := newVersionTestRouter("1.0.0", "", "")

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/version", nil) // no Authorization header
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 without auth, got %d", w.Code)
	}
}

func TestVersion_EmptyIOSStoreURL(t *testing.T) {
	r := newVersionTestRouter("1.0.0", "https://play.google.com/store/apps/details?id=com.dreamlog.app", "")

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/version", nil)
	r.ServeHTTP(w, req)

	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON response: %v", err)
	}
	if v, ok := body["ios_store_url"]; !ok || v != "" {
		t.Errorf("ios_store_url should be present and empty, got %q (present=%v)", v, ok)
	}
}
