package services

import (
	"context"
	"errors"
	"github.com/dreamlog/backend/pkg/apierr"
	"strings"
	"testing"
)

type waitlistRecorder struct {
	emails []string
	err    error
}

func (r *waitlistRecorder) Join(_ context.Context, email string) error {
	r.emails = append(r.emails, email)
	return r.err
}

func TestWaitlistValidationAndNormalization(t *testing.T) {
	for _, email := range []string{"", "invalid", "Name <you@example.com>", "a@localhost", strings.Repeat("x", 250) + "@example.com", "a\r\nb@example.com"} {
		store := &waitlistRecorder{}
		err := NewWaitlistService(store).Join(context.Background(), email, "")
		api, ok := apierr.As(err)
		if !ok || api.Code != 400 || len(store.emails) != 0 {
			t.Fatalf("invalid email accepted: %q", email)
		}
	}
	store := &waitlistRecorder{}
	if err := NewWaitlistService(store).Join(context.Background(), "  You+ode@Example.COM  ", ""); err != nil {
		t.Fatal(err)
	}
	if len(store.emails) != 1 || store.emails[0] != "you+ode@example.com" {
		t.Fatal(store.emails)
	}
}

func TestWaitlistHoneypotAndFailure(t *testing.T) {
	store := &waitlistRecorder{err: errors.New("email sensitive@example.com database unavailable")}
	svc := NewWaitlistService(store)
	if err := svc.Join(context.Background(), "you@example.com", "bot.example"); err != nil || len(store.emails) != 0 {
		t.Fatal("honeypot stored")
	}
	err := svc.Join(context.Background(), "you@example.com", "")
	api, ok := apierr.As(err)
	if !ok || api.Code != 500 || strings.Contains(err.Error(), "sensitive") || api.Cause != nil {
		t.Fatal("storage failure must be generic")
	}
}
