package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/dreamlog/backend/internal/middleware"
	"github.com/dreamlog/backend/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// ── Fakes ────────────────────────────────────────────────────────────────────

type fakeRelationshipRepo struct {
	people    []*models.Person
	mapErr    error
	detail    *models.PersonDetail
	detailErr error
}

func (f *fakeRelationshipRepo) GetMap(_ context.Context, _ uuid.UUID) ([]*models.Person, error) {
	return f.people, f.mapErr
}
func (f *fakeRelationshipRepo) GetDetail(_ context.Context, _, _ uuid.UUID) (*models.PersonDetail, error) {
	return f.detail, f.detailErr
}

// ── Router & JWT ──────────────────────────────────────────────────────────────

const relTestSecret = "relationship-test-jwt-secret-32-b!"

func relTestRouter(repo relationshipMapRepo) *gin.Engine {
	gin.SetMode(gin.TestMode)
	log := zap.NewNop()

	testUser := &models.User{ID: uuid.New(), Email: "test@dreamlog.dev", Name: "Tester", Plan: models.PlanFree}

	r := gin.New()
	r.Use(middleware.ErrorHandler(log))
	r.Use(middleware.AuthMiddleware(relTestSecret, "", &fakeProvisioner{user: testUser}, log))

	h := NewRelationshipHandler(repo)
	r.GET("/relationships", h.GetMap)
	r.GET("/relationships/:id", h.GetPersonDetail)
	return r
}

func relJWT(t *testing.T) string {
	t.Helper()
	claims := jwt.MapClaims{
		"sub":   "test-sub-rel-001",
		"email": "test@dreamlog.dev",
		"exp":   time.Now().Add(time.Hour).Unix(),
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	str, err := tok.SignedString([]byte(relTestSecret))
	if err != nil {
		t.Fatal(err)
	}
	return str
}

func samplePerson() *models.Person {
	return &models.Person{
		ID:              uuid.New(),
		UserID:          uuid.New(),
		Name:            "Sarah",
		Role:            models.PersonRoleFriend,
		MentionCount:    5,
		PositiveCount:   3,
		NegativeCount:   1,
		LastMentionedAt: time.Now(),
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
}

func samplePersonDetail() *models.PersonDetail {
	p := samplePerson()
	return &models.PersonDetail{
		Person: p,
		Mentions: []models.PersonMention{
			{
				ID:        uuid.New(),
				PersonID:  p.ID,
				EntryID:   uuid.New(),
				UserID:    p.UserID,
				Sentiment: models.PersonSentimentPositive,
				Context:   "Sarah helped me finish the report",
				CreatedAt: time.Now(),
			},
		},
	}
}

// ── Tests: GetMap ─────────────────────────────────────────────────────────────

func TestGetRelationshipMap_Returns200WithPeople(t *testing.T) {
	repo := &fakeRelationshipRepo{people: []*models.Person{samplePerson(), samplePerson()}}
	r := relTestRouter(repo)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/relationships", nil)
	req.Header.Set("Authorization", "Bearer "+relJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var body struct {
		People []*models.Person `json:"people"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(body.People) != 2 {
		t.Errorf("expected 2 people, got %d", len(body.People))
	}
}

func TestGetRelationshipMap_EmptyReturnsEmptyArray(t *testing.T) {
	repo := &fakeRelationshipRepo{people: nil}
	r := relTestRouter(repo)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/relationships", nil)
	req.Header.Set("Authorization", "Bearer "+relJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var body struct {
		People []*models.Person `json:"people"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body.People == nil {
		t.Error("expected empty array, got nil")
	}
	if len(body.People) != 0 {
		t.Errorf("expected 0 people, got %d", len(body.People))
	}
}

func TestGetRelationshipMap_RepoErrorReturns500(t *testing.T) {
	repo := &fakeRelationshipRepo{mapErr: errors.New("db error")}
	r := relTestRouter(repo)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/relationships", nil)
	req.Header.Set("Authorization", "Bearer "+relJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestGetRelationshipMap_MissingAuthReturns401(t *testing.T) {
	repo := &fakeRelationshipRepo{}
	r := relTestRouter(repo)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/relationships", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

// ── Tests: GetPersonDetail ────────────────────────────────────────────────────

func TestGetPersonDetail_Returns200(t *testing.T) {
	detail := samplePersonDetail()
	repo := &fakeRelationshipRepo{detail: detail}
	r := relTestRouter(repo)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/relationships/"+detail.Person.ID.String(), nil)
	req.Header.Set("Authorization", "Bearer "+relJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var body models.PersonDetail
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body.Person == nil {
		t.Fatal("expected person in detail, got nil")
	}
	if body.Person.Name != "Sarah" {
		t.Errorf("expected name 'Sarah', got %q", body.Person.Name)
	}
	if len(body.Mentions) != 1 {
		t.Errorf("expected 1 mention, got %d", len(body.Mentions))
	}
	if body.Mentions[0].Sentiment != models.PersonSentimentPositive {
		t.Errorf("expected positive sentiment, got %q", body.Mentions[0].Sentiment)
	}
}

func TestGetPersonDetail_NotFoundReturns404(t *testing.T) {
	repo := &fakeRelationshipRepo{detail: nil}
	r := relTestRouter(repo)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/relationships/"+uuid.New().String(), nil)
	req.Header.Set("Authorization", "Bearer "+relJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestGetPersonDetail_InvalidIDReturns400(t *testing.T) {
	repo := &fakeRelationshipRepo{}
	r := relTestRouter(repo)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/relationships/not-a-uuid", nil)
	req.Header.Set("Authorization", "Bearer "+relJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestGetPersonDetail_RepoErrorReturns500(t *testing.T) {
	repo := &fakeRelationshipRepo{detailErr: errors.New("db error")}
	r := relTestRouter(repo)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/relationships/"+uuid.New().String(), nil)
	req.Header.Set("Authorization", "Bearer "+relJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestGetPersonDetail_MissingAuthReturns401(t *testing.T) {
	repo := &fakeRelationshipRepo{}
	r := relTestRouter(repo)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/relationships/"+uuid.New().String(), nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

// ── Tests: person fields ──────────────────────────────────────────────────────

func TestGetRelationshipMap_PersonFields(t *testing.T) {
	p := samplePerson()
	p.MentionCount = 7
	p.PositiveCount = 5
	p.NegativeCount = 2
	p.Role = models.PersonRoleFamily

	repo := &fakeRelationshipRepo{people: []*models.Person{p}}
	r := relTestRouter(repo)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/relationships", nil)
	req.Header.Set("Authorization", "Bearer "+relJWT(t))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var body struct {
		People []models.Person `json:"people"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	got := body.People[0]
	if got.MentionCount != 7 {
		t.Errorf("mention_count: want 7, got %d", got.MentionCount)
	}
	if got.PositiveCount != 5 {
		t.Errorf("positive_count: want 5, got %d", got.PositiveCount)
	}
	if got.NegativeCount != 2 {
		t.Errorf("negative_count: want 2, got %d", got.NegativeCount)
	}
	if got.Role != models.PersonRoleFamily {
		t.Errorf("role: want family, got %q", got.Role)
	}
}
