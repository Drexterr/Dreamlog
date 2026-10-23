//go:build integration

package blackbox

import (
	"fmt"
	"github.com/google/uuid"
	"testing"
)

func TestBlackboxNotes(t *testing.T) {
	h := newAPI(t)
	h.request(t, "notes.nontherapist-profile", "GET", "/therapists/me", "", h.owner, 404)
	h.request(t, "notes.nontherapist-denied", "GET", "/therapists/external-clients", "", h.owner, 403)
	h.request(t, "notes.registration-validation", "POST", "/therapists/register", `{}`, h.owner, 400)
	therapist := h.request(t, "notes.register-therapist", "POST", "/therapists/register", `{"name":"QA therapist","email":"qa@example.test","credentials":"Synthetic QA"}`, h.owner, 201)
	tid := therapist["id"].(string)
	h.request(t, "notes.profile-after-registration", "GET", "/therapists/me", "", h.owner, 200)
	h.request(t, "notes.consent-required", "POST", "/therapists/external-clients", `{"name":"QA client"}`, h.owner, 403)
	h.request(t, "notes.accept-client-consent", "POST", "/therapists/consent", `{"version":"qa-v1"}`, h.owner, 200)
	h.request(t, "notes.reject-empty-client", "POST", "/therapists/external-clients", `{"name":" "}`, h.owner, 400)
	client := h.request(t, "notes.create-encrypted-client", "POST", "/therapists/external-clients", `{"name":"QA synthetic client","role":"client"}`, h.owner, 201)
	id := client["id"].(string)
	got := h.request(t, "notes.read-decrypted-client", "GET", "/therapists/external-clients/"+id, "", h.owner, 200)
	if got["name"] != "QA synthetic client" {
		t.Fatal("client decrypt round trip")
	}
	h.request(t, "notes.list-clients", "GET", "/therapists/external-clients", "", h.owner, 200)
	h.request(t, "notes.rename-client", "PATCH", "/therapists/external-clients/"+id, `{"name":"QA renamed"}`, h.owner, 200)
	h.request(t, "notes.client-invalid-id", "GET", "/therapists/external-clients/bad", "", h.owner, 400)
	h.request(t, "notes.client-not-found", "GET", "/therapists/external-clients/"+uuid.NewString(), "", h.owner, 404)
	h.request(t, "notes.session-missing-client", "POST", "/therapists/sessions", `{"bullets":["QA note"]}`, h.owner, 400)
	h.request(t, "notes.session-invalid-date", "POST", "/therapists/sessions", fmt.Sprintf(`{"external_client_id":%q,"session_date":"bad","bullets":["QA note"]}`, id), h.owner, 400)
	h.request(t, "notes.session-empty-bullets", "POST", "/therapists/sessions", fmt.Sprintf(`{"external_client_id":%q,"bullets":[" "]}`, id), h.owner, 400)
	session := h.request(t, "notes.create-typed-session", "POST", "/therapists/sessions", fmt.Sprintf(`{"external_client_id":%q,"session_date":"2026-10-15","bullets":["QA synthetic note"]}`, id), h.owner, 201)
	sid := session["id"].(string)
	h.request(t, "notes.read-session", "GET", "/therapists/sessions/"+sid, "", h.owner, 200)
	h.request(t, "notes.list-session-filter", "GET", "/therapists/sessions?external_client_id="+id, "", h.owner, 200)
	h.request(t, "notes.update-bullets", "PATCH", "/therapists/sessions/"+sid, `{"bullets":["QA revised note"]}`, h.owner, 200)
	h.request(t, "notes.summarize-session-controlled-ai", "POST", "/therapists/sessions/"+sid+"/summarize", `{}`, h.owner, 200)
	h.request(t, "notes.overview", "GET", "/therapists/overview", "", h.owner, 200)
	h.request(t, "notes.presign-reject-type", "POST", "/therapists/sessions/presign", `{"filename":"qa.exe","content_type":"application/octet-stream"}`, h.owner, 400)
	h.request(t, "notes.presign-image", "POST", "/therapists/sessions/presign", `{"filename":"qa.jpg","content_type":"image/jpeg"}`, h.owner, 200)
	h.request(t, "notes.reject-foreign-image-key", "POST", "/therapists/sessions", fmt.Sprintf(`{"external_client_id":%q,"image_key":"notes/foreign/photo.jpg"}`, id), h.owner, 400)
	h.request(t, "notes.create-ocr-job", "POST", "/therapists/sessions", fmt.Sprintf(`{"external_client_id":%q,"image_key":"notes/%s/qa.jpg"}`, id, tid), h.owner, 201)
	// The second therapist has consent but cannot access the first therapist's data.
	h.request(t, "notes.register-second-therapist", "POST", "/therapists/register", `{"name":"QA other","email":"other@example.test"}`, h.other, 201)
	h.request(t, "notes.second-consent", "POST", "/therapists/consent", `{}`, h.other, 200)
	h.request(t, "notes.cross-therapist-client-denied", "GET", "/therapists/external-clients/"+id, "", h.other, 404)
	h.request(t, "notes.cross-therapist-session-denied", "GET", "/therapists/sessions/"+sid, "", h.other, 404)
	h.request(t, "notes.cross-therapist-update-denied", "PATCH", "/therapists/sessions/"+sid, `{"bullets":["forbidden"]}`, h.other, 404)
	h.request(t, "notes.archive-client", "PATCH", "/therapists/external-clients/"+id, `{"archived":true}`, h.owner, 200)
	h.request(t, "notes.list-archived-clients", "GET", "/therapists/external-clients?include_archived=true", "", h.owner, 200)
	h.request(t, "notes.restore-client", "PATCH", "/therapists/external-clients/"+id, `{"archived":false}`, h.owner, 200)
	h.request(t, "notes.delete-session", "DELETE", "/therapists/sessions/"+sid, "", h.owner, 204)
	h.request(t, "notes.deleted-session-hidden", "GET", "/therapists/sessions/"+sid, "", h.owner, 404)
	h.request(t, "notes.delete-client", "DELETE", "/therapists/external-clients/"+id, "", h.owner, 204)
	h.request(t, "notes.deleted-client-hidden", "GET", "/therapists/external-clients/"+id, "", h.owner, 404)
}

func TestBlackboxSharingTherapy(t *testing.T) {
	h := newAPI(t)
	h.request(t, "sharing.free-plan-denied", "POST", "/share", `{}`, h.other, 403)
	link := h.request(t, "sharing.create-link", "POST", "/share", `{}`, h.owner, 201)
	token := link["token"].(string)
	pass := link["passcode"].(string)
	h.request(t, "sharing.reject-no-passcode", "GET", "/share/"+token, "", "", 401)
	h.request(t, "sharing.reject-wrong-passcode", "GET", "/share/"+token+"?p=wrong", "", "", 401)
	h.request(t, "sharing.authorized-view", "GET", "/share/"+token+"?p="+pass, "", "", 200)
	links := h.request(t, "sharing.list-owner", "GET", "/share", "", h.owner, 200)
	id := links["links"].([]any)[0].(map[string]any)["id"].(string)
	h.request(t, "sharing.reject-invalid-revoke-id", "DELETE", "/share/bad", "", h.owner, 400)
	h.request(t, "sharing.revoke-link", "DELETE", "/share/"+id, "", h.owner, 204)
	h.request(t, "sharing.revoked-view-denied", "GET", "/share/"+token+"?p="+pass, "", "", 404)
	h.request(t, "sharing.missing-link", "GET", "/share/absent?p=1234", "", "", 404)
	h.request(t, "therapy.invalid-persona", "POST", "/therapy/sessions", `{"persona":"unknown"}`, h.owner, 400)
	h.request(t, "therapy.empty-sessions", "GET", "/therapy/sessions", "", h.owner, 200)
	session := h.request(t, "therapy.start-comforting", "POST", "/therapy/sessions", `{"persona":"comforting"}`, h.owner, 201)
	id = session["id"].(string)
	h.request(t, "therapy.read-session", "GET", "/therapy/sessions/"+id, "", h.owner, 200)
	h.request(t, "therapy.cross-account-denied", "GET", "/therapy/sessions/"+id, "", h.other, 404)
	h.request(t, "therapy.invalid-session-id", "GET", "/therapy/sessions/bad", "", h.owner, 400)
	h.request(t, "therapy.reject-empty-message", "POST", "/therapy/sessions/"+id+"/messages", `{}`, h.owner, 400)
	h.request(t, "therapy.text-turn-controlled-ai", "POST", "/therapy/sessions/"+id+"/messages", `{"content":"QA I had a calm day","input_mode":"text"}`, h.owner, 201)
	h.request(t, "therapy.end-session", "POST", "/therapy/sessions/"+id+"/end", `{}`, h.owner, 200)
	h.request(t, "therapy.reject-turn-after-end", "POST", "/therapy/sessions/"+id+"/messages", `{"content":"QA extra turn","input_mode":"text"}`, h.owner, 409)
}

// Therapy pay-per-use: credits bought through POST /billing/therapy-credits
// (dev stub - no store credentials) pay for sessions beyond the free first one.
func TestBlackboxTherapyCredits(t *testing.T) {
	h := newAPI(t)
	credits := func(name string) float64 {
		t.Helper()
		plan := h.request(t, name, "GET", "/billing/plan", "", h.other, 200)
		n, ok := plan["therapy_credits"].(float64)
		if !ok {
			t.Fatalf("therapy_credits missing: %v", plan)
		}
		return n
	}
	if credits("credits.initial-balance") != 0 {
		t.Fatal("new accounts start without credits")
	}
	h.request(t, "credits.reject-unknown-product", "POST", "/billing/therapy-credits", `{"product_id":"com.example.other"}`, h.other, 400)
	h.request(t, "credits.reject-plan-product", "POST", "/billing/therapy-credits", `{"product_id":"com.talktoode.app.plus.monthly"}`, h.other, 400)
	h.request(t, "credits.member-price-pro-only", "POST", "/billing/therapy-credits", `{"product_id":"com.talktoode.app.therapy.member"}`, h.other, 403)
	got := h.request(t, "credits.dev-grant", "POST", "/billing/therapy-credits", `{"product_id":"com.talktoode.app.therapy.single"}`, h.other, 200)
	if got["therapy_credits"] != float64(1) || got["credits_granted"] != float64(1) {
		t.Fatalf("grant: %v", got)
	}
	h.request(t, "credits.member-price-for-pro", "POST", "/billing/therapy-credits", `{"product_id":"com.talktoode.app.therapy.member"}`, h.owner, 200)

	first := h.request(t, "credits.first-session-free", "POST", "/therapy/sessions", `{}`, h.other, 201)
	if first["billing_amount_paise"] != float64(0) || credits("credits.free-session-keeps-credit") != 1 {
		t.Fatalf("the free first session must not spend a credit: %v", first)
	}
	paid := h.request(t, "credits.paid-session-spends-credit", "POST", "/therapy/sessions", `{}`, h.other, 201)
	if paid["billing_amount_paise"] != float64(49900) || credits("credits.balance-after-paid-session") != 0 {
		t.Fatalf("paid session: %v", paid)
	}
}
