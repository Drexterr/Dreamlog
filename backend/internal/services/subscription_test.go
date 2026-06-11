package services_test

import (
	"context"
	"testing"
	"time"

	"github.com/dreamlog/backend/internal/models"
	"github.com/dreamlog/backend/internal/services"
	"github.com/google/uuid"
)

// ── fakes ─────────────────────────────────────────────────────────────────────

type fakeSubUserRepo struct {
	monthlyCount int
	updatedPlan  models.Plan
	updatedAt    *time.Time
}

func (f *fakeSubUserRepo) UpdatePlan(_ context.Context, _ uuid.UUID, plan models.Plan, expiresAt *time.Time) (*models.User, error) {
	f.updatedPlan = plan
	f.updatedAt = expiresAt
	return &models.User{Plan: plan, PlanExpiresAt: expiresAt}, nil
}

func (f *fakeSubUserRepo) CountMonthlyEntries(_ context.Context, _ uuid.UUID) (int, error) {
	return f.monthlyCount, nil
}

type fakeSubShareRepo struct {
	monthlyCount int
}

func (f *fakeSubShareRepo) CountMonthlyByUser(_ context.Context, _ uuid.UUID) (int, error) {
	return f.monthlyCount, nil
}

func newSvc(entryCount, shareCount int) *services.SubscriptionService {
	return services.NewSubscriptionService(
		&fakeSubUserRepo{monthlyCount: entryCount},
		&fakeSubShareRepo{monthlyCount: shareCount},
	)
}

// ── GetPlanDetails ────────────────────────────────────────────────────────────

func TestGetPlanDetails_Free(t *testing.T) {
	svc := newSvc(0, 0)
	limits := svc.GetPlanDetails(models.PlanFree)
	if limits.MonthlyEntries != models.FreeMonthlyEntries {
		t.Fatalf("expected %d monthly entries for free plan, got %d", models.FreeMonthlyEntries, limits.MonthlyEntries)
	}
	if limits.HasPDFExport {
		t.Fatal("free plan must not have PDF export")
	}
	if limits.HasWeeklyReview {
		t.Fatal("free plan must not have weekly review")
	}
	if limits.MonthlyShares != 0 {
		t.Fatal("free plan must not allow shares")
	}
}

func TestGetPlanDetails_Plus(t *testing.T) {
	svc := newSvc(0, 0)
	limits := svc.GetPlanDetails(models.PlanPlus)
	if limits.MonthlyEntries != -1 {
		t.Fatal("plus plan should have unlimited entries")
	}
	if limits.MonthlyShares != models.PlusMonthlyShares {
		t.Fatalf("expected %d shares for plus plan, got %d", models.PlusMonthlyShares, limits.MonthlyShares)
	}
	if !limits.HasPDFExport {
		t.Fatal("plus plan must have PDF export (Plus is the complete journal product)")
	}
	if !limits.HasWeeklyReview {
		t.Fatal("plus plan must have weekly review")
	}
}

func TestGetPlanDetails_Pro(t *testing.T) {
	svc := newSvc(0, 0)
	limits := svc.GetPlanDetails(models.PlanPro)
	if !limits.HasPDFExport {
		t.Fatal("pro plan must have PDF export")
	}
	if limits.MonthlyShares != -1 {
		t.Fatal("pro plan should have unlimited shares")
	}
}

// ── CheckEntryQuota ───────────────────────────────────────────────────────────

func TestCheckEntryQuota_FreeBelowLimit(t *testing.T) {
	svc := newSvc(models.FreeMonthlyEntries-1, 0)
	if err := svc.CheckEntryQuota(context.Background(), uuid.New(), models.PlanFree); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestCheckEntryQuota_FreeAtLimit(t *testing.T) {
	svc := newSvc(models.FreeMonthlyEntries, 0)
	if err := svc.CheckEntryQuota(context.Background(), uuid.New(), models.PlanFree); err == nil {
		t.Fatal("expected error at limit, got nil")
	}
}

func TestCheckEntryQuota_FreeOverLimit(t *testing.T) {
	svc := newSvc(models.FreeMonthlyEntries+5, 0)
	if err := svc.CheckEntryQuota(context.Background(), uuid.New(), models.PlanFree); err == nil {
		t.Fatal("expected error over limit, got nil")
	}
}

func TestCheckEntryQuota_PlusUnlimited(t *testing.T) {
	// Even with 1000 entries, plus should pass.
	svc := newSvc(1000, 0)
	if err := svc.CheckEntryQuota(context.Background(), uuid.New(), models.PlanPlus); err != nil {
		t.Fatalf("plus plan should have unlimited entries, got %v", err)
	}
}

func TestCheckEntryQuota_ProUnlimited(t *testing.T) {
	svc := newSvc(1000, 0)
	if err := svc.CheckEntryQuota(context.Background(), uuid.New(), models.PlanPro); err != nil {
		t.Fatalf("pro plan should have unlimited entries, got %v", err)
	}
}

// ── CheckShareQuota ───────────────────────────────────────────────────────────

func TestCheckShareQuota_FreeNotAllowed(t *testing.T) {
	svc := newSvc(0, 0)
	if err := svc.CheckShareQuota(context.Background(), uuid.New(), models.PlanFree); err == nil {
		t.Fatal("free plan must not allow share links")
	}
}

func TestCheckShareQuota_PlusBelowLimit(t *testing.T) {
	svc := newSvc(0, models.PlusMonthlyShares-1)
	if err := svc.CheckShareQuota(context.Background(), uuid.New(), models.PlanPlus); err != nil {
		t.Fatalf("plus under limit should pass, got %v", err)
	}
}

func TestCheckShareQuota_PlusAtLimit(t *testing.T) {
	svc := newSvc(0, models.PlusMonthlyShares)
	if err := svc.CheckShareQuota(context.Background(), uuid.New(), models.PlanPlus); err == nil {
		t.Fatal("plus at monthly share limit should fail")
	}
}

func TestCheckShareQuota_ProUnlimited(t *testing.T) {
	svc := newSvc(0, 1000)
	if err := svc.CheckShareQuota(context.Background(), uuid.New(), models.PlanPro); err != nil {
		t.Fatalf("pro should have unlimited shares, got %v", err)
	}
}

func TestCheckShareQuota_B2BUnlimited(t *testing.T) {
	svc := newSvc(0, 1000)
	if err := svc.CheckShareQuota(context.Background(), uuid.New(), models.PlanB2B); err != nil {
		t.Fatalf("b2b should have unlimited shares, got %v", err)
	}
}

// ── UpgradePlan ───────────────────────────────────────────────────────────────

func TestUpgradePlan_SetsCorrectPlan(t *testing.T) {
	repo := &fakeSubUserRepo{}
	svc := services.NewSubscriptionService(repo, &fakeSubShareRepo{})
	user, err := svc.UpgradePlan(context.Background(), uuid.New(), models.PlanPlus, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.Plan != models.PlanPlus {
		t.Fatalf("expected plus plan, got %s", user.Plan)
	}
	if repo.updatedPlan != models.PlanPlus {
		t.Fatalf("repo not updated correctly, got %s", repo.updatedPlan)
	}
}

func TestUpgradePlan_WithExpiry(t *testing.T) {
	repo := &fakeSubUserRepo{}
	svc := services.NewSubscriptionService(repo, &fakeSubShareRepo{})
	expiry := time.Now().Add(30 * 24 * time.Hour)
	_, err := svc.UpgradePlan(context.Background(), uuid.New(), models.PlanPro, &expiry)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.updatedAt == nil {
		t.Fatal("expected expiry to be set")
	}
}

func TestUpgradePlan_NilExpiry_Works(t *testing.T) {
	repo := &fakeSubUserRepo{}
	svc := services.NewSubscriptionService(repo, &fakeSubShareRepo{})
	user, err := svc.UpgradePlan(context.Background(), uuid.New(), models.PlanFree, nil)
	if err != nil {
		t.Fatalf("nil expiry must not error: %v", err)
	}
	if user.PlanExpiresAt != nil {
		t.Error("nil expiry must produce nil PlanExpiresAt")
	}
}

// ── GetPlanDetails additional plans ──────────────────────────────────────────

func TestGetPlanDetails_B2B(t *testing.T) {
	svc := newSvc(0, 0)
	limits := svc.GetPlanDetails(models.PlanB2B)
	if limits.MonthlyEntries != -1 {
		t.Fatal("b2b plan should have unlimited entries")
	}
	if limits.MonthlyShares != -1 {
		t.Fatal("b2b plan should have unlimited shares")
	}
	if !limits.HasPDFExport {
		t.Fatal("b2b plan must have PDF export")
	}
	if !limits.HasWeeklyReview {
		t.Fatal("b2b plan must have weekly review")
	}
}

func TestCheckEntryQuota_B2BUnlimited(t *testing.T) {
	svc := newSvc(1000, 0)
	if err := svc.CheckEntryQuota(context.Background(), uuid.New(), models.PlanB2B); err != nil {
		t.Fatalf("b2b plan should have unlimited entries, got %v", err)
	}
}

// ── ErrorSentinels ────────────────────────────────────────────────────────────

func TestIsEntryQuotaExceeded_TrueForQuotaError(t *testing.T) {
	svc := newSvc(models.FreeMonthlyEntries, 0)
	err := svc.CheckEntryQuota(context.Background(), uuid.New(), models.PlanFree)
	if !services.IsEntryQuotaExceeded(err) {
		t.Error("IsEntryQuotaExceeded must return true for free plan quota error")
	}
}

func TestIsEntryQuotaExceeded_FalseForNil(t *testing.T) {
	if services.IsEntryQuotaExceeded(nil) {
		t.Error("IsEntryQuotaExceeded must return false for nil")
	}
}

func TestIsShareError_TrueForFreeNotAllowed(t *testing.T) {
	svc := newSvc(0, 0)
	err := svc.CheckShareQuota(context.Background(), uuid.New(), models.PlanFree)
	if !services.IsShareError(err) {
		t.Error("IsShareError must return true for free plan share error")
	}
}

func TestIsShareError_TrueForQuotaExceeded(t *testing.T) {
	svc := newSvc(0, models.PlusMonthlyShares)
	err := svc.CheckShareQuota(context.Background(), uuid.New(), models.PlanPlus)
	if !services.IsShareError(err) {
		t.Error("IsShareError must return true for plus quota exceeded error")
	}
}

// ── Plan.AtLeast ──────────────────────────────────────────────────────────────

func TestPlanAtLeast(t *testing.T) {
	cases := []struct {
		plan     models.Plan
		required models.Plan
		want     bool
	}{
		{models.PlanFree, models.PlanFree, true},
		{models.PlanFree, models.PlanPlus, false},
		{models.PlanPlus, models.PlanFree, true},
		{models.PlanPlus, models.PlanPlus, true},
		{models.PlanPlus, models.PlanPro, false},
		{models.PlanPro, models.PlanPlus, true},
		{models.PlanPro, models.PlanPro, true},
		{models.PlanB2B, models.PlanPro, true},
	}

	for _, tc := range cases {
		got := tc.plan.AtLeast(tc.required)
		if got != tc.want {
			t.Errorf("%s.AtLeast(%s) = %v, want %v", tc.plan, tc.required, got, tc.want)
		}
	}
}
