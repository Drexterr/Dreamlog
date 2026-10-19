package storage

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	appconfig "github.com/dreamlog/backend/internal/config"
)

func storageTestConfig(t *testing.T, endpoint string) *appconfig.StorageConfig {
	t.Helper()
	t.Setenv("AWS_CONFIG_FILE", filepath.Join(t.TempDir(), "absent-config"))
	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", filepath.Join(t.TempDir(), "absent-credentials"))
	t.Setenv("AWS_EC2_METADATA_DISABLED", "true")
	t.Setenv("AWS_MAX_ATTEMPTS", "1")
	return &appconfig.StorageConfig{Endpoint: endpoint, Bucket: "test-bucket", Region: "us-east-1", AccessKeyID: "synthetic-key", SecretAccessKey: "synthetic-secret", UsePathStyle: true, PresignExpiry: 7 * time.Minute}
}

func TestStorageExpansionObjectLifecycle(t *testing.T) {
	var calls atomic.Int32
	var mu sync.Mutex
	var stored []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		calls.Add(1)
		if r.URL.Path != "/test-bucket/audio/a clip.aac" || !strings.Contains(r.Header.Get("Authorization"), "Credential=synthetic-key/") {
			t.Errorf("unexpected signed request: %s %s", r.Method, r.URL.Path)
		}
		switch r.Method {
		case http.MethodPut:
			if r.Header.Get("Content-Type") != "audio/aac" || r.ContentLength != 13 {
				t.Errorf("upload metadata: %v, length %d", r.Header, r.ContentLength)
			}
			var err error
			stored, err = io.ReadAll(r.Body)
			if err != nil {
				t.Error(err)
			}
			w.Header().Set("ETag", `"fixture"`)
		case http.MethodGet:
			_, _ = w.Write(stored)
		case http.MethodHead:
			if stored == nil {
				w.WriteHeader(http.StatusNotFound)
			}
		case http.MethodDelete:
			stored = nil
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("unexpected method %s", r.Method)
		}
	}))
	defer server.Close()
	client, err := New(storageTestConfig(t, server.URL))
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	key := "audio/a clip.aac"
	if err := client.Upload(ctx, key, "audio/aac", strings.NewReader("fixture-audio")); err != nil {
		t.Fatal(err)
	}
	if ok, err := client.Exists(ctx, key); err != nil || !ok {
		t.Fatalf("exists = %v, %v", ok, err)
	}
	body, err := client.Download(ctx, key)
	if err != nil {
		t.Fatal(err)
	}
	data, readErr := io.ReadAll(body)
	closeErr := body.Close()
	if readErr != nil || closeErr != nil || string(data) != "fixture-audio" {
		t.Fatalf("download = %q, %v, %v", data, readErr, closeErr)
	}
	for i := 0; i < 2; i++ {
		if err := client.Delete(ctx, key); err != nil {
			t.Fatal(err)
		}
	}
	if ok, err := client.Exists(ctx, key); err != nil || ok {
		t.Fatalf("deleted exists = %v, %v", ok, err)
	}
	if calls.Load() != 6 {
		t.Fatalf("requests = %d", calls.Load())
	}
}

type storageFailReader struct{ err error }

func (r storageFailReader) Read([]byte) (int, error) { return 0, r.err }

func TestStorageExpansionFailuresAndCancellation(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusForbidden)
		if r.Method != http.MethodHead {
			_, _ = io.WriteString(w, `<Error><Code>AccessDenied</Code><Message>local denial</Message></Error>`)
		}
	}))
	defer server.Close()
	client, err := New(storageTestConfig(t, server.URL))
	if err != nil {
		t.Fatal(err)
	}
	sentinel := errors.New("fixture reader failure")
	if err := client.Upload(context.Background(), "key", "audio/aac", storageFailReader{sentinel}); !errors.Is(err, sentinel) || !strings.Contains(err.Error(), "read upload body") {
		t.Fatalf("reader error = %v", err)
	}
	if calls.Load() != 0 {
		t.Fatal("reader failure contacted server")
	}
	for _, canceled := range []bool{false, true} {
		t.Run(map[bool]string{false: "provider", true: "canceled"}[canceled], func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if canceled {
				cancel()
			}
			before := calls.Load()
			for _, op := range []struct {
				name string
				run  func() error
			}{
				{"upload", func() error { return client.Upload(ctx, "key", "audio/aac", strings.NewReader("data")) }},
				{"download", func() error {
					body, err := client.Download(ctx, "key")
					if body != nil {
						body.Close()
						t.Error("body returned on failure")
					}
					return err
				}},
				{"delete", func() error { return client.Delete(ctx, "key") }},
			} {
				err := op.run()
				if err == nil || !strings.Contains(err.Error(), "storage: "+op.name) {
					t.Fatalf("%s error = %v", op.name, err)
				}
				if canceled && !errors.Is(err, context.Canceled) {
					t.Fatalf("lost cancellation: %v", err)
				}
			}
			// Characterize the current API: all HEAD errors are treated as absence.
			if ok, err := client.Exists(ctx, "key"); ok || err != nil {
				t.Fatalf("exists = %v, %v", ok, err)
			}
			if canceled && calls.Load() != before {
				t.Fatal("canceled operation contacted server")
			}
		})
	}
}

func TestStorageExpansionPresign(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { t.Error("presigning must not send HTTP requests") }))
	defer server.Close()
	public := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { t.Error("public presigning must be offline") }))
	defer public.Close()
	for _, separate := range []bool{false, true} {
		cfg := storageTestConfig(t, server.URL)
		if separate {
			cfg.PublicBaseURL = public.URL
		}
		client, err := New(cfg)
		if err != nil {
			t.Fatal(err)
		}
		for _, upload := range []bool{false, true} {
			var raw string
			if upload {
				raw, err = client.PresignUpload(context.Background(), "folder/a +b.aac")
			} else {
				raw, err = client.PresignDownload(context.Background(), "folder/a +b.aac", 91*time.Second)
			}
			if err != nil {
				t.Fatal(err)
			}
			u, err := url.Parse(raw)
			if err != nil {
				t.Fatal(err)
			}
			wantHost := strings.TrimPrefix(server.URL, "http://")
			wantExpiry := "91"
			if upload {
				wantExpiry = "420"
				if separate {
					wantHost = strings.TrimPrefix(public.URL, "http://")
				}
			}
			if u.Host != wantHost || u.Path != "/test-bucket/folder/a +b.aac" || u.Query().Get("X-Amz-Expires") != wantExpiry || u.Query().Get("X-Amz-Signature") == "" || !strings.HasPrefix(u.Query().Get("X-Amz-Credential"), "synthetic-key/") {
				t.Fatalf("bad signed URL: %s", raw)
			}
		}
		client.expiry = -time.Second
		if raw, err := client.PresignUpload(context.Background(), "key"); raw != "" || err == nil || !strings.Contains(err.Error(), "presign upload") {
			t.Fatalf("upload expiry error = %q, %v", raw, err)
		}
		if raw, err := client.PresignDownload(context.Background(), "key", -time.Second); raw != "" || err == nil || !strings.Contains(err.Error(), "presign download") {
			t.Fatalf("download expiry error = %q, %v", raw, err)
		}
	}
}

func TestStorageExpansionPresignCanceledCredentials(t *testing.T) {
	storageTestConfig(t, "http://127.0.0.1:1")
	provider := aws.CredentialsProviderFunc(func(ctx context.Context) (aws.Credentials, error) { return aws.Credentials{}, ctx.Err() })
	sdk := s3.New(s3.Options{Region: "us-east-1", BaseEndpoint: aws.String("http://127.0.0.1:1"), UsePathStyle: true, Credentials: provider})
	client := &Client{pre: s3.NewPresignClient(sdk), prePub: s3.NewPresignClient(sdk), bucket: "test-bucket", expiry: time.Minute}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := client.PresignUpload(ctx, "key"); !errors.Is(err, context.Canceled) {
		t.Fatalf("upload cancellation = %v", err)
	}
	if _, err := client.PresignDownload(ctx, "key", time.Minute); !errors.Is(err, context.Canceled) {
		t.Fatalf("download cancellation = %v", err)
	}
}

func TestStorageExpansionConfigFailure(t *testing.T) {
	cfg := storageTestConfig(t, "http://127.0.0.1:1")
	t.Setenv("AWS_DEFAULTS_MODE", "invalid-fixture-mode")
	client, err := New(cfg)
	if client != nil || err == nil || !strings.Contains(err.Error(), "storage: load aws config:") {
		t.Fatalf("invalid SDK configuration = %v, %v", client, err)
	}
}
