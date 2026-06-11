package services

import (
	"context"
	"strings"
	"testing"

	appconfig "github.com/dreamlog/backend/internal/config"
)

func TestFCM_SendToToken_SkipsWhenNotConfigured(t *testing.T) {
	svc := NewFCMService(&appconfig.FCMConfig{})
	if err := svc.SendToToken(context.Background(), "tok", "title", "body", nil); err != nil {
		t.Fatalf("expected silent skip with no credentials, got error: %v", err)
	}
}

func TestFCM_GetAccessToken_InvalidCredentials(t *testing.T) {
	svc := NewFCMService(&appconfig.FCMConfig{
		ProjectID:       "test-project",
		CredentialsJSON: "not-json",
	})
	_, err := svc.getAccessToken(context.Background())
	if err == nil {
		t.Fatal("expected error for malformed credentials JSON")
	}
	if !strings.Contains(err.Error(), "parse credentials") {
		t.Fatalf("expected parse credentials error, got: %v", err)
	}
}

func TestFCM_GetAccessToken_ErrorIsSticky(t *testing.T) {
	// The TokenSource is built once; a credentials parse failure must be
	// returned consistently on subsequent calls, not panic on a nil source.
	svc := NewFCMService(&appconfig.FCMConfig{
		ProjectID:       "test-project",
		CredentialsJSON: `{"type":"service_account"}`, // valid JSON, missing key material
	})
	_, err1 := svc.getAccessToken(context.Background())
	_, err2 := svc.getAccessToken(context.Background())
	if err1 == nil || err2 == nil {
		t.Fatal("expected errors for incomplete service account JSON")
	}
}
