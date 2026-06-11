package apierr

import (
	"errors"
	"fmt"
	"net/http"
	"testing"
)

func TestConstructors_SetCodeAndMessage(t *testing.T) {
	cases := []struct {
		err  *APIError
		code int
		msg  string
	}{
		{BadRequest("bad input"), http.StatusBadRequest, "bad input"},
		{Unauthorized("no token"), http.StatusUnauthorized, "no token"},
		{Forbidden("not yours"), http.StatusForbidden, "not yours"},
		{NotFound("entry"), http.StatusNotFound, "entry not found"},
		{Conflict("already exists"), http.StatusConflict, "already exists"},
		{Internal("oops"), http.StatusInternalServerError, "oops"},
		{New(http.StatusPaymentRequired, "pay up"), http.StatusPaymentRequired, "pay up"},
	}
	for _, tc := range cases {
		if tc.err.Code != tc.code {
			t.Errorf("%q: want code %d, got %d", tc.msg, tc.code, tc.err.Code)
		}
		if tc.err.Error() != tc.msg {
			t.Errorf("want message %q, got %q", tc.msg, tc.err.Error())
		}
	}
}

func TestNew_WithDetail(t *testing.T) {
	e := BadRequest("invalid body", "field x is required")
	if e.Detail != "field x is required" {
		t.Errorf("want detail set, got %q", e.Detail)
	}
	if BadRequest("no detail").Detail != "" {
		t.Error("detail must be empty when not provided")
	}
}

func TestAs_UnwrapsFromChain(t *testing.T) {
	base := NotFound("user")
	wrapped := fmt.Errorf("handler: %w", base)

	got, ok := As(wrapped)
	if !ok {
		t.Fatal("As must unwrap APIError from a wrapped chain")
	}
	if got.Code != http.StatusNotFound {
		t.Errorf("want 404, got %d", got.Code)
	}

	if _, ok := As(errors.New("plain")); ok {
		t.Error("As must return false for non-APIError")
	}
}
