//go:build integration

package blackbox

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestBlackboxAccountEntries(t *testing.T) {
	h := newAPI(t)
	me := h.request(t, "account.read-own-profile", "GET", "/me", "", h.owner, 200)
	if me["id"] != h.ownerID.String() {
		t.Fatalf("wrong profile: %v", me)
	}
	h.request(t, "account.update-name-language", "PUT", "/me", `{"name":"QA changed","voice_language":"hindi","timezone":"Asia/Kolkata"}`, h.owner, 200)
	me = h.request(t, "account.read-persisted-profile", "GET", "/me", "", h.owner, 200)
	if me["name"] != "QA changed" || me["voice_language"] != "hindi" {
		t.Fatalf("profile not persisted: %v", me)
	}
	h.request(t, "account.reject-invalid-hour", "PUT", "/me", `{"fcm_nudge_hour":25}`, h.owner, 400)
	h.request(t, "account.reject-invalid-timezone", "PUT", "/me", `{"timezone":"Not/AZone"}`, h.owner, 400)
	h.request(t, "account.reject-malformed-json", "PUT", "/me", `{`, h.owner, 400)
	other := h.request(t, "account.other-user-isolation", "GET", "/me", "", h.other, 200)
	if other["id"] == me["id"] || other["name"] == "QA changed" {
		t.Fatal("cross-account profile leak")
	}
	h.request(t, "account.initial-terms", "GET", "/me/terms", "", h.owner, 200)
	h.request(t, "account.accept-terms", "POST", "/me/accept-terms", `{"version":"qa-v1"}`, h.owner, 200)
	terms := h.request(t, "account.persisted-terms", "GET", "/me/terms", "", h.owner, 200)
	if terms["tos_version"] != "qa-v1" {
		t.Fatal("terms version not persisted")
	}
	plan := h.request(t, "billing.paid-plan", "GET", "/billing/plan", "", h.owner, 200)
	if plan["plan"] != "pro" {
		t.Fatal("wrong paid plan")
	}
	plan = h.request(t, "billing.free-plan", "GET", "/billing/plan", "", h.other, 200)
	if plan["plan"] != "free" {
		t.Fatal("wrong free plan")
	}
	h.request(t, "billing.reject-invalid-plan", "POST", "/billing/upgrade", `{"plan":"unlimited"}`, h.other, 400)
	h.request(t, "billing.reject-malformed-json", "POST", "/billing/upgrade", `{`, h.other, 400)
	h.request(t, "entries.empty-timeline", "GET", "/timeline", "", h.owner, 200)
	list := h.request(t, "entries.empty-list", "GET", "/entries", "", h.owner, 200)
	if list["total"] != float64(0) {
		t.Fatal("empty fixture contains entries")
	}
	presign := h.request(t, "entries.presign-owned-key", "POST", "/entries/presign", `{}`, h.owner, 200)
	key := presign["audio_key"].(string)
	if !strings.HasPrefix(key, "audio/"+h.ownerID.String()+"/") {
		t.Fatal("presign key ownership")
	}
	h.request(t, "entries.reject-missing-fields", "POST", "/entries", `{}`, h.owner, 400)
	h.request(t, "entries.reject-other-user-audio", "POST", "/entries", fmt.Sprintf(`{"audio_key":"audio/%s/clip.aac","audio_size_bytes":20,"duration_sec":2}`, h.otherID), h.owner, 400)
	h.request(t, "entries.reject-overlong-recording", "POST", "/entries", fmt.Sprintf(`{"audio_key":%q,"audio_size_bytes":20,"duration_sec":1801}`, key), h.owner, 400)
	h.request(t, "entries.reject-missing-object", "POST", "/entries", fmt.Sprintf(`{"audio_key":"audio/%s/missing.aac","audio_size_bytes":20,"duration_sec":2}`, h.ownerID), h.owner, 400)
	entry := h.request(t, "entries.create-and-queue", "POST", "/entries", fmt.Sprintf(`{"audio_key":%q,"audio_size_bytes":20,"duration_sec":2,"mode":"gratitude"}`, key), h.owner, 201)
	if entry["status"] != "pending" || entry["mode"] != "gratitude" {
		t.Fatalf("entry: %v", entry)
	}
	id := entry["id"].(string)
	got := h.request(t, "entries.read-created-entry", "GET", "/entries/"+id, "", h.owner, 200)
	if got["id"] != id {
		t.Fatal("wrong entry")
	}
	h.request(t, "entries.reject-cross-account-read", "GET", "/entries/"+id, "", h.other, 404)
	h.request(t, "entries.reject-invalid-id", "GET", "/entries/not-a-uuid", "", h.owner, 400)
	h.request(t, "entries.missing-id", "GET", "/entries/"+uuid.NewString(), "", h.owner, 404)
	h.request(t, "entries.pending-analysis", "GET", "/entries/"+id+"/analysis", "", h.owner, 404)
	h.request(t, "entries.search-empty-query", "GET", "/entries/search?q=", "", h.owner, 400)
	h.request(t, "entries.search-no-match", "GET", "/entries/search?q="+url.QueryEscape("QA definitely absent"), "", h.owner, 200)
	list = h.request(t, "entries.paginated-list", "GET", "/entries?page=1&page_size=1", "", h.owner, 200)
	if list["total"] != float64(1) {
		t.Fatal("created entry missing from list")
	}
	h.request(t, "entries.other-user-empty-list", "GET", "/entries", "", h.other, 200)
	h.request(t, "entries.checkin-requires-completed", "POST", "/entries/"+id+"/checkin", `{}`, h.owner, 409)
	h.request(t, "devices.reject-invalid-platform", "POST", "/devices", `{"fcm_token":"qa-token","platform":"desktop"}`, h.owner, 400)
	h.request(t, "devices.register-android", "POST", "/devices", `{"fcm_token":"qa-token","platform":"android"}`, h.owner, 200)
	h.request(t, "devices.retry-registration", "POST", "/devices", `{"fcm_token":"qa-token","platform":"android"}`, h.owner, 200)
	// Seed completed content as an external dependency fixture; all observations use HTTP.
	if _, err := h.db.Exec(context.Background(), `UPDATE entries SET status='completed',transcript='QA searchable gentle walk' WHERE id=$1`, id); err != nil {
		t.Fatal(err)
	}
	if _, err := h.db.Exec(context.Background(), `INSERT INTO entry_analysis(entry_id,mood_score,emotional_tone,topics,key_quotes,summary,reflection,morning_nudge) VALUES($1,7,'[{"emotion":"hope","intensity":0.7}]',ARRAY['rest'],ARRAY['gentle walk'],'QA summary','QA reflection','QA checkin')`, id); err != nil {
		t.Fatal(err)
	}
	h.request(t, "entries.completed-analysis", "GET", "/entries/"+id+"/analysis", "", h.owner, 200)
	h.request(t, "entries.analysis-ownership", "GET", "/entries/"+id+"/analysis", "", h.other, 404)
	conv := h.request(t, "conversation.create", "POST", "/entries/"+id+"/conversation", `{}`, h.owner, 200)
	cid := conv["id"].(string)
	again := h.request(t, "conversation.resume-idempotent", "POST", "/entries/"+id+"/conversation", `{}`, h.owner, 200)
	if again["id"] != cid {
		t.Fatal("conversation duplicated")
	}
	h.request(t, "conversation.reject-empty-message", "POST", "/conversations/"+cid+"/messages", `{"content":""}`, h.owner, 400)
	h.request(t, "conversation.reply-first", "POST", "/conversations/"+cid+"/messages", `{"content":"QA I enjoyed a walk"}`, h.owner, 200)
	h.request(t, "conversation.reply-second", "POST", "/conversations/"+cid+"/messages", `{"content":"QA I will rest"}`, h.owner, 200)
	h.request(t, "conversation.reply-third", "POST", "/conversations/"+cid+"/messages", `{"content":"QA thank you"}`, h.owner, 200)
	h.request(t, "conversation.reject-fourth-turn", "POST", "/conversations/"+cid+"/messages", `{"content":"QA extra"}`, h.owner, 409)
}
