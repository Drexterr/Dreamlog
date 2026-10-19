//go:build integration

package repositories

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/dreamlog/backend/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

func therapistDataOK(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func therapistDataPG(t *testing.T, err error, code string) {
	t.Helper()
	var pe *pgconn.PgError
	if !errors.As(err, &pe) || pe.Code != code {
		t.Fatalf("expected PostgreSQL %s, got %v", code, err)
	}
}

func therapistDataFixture(t *testing.T, db *pgxpool.Pool) (uuid.UUID, uuid.UUID) {
	t.Helper()
	user := regressionUser(t, db)
	profile, err := NewTherapistRepository(db).Register(context.Background(), user, "QA therapist", user.String()+"@example.test", "QA credentials")
	therapistDataOK(t, err)
	return user, profile.ID
}

func TestIntegration_TherapistDataLinksAndConsent(t *testing.T) {
	t.Parallel()
	db := regressionDB(t)
	ctx := context.Background()
	user, therapist := therapistDataFixture(t, db)
	_, otherTherapist := therapistDataFixture(t, db)
	client, foreign := regressionUser(t, db), regressionUser(t, db)
	r, notes := NewTherapistRepository(db), NewTherapistNotesRepository(db)
	original, err := r.GetByUserID(ctx, user)
	therapistDataOK(t, err)
	updated, err := r.Register(ctx, user, "Updated", "updated@example.test", "New credentials")
	therapistDataOK(t, err)
	if updated.ID != therapist || updated.Name != "Updated" || updated.Email != "updated@example.test" || !updated.CreatedAt.Equal(original.CreatedAt) {
		t.Fatalf("upsert: %+v", updated)
	}
	if got, err := r.GetByUserID(ctx, client); err != nil || got != nil {
		t.Fatalf("missing profile: %+v %v", got, err)
	}
	_, err = r.Register(ctx, uuid.New(), "Missing", "missing@example.test", "")
	therapistDataPG(t, err, "23503")
	for _, id := range []uuid.UUID{therapist, otherTherapist} {
		therapistDataOK(t, r.LinkClient(ctx, id, client))
	}
	therapistDataOK(t, r.LinkClient(ctx, therapist, client))
	if link, err := r.GetClientLink(ctx, therapist, client); err != nil || link != nil {
		t.Fatalf("pending link granted access: %+v %v", link, err)
	}
	requests, err := r.ListPendingRequests(ctx, client)
	therapistDataOK(t, err)
	if len(requests) != 2 || requests[0].RequestedAt.Before(requests[1].RequestedAt) {
		t.Fatalf("pending count/order: %+v", requests)
	}
	if got, err := r.ListPendingRequests(ctx, foreign); err != nil || len(got) != 0 {
		t.Fatalf("foreign requests: %+v %v", got, err)
	}
	if ok, err := r.ApproveLink(ctx, foreign, therapist); err != nil || ok {
		t.Fatalf("foreign approval: %v %v", ok, err)
	}
	if ok, err := r.ApproveLink(ctx, client, therapist); err != nil || !ok {
		t.Fatalf("approval: %v %v", ok, err)
	}
	if ok, err := r.ApproveLink(ctx, client, therapist); err != nil || ok {
		t.Fatalf("duplicate approval: %v %v", ok, err)
	}
	link, err := r.GetClientLink(ctx, therapist, client)
	therapistDataOK(t, err)
	if link == nil || link.Status != "active" || link.RevokedAt != nil || link.ClientID != client {
		t.Fatalf("active link: %+v", link)
	}
	therapistDataOK(t, r.LinkClient(ctx, therapist, client))
	if got, err := r.GetClientLink(ctx, therapist, client); err != nil || got == nil || got.ID != link.ID {
		t.Fatalf("active relink: %+v %v", got, err)
	}
	if ok, err := r.DeclineLink(ctx, foreign, therapist); err != nil || ok {
		t.Fatalf("foreign decline: %v %v", ok, err)
	}
	if ok, err := r.DeclineLink(ctx, client, therapist); err != nil || !ok {
		t.Fatalf("revoke: %v %v", ok, err)
	}
	if ok, err := r.DeclineLink(ctx, client, therapist); err != nil || ok {
		t.Fatalf("duplicate revoke: %v %v", ok, err)
	}
	if got, err := r.GetClientLink(ctx, therapist, client); err != nil || got != nil {
		t.Fatalf("revoked access: %+v %v", got, err)
	}
	therapistDataOK(t, r.LinkClient(ctx, therapist, client))
	var status string
	var revoked *time.Time
	therapistDataOK(t, db.QueryRow(ctx, `SELECT status,revoked_at FROM client_therapist_links WHERE therapist_id=$1 AND client_id=$2`, therapist, client).Scan(&status, &revoked))
	if status != "pending" || revoked != nil {
		t.Fatalf("relink bypassed consent: %s %v", status, revoked)
	}
	therapistDataOK(t, r.UnlinkClient(ctx, therapist, client))
	if ok, err := r.DeclineLink(ctx, client, otherTherapist); err != nil || !ok {
		t.Fatalf("decline pending: %v %v", ok, err)
	}
	if got, err := r.ListPendingRequests(ctx, client); err != nil || len(got) != 0 {
		t.Fatalf("requests remain: %+v %v", got, err)
	}
	therapistDataPG(t, r.LinkClient(ctx, therapist, uuid.New()), "23503")
	if at, version, err := notes.GetUserTerms(ctx, client); err != nil || at != nil || version != nil {
		t.Fatalf("initial terms: %v %v %v", at, version, err)
	}
	if at, version, err := notes.GetUserTerms(ctx, uuid.New()); err != nil || at != nil || version != nil {
		t.Fatalf("missing terms: %v %v %v", at, version, err)
	}
	for _, version := range []string{"qa-1", "qa-2"} {
		therapistDataOK(t, notes.AcceptUserTerms(ctx, client, version))
		at, got, err := notes.GetUserTerms(ctx, client)
		if err != nil || at == nil || got == nil || *got != version {
			t.Fatalf("terms persistence: %v %v %v", at, got, err)
		}
	}
	for _, id := range []uuid.UUID{therapist, uuid.New()} {
		if ok, err := notes.TherapistClientConsentAccepted(ctx, id); err != nil || ok {
			t.Fatalf("initial consent: %v %v", ok, err)
		}
	}
	therapistDataOK(t, notes.AcceptTherapistClientConsent(ctx, therapist, "qa-consent"))
	if ok, err := notes.TherapistClientConsentAccepted(ctx, therapist); err != nil || !ok {
		t.Fatalf("consent persistence: %v %v", ok, err)
	}
	if ok, err := notes.TherapistClientConsentAccepted(ctx, otherTherapist); err != nil || ok {
		t.Fatalf("foreign consent changed: %v %v", ok, err)
	}
}

func TestIntegration_TherapistDataNotesLifecycle(t *testing.T) {
	t.Parallel()
	db := regressionDB(t)
	ctx := context.Background()
	_, owner := therapistDataFixture(t, db)
	_, other := therapistDataFixture(t, db)
	r := NewTherapistNotesRepository(db)
	if key, err := r.GetWrappedDEK(ctx, owner); err != nil || key != nil {
		t.Fatalf("initial DEK: %x %v", key, err)
	}
	for _, candidate := range [][]byte{{0, 1, 255, 2}, {9, 8, 7}} {
		key, err := r.InsertWrappedDEK(ctx, owner, candidate)
		if err != nil || !bytes.Equal(key, []byte{0, 1, 255, 2}) {
			t.Fatalf("first key must win: %x %v", key, err)
		}
	}
	if key, err := r.GetWrappedDEK(ctx, owner); err != nil || !bytes.Equal(key, []byte{0, 1, 255, 2}) {
		t.Fatalf("stored DEK: %x %v", key, err)
	}
	if key, err := r.GetWrappedDEK(ctx, other); err != nil || key != nil {
		t.Fatalf("foreign DEK: %x %v", key, err)
	}
	_, err := r.InsertWrappedDEK(ctx, uuid.New(), []byte{1})
	therapistDataPG(t, err, "23503")
	client, err := r.CreateExternalClient(ctx, owner, []byte{0, 255, 42}, "client")
	therapistDataOK(t, err)
	quiet, err := r.CreateExternalClient(ctx, owner, []byte{3}, "minor")
	therapistDataOK(t, err)
	_, err = r.CreateExternalClient(ctx, other, []byte{4}, "couple")
	therapistDataOK(t, err)
	if client.Archived || client.TherapistID != owner || !bytes.Equal(client.NameEnc, []byte{0, 255, 42}) {
		t.Fatalf("client: %+v", client)
	}
	for _, id := range []uuid.UUID{client.ID, uuid.New()} {
		if got, err := r.GetExternalClient(ctx, other, id); err != nil || got != nil {
			t.Fatalf("foreign/missing client: %+v %v", got, err)
		}
	}
	role, archived := "couple", true
	if ok, err := r.UpdateExternalClient(ctx, other, client.ID, []byte{9}, &role, &archived); err != nil || ok {
		t.Fatalf("foreign update: %v %v", ok, err)
	}
	if ok, err := r.UpdateExternalClient(ctx, owner, client.ID, []byte{5, 0}, &role, &archived); err != nil || !ok {
		t.Fatalf("archive: %v %v", ok, err)
	}
	if got, err := r.ListExternalClients(ctx, owner, false); err != nil || len(got) != 1 || got[0].ID != quiet.ID || got[0].LastSessionAt != nil || got[0].SessionCount != 0 {
		t.Fatalf("archive filter/nullable stats: %+v %v", got, err)
	}
	if got, err := r.ListExternalClients(ctx, owner, true); err != nil || len(got) != 2 {
		t.Fatalf("include archived: %+v %v", got, err)
	}
	archived = false
	if ok, err := r.UpdateExternalClient(ctx, owner, client.ID, nil, nil, &archived); err != nil || !ok {
		t.Fatalf("restore: %v %v", ok, err)
	}
	gotClient, err := r.GetExternalClient(ctx, owner, client.ID)
	if err != nil || gotClient.Archived || gotClient.Role != role || !bytes.Equal(gotClient.NameEnc, []byte{5, 0}) {
		t.Fatalf("partial update: %+v %v", gotClient, err)
	}
	image := "qa/not-uploaded"
	day := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	session, err := r.CreateSession(ctx, &models.ClientSessionRow{TherapistID: owner, ExternalClientID: &client.ID, SessionDate: day, Status: models.ClientSessionPending, ImageKey: &image})
	therapistDataOK(t, err)
	if got, err := r.GetSession(ctx, other, session.ID); err != nil || got != nil {
		t.Fatalf("foreign session: %+v %v", got, err)
	}
	if got, err := r.GetSessionByID(ctx, uuid.New()); err != nil || got != nil {
		t.Fatalf("missing worker session: %+v %v", got, err)
	}
	for _, want := range []bool{true, false} {
		if ok, err := r.SetSessionProcessing(ctx, session.ID); err != nil || ok != want {
			t.Fatalf("claim: %v %v want %v", ok, err, want)
		}
	}
	therapistDataOK(t, r.SetSessionFailed(ctx, session.ID, "retryable OCR", false))
	failed, err := r.GetSessionByID(ctx, session.ID)
	if err != nil || failed.Status != models.ClientSessionPending || failed.RetryCount != 1 || failed.ErrorMsg == nil || *failed.ErrorMsg != "retryable OCR" || failed.ImageKey == nil {
		t.Fatalf("retry state: %+v %v", failed, err)
	}
	if ok, err := r.SetSessionProcessing(ctx, session.ID); err != nil || !ok {
		t.Fatalf("retry claim: %v %v", ok, err)
	}
	therapistDataOK(t, r.CompleteSessionOCR(ctx, session.ID, []byte{0, 8}, []byte{2, 255}))
	for _, operation := range []func(uuid.UUID) (bool, error){
		func(id uuid.UUID) (bool, error) { return r.UpdateSessionBullets(ctx, id, session.ID, []byte{4, 0}) },
		func(id uuid.UUID) (bool, error) { return r.UpdateSessionSummary(ctx, id, session.ID, []byte{5, 255}) },
	} {
		if ok, err := operation(other); err != nil || ok {
			t.Fatalf("foreign notes edit: %v %v", ok, err)
		}
		if ok, err := operation(owner); err != nil || !ok {
			t.Fatalf("notes edit: %v %v", ok, err)
		}
	}
	completed, err := r.GetSession(ctx, owner, session.ID)
	if err != nil || completed.Status != models.ClientSessionCompleted || completed.ImageKey != nil || completed.ErrorMsg != nil || completed.RetryCount != 1 || !bytes.Equal(completed.RawTextEnc, []byte{0, 8}) || !bytes.Equal(completed.BulletsEnc, []byte{4, 0}) || !bytes.Equal(completed.SummaryEnc, []byte{5, 255}) || completed.LinkedUserID != nil || completed.ExternalClientID == nil || *completed.ExternalClientID != client.ID {
		t.Fatalf("completed ciphertext/nullable fields: %+v %v", completed, err)
	}
	if ok, err := r.SetSessionProcessing(ctx, session.ID); err != nil || ok {
		t.Fatalf("completed reclaimed: %v %v", ok, err)
	}
	terminal, err := r.CreateSession(ctx, &models.ClientSessionRow{TherapistID: owner, ExternalClientID: &client.ID, SessionDate: day.AddDate(0, 0, 1), Status: models.ClientSessionPending, ImageKey: &image})
	therapistDataOK(t, err)
	therapistDataOK(t, r.SetSessionFailed(ctx, terminal.ID, "terminal OCR", true))
	if got, err := r.GetSessionByID(ctx, terminal.ID); err != nil || got.Status != models.ClientSessionFailed || got.RetryCount != 1 || got.ErrorMsg == nil || *got.ErrorMsg != "terminal OCR" {
		t.Fatalf("terminal state: %+v %v", got, err)
	}
	if ok, err := r.SetSessionProcessing(ctx, terminal.ID); err != nil || ok {
		t.Fatalf("terminal reclaimed: %v %v", ok, err)
	}
	listed, err := r.ListExternalClients(ctx, owner, false)
	if err != nil || len(listed) != 2 || listed[0].ID != client.ID || listed[0].SessionCount != 2 || listed[0].LastSessionAt == nil {
		t.Fatalf("activity stats/order: %+v %v", listed, err)
	}
	if key, ok, err := r.DeleteSession(ctx, other, terminal.ID); err != nil || ok || key != nil {
		t.Fatalf("foreign delete: %v %v %v", key, ok, err)
	}
	if key, ok, err := r.DeleteSession(ctx, owner, terminal.ID); err != nil || !ok || key == nil || *key != image {
		t.Fatalf("image cleanup key: %v %v %v", key, ok, err)
	}
	if key, ok, err := r.DeleteSession(ctx, owner, session.ID); err != nil || !ok || key != nil {
		t.Fatalf("completed delete: %v %v %v", key, ok, err)
	}
	if _, ok, err := r.DeleteSession(ctx, owner, session.ID); err != nil || ok {
		t.Fatalf("duplicate delete: %v %v", ok, err)
	}
	if ok, err := r.DeleteExternalClient(ctx, other, client.ID); err != nil || ok {
		t.Fatalf("foreign client delete: %v %v", ok, err)
	}
	child, err := r.CreateSession(ctx, &models.ClientSessionRow{TherapistID: owner, ExternalClientID: &client.ID, SessionDate: day, Status: models.ClientSessionCompleted})
	therapistDataOK(t, err)
	if ok, err := r.DeleteExternalClient(ctx, owner, client.ID); err != nil || !ok {
		t.Fatalf("client delete: %v %v", ok, err)
	}
	if got, err := r.GetSessionByID(ctx, child.ID); err != nil || got != nil {
		t.Fatalf("cascade: %+v %v", got, err)
	}
	if ok, err := r.DeleteExternalClient(ctx, owner, client.ID); err != nil || ok {
		t.Fatalf("duplicate client delete: %v %v", ok, err)
	}
}

func TestIntegration_TherapistDataSessionListsConstraintsOverview(t *testing.T) {
	t.Parallel()
	db := regressionDB(t)
	ctx := context.Background()
	_, owner := therapistDataFixture(t, db)
	_, other := therapistDataFixture(t, db)
	linked := regressionUser(t, db)
	r := NewTherapistNotesRepository(db)
	client, err := r.CreateExternalClient(ctx, owner, []byte{1}, "client")
	therapistDataOK(t, err)
	if got, err := r.Overview(ctx, other); err != nil || got.TotalSessions != 0 || got.LastSessionAt != nil || got.ExternalClients != 0 {
		t.Fatalf("empty overview: %+v %v", got, err)
	}
	tr := NewTherapistRepository(db)
	therapistDataOK(t, tr.LinkClient(ctx, owner, linked))
	if ok, err := tr.ApproveLink(ctx, linked, owner); err != nil || !ok {
		t.Fatalf("approve fixture: %v %v", ok, err)
	}
	day := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	var ids []uuid.UUID
	for i := 0; i < 3; i++ {
		s, err := r.CreateSession(ctx, &models.ClientSessionRow{TherapistID: owner, ExternalClientID: &client.ID, SessionDate: day.AddDate(0, 0, i), Status: models.ClientSessionCompleted})
		therapistDataOK(t, err)
		ids = append(ids, s.ID)
	}
	ls, err := r.CreateSession(ctx, &models.ClientSessionRow{TherapistID: owner, LinkedUserID: &linked, SessionDate: day.AddDate(0, 0, 4), Status: models.ClientSessionCompleted})
	therapistDataOK(t, err)
	for name, fn := range map[string]func() ([]*models.ClientSessionRow, error){
		"external": func() ([]*models.ClientSessionRow, error) {
			return r.ListSessionsForExternalClient(ctx, owner, client.ID, 2)
		},
		"linked": func() ([]*models.ClientSessionRow, error) {
			return r.ListSessionsForLinkedClient(ctx, owner, linked, 2)
		},
		"recent": func() ([]*models.ClientSessionRow, error) { return r.ListRecentSessions(ctx, owner, 2) },
	} {
		t.Run(name, func(t *testing.T) {
			got, err := fn()
			therapistDataOK(t, err)
			want := []uuid.UUID{ids[2], ids[1]}
			if name == "linked" {
				want = []uuid.UUID{ls.ID}
			}
			if name == "recent" {
				want = []uuid.UUID{ls.ID, ids[2]}
			}
			if len(got) != len(want) {
				t.Fatalf("count %d want %d", len(got), len(want))
			}
			for i, s := range got {
				if s.ID != want[i] || s.TherapistID != owner {
					t.Fatalf("order/owner: %+v", s)
				}
			}
		})
	}
	if got, err := r.ListSessionsForExternalClient(ctx, other, client.ID, 10); err != nil || len(got) != 0 {
		t.Fatalf("foreign external list: %+v %v", got, err)
	}
	if got, err := r.ListSessionsForLinkedClient(ctx, other, linked, 10); err != nil || len(got) != 0 {
		t.Fatalf("foreign linked list: %+v %v", got, err)
	}
	if got, err := r.ListRecentSessions(ctx, owner, 0); err != nil || len(got) != 0 {
		t.Fatalf("zero limit: %+v %v", got, err)
	}
	_, err = r.ListRecentSessions(ctx, owner, -1)
	therapistDataPG(t, err, "2201W")
	for name, s := range map[string]*models.ClientSessionRow{
		"neither": {TherapistID: owner, SessionDate: day, Status: models.ClientSessionPending},
		"both":    {TherapistID: owner, ExternalClientID: &client.ID, LinkedUserID: &linked, SessionDate: day, Status: models.ClientSessionPending},
	} {
		t.Run(name, func(t *testing.T) { _, err := r.CreateSession(ctx, s); therapistDataPG(t, err, "23514") })
	}
	missing := uuid.New()
	_, err = r.CreateSession(ctx, &models.ClientSessionRow{TherapistID: owner, ExternalClientID: &missing, SessionDate: day, Status: models.ClientSessionPending})
	therapistDataPG(t, err, "23503")
	therapistDataOK(t, db.QueryRow(ctx, `SELECT NOW()`).Scan(&day))
	therapistDataOK(t, func() error {
		_, err := db.Exec(ctx, `UPDATE client_sessions SET created_at=$1::timestamptz-INTERVAL '10 days' WHERE id=$2`, day, ids[0])
		return err
	}())
	therapistDataOK(t, func() error {
		_, err := db.Exec(ctx, `UPDATE client_sessions SET created_at=$1::timestamptz-INTERVAL '40 days' WHERE id=$2`, day, ids[1])
		return err
	}())
	o, err := r.Overview(ctx, owner)
	if err != nil || o.ExternalClients != 1 || o.LinkedClients != 1 || o.TotalSessions != 4 || o.SessionsThisWeek != 2 || o.SessionsThisMonth != 3 || o.LastSessionAt == nil {
		t.Fatalf("overview: %+v %v", o, err)
	}
}

func TestIntegration_TherapistDataClientAnalytics(t *testing.T) {
	t.Parallel()
	db := regressionDB(t)
	ctx := context.Background()
	_, owner := therapistDataFixture(t, db)
	r := NewTherapistRepository(db)
	client := regressionUser(t, db)
	quiet := regressionUser(t, db)
	var now time.Time
	therapistDataOK(t, db.QueryRow(ctx, `SELECT NOW()`).Scan(&now))
	for i := 0; i < 8; i++ {
		exportRegressionEntry(t, db, client, now.Add(-time.Duration(i)*time.Hour), 80, false, fmt.Sprintf("summary-%d", i), `[{"emotion":"calm","intensity":0.5},{"emotion":"excluded","intensity":0.49}]`, []string{"first", "second"})
	}
	exportRegressionEntry(t, db, client, now.Add(-10*24*time.Hour), 40, false, "prior", `[]`, []string{})
	exportRegressionEntry(t, db, client, now.Add(-time.Minute), 1, true, "crisis", `[]`, []string{})
	exportRegressionEntry(t, db, quiet, now.Add(time.Second), 100, false, "foreign", `[]`, []string{})
	therapistDataOK(t, func() error {
		_, err := db.Exec(ctx, `UPDATE users SET preferred_name='Preferred QA' WHERE id=$1`, client)
		return err
	}())
	for _, id := range []uuid.UUID{client, quiet} {
		therapistDataOK(t, r.LinkClient(ctx, owner, id))
		if ok, err := r.ApproveLink(ctx, id, owner); err != nil || !ok {
			t.Fatalf("approve: %v %v", ok, err)
		}
	}
	clients, err := r.ListClients(ctx, owner)
	if err != nil || len(clients) != 2 || clients[0].ClientID != quiet || clients[1].ClientID != client || clients[1].Name != "Preferred QA" || clients[1].EntryCount != 10 || clients[1].AvgMood30d == nil || *clients[1].AvgMood30d != 76 {
		t.Fatalf("client stats/order: %+v %v", clients, err)
	}
	if got, err := r.ClientDisplayName(ctx, client); err != nil || got != "Preferred QA" {
		t.Fatalf("preferred name: %q %v", got, err)
	}
	if got, err := r.ClientDisplayName(ctx, uuid.New()); err != nil || got != "" {
		t.Fatalf("missing name: %q %v", got, err)
	}
	if n, err := r.ClientEntryCount(ctx, client); err != nil || n != 9 {
		t.Fatalf("noncrisis count: %d %v", n, err)
	}
	entries, err := r.ClientRecentEntries(ctx, client)
	if err != nil || len(entries) != 5 {
		t.Fatalf("recent entries: %+v %v", entries, err)
	}
	for i, e := range entries {
		if e.Summary != fmt.Sprintf("summary-%d", i) || e.KeyQuote != "first" || !reflect.DeepEqual(e.Topics, []string{"fixture-topic"}) {
			t.Fatalf("recent order/quote: %+v", e)
		}
	}
	text, err := r.ClientRecentSummariesText(ctx, client, now.Add(-24*time.Hour))
	therapistDataOK(t, err)
	if strings.Count(text, "\n") != 7 || !strings.Contains(text, "mood 80] summary-0") || strings.Contains(text, "summary-7") || strings.Contains(text, "crisis") || strings.Contains(text, "foreign") {
		t.Fatalf("summary window: %q", text)
	}
	for _, tc := range []struct {
		current int
		trend   string
	}{{80, "improving"}, {35, "declining"}, {42, "stable"}} {
		therapistDataOK(t, func() error {
			_, err := db.Exec(ctx, `UPDATE entry_analysis SET mood_score=$2 WHERE entry_id IN (SELECT id FROM entries WHERE user_id=$1 AND created_at >= $3) AND is_crisis=false`, client, tc.current, now.Add(-24*time.Hour))
			return err
		}())
		avg, emotions, trend, err := r.ClientMoodStats(ctx, client)
		if err != nil || avg == nil || *avg != tc.current || trend != tc.trend || !reflect.DeepEqual(emotions, []string{"calm"}) {
			t.Fatalf("mood trend: %v %v %s %v", avg, emotions, trend, err)
		}
	}
	if avg, emotions, trend, err := r.ClientMoodStats(ctx, uuid.New()); err != nil || avg != nil || len(emotions) != 0 || trend != "insufficient_data" {
		t.Fatalf("empty mood: %v %v %s %v", avg, emotions, trend, err)
	}
}

func TestIntegration_TherapistDataTherapyPersistence(t *testing.T) {
	t.Parallel()
	db := regressionDB(t)
	ctx := context.Background()
	owner, other := regressionUser(t, db), regressionUser(t, db)
	r := NewTherapyRepository(db)
	mood := 65.5
	snapshot := models.TherapyContextSnapshot{MoodAvg30d: &mood, TopEmotions: []string{"calm"}, TopTopics: []string{"work"}, RecentSummaries: []string{"journal"}, PastSessionSummaries: []string{"previous"}, TopPeople: []string{"QA"}, Country: "IN", VoiceLanguage: "hindi"}
	s, err := r.Create(ctx, owner, models.PersonaCBT, snapshot, 29900)
	therapistDataOK(t, err)
	if s.Status != models.TherapyStatusActive || s.Persona != models.PersonaCBT || s.BillingAmountPaise != 29900 || s.EndedAt != nil || s.DurationSec != nil || s.PostSessionSummary != nil || s.SessionMoodScore != nil || s.TurnCount != 0 || s.CrisisWarnings != 0 || !reflect.DeepEqual(s.ContextSnapshot, snapshot) || s.ExpiresAt.Sub(s.StartedAt) != time.Hour || s.TimeRemainingSec < 3500 || s.TimeRemainingSec > 3600 {
		t.Fatalf("initial session: %+v", s)
	}
	for _, id := range []uuid.UUID{s.ID, uuid.New()} {
		if got, err := r.GetByID(ctx, id, other); err != nil || got != nil {
			t.Fatalf("foreign/missing therapy: %+v %v", got, err)
		}
	}
	for i := 1; i <= 2; i++ {
		if n, err := r.IncrementTurn(ctx, s.ID); err != nil || n != i {
			t.Fatalf("turn: %d %v", n, err)
		}
		if n, err := r.IncrementCrisisWarning(ctx, s.ID); err != nil || n != i {
			t.Fatalf("warning: %d %v", n, err)
		}
	}
	for _, fn := range []func(context.Context, uuid.UUID) (int, error){r.IncrementTurn, r.IncrementCrisisWarning} {
		if _, err := fn(ctx, uuid.New()); !errors.Is(err, pgx.ErrNoRows) {
			t.Fatalf("missing increment: %v", err)
		}
	}
	var messageIDs []uuid.UUID
	for i, tc := range []struct{ role, content, mode string }{{"user", "Text QA", "text"}, {"assistant", "Response QA", "system"}, {"user", "Voice QA", "voice"}} {
		m, err := r.AddMessage(ctx, s.ID, tc.role, tc.content, tc.mode)
		therapistDataOK(t, err)
		if m.SessionID != s.ID || m.Role != tc.role || m.Content != tc.content || m.InputMode != tc.mode || m.TTSUrl != nil {
			t.Fatalf("message: %+v", m)
		}
		messageIDs = append(messageIDs, m.ID)
		therapistDataOK(t, func() error {
			_, err := db.Exec(ctx, `UPDATE therapy_session_messages SET created_at=$1 WHERE id=$2`, s.StartedAt.Add(time.Duration(i)*time.Second), m.ID)
			return err
		}())
	}
	msgs, err := r.ListMessages(ctx, s.ID)
	if err != nil || len(msgs) != 3 {
		t.Fatalf("messages: %+v %v", msgs, err)
	}
	for i, m := range msgs {
		if m.ID != messageIDs[i] {
			t.Fatalf("message order: %+v", msgs)
		}
	}
	if got, err := r.ListMessages(ctx, uuid.New()); err != nil || len(got) != 0 {
		t.Fatalf("missing messages: %+v %v", got, err)
	}
	for _, tc := range []struct{ role, mode, code string }{{"invalid", "text", "23514"}, {"user", "invalid", "23514"}} {
		_, err := r.AddMessage(ctx, s.ID, tc.role, "QA", tc.mode)
		therapistDataPG(t, err, tc.code)
	}
	_, err = r.AddMessage(ctx, uuid.New(), "user", "QA", "text")
	therapistDataPG(t, err, "23503")
	a := &models.TherapySessionAnalysis{MoodScore: 78, EmotionalTone: []models.EmotionalTone{{Emotion: "hope", Intensity: 0.8}}, Topics: []string{"work"}, KeyInsights: []string{"pause"}, SessionNarrative: "QA narrative"}
	therapistDataOK(t, r.SetSessionAnalysis(ctx, s.ID, a))
	bad := *a
	bad.MoodScore = 101
	therapistDataPG(t, r.SetSessionAnalysis(ctx, s.ID, &bad), "23514")
	ended := s.StartedAt.Add(12 * time.Minute)
	duration := 720
	therapistDataOK(t, r.UpdateStatus(ctx, s.ID, models.TherapyStatusCompleted, &ended, &duration))
	got, err := r.GetByID(ctx, s.ID, owner)
	if err != nil || got.Status != models.TherapyStatusCompleted || got.TimeRemainingSec != 0 || got.EndedAt == nil || !got.EndedAt.Equal(ended) || got.DurationSec == nil || *got.DurationSec != duration || got.TurnCount != 2 || got.CrisisWarnings != 2 || got.SessionMoodScore == nil || *got.SessionMoodScore != 78 || got.PostSessionSummary == nil || *got.PostSessionSummary != a.SessionNarrative || !reflect.DeepEqual(got.SessionEmotionalTone, a.EmotionalTone) || !reflect.DeepEqual(got.SessionTopics, a.Topics) || !reflect.DeepEqual(got.SessionKeyInsights, a.KeyInsights) {
		t.Fatalf("analysis/end persistence: %+v %v", got, err)
	}
	therapistDataPG(t, r.UpdateStatus(ctx, s.ID, models.TherapySessionStatus("invalid"), nil, nil), "22P02")
	_, err = r.Create(ctx, uuid.New(), models.PersonaComforting, models.TherapyContextSnapshot{}, 0)
	therapistDataPG(t, err, "23503")
	_, err = r.Create(ctx, owner, models.TherapyPersona("invalid"), models.TherapyContextSnapshot{}, 0)
	therapistDataPG(t, err, "23514")
	nan := math.NaN()
	_, err = r.Create(ctx, owner, models.PersonaComforting, models.TherapyContextSnapshot{MoodAvg30d: &nan}, 0)
	if err == nil || !strings.Contains(err.Error(), "marshal snapshot") {
		t.Fatalf("invalid snapshot: %v", err)
	}
	badTone := *a
	badTone.EmotionalTone = []models.EmotionalTone{{Emotion: "invalid", Intensity: math.NaN()}}
	if err := r.SetSessionAnalysis(ctx, s.ID, &badTone); err == nil || !strings.Contains(err.Error(), "marshal tone") {
		t.Fatalf("invalid tone: %v", err)
	}
	therapistDataOK(t, func() error {
		_, err := db.Exec(ctx, `UPDATE therapy_sessions SET context_snapshot='[]'::jsonb WHERE id=$1`, s.ID)
		return err
	}())
	if got, err := r.GetByID(ctx, s.ID, owner); err == nil || got != nil || !strings.Contains(err.Error(), "unmarshal context_snapshot") {
		t.Fatalf("malformed snapshot: %+v %v", got, err)
	}
	if got, err := r.ListByUser(ctx, owner); err == nil || got != nil {
		t.Fatalf("malformed list scan: %+v %v", got, err)
	}
}

func TestIntegration_TherapistDataTherapyHistoryCountsExpiry(t *testing.T) {
	t.Parallel()
	db := regressionDB(t)
	ctx := context.Background()
	owner, other := regressionUser(t, db), regressionUser(t, db)
	r := NewTherapyRepository(db)
	var now time.Time
	therapistDataOK(t, db.QueryRow(ctx, `SELECT NOW()`).Scan(&now))
	var ids []uuid.UUID
	for i := 0; i < 23; i++ {
		s, err := r.Create(ctx, owner, models.PersonaMindful, models.TherapyContextSnapshot{}, 0)
		therapistDataOK(t, err)
		ids = append(ids, s.ID)
		therapistDataOK(t, func() error {
			_, err := db.Exec(ctx, `UPDATE therapy_sessions SET created_at=$1 WHERE id=$2`, now.Add(time.Duration(i)*time.Minute), s.ID)
			return err
		}())
		if i < 5 {
			therapistDataOK(t, r.UpdateStatus(ctx, s.ID, models.TherapyStatusCompleted, nil, nil))
			if i != 0 {
				summary := fmt.Sprintf("summary-%d", i)
				if i == 1 {
					summary = ""
				}
				therapistDataOK(t, r.SetPostSessionSummary(ctx, s.ID, summary))
			}
		}
	}
	foreign, err := r.Create(ctx, other, models.PersonaRational, models.TherapyContextSnapshot{}, 0)
	therapistDataOK(t, err)
	therapistDataOK(t, r.UpdateStatus(ctx, foreign.ID, models.TherapyStatusCompleted, nil, nil))
	therapistDataOK(t, r.SetPostSessionSummary(ctx, foreign.ID, "foreign"))
	therapistDataOK(t, r.SetPostSessionSummary(ctx, ids[22], "active-not-history"))
	summaries, err := r.PastCompletedSummaries(ctx, owner, 2)
	if err != nil || !reflect.DeepEqual(summaries, []string{"summary-3", "summary-4"}) {
		t.Fatalf("history filtering/order: %v %v", summaries, err)
	}
	if got, err := r.PastCompletedSummaries(ctx, owner, 0); err != nil || len(got) != 0 {
		t.Fatalf("zero summaries: %v %v", got, err)
	}
	_, err = r.PastCompletedSummaries(ctx, owner, -1)
	therapistDataPG(t, err, "2201W")
	list, err := r.ListByUser(ctx, owner)
	if err != nil || len(list) != 20 {
		t.Fatalf("session list cap: %d %v", len(list), err)
	}
	for i, s := range list {
		if s.ID != ids[22-i] || s.UserID != owner {
			t.Fatalf("list order/owner at %d: %+v", i, s)
		}
	}
	if n, err := r.CountAll(ctx, owner); err != nil || n != 23 {
		t.Fatalf("all count: %d %v", n, err)
	}
	therapistDataOK(t, func() error {
		_, err := db.Exec(ctx, `UPDATE therapy_sessions SET started_at=date_trunc('month',$1::timestamptz)-INTERVAL '1 microsecond' WHERE id=$2`, now, ids[0])
		return err
	}())
	if n, err := r.CountThisMonth(ctx, owner); err != nil || n != 22 {
		t.Fatalf("month boundary: %d %v", n, err)
	}
	if n, err := r.CountAll(ctx, other); err != nil || n != 1 {
		t.Fatalf("foreign count: %d %v", n, err)
	}
	therapistDataOK(t, func() error {
		_, err := db.Exec(ctx, `UPDATE therapy_sessions SET expires_at=$1::timestamptz-INTERVAL '1 second' WHERE id=$2`, now, ids[22])
		return err
	}())
	if got, err := r.GetByID(ctx, ids[22], owner); err != nil || got.Status != models.TherapyStatusActive || got.TimeRemainingSec != 0 {
		t.Fatalf("expired active remaining: %+v %v", got, err)
	}
	for _, status := range []models.TherapySessionStatus{models.TherapyStatusExpired, models.TherapyStatusCrisisDetected} {
		therapistDataOK(t, r.UpdateStatus(ctx, ids[22], status, nil, nil))
		if got, err := r.GetByID(ctx, ids[22], owner); err != nil || got.Status != status || got.TimeRemainingSec != 0 || got.EndedAt != nil || got.DurationSec != nil {
			t.Fatalf("closed status: %+v %v", got, err)
		}
	}
}

func TestIntegration_TherapistDataCanceledOperations(t *testing.T) {
	t.Parallel()
	db := regressionDB(t)
	background := context.Background()
	user, therapist := therapistDataFixture(t, db)
	client := regressionUser(t, db)
	tr, nr, ar := NewTherapistRepository(db), NewTherapistNotesRepository(db), NewTherapyRepository(db)
	ec, err := nr.CreateExternalClient(background, therapist, []byte{1}, "client")
	therapistDataOK(t, err)
	note, err := nr.CreateSession(background, &models.ClientSessionRow{TherapistID: therapist, ExternalClientID: &ec.ID, SessionDate: time.Now(), Status: models.ClientSessionPending})
	therapistDataOK(t, err)
	session, err := ar.Create(background, user, models.PersonaComforting, models.TherapyContextSnapshot{}, 0)
	therapistDataOK(t, err)
	ctx, cancel := context.WithCancel(background)
	cancel()
	operations := map[string]func() error{
		"therapist.GetByUserID":               func() error { _, e := tr.GetByUserID(ctx, user); return e },
		"therapist.Register":                  func() error { _, e := tr.Register(ctx, user, "changed", "changed@example.test", ""); return e },
		"therapist.LinkClient":                func() error { return tr.LinkClient(ctx, therapist, client) },
		"therapist.ApproveLink":               func() error { _, e := tr.ApproveLink(ctx, client, therapist); return e },
		"therapist.DeclineLink":               func() error { _, e := tr.DeclineLink(ctx, client, therapist); return e },
		"therapist.ListPendingRequests":       func() error { _, e := tr.ListPendingRequests(ctx, client); return e },
		"therapist.UnlinkClient":              func() error { return tr.UnlinkClient(ctx, therapist, client) },
		"therapist.ListClients":               func() error { _, e := tr.ListClients(ctx, therapist); return e },
		"therapist.GetClientLink":             func() error { _, e := tr.GetClientLink(ctx, therapist, client); return e },
		"therapist.ClientRecentEntries":       func() error { _, e := tr.ClientRecentEntries(ctx, client); return e },
		"therapist.ClientEntryCount":          func() error { _, e := tr.ClientEntryCount(ctx, client); return e },
		"therapist.ClientDisplayName":         func() error { _, e := tr.ClientDisplayName(ctx, client); return e },
		"therapist.ClientRecentSummariesText": func() error { _, e := tr.ClientRecentSummariesText(ctx, client, time.Now()); return e },
		"notes.GetWrappedDEK":                 func() error { _, e := nr.GetWrappedDEK(ctx, therapist); return e },
		"notes.InsertWrappedDEK":              func() error { _, e := nr.InsertWrappedDEK(ctx, therapist, []byte{9}); return e },
		"notes.CreateExternalClient":          func() error { _, e := nr.CreateExternalClient(ctx, therapist, []byte{9}, "minor"); return e },
		"notes.ListExternalClients":           func() error { _, e := nr.ListExternalClients(ctx, therapist, true); return e },
		"notes.GetExternalClient":             func() error { _, e := nr.GetExternalClient(ctx, therapist, ec.ID); return e },
		"notes.UpdateExternalClient":          func() error { _, e := nr.UpdateExternalClient(ctx, therapist, ec.ID, []byte{9}, nil, nil); return e },
		"notes.DeleteExternalClient":          func() error { _, e := nr.DeleteExternalClient(ctx, therapist, ec.ID); return e },
		"notes.CreateSession": func() error {
			_, e := nr.CreateSession(ctx, &models.ClientSessionRow{TherapistID: therapist, ExternalClientID: &ec.ID, SessionDate: time.Now(), Status: models.ClientSessionPending})
			return e
		},
		"notes.GetSession":                     func() error { _, e := nr.GetSession(ctx, therapist, note.ID); return e },
		"notes.GetSessionByID":                 func() error { _, e := nr.GetSessionByID(ctx, note.ID); return e },
		"notes.ListSessionsForExternalClient":  func() error { _, e := nr.ListSessionsForExternalClient(ctx, therapist, ec.ID, 10); return e },
		"notes.ListSessionsForLinkedClient":    func() error { _, e := nr.ListSessionsForLinkedClient(ctx, therapist, client, 10); return e },
		"notes.ListRecentSessions":             func() error { _, e := nr.ListRecentSessions(ctx, therapist, 10); return e },
		"notes.SetSessionProcessing":           func() error { _, e := nr.SetSessionProcessing(ctx, note.ID); return e },
		"notes.CompleteSessionOCR":             func() error { return nr.CompleteSessionOCR(ctx, note.ID, []byte{9}, []byte{9}) },
		"notes.SetSessionFailed":               func() error { return nr.SetSessionFailed(ctx, note.ID, "canceled", true) },
		"notes.UpdateSessionBullets":           func() error { _, e := nr.UpdateSessionBullets(ctx, therapist, note.ID, []byte{9}); return e },
		"notes.UpdateSessionSummary":           func() error { _, e := nr.UpdateSessionSummary(ctx, therapist, note.ID, []byte{9}); return e },
		"notes.DeleteSession":                  func() error { _, _, e := nr.DeleteSession(ctx, therapist, note.ID); return e },
		"notes.Overview":                       func() error { _, e := nr.Overview(ctx, therapist); return e },
		"notes.AcceptUserTerms":                func() error { return nr.AcceptUserTerms(ctx, client, "canceled") },
		"notes.GetUserTerms":                   func() error { _, _, e := nr.GetUserTerms(ctx, client); return e },
		"notes.AcceptTherapistClientConsent":   func() error { return nr.AcceptTherapistClientConsent(ctx, therapist, "canceled") },
		"notes.TherapistClientConsentAccepted": func() error { _, e := nr.TherapistClientConsentAccepted(ctx, therapist); return e },
		"therapy.Create": func() error {
			_, e := ar.Create(ctx, user, models.PersonaComforting, models.TherapyContextSnapshot{}, 0)
			return e
		},
		"therapy.GetByID":                func() error { _, e := ar.GetByID(ctx, session.ID, user); return e },
		"therapy.ListByUser":             func() error { _, e := ar.ListByUser(ctx, user); return e },
		"therapy.UpdateStatus":           func() error { return ar.UpdateStatus(ctx, session.ID, models.TherapyStatusCompleted, nil, nil) },
		"therapy.IncrementTurn":          func() error { _, e := ar.IncrementTurn(ctx, session.ID); return e },
		"therapy.IncrementCrisisWarning": func() error { _, e := ar.IncrementCrisisWarning(ctx, session.ID); return e },
		"therapy.PastCompletedSummaries": func() error { _, e := ar.PastCompletedSummaries(ctx, user, 3); return e },
		"therapy.SetSessionAnalysis": func() error {
			return ar.SetSessionAnalysis(ctx, session.ID, &models.TherapySessionAnalysis{MoodScore: 50})
		},
		"therapy.SetPostSessionSummary": func() error { return ar.SetPostSessionSummary(ctx, session.ID, "canceled") },
		"therapy.CountAll":              func() error { _, e := ar.CountAll(ctx, user); return e },
		"therapy.CountThisMonth":        func() error { _, e := ar.CountThisMonth(ctx, user); return e },
		"therapy.AddMessage":            func() error { _, e := ar.AddMessage(ctx, session.ID, "user", "canceled", "text"); return e },
		"therapy.ListMessages":          func() error { _, e := ar.ListMessages(ctx, session.ID); return e },
	}
	for name, operation := range operations {
		t.Run(name, func(t *testing.T) {
			if err := operation(); !errors.Is(err, context.Canceled) {
				t.Fatalf("canceled operation did not preserve context error: %v", err)
			}
		})
	}
	// Read through an uncanceled context to prove canceled writes did not persist.
	gotProfile, err := tr.GetByUserID(background, user)
	if err != nil || gotProfile.Name != "QA therapist" {
		t.Fatalf("canceled profile mutation: %+v %v", gotProfile, err)
	}
	gotClient, err := nr.GetExternalClient(background, therapist, ec.ID)
	if err != nil || gotClient == nil || !bytes.Equal(gotClient.NameEnc, []byte{1}) {
		t.Fatalf("canceled client mutation: %+v %v", gotClient, err)
	}
	gotNote, err := nr.GetSession(background, therapist, note.ID)
	if err != nil || gotNote == nil || gotNote.Status != models.ClientSessionPending || gotNote.RetryCount != 0 || gotNote.BulletsEnc != nil || gotNote.SummaryEnc != nil {
		t.Fatalf("canceled notes mutation: %+v %v", gotNote, err)
	}
	gotSession, err := ar.GetByID(background, session.ID, user)
	if err != nil || gotSession == nil || gotSession.Status != models.TherapyStatusActive || gotSession.TurnCount != 0 || gotSession.CrisisWarnings != 0 || gotSession.PostSessionSummary != nil {
		t.Fatalf("canceled therapy mutation: %+v %v", gotSession, err)
	}
	if n, err := ar.CountAll(background, user); err != nil || n != 1 {
		t.Fatalf("canceled create persisted: %d %v", n, err)
	}
	if messages, err := ar.ListMessages(background, session.ID); err != nil || len(messages) != 0 {
		t.Fatalf("canceled message persisted: %+v %v", messages, err)
	}
}

func TestIntegration_TherapistDataMoodStatsCancellation(t *testing.T) {
	db := regressionDB(t)
	client := regressionUser(t, db)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	avg, emotions, trend, err := NewTherapistRepository(db).ClientMoodStats(ctx, client)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled analytics must return context.Canceled, not successful empty data: avg=%v emotions=%v trend=%q err=%v", avg, emotions, trend, err)
	}
}

func TestIntegration_TherapistDataMoodStatsNullablePeriods(t *testing.T) {
	t.Parallel()
	db := regressionDB(t)
	ctx := context.Background()
	r := NewTherapistRepository(db)
	var now time.Time
	therapistDataOK(t, db.QueryRow(ctx, `SELECT NOW()`).Scan(&now))
	for _, tc := range []struct {
		name           string
		current, prior bool
		wantAverage    *int
		wantEmotions   []string
		wantTrend      string
	}{
		{name: "empty", wantTrend: "insufficient_data"},
		{name: "current only", current: true, wantAverage: therapistDataInt(70), wantEmotions: []string{"calm"}, wantTrend: "insufficient_data"},
		{name: "prior only", prior: true, wantEmotions: []string{"calm"}, wantTrend: "insufficient_data"},
		{name: "both periods", current: true, prior: true, wantAverage: therapistDataInt(70), wantEmotions: []string{"calm"}, wantTrend: "improving"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			user := regressionUser(t, db)
			if tc.current {
				exportRegressionEntry(t, db, user, now.Add(-time.Hour), 70, false, "current", `[{"emotion":"calm","intensity":0.8}]`, []string{})
			}
			if tc.prior {
				exportRegressionEntry(t, db, user, now.Add(-10*24*time.Hour), 40, false, "prior", `[{"emotion":"calm","intensity":0.8}]`, []string{})
			}
			// Legacy malformed tone data must not discard otherwise usable mood data.
			exportRegressionEntry(t, db, user, now.Add(-20*24*time.Hour), 90, false, "legacy tone", `{"unexpected":"shape"}`, []string{})
			avg, emotions, trend, err := r.ClientMoodStats(ctx, user)
			if err != nil || !reflect.DeepEqual(avg, tc.wantAverage) || len(emotions) != len(tc.wantEmotions) || (len(emotions) > 0 && !reflect.DeepEqual(emotions, tc.wantEmotions)) || trend != tc.wantTrend {
				t.Fatalf("nullable periods: avg=%v emotions=%v trend=%q err=%v", avg, emotions, trend, err)
			}
		})
	}
}

func therapistDataInt(value int) *int { return &value }

func TestIntegration_TherapistDataMoodStatsPostgresErrors(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct{ name, column, stage string }{
		{"current aggregate", "mood_score", "current"},
		{"emotion query", "emotional_tone", "emotions"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := regressionDB(t)
			ctx := context.Background()
			user := regressionUser(t, db)
			// Only the disposable QA schema is altered to provoke a real SQL error.
			_, err := db.Exec(ctx, "ALTER TABLE entry_analysis RENAME COLUMN "+pgx.Identifier{tc.column}.Sanitize()+" TO qa_missing_column")
			therapistDataOK(t, err)
			avg, emotions, trend, err := NewTherapistRepository(db).ClientMoodStats(ctx, user)
			therapistDataPG(t, err, "42703")
			if !strings.Contains(err.Error(), "therapist.ClientMoodStats "+tc.stage+":") || avg != nil || emotions != nil || trend != "" {
				t.Fatalf("SQL error must remain wrapped without partial success: avg=%v emotions=%v trend=%q err=%v", avg, emotions, trend, err)
			}
		})
	}
}
