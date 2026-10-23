package services

import (
	"context"
	"errors"
	"testing"

	appconfig "github.com/dreamlog/backend/internal/config"
	"github.com/dreamlog/backend/internal/models"
	"github.com/google/uuid"
)

// newProdBillingTherapyService builds a TherapyService with real (non-stub)
// billing: paid sessions need a prepaid credit.
func newProdBillingTherapyService(repo *fakeTherapyRepo) *TherapyService {
	claude := NewClaudeService(&appconfig.AnthropicConfig{StubAnalysis: true, Model: "stub"})
	transcription := NewTranscriptionService(&appconfig.OpenAIConfig{BaseURL: "http://localhost:9999"})
	return NewTherapyService(repo, &fakeTherapyAnalysisRepo{}, nil, claude, transcription, &fakeTherapyStorage{}, NewCrisisDetector(nil, nil), nil, false, nil)
}

func startWithPlan(svc *TherapyService, plan models.Plan) (*models.TherapySession, error) {
	return svc.StartSession(context.Background(), uuid.New(), plan, models.PersonaComforting, "", "auto", "")
}

func TestTherapyCredits_PaidSession_ConsumesOneCredit_StandalonePrice(t *testing.T) {
	repo := newFakeTherapyRepo()
	repo.countAll, repo.countMonth, repo.credits = 4, 4, 2
	svc := newProdBillingTherapyService(repo)

	s, err := startWithPlan(svc, models.PlanFree)
	if err != nil {
		t.Fatalf("a user with credits must be able to start a paid session: %v", err)
	}
	if repo.credits != 1 || repo.creditSessions != 1 {
		t.Fatalf("exactly one credit must be consumed: credits=%d sessions=%d", repo.credits, repo.creditSessions)
	}
	if s.BillingAmountPaise != models.TherapySessionPricePaise {
		t.Fatalf("billing_amount_paise = %d, want standalone %d", s.BillingAmountPaise, models.TherapySessionPricePaise)
	}
}

func TestTherapyCredits_ProBeyondAllowance_ConsumesCreditAtMemberPrice(t *testing.T) {
	repo := newFakeTherapyRepo()
	repo.countAll, repo.countMonth, repo.credits = 3, models.TherapyProMonthlyAllowance, 1
	svc := newProdBillingTherapyService(repo)

	s, err := startWithPlan(svc, models.PlanPro)
	if err != nil {
		t.Fatal(err)
	}
	if repo.credits != 0 {
		t.Fatalf("credit not consumed: %d", repo.credits)
	}
	if s.BillingAmountPaise != models.TherapyMemberSessionPricePaise {
		t.Fatalf("billing_amount_paise = %d, want member %d", s.BillingAmountPaise, models.TherapyMemberSessionPricePaise)
	}
}

func TestTherapyCredits_FreeSessionsNeverConsumeCredits(t *testing.T) {
	t.Run("first session ever", func(t *testing.T) {
		repo := newFakeTherapyRepo()
		repo.credits = 3
		s, err := startWithPlan(newProdBillingTherapyService(repo), models.PlanFree)
		if err != nil || s.BillingAmountPaise != 0 {
			t.Fatalf("first session must be free: %+v %v", s, err)
		}
		if repo.credits != 3 {
			t.Fatalf("free first session consumed a credit: %d", repo.credits)
		}
	})
	t.Run("pro monthly allowance", func(t *testing.T) {
		repo := newFakeTherapyRepo()
		repo.countAll, repo.countMonth, repo.credits = 5, 0, 3
		s, err := startWithPlan(newProdBillingTherapyService(repo), models.PlanPro)
		if err != nil || s.BillingAmountPaise != 0 {
			t.Fatalf("included Pro session must be free: %+v %v", s, err)
		}
		if repo.credits != 3 {
			t.Fatalf("included Pro session consumed a credit: %d", repo.credits)
		}
	})
}

func TestTherapyCredits_NoCredit_Returns402_NoSessionCreated(t *testing.T) {
	repo := newFakeTherapyRepo()
	repo.countAll, repo.countMonth = 2, 2
	svc := newProdBillingTherapyService(repo)

	_, err := startWithPlan(svc, models.PlanPlus)
	if !errors.Is(err, ErrTherapyPaymentRequired) {
		t.Fatalf("want ErrTherapyPaymentRequired, got %v", err)
	}
	if len(repo.sessions) != 0 {
		t.Fatalf("no session may be created without payment, got %d", len(repo.sessions))
	}
}

func TestTherapyCredits_CreateFailure_KeepsCredit(t *testing.T) {
	repo := newFakeTherapyRepo()
	repo.countAll, repo.countMonth, repo.credits = 2, 2, 1
	repo.createErr = errors.New("insert failed")
	svc := newProdBillingTherapyService(repo)

	if _, err := startWithPlan(svc, models.PlanFree); err == nil || errors.Is(err, ErrTherapyPaymentRequired) {
		t.Fatalf("want the storage error, got %v", err)
	}
	if repo.credits != 1 {
		t.Fatalf("a failed session start must not spend the credit: %d", repo.credits)
	}
}

func TestTherapyCredits_LastCreditSpentThenPaymentRequired(t *testing.T) {
	repo := newFakeTherapyRepo()
	repo.countAll, repo.countMonth, repo.credits = 1, 1, 1
	svc := newProdBillingTherapyService(repo)

	if _, err := startWithPlan(svc, models.PlanFree); err != nil {
		t.Fatal(err)
	}
	if _, err := startWithPlan(svc, models.PlanFree); !errors.Is(err, ErrTherapyPaymentRequired) {
		t.Fatalf("second paid session without credits must be 402, got %v", err)
	}
	if repo.creditSessions != 1 || len(repo.sessions) != 1 {
		t.Fatalf("sessions=%d credit sessions=%d, want 1/1", len(repo.sessions), repo.creditSessions)
	}
}

func TestTherapyCredits_StubBilling_UsesCreditWhenPresentElseAllows(t *testing.T) {
	repo := newFakeTherapyRepo()
	repo.countAll, repo.countMonth, repo.credits = 2, 2, 1
	svc := newStubTherapyService(repo, nil)

	if _, err := startWithPlan(svc, models.PlanFree); err != nil {
		t.Fatal(err)
	}
	if repo.credits != 0 {
		t.Fatalf("dev stub must still consume an available credit: %d", repo.credits)
	}
	s, err := startWithPlan(svc, models.PlanFree)
	if err != nil {
		t.Fatalf("dev stub allows paid sessions without credits: %v", err)
	}
	if s.BillingAmountPaise != models.TherapySessionPricePaise {
		t.Fatalf("billing_amount_paise = %d", s.BillingAmountPaise)
	}
}

func TestIAPTherapyCatalog_DisjointFromPlanCatalog(t *testing.T) {
	for sku, p := range IAPTherapyCatalog {
		if _, clash := IAPCatalog[sku]; clash {
			t.Errorf("%s is both a plan and a therapy-credit SKU", sku)
		}
		if p.Credits != 1 {
			t.Errorf("%s grants %d credits; session packs are not sold (docs/PRICING.md)", sku, p.Credits)
		}
	}
	if !IAPTherapyCatalog[IAPProductTherapyMember].MemberPrice || IAPTherapyCatalog[IAPProductTherapySingle].MemberPrice {
		t.Error("only the member SKU is member-priced")
	}
}
