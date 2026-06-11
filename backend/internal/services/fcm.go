package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	appconfig "github.com/dreamlog/backend/internal/config"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// FCMService sends push notifications via Firebase Cloud Messaging HTTP v1 API.
// Authentication uses a service account JSON credential file that is exchanged
// for a short-lived OAuth2 access token before each batch of sends.
type FCMService struct {
	cfg        *appconfig.FCMConfig
	httpClient *http.Client

	tokenOnce   sync.Once
	tokenSource oauth2.TokenSource
	tokenErr    error
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
		// FCM not configured - skip silently in dev.
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
// OAuth2 bearer token. The TokenSource is created once and reused; it caches
// the token internally and refreshes it automatically before expiry.
func (s *FCMService) getAccessToken(ctx context.Context) (string, error) {
	s.tokenOnce.Do(func() {
		conf, err := google.JWTConfigFromJSON(
			[]byte(s.cfg.CredentialsJSON),
			"https://www.googleapis.com/auth/firebase.messaging",
		)
		if err != nil {
			s.tokenErr = fmt.Errorf("fcm: parse credentials: %w", err)
			return
		}
		// Background context: the source outlives any single request.
		s.tokenSource = conf.TokenSource(context.Background())
	})
	if s.tokenErr != nil {
		return "", s.tokenErr
	}

	token, err := s.tokenSource.Token()
	if err != nil {
		return "", fmt.Errorf("fcm: fetch access token: %w", err)
	}
	return token.AccessToken, nil
}
