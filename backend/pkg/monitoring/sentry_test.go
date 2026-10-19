package monitoring

import (
	"testing"

	"github.com/getsentry/sentry-go"
)

// M17: request bodies, cookies, query strings and auth headers never reach Sentry.
func TestScrubEvent(t *testing.T) {
	ev := &sentry.Event{Request: &sentry.Request{
		Data:        `{"content":"my private journal"}`,
		Cookies:     "session=abc",
		QueryString: "p=1234",
		Headers:     map[string]string{"Authorization": "Bearer x", "User-Agent": "ua"},
	}}
	out := scrubEvent(ev, nil)
	if out.Request.Data != "" || out.Request.Cookies != "" || out.Request.QueryString != "" {
		t.Errorf("body/cookies/query must be scrubbed: %+v", out.Request)
	}
	if _, ok := out.Request.Headers["Authorization"]; ok {
		t.Error("Authorization header must be removed")
	}
	if out.Request.Headers["User-Agent"] != "ua" {
		t.Error("non-sensitive headers are kept")
	}
	if scrubEvent(&sentry.Event{}, nil) == nil {
		t.Error("events without a request pass through")
	}
}
