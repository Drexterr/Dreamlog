//go:build integration

package blackbox

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/dreamlog/backend/internal/models"
	"github.com/google/uuid"
)

// Operator DLQ tools over real PostgreSQL + Redis: admin-only (verified email
// on ADMIN_EMAILS), privacy-safe listing, single-shot requeue, discard.
func TestBlackboxAdminDLQ(t *testing.T) {
	h := newAPI(t)
	ctx := context.Background()
	seed := func(sql string, args ...any) {
		t.Helper()
		if _, err := h.db.Exec(ctx, sql, args...); err != nil {
			t.Fatal(err)
		}
	}
	deadLetter := func(job models.TranscriptionJob) {
		t.Helper()
		payload, _ := json.Marshal(job)
		if err := h.jobs.EnqueueDLQ(ctx, payload, "max retries exceeded: claude: timeout"); err != nil {
			t.Fatal(err)
		}
	}
	failedEntry := func(transcript *string, audioKey string) uuid.UUID {
		t.Helper()
		id := uuid.New()
		seed(`INSERT INTO entries(id,user_id,audio_key,status,transcript,error_msg,retry_count) VALUES($1,$2,$3,'failed',$4,'max retries exceeded',3)`,
			id, h.otherID, audioKey, transcript)
		deadLetter(models.TranscriptionJob{EntryID: id, UserID: h.otherID, AudioKey: audioKey, UserCountry: "IN", Attempt: 2})
		return id
	}
	transcript := "QA private journal words"
	retryable := failedEntry(&transcript, "audio/"+h.otherID.String()+"/retry.m4a")
	audioGone := failedEntry(nil, "audio/"+h.otherID.String()+"/missing.m4a")
	completed := failedEntry(&transcript, "audio/"+h.otherID.String()+"/done.m4a")
	seed(`UPDATE entries SET status='completed' WHERE id=$1`, completed)

	// Local signups are unverified: the admin email alone is not enough.
	h.request(t, "admin.unverified-admin-hidden", "GET", "/admin/dlq/entries", "", h.owner, 404)
	seed(`UPDATE users SET email_verified=true WHERE id=$1`, h.ownerID)
	h.request(t, "admin.non-admin-hidden", "GET", "/admin/dlq/entries", "", h.other, 404)
	h.request(t, "admin.requires-auth", "GET", "/admin/dlq/entries", "", "", 401)
	h.request(t, "admin.unknown-queue", "GET", "/admin/dlq/payments", "", h.owner, 400)
	h.request(t, "admin.bad-page-size", "GET", "/admin/dlq/entries?page_size=1000", "", h.owner, 400)

	list := h.request(t, "admin.list-entries", "GET", "/admin/dlq/entries?page_size=2", "", h.owner, 200)
	if list["total"] != float64(3) || list["has_more"] != true || len(list["items"].([]any)) != 2 {
		t.Fatalf("listing: %v", list)
	}
	raw, _ := json.Marshal(list)
	for _, leak := range []string{transcript, "audio/", h.otherID.String()} {
		if strings.Contains(string(raw), leak) {
			t.Fatalf("DLQ listing leaks %q: %s", leak, raw)
		}
	}
	h.request(t, "admin.list-notes-empty", "GET", "/admin/dlq/notes", "", h.owner, 200)

	h.request(t, "admin.requeue-audio-gone", "POST", "/admin/dlq/entries/"+audioGone.String()+"/requeue", "", h.owner, 409)
	h.request(t, "admin.requeue-completed", "POST", "/admin/dlq/entries/"+completed.String()+"/requeue", "", h.owner, 409)
	h.request(t, "admin.requeue-missing-item", "POST", "/admin/dlq/entries/"+uuid.NewString()+"/requeue", "", h.owner, 404)

	// A double click: concurrent requeues of one dead letter enqueue one job.
	before, err := h.rdb.LLen(ctx, h.jobsKey).Result()
	if err != nil {
		t.Fatal(err)
	}
	const n = 5
	codes := make(chan int, n)
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			codes <- h.status(t, "POST", "/admin/dlq/entries/"+retryable.String()+"/requeue", h.owner)
		}()
	}
	wg.Wait()
	close(codes)
	ok := 0
	for c := range codes {
		switch c {
		case 200:
			ok++
		case 404, 409:
		default:
			t.Fatalf("unexpected status %d", c)
		}
	}
	if ok != 1 {
		t.Fatalf("requeue succeeded %d times, want exactly 1", ok)
	}
	after, _ := h.rdb.LLen(ctx, h.jobsKey).Result()
	if after-before != 1 {
		t.Fatalf("enqueued %d jobs, want 1", after-before)
	}
	var status string
	var retries int
	var errMsg *string
	if err := h.db.QueryRow(ctx, `SELECT status, retry_count, error_msg FROM entries WHERE id=$1`, retryable).Scan(&status, &retries, &errMsg); err != nil {
		t.Fatal(err)
	}
	if status != "pending" || retries != 0 || errMsg != nil {
		t.Fatalf("entry not reset: %s %d %v", status, retries, errMsg)
	}
	job, _ := h.rdb.LIndex(ctx, h.jobsKey, 0).Result()
	if !strings.Contains(job, retryable.String()) || !strings.Contains(job, `"attempt":0`) {
		t.Fatalf("fresh job: %s", job)
	}

	h.request(t, "admin.discard", "DELETE", "/admin/dlq/entries/"+audioGone.String(), "", h.owner, 204)
	h.request(t, "admin.discard-again", "DELETE", "/admin/dlq/entries/"+audioGone.String(), "", h.owner, 404)
	h.request(t, "admin.non-admin-cannot-discard", "DELETE", "/admin/dlq/entries/"+completed.String(), "", h.other, 404)
	left := h.request(t, "admin.list-after", "GET", "/admin/dlq/entries", "", h.owner, 200)
	if left["total"] != float64(1) {
		t.Fatalf("one dead letter (the completed entry) should remain: %v", left)
	}
}

// status performs a request and returns only its status code (safe to call
// from several goroutines).
func (h *apiHarness) status(t *testing.T, method, path, token string) int {
	t.Helper()
	req, err := http.NewRequest(method, h.server.URL+path, nil)
	if err != nil {
		t.Error(err)
		return 0
	}
	req.Header.Set("Authorization", "Bearer "+token)
	res, err := h.server.Client().Do(req)
	if err != nil {
		t.Error(err)
		return 0
	}
	res.Body.Close()
	return res.StatusCode
}
