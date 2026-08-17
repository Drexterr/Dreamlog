package middleware

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestMaxBodySize_RejectsOversizedBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(MaxBodySize(10)) // tiny limit for the test
	r.POST("/x", func(c *gin.Context) {
		_, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.Status(http.StatusRequestEntityTooLarge)
			return
		}
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/x", strings.NewReader(strings.Repeat("a", 1000)))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("oversized body: want 413, got %d", w.Code)
	}
}

func TestMaxBodySize_AllowsBodyWithinLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(MaxBodySize(MaxRequestBodyBytes))
	r.POST("/x", func(c *gin.Context) {
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.Status(http.StatusRequestEntityTooLarge)
			return
		}
		c.String(http.StatusOK, "%d", len(body))
	})

	payload := bytes.Repeat([]byte("a"), 1000)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/x", bytes.NewReader(payload))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("body within limit: want 200, got %d (%s)", w.Code, w.Body.String())
	}
	if w.Body.String() != "1000" {
		t.Errorf("want body length 1000 echoed back, got %s", w.Body.String())
	}
}

func TestMaxBodySize_ExemptsDevUploadProxy(t *testing.T) {
	// The dev-only /upload proxy streams raw audio straight through the API
	// (see router.go) - a 30-minute recording is well over the JSON body
	// cap, so this route must bypass it.
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(MaxBodySize(10)) // tiny limit - would reject anything else
	r.PUT("/upload", func(c *gin.Context) {
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.Status(http.StatusRequestEntityTooLarge)
			return
		}
		c.String(http.StatusOK, "%d", len(body))
	})

	payload := bytes.Repeat([]byte("a"), 5000)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/upload", bytes.NewReader(payload))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("/upload must be exempt from the body cap: want 200, got %d (%s)", w.Code, w.Body.String())
	}
}
