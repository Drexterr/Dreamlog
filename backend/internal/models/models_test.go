package models

import (
	"testing"
	"time"
)

// ── Plan ──────────────────────────────────────────────────────────────────────

func TestPlan_AtLeast(t *testing.T) {
	cases := []struct {
		p, required Plan
		want        bool
	}{
		{PlanFree, PlanFree, true},
		{PlanFree, PlanPlus, false},
		{PlanPlus, PlanFree, true},
		{PlanPlus, PlanPlus, true},
		{PlanPlus, PlanPro, false},
		{PlanPro, PlanPlus, true},
		{PlanPro, PlanPro, true},
		{PlanB2B, PlanPro, true},
		{PlanFree, PlanB2B, false},
	}
	for _, tc := range cases {
		if got := tc.p.AtLeast(tc.required); got != tc.want {
			t.Errorf("%s.AtLeast(%s): want %v, got %v", tc.p, tc.required, tc.want, got)
		}
	}
}

func TestGetPlanLimits_AllPlans(t *testing.T) {
	free := GetPlanLimits(PlanFree)
	if free.MonthlyEntries != FreeMonthlyEntries {
		t.Errorf("free monthly entries: want %d, got %d", FreeMonthlyEntries, free.MonthlyEntries)
	}
	if free.MonthlyShares != 0 || free.HasPDFExport || free.HasWeeklyReview {
		t.Error("free plan must have no shares, no PDF export, no weekly review")
	}

	plus := GetPlanLimits(PlanPlus)
	if plus.MonthlyEntries != -1 {
		t.Error("plus plan must have unlimited entries")
	}
	if plus.MonthlyShares != PlusMonthlyShares {
		t.Errorf("plus shares: want %d, got %d", PlusMonthlyShares, plus.MonthlyShares)
	}
	if !plus.HasPDFExport {
		t.Error("plus plan must include PDF export (Plus is the complete journal product)")
	}

	pro := GetPlanLimits(PlanPro)
	if !pro.HasPDFExport || pro.MonthlyShares != -1 {
		t.Error("pro plan must include PDF export and unlimited shares")
	}

	b2b := GetPlanLimits(PlanB2B)
	if !b2b.HasPDFExport || b2b.MonthlyEntries != -1 {
		t.Error("b2b plan must include PDF export and unlimited entries")
	}

	// Unknown plan falls back to free limits.
	unknown := GetPlanLimits(Plan("nonsense"))
	if unknown.Plan != PlanFree {
		t.Errorf("unknown plan must fall back to free, got %s", unknown.Plan)
	}
}

func TestUser_EffectivePlan_Expiry(t *testing.T) {
	past := time.Now().Add(-time.Minute)
	future := time.Now().Add(time.Minute)

	cases := []struct {
		name   string
		plan   Plan
		expiry *time.Time
		want   Plan
	}{
		{"free nil expiry", PlanFree, nil, PlanFree},
		{"free with past expiry stays free", PlanFree, &past, PlanFree},
		{"plus nil expiry never expires", PlanPlus, nil, PlanPlus},
		{"plus future expiry active", PlanPlus, &future, PlanPlus},
		{"plus past expiry degrades", PlanPlus, &past, PlanFree},
		{"pro past expiry degrades", PlanPro, &past, PlanFree},
		{"b2b past expiry degrades", PlanB2B, &past, PlanFree},
	}
	for _, tc := range cases {
		u := &User{Plan: tc.plan, PlanExpiresAt: tc.expiry}
		if got := u.EffectivePlan(); got != tc.want {
			t.Errorf("%s: want %s, got %s", tc.name, tc.want, got)
		}
	}
}

// ── EntryMode ─────────────────────────────────────────────────────────────────

func TestEntryMode_Valid(t *testing.T) {
	for _, m := range []EntryMode{EntryModeProcessing, EntryModeRant, EntryModeGratitude, EntryModeDecision, EntryModeDream} {
		if !m.Valid() {
			t.Errorf("%s must be valid", m)
		}
	}
	for _, m := range []EntryMode{"", "therapy", "PROCESSING", "dreams"} {
		if m.Valid() {
			t.Errorf("%q must be invalid", m)
		}
	}
}

// ── TherapyPersona ────────────────────────────────────────────────────────────

func TestValidPersona(t *testing.T) {
	for _, p := range []string{"comforting", "rational", "cbt", "mindful"} {
		if !ValidPersona(p) {
			t.Errorf("%s must be a valid persona", p)
		}
	}
	for _, p := range []string{"", "freudian", "Comforting", "CBT"} {
		if ValidPersona(p) {
			t.Errorf("%q must be invalid", p)
		}
	}
}

// ── Streak milestones ─────────────────────────────────────────────────────────

func TestNextStreakMilestone(t *testing.T) {
	cases := []struct{ current, want int }{
		{0, 7}, {6, 7}, {7, 21}, {20, 21}, {21, 50}, {49, 50}, {50, 100}, {99, 100}, {100, 0}, {500, 0},
	}
	for _, tc := range cases {
		if got := NextStreakMilestone(tc.current); got != tc.want {
			t.Errorf("NextStreakMilestone(%d): want %d, got %d", tc.current, tc.want, got)
		}
	}
}

func TestIsStreakMilestone(t *testing.T) {
	for _, m := range StreakMilestones {
		if !IsStreakMilestone(m) {
			t.Errorf("%d must be a milestone", m)
		}
	}
	for _, n := range []int{0, 1, 6, 8, 22, 99, 101} {
		if IsStreakMilestone(n) {
			t.Errorf("%d must not be a milestone", n)
		}
	}
}
