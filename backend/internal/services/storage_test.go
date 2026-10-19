package services

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
	"time"

	appconfig "github.com/dreamlog/backend/internal/config"
	pkgstorage "github.com/dreamlog/backend/pkg/storage"
	"github.com/google/uuid"
)

func expansionStorageService(t *testing.T, endpoint string) (*StorageService, *appconfig.StorageConfig) {
	t.Helper()
	t.Setenv("AWS_CONFIG_FILE", filepath.Join(t.TempDir(), "absent-config"))
	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", filepath.Join(t.TempDir(), "absent-credentials"))
	t.Setenv("AWS_EC2_METADATA_DISABLED", "true")
	t.Setenv("AWS_MAX_ATTEMPTS", "1")
	cfg := &appconfig.StorageConfig{Endpoint: endpoint, Bucket: "test-bucket", Region: "us-east-1", AccessKeyID: "synthetic-key", SecretAccessKey: "synthetic-secret", UsePathStyle: true, PresignExpiry: 12500 * time.Millisecond}
	client, err := pkgstorage.New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	return NewStorageService(client, cfg), cfg
}

func TestStorageExpansionSanitizeFilename(t *testing.T) {
	for _, tc := range []struct{ input, want string }{
		{"clip.aac", "clip.aac"}, {"../../private/clip.aac", "clip.aac"},
		{"...clip.aac", "clip.aac"}, {"a b?\x00.aac", "a_b__.aac"},
		{`C:\private\clip.aac`, `C__private_clip.aac`},
		{"/", "_"}, {"voice-01_test.M4A", "voice-01_test.M4A"},
		{strings.Repeat("a", 140) + ".aac", strings.Repeat("a", 124) + ".aac"},
	} {
		t.Run(tc.input, func(t *testing.T) {
			if got := sanitizeFilename(tc.input); got != tc.want {
				t.Fatalf("sanitize = %q, want %q", got, tc.want)
			}
		})
	}
	seen := map[string]bool{}
	for _, input := range []string{"", ".", "..", "...", "/dir/.."} {
		got := sanitizeFilename(input)
		if !strings.HasSuffix(got, ".aac") {
			t.Fatalf("missing fallback extension: %q", got)
		}
		if _, err := uuid.Parse(strings.TrimSuffix(got, ".aac")); err != nil {
			t.Fatalf("fallback = %q: %v", got, err)
		}
		if seen[got] {
			t.Fatalf("fallback reused: %q", got)
		}
		seen[got] = true
	}
}

func expansionSignedURL(t *testing.T, raw, key, expiry string) {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if u.Path != "/test-bucket/"+key || u.Query().Get("X-Amz-Expires") != expiry || u.Query().Get("X-Amz-Signature") == "" {
		t.Fatalf("unexpected signed URL %s", raw)
	}
}

func TestStorageExpansionServicePresignKeys(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { t.Error("presigning contacted server") }))
	defer server.Close()
	svc, cfg := expansionStorageService(t, server.URL)
	user := uuid.MustParse("11111111-1111-4111-8111-111111111111")
	entry := uuid.MustParse("22222222-2222-4222-8222-222222222222")
	if got := AudioKey(user, entry); got != "audio/"+user.String()+"/"+entry.String()+".aac" {
		t.Fatalf("audio key = %q", got)
	}
	if svc.PresignExpirySec() != 12 {
		t.Fatalf("expiry = %d", svc.PresignExpirySec())
	}
	ctx := context.Background()
	priorKey := ""
	for i := 0; i < 2; i++ {
		raw, key, err := svc.PresignUpload(ctx, user)
		if err != nil {
			t.Fatal(err)
		}
		prefix := "audio/" + user.String() + "/"
		if !strings.HasPrefix(key, prefix) || !strings.HasSuffix(key, ".aac") || key == priorKey {
			t.Fatalf("bad audio key %q", key)
		}
		if _, err := uuid.Parse(strings.TrimSuffix(strings.TrimPrefix(key, prefix), ".aac")); err != nil {
			t.Fatal(err)
		}
		expansionSignedURL(t, raw, key, "12")
		priorKey = key
	}
	raw, key, err := svc.PresignPut(ctx, "../../a voice.aac", "audio/custom", 93*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.Split(key, "/")
	if len(parts) != 3 || parts[0] != "therapy" || parts[2] != "a_voice.aac" {
		t.Fatalf("therapy key %q", key)
	}
	if _, err := uuid.Parse(parts[1]); err != nil {
		t.Fatal(err)
	}
	// Characterize existing behavior: per-call expiry/content type are ignored.
	expansionSignedURL(t, raw, key, "12")
	u, _ := url.Parse(raw)
	if strings.Contains(u.Query().Get("X-Amz-SignedHeaders"), "content-type") {
		t.Fatal("unexpected content-type binding")
	}
	exact := "therapist-notes/id/photo +1.jpg"
	raw, err = svc.PresignPutKey(ctx, exact)
	if err != nil {
		t.Fatal(err)
	}
	expansionSignedURL(t, raw, exact, "12")
	raw, err = svc.PresignDownload(ctx, exact, 83*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	expansionSignedURL(t, raw, exact, "83")

	cfg.ProxyBaseURL = server.URL + "/proxy"
	// A nil client proves all proxy paths avoid SDK access.
	svc = NewStorageService(nil, cfg)
	for _, run := range []func() (string, string, error){
		func() (string, string, error) { return svc.PresignUpload(ctx, user) },
		func() (string, string, error) { return svc.PresignPut(ctx, "../clip.aac", "audio/aac", time.Minute) },
		func() (string, string, error) { raw, err := svc.PresignPutKey(ctx, exact); return raw, exact, err },
	} {
		raw, key, err := run()
		if err != nil {
			t.Fatal(err)
		}
		u, err := url.Parse(raw)
		if err != nil {
			t.Fatal(err)
		}
		if u.Path != "/proxy/upload" || u.Query().Get("key") != key || len(u.Query()) != 1 {
			t.Fatalf("proxy URL %q for %q", raw, key)
		}
	}
}

func TestStorageExpansionServiceIO(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/test-bucket/audio/fixture.aac" {
			t.Errorf("path = %s", r.URL.Path)
		}
		switch r.Method {
		case http.MethodPut:
			data, err := io.ReadAll(r.Body)
			if err != nil || string(data) != "service-audio" || r.Header.Get("Content-Type") != "audio/aac" {
				t.Errorf("upload = %q, %v, %s", data, err, r.Header.Get("Content-Type"))
			}
		case http.MethodGet:
			_, _ = io.WriteString(w, "service-audio")
		case http.MethodHead:
			w.WriteHeader(http.StatusOK)
		case http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("method %s", r.Method)
		}
	}))
	defer server.Close()
	svc, _ := expansionStorageService(t, server.URL)
	ctx := context.Background()
	key := "audio/fixture.aac"
	if err := svc.Upload(ctx, key, strings.NewReader("service-audio")); err != nil {
		t.Fatal(err)
	}
	if ok, err := svc.Exists(ctx, key); err != nil || !ok {
		t.Fatalf("exists = %v, %v", ok, err)
	}
	body, err := svc.GetObject(ctx, key)
	if err != nil {
		t.Fatal(err)
	}
	data, err := io.ReadAll(body)
	closeErr := body.Close()
	if err != nil || closeErr != nil || string(data) != "service-audio" {
		t.Fatalf("get = %q, %v, %v", data, err, closeErr)
	}
	if err := svc.Delete(ctx, key); err != nil {
		t.Fatal(err)
	}
}

func TestStorageExpansionServiceErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusForbidden) }))
	defer server.Close()
	svc, cfg := expansionStorageService(t, server.URL)
	for _, canceled := range []bool{false, true} {
		ctx, cancel := context.WithCancel(context.Background())
		if canceled {
			cancel()
		}
		for _, tc := range []struct {
			name string
			run  func() error
		}{
			{"upload", func() error { return svc.Upload(ctx, "key", strings.NewReader("data")) }},
			{"delete", func() error { return svc.Delete(ctx, "key") }},
			{"get object", func() error {
				body, err := svc.GetObject(ctx, "key")
				if body != nil {
					body.Close()
					t.Error("body returned on error")
				}
				return err
			}},
		} {
			err := tc.run()
			if err == nil || !strings.Contains(err.Error(), "storage: "+tc.name) {
				t.Fatalf("%s = %v", tc.name, err)
			}
			if canceled && !errors.Is(err, context.Canceled) {
				t.Fatalf("lost cancellation: %v", err)
			}
		}
		if ok, err := svc.Exists(ctx, "key"); ok || err != nil {
			t.Fatalf("exists = %v, %v", ok, err)
		}
		cancel()
	}
	// Invalid expiry fails locally during SDK signing, not over the network.
	cfg.PresignExpiry = -time.Second
	client, err := pkgstorage.New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	svc = NewStorageService(client, cfg)
	ctx := context.Background()
	for _, tc := range []struct {
		name string
		run  func() (string, string, error)
	}{
		{"presign upload", func() (string, string, error) { return svc.PresignUpload(ctx, uuid.New()) }},
		{"presign therapy put", func() (string, string, error) { return svc.PresignPut(ctx, "clip.aac", "audio/aac", time.Minute) }},
		{"presign put key", func() (string, string, error) { raw, err := svc.PresignPutKey(ctx, "key"); return raw, "", err }},
		{"presign download", func() (string, string, error) {
			raw, err := svc.PresignDownload(ctx, "key", -time.Second)
			return raw, "", err
		}},
	} {
		raw, key, err := tc.run()
		if raw != "" || key != "" || err == nil || !strings.Contains(err.Error(), "storage: "+tc.name) {
			t.Fatalf("%s = %q, %q, %v", tc.name, raw, key, err)
		}
	}
}
