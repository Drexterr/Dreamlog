//go:build integration

package blackbox

import (
	"fmt"
	"testing"
	"time"
)

func TestBlackboxConsent(t *testing.T) {
	h := newAPI(t)
	therapist := h.request(t, "consent.register-therapist", "POST", "/therapists/register", `{"name":"QA consent therapist","email":"consent@example.test"}`, h.owner, 201)
	id := therapist["id"].(string)
	h.request(t, "consent.reject-nontherapist-list", "GET", "/therapists/clients", "", h.other, 403)
	h.request(t, "consent.invalid-client-id", "POST", "/therapists/clients/link", `{"client_id":"bad"}`, h.owner, 400)
	h.request(t, "consent.request-link", "POST", "/therapists/clients/link", fmt.Sprintf(`{"client_id":%q}`, h.otherID), h.owner, 200)
	pending := h.request(t, "consent.pending-request-visible-to-client", "GET", "/therapists/requests", "", h.other, 200)
	requests := pending["requests"].([]any)
	if len(requests) != 1 || requests[0].(map[string]any)["therapist_id"] != id {
		t.Fatal("pending request identity mismatch")
	}
	h.request(t, "consent.pending-client-brief-hidden", "GET", "/therapists/clients/"+h.otherID.String()+"/brief", "", h.owner, 404)
	h.request(t, "consent.invalid-approval-id", "POST", "/therapists/requests/bad/approve", `{}`, h.other, 400)
	h.request(t, "consent.approve-link", "POST", "/therapists/requests/"+id+"/approve", `{}`, h.other, 200)
	clients := h.request(t, "consent.approved-client-visible", "GET", "/therapists/clients", "", h.owner, 200)
	rows := clients["clients"].([]any)
	if len(rows) != 1 || rows[0].(map[string]any)["client_id"] != h.otherID.String() {
		t.Fatal("approved client missing")
	}
	brief := h.request(t, "consent.approved-brief", "GET", "/therapists/clients/"+h.otherID.String()+"/brief", "", h.owner, 200)
	if brief["client_id"] != h.otherID.String() {
		t.Fatal("brief client mismatch")
	}
	h.request(t, "consent.client-revokes", "POST", "/therapists/requests/"+id+"/decline", `{}`, h.other, 200)
	h.request(t, "consent.revoked-brief-hidden", "GET", "/therapists/clients/"+h.otherID.String()+"/brief", "", h.owner, 404)
	h.request(t, "consent.therapist-unlink", "DELETE", "/therapists/clients/"+h.otherID.String(), "", h.owner, 204)
}

func TestBlackboxConversationOwnership(t *testing.T) {
	h := newAPI(t)
	first := exploreEntry(t, h, h.ownerID, time.Now(), 70, false)
	second := exploreEntry(t, h, h.ownerID, time.Now(), 60, false)
	exploreSeed(t, h, `UPDATE entry_analysis SET reflection='QA reflection',morning_nudge='QA nudge',topics='{}',key_quotes='{}' WHERE entry_id=$1`, first)
	conv := h.request(t, "conversation.ownership-owner-create", "POST", "/entries/"+first.String()+"/conversation", `{}`, h.owner, 200)
	id := conv["id"].(string)
	h.request(t, "conversation.ownership-owner-reply", "POST", "/conversations/"+id+"/messages", `{"content":"QA private synthetic text"}`, h.owner, 200)
	h.request(t, "conversation.ownership-existing-denied", "POST", "/entries/"+first.String()+"/conversation", `{}`, h.other, 404)
	h.request(t, "conversation.ownership-new-denied", "POST", "/entries/"+second.String()+"/conversation", `{}`, h.other, 404)
	again := h.request(t, "conversation.ownership-owner-preserved", "POST", "/entries/"+first.String()+"/conversation", `{}`, h.owner, 200)
	if again["id"] != id || again["user_id"] != h.ownerID.String() {
		t.Fatal("foreign request changed conversation ownership")
	}
}
