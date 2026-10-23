//go:build integration

package repositories

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestIntegration_NudgeDeliveryDevicesAndCheckin(t *testing.T) {
	db := regressionDB(t)
	ctx := context.Background()
	owner, other := regressionUser(t, db), regressionUser(t, db)
	r := NewNudgeRepository(db)
	entry := uuid.New()
	if _, err := db.Exec(ctx, `INSERT INTO entries(id,user_id,audio_key,status) VALUES($1,$2,'fixture/nudge/' || gen_random_uuid()::text,'completed')`, entry, owner); err != nil {
		t.Fatal(err)
	}
	due := time.Now().Add(-time.Hour)
	morning, err := r.Create(ctx, owner, &entry, "QA morning", due, "UTC")
	if err != nil || morning == nil || morning.NudgeType != "morning" || morning.Message != "QA morning" || morning.EntryID == nil || *morning.EntryID != entry || morning.SentAt != nil {
		t.Fatalf("create: %+v %v", morning, err)
	}
	future, err := r.CreateWithType(ctx, owner, nil, "QA future", time.Now().Add(time.Hour), "UTC", "reengagement")
	if err != nil {
		t.Fatal(err)
	}
	pending, err := r.PendingDue(ctx)
	if err != nil || len(pending) != 1 || pending[0].ID != morning.ID {
		t.Fatalf("due: %+v %v", pending, err)
	}
	if err := r.MarkSent(ctx, morning.ID); err != nil {
		t.Fatal(err)
	}
	var sent *time.Time
	var status string
	if err := db.QueryRow(ctx, `SELECT status,sent_at FROM nudges WHERE id=$1`, morning.ID).Scan(&status, &sent); err != nil || status != "sent" || sent == nil {
		t.Fatalf("sent: %s %v %v", status, sent, err)
	}
	if err := r.MarkFailed(ctx, future.ID, "QA failure"); err != nil {
		t.Fatal(err)
	}
	var failure string
	if err := db.QueryRow(ctx, `SELECT status,error_msg FROM nudges WHERE id=$1`, future.ID).Scan(&status, &failure); err != nil || status != "failed" || failure != "QA failure" {
		t.Fatalf("failed: %s %s %v", status, failure, err)
	}
	if pending, err := r.PendingDue(ctx); err != nil || len(pending) != 0 {
		t.Fatalf("terminal excluded: %+v %v", pending, err)
	}
	if exists, err := r.HasPendingCheckinForEntry(ctx, entry); err != nil || exists {
		t.Fatalf("no checkin: %v %v", exists, err)
	}
	checkin, err := r.CreateWithType(ctx, owner, &entry, "QA checkin", due, "UTC", "checkin")
	if err != nil {
		t.Fatal(err)
	}
	if exists, err := r.HasPendingCheckinForEntry(ctx, entry); err != nil || !exists {
		t.Fatalf("pending checkin: %v %v", exists, err)
	}
	if err := r.MarkSent(ctx, checkin.ID); err != nil {
		t.Fatal(err)
	}
	if exists, err := r.HasPendingCheckinForEntry(ctx, entry); err != nil || exists {
		t.Fatalf("sent checkin: %v %v", exists, err)
	}
	if err := r.UpsertDevice(ctx, owner, "qa-token", "android"); err != nil {
		t.Fatal(err)
	}
	if err := r.UpsertDevice(ctx, owner, "qa-token", "android"); err != nil {
		t.Fatal(err)
	}
	if tokens, err := r.GetDeviceTokens(ctx, owner); err != nil || len(tokens) != 1 || tokens[0] != "qa-token" {
		t.Fatalf("tokens: %v %v", tokens, err)
	}
	if err := r.UpsertDevice(ctx, other, "qa-token", "ios"); err != nil {
		t.Fatal(err)
	}
	if tokens, err := r.GetDeviceTokens(ctx, owner); err != nil || len(tokens) != 0 {
		t.Fatalf("transferred token: %v %v", tokens, err)
	}
	if tokens, err := r.GetDeviceTokens(ctx, other); err != nil || len(tokens) != 1 {
		t.Fatalf("new token owner: %v %v", tokens, err)
	}
	if _, err := r.CreateWithType(ctx, uuid.New(), nil, "bad", due, "UTC", "morning"); err == nil {
		t.Fatal("missing user accepted")
	}
	if err := r.UpsertDevice(ctx, uuid.New(), "missing-user-token", "android"); err == nil {
		t.Fatal("missing device owner accepted")
	}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := r.Create(canceled, owner, nil, "", due, "UTC"); !errors.Is(err, context.Canceled) {
		t.Fatalf("create cancel: %v", err)
	}
	if _, err := r.PendingDue(canceled); !errors.Is(err, context.Canceled) {
		t.Fatalf("due cancel: %v", err)
	}
	if err := r.MarkSent(canceled, morning.ID); !errors.Is(err, context.Canceled) {
		t.Fatalf("sent cancel: %v", err)
	}
	if err := r.MarkFailed(canceled, morning.ID, ""); !errors.Is(err, context.Canceled) {
		t.Fatalf("failed cancel: %v", err)
	}
	if err := r.UpsertDevice(canceled, owner, "qa-cancel", "android"); !errors.Is(err, context.Canceled) {
		t.Fatalf("device cancel: %v", err)
	}
	if _, err := r.GetDeviceTokens(canceled, owner); !errors.Is(err, context.Canceled) {
		t.Fatalf("tokens cancel: %v", err)
	}
	if _, err := r.HasPendingCheckinForEntry(canceled, entry); !errors.Is(err, context.Canceled) {
		t.Fatalf("checkin cancel: %v", err)
	}
}

func TestIntegration_NudgeCandidateFiltersAndTypicalHour(t *testing.T) {
	db := regressionDB(t)
	ctx := context.Background()
	owner := regressionUser(t, db)
	r := NewNudgeRepository(db)
	var hour int
	if err := db.QueryRow(ctx, `SELECT EXTRACT(HOUR FROM NOW() AT TIME ZONE 'UTC')::int`).Scan(&hour); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(ctx, `UPDATE users SET timezone='UTC',fcm_nudge_hour=$2,nudge_enabled=true,plan='pro',plan_expires_at=NOW()+INTERVAL '2 days' WHERE id=$1`, owner, hour); err != nil {
		t.Fatal(err)
	}
	if err := r.UpsertDevice(ctx, owner, "qa-candidate", "android"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(ctx, `INSERT INTO entries(user_id,audio_key,status,created_at) SELECT $1,'fixture/old/' || gen_random_uuid()::text,'completed', DATE_TRUNC('day',NOW() AT TIME ZONE 'UTC') AT TIME ZONE 'UTC' - INTERVAL '1 day' + INTERVAL '8 hours' FROM generate_series(1,3)`, owner); err != nil {
		t.Fatal(err)
	}
	typical, ok, err := r.TypicalEntryHour(ctx, owner, "UTC")
	if err != nil || !ok || typical != 8 {
		t.Fatalf("typical: %d %v %v", typical, ok, err)
	}
	if _, ok, err := r.TypicalEntryHour(ctx, uuid.New(), "UTC"); err != nil || ok {
		t.Fatalf("empty signal: %v %v", ok, err)
	}
	lapsed, err := r.LapsedUsersAtNudgeHour(ctx, 1)
	if err != nil || len(lapsed) != 1 || lapsed[0].UserID != owner || lapsed[0].Timezone != "UTC" {
		t.Fatalf("lapsed: %+v %v", lapsed, err)
	}
	risk, err := r.StreakRiskUsersAtLocalHour(ctx, hour)
	if err != nil || len(risk) != 1 || risk[0].UserID != owner {
		t.Fatalf("risk: %+v %v", risk, err)
	}
	expiring, err := r.PlanExpiringSoonUsersAtLocalHour(ctx, hour, 3)
	if err != nil || len(expiring) != 1 || expiring[0].UserID != owner || expiring[0].Plan != "pro" {
		t.Fatalf("expiring: %+v %v", expiring, err)
	}
	if _, err := r.CreateWithType(ctx, owner, nil, "QA dedup", time.Now(), "UTC", "reengagement"); err != nil {
		t.Fatal(err)
	}
	if got, err := r.LapsedUsersAtNudgeHour(ctx, 1); err != nil || len(got) != 0 {
		t.Fatalf("reengagement dedup: %+v %v", got, err)
	}
	if _, err := r.CreateWithType(ctx, owner, nil, "QA dedup", time.Now(), "UTC", "streak_risk"); err != nil {
		t.Fatal(err)
	}
	if got, err := r.StreakRiskUsersAtLocalHour(ctx, hour); err != nil || len(got) != 0 {
		t.Fatalf("risk dedup: %+v %v", got, err)
	}
	if _, err := r.CreateWithType(ctx, owner, nil, "QA dedup", time.Now(), "UTC", "plan_expiring_soon"); err != nil {
		t.Fatal(err)
	}
	if got, err := r.PlanExpiringSoonUsersAtLocalHour(ctx, hour, 3); err != nil || len(got) != 0 {
		t.Fatalf("expiry dedup: %+v %v", got, err)
	}
	if _, err := db.Exec(ctx, `UPDATE users SET plan_expires_at=NOW()-INTERVAL '1 day' WHERE id=$1`, owner); err != nil {
		t.Fatal(err)
	}
	expired, err := r.PlanExpiredUsersAtLocalHour(ctx, hour)
	if err != nil || len(expired) != 1 || expired[0].UserID != owner {
		t.Fatalf("expired: %+v %v", expired, err)
	}
	if _, err := r.CreateWithType(ctx, owner, nil, "QA dedup", time.Now(), "UTC", "plan_expired"); err != nil {
		t.Fatal(err)
	}
	if got, err := r.PlanExpiredUsersAtLocalHour(ctx, hour); err != nil || len(got) != 0 {
		t.Fatalf("expired dedup: %+v %v", got, err)
	}
	if _, err := db.Exec(ctx, `UPDATE users SET nudge_enabled=false WHERE id=$1`, owner); err != nil {
		t.Fatal(err)
	}
	if got, err := r.StreakRiskUsersAtLocalHour(ctx, hour); err != nil || len(got) != 0 {
		t.Fatalf("disabled: %+v %v", got, err)
	}
	if _, _, err := r.TypicalEntryHour(ctx, owner, "invalid/timezone"); err == nil {
		t.Fatal("invalid timezone accepted")
	}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := r.LapsedUsersAtNudgeHour(canceled, 1); !errors.Is(err, context.Canceled) {
		t.Fatalf("lapse cancel: %v", err)
	}
	if _, err := r.StreakRiskUsersAtLocalHour(canceled, hour); !errors.Is(err, context.Canceled) {
		t.Fatalf("risk cancel: %v", err)
	}
	if _, err := r.PlanExpiringSoonUsersAtLocalHour(canceled, hour, 3); !errors.Is(err, context.Canceled) {
		t.Fatalf("expiry cancel: %v", err)
	}
	if _, err := r.PlanExpiredUsersAtLocalHour(canceled, hour); !errors.Is(err, context.Canceled) {
		t.Fatalf("expired cancel: %v", err)
	}
	if _, _, err := r.TypicalEntryHour(canceled, owner, "UTC"); !errors.Is(err, context.Canceled) {
		t.Fatalf("typical cancel: %v", err)
	}
}
