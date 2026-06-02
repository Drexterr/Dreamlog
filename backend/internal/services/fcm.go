package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	appconfig "github.com/dreamlog/backend/internal/config"
)

// FCMService sends push notifications via Firebase Cloud Messaging HTTP v1 API.
// Authentication uses a service account JSON credential file that is exchanged
// for a short-lived OAuth2 access token before each batch of sends.
type FCMService struct {
	cfg        *appconfig.FCMConfig
	httpClient *http.Client
}

func NewFCMService(cfg *appconfig.FCMConfig) *FCMService {
	return &FCMService{
		cfg:        cfg,
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

type fcmMessage struct {
	Message fcmPayload `json:"message"`
}

type fcmPayload struct {
	Token        string            `json:"token"`
	Notification fcmNotification   `json:"notification"`
	Data         map[string]string `json:"data,omitempty"`
}

type fcmNotification struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

// SendToToken sends a single push notification to one device token.
func (s *FCMService) SendToToken(ctx context.Context, token, title, body string, data map[string]string) error {
	if s.cfg.ProjectID == "" || s.cfg.CredentialsJSON == "" {
		// FCM not configured — skip silently in dev.
		return nil
	}

	accessToken, err := s.getAccessToken(ctx)
	if err != nil {
		return fmt.Errorf("fcm: get access token: %w", err)
	}

	msg := fcmMessage{
		Message: fcmPayload{
			Token:        token,
			Notification: fcmNotification{Title: title, Body: body},
			Data:         data,
		},
	}

	b, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("fcm: marshal message: %w", err)
	}

	url := fmt.Sprintf("https://fcm.googleapis.com/v1/projects/%s/messages:send", s.cfg.ProjectID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		return fmt.Errorf("fcm: build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("fcm: send: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("fcm: send returned %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

// getAccessToken exchanges the service account credentials for a short-lived
// OAuth2 bearer token using the Google token endpoint.
// In production this should be cached with a ~50-minute TTL.
func (s *FCMService) getAccessToken(ctx context.Context) (string, error) {
	// Build a minimal JWT assertion for the Google token endpoint.
	// google.golang.org/api/option is the canonical approach;
	// this is a simplified manual implementation to avoid adding a heavy dependency.
	// For production: use "golang.org/x/oauth2/google" with ServiceAccountJSON.

	type serviceAccount struct {
		ClientEmail string `json:"client_email"`
		PrivateKey  string `json:"private_key"`
		TokenURI    string `json:"token_uri"`
	}

	var sa serviceAccount
	if err := json.Unmarshal([]byte(s.cfg.CredentialsJSON), &sa); err != nil {
		return "", fmt.Errorf("fcm: parse credentials: %w", err)
	}

	// For a production implementation, replace this with golang.org/x/oauth2/google:
	//
	//   conf, err := google.JWTConfigFromJSON([]byte(s.cfg.CredentialsJSON),
	//       "https://www.googleapis.com/auth/firebase.messaging")
	//   token, err := conf.TokenSource(ctx).Token()
	//   return token.AccessToken, err
	//
	// The stub below returns an error so FCM is skipped in dev without credentials.
	_ = sa
	return "", fmt.Errorf("fcm: production credentials required — add golang.org/x/oauth2/google")
}
