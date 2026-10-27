package services

import (
	"context"
	"github.com/dreamlog/backend/pkg/apierr"
	"net/mail"
	"strings"
)

type waitlistStore interface {
	Join(context.Context, string) error
}
type WaitlistService struct{ store waitlistStore }

func NewWaitlistService(store waitlistStore) *WaitlistService { return &WaitlistService{store: store} }

func (s *WaitlistService) Join(ctx context.Context, email, website string) error {
	email = strings.ToLower(strings.TrimSpace(email))
	address, err := mail.ParseAddress(email)
	if err != nil || len(email) > 254 || address.Address != email || !strings.Contains(email, ".") {
		return apierr.BadRequest("Enter a valid email address.")
	}
	// Honeypot submissions get the same response without storing anything.
	if website != "" {
		return nil
	}
	if err := s.store.Join(ctx, email); err != nil {
		// Do not attach raw database errors: constraint failures can include emails.
		return apierr.Internal("We couldn't save your place. Please try again.")
	}
	return nil
}
