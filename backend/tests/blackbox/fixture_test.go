//go:build integration

package blackbox

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/dreamlog/backend/internal/config"
	"github.com/dreamlog/backend/internal/handlers"
	"github.com/dreamlog/backend/internal/repositories"
	"github.com/dreamlog/backend/internal/services"
	"github.com/dreamlog/backend/pkg/queue"
	pkgstorage "github.com/dreamlog/backend/pkg/storage"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type apiHarness struct {
	db               *pgxpool.Pool
	server           *httptest.Server
	owner, other     string
	ownerID, otherID uuid.UUID
	routes           gin.RoutesInfo
}

func newAPI(t *testing.T) *apiHarness {
	t.Helper()
	dsn := os.Getenv("ODE_TEST_DATABASE_URL")
	u, err := url.Parse(dsn)
	if err != nil || !strings.HasSuffix(u.Path, "_test") || (u.Hostname() != "127.0.0.1" && u.Hostname() != "localhost") {
		t.Fatal("ODE_TEST_DATABASE_URL must point to a loopback _test database")
	}
	ctx := context.Background()
	base, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	schema := "qa_bb_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	quoted := pgx.Identifier{schema}.Sanitize()
	if _, err = base.Exec(ctx, "CREATE EXTENSION IF NOT EXISTS pgcrypto WITH SCHEMA public"); err != nil {
		t.Fatal(err)
	}
	if _, err = base.Exec(ctx, "CREATE SCHEMA "+quoted); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := base.Exec(ctx, "DROP SCHEMA "+quoted+" CASCADE"); err != nil {
			t.Error(err)
		}
		base.Close()
	})
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schema + ",public"
	db, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(db.Close)
	files, _ := filepath.Glob("../../migrations/*.up.sql")
	sort.Strings(files)
	if len(files) == 0 {
		t.Fatal("migrations missing")
	}
	for _, file := range files {
		sql, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = db.Exec(ctx, string(sql)); err != nil {
			t.Fatalf("%s: %v", file, err)
		}
	}
	addr := os.Getenv("ODE_TEST_REDIS_ADDR")
	if !strings.HasPrefix(addr, "127.0.0.1:") && !strings.HasPrefix(addr, "localhost:") {
		t.Fatal("loopback ODE_TEST_REDIS_ADDR required")
	}
	rdb := redis.NewClient(&redis.Options{Addr: addr})
	t.Cleanup(func() { rdb.Close() })
	if err = rdb.Ping(ctx).Err(); err != nil {
		t.Fatal(err)
	}
	key := schema + ":entries"
	notesKey := schema + ":notes"
	t.Cleanup(func() { rdb.Del(ctx, key, notesKey, key+":dlq", notesKey+":dlq") })
	jobs := queue.New(rdb, key, key+":dlq", time.Second)
	notesJobs := queue.New(rdb, notesKey, notesKey+":dlq", time.Second)
	// Object storage is a controlled HTTP boundary; PostgreSQL, Redis and application layers are real.
	objects := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "missing") {
			w.WriteHeader(404)
			return
		}
		if r.Method == http.MethodGet {
			w.Write([]byte("synthetic fixture"))
			return
		}
		w.WriteHeader(200)
	}))
	t.Cleanup(objects.Close)
	storageCfg := &config.StorageConfig{Endpoint: objects.URL, AccessKeyID: "qa-local", SecretAccessKey: "qa-local-secret", Bucket: "qa", Region: "us-east-1", UsePathStyle: true, PresignExpiry: time.Minute}
	sc, err := pkgstorage.New(storageCfg)
	if err != nil {
		t.Fatal(err)
	}
	storage := services.NewStorageService(sc, storageCfg)
	users := repositories.NewUserRepository(db)
	entries := repositories.NewEntryRepository(db)
	analysis := repositories.NewAnalysisRepository(db)
	shares := repositories.NewShareRepository(db)
	therapists := repositories.NewTherapistRepository(db)
	notes := repositories.NewTherapistNotesRepository(db)
	relationships := repositories.NewRelationshipRepository(db)
	therapyRepo := repositories.NewTherapyRepository(db)
	claude := services.NewClaudeService(&config.AnthropicConfig{StubAnalysis: true})
	userSvc := services.NewUserService(users)
	deps := handlers.Deps{
		UserSvc: userSvc, AuthSvc: services.NewAuthService(users, "qa-blackbox-local-only", zap.NewNop()),
		EntrySvc: services.NewEntryService(entries, storage, jobs), StorageSvc: storage,
		ConvSvc:            services.NewConversationService(repositories.NewConversationRepository(db), entries, analysis, claude),
		SubscriptionSvc:    services.NewSubscriptionService(users, shares),
		TherapySvc:         services.NewTherapyService(therapyRepo, analysis, relationships, claude, services.NewTranscriptionService(&config.OpenAIConfig{}), storage, services.NewCrisisDetector(claude, zap.NewNop()), nil, true, zap.NewNop()),
		TherapistNotesSvc:  services.NewTherapistNotesService(notes, therapists, notesJobs, storage, claude, services.NewNotesCipher([]byte("01234567890123456789012345678901"), notes)),
		TherapistNotesRepo: notes, EntryRepo: entries, AnalysisRepo: analysis, NudgeRepo: repositories.NewNudgeRepository(db), UserRepo: users,
		WeeklyReviewRepo: repositories.NewWeeklyReviewRepository(db), ShareRepo: shares, CompanyRepo: repositories.NewCompanyRepository(db), TherapistRepo: therapists,
		InsightShareRepo: repositories.NewInsightShareRepository(db), JourneyRepo: repositories.NewJourneyRepository(db), AnnualReviewRepo: repositories.NewAnnualReviewRepository(db),
		LifeChapterRepo: repositories.NewLifeChapterRepository(db), RelationshipRepo: relationships, PaymentRepo: repositories.NewPaymentRepository(db),
		AnalyticsRepo: repositories.NewAnalyticsRepository(db), ClaudeSvc: claude, IAPSvc: services.NewIAPService(&config.IAPConfig{}), JWTSecret: "qa-blackbox-local-only", AppBaseURL: "https://qa.invalid", Log: zap.NewNop(), MinimumAppVersion: "1.0.0",
	}
	deps.AnalyticsSvc = services.NewAnalyticsService(deps.AnalyticsRepo)
	router := handlers.NewRouter(deps).(*gin.Engine)
	h := &apiHarness{db: db, server: httptest.NewServer(router), routes: router.Routes()}
	t.Cleanup(h.server.Close)
	for i := 0; i < 2; i++ {
		email := uuid.NewString() + "@example.test"
		body := fmt.Sprintf(`{"email":%q,"name":"QA synthetic","password":"QA-local-12345"}`, email)
		got := h.raw(t, "POST", "/auth/register", body, "", 201)
		token := got["token"].(string)
		user := got["user"].(map[string]any)
		id := uuid.MustParse(user["id"].(string))
		if i == 0 {
			h.owner = token
			h.ownerID = id
		} else {
			h.other = token
			h.otherID = id
		}
	}
	if _, err = db.Exec(ctx, `UPDATE users SET plan='pro',plan_expires_at=NOW()+INTERVAL '30 days' WHERE id=$1`, h.ownerID); err != nil {
		t.Fatal(err)
	}
	return h
}

func (h *apiHarness) raw(t *testing.T, method, path, body, token string, want int) map[string]any {
	t.Helper()
	req, err := http.NewRequest(method, h.server.URL+path, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	client := &http.Client{Timeout: 15 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	data, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != want {
		t.Fatalf("%s %s: want %d got %d: %s", method, path, want, res.StatusCode, data)
	}
	out := map[string]any{}
	if len(data) > 0 && strings.Contains(res.Header.Get("Content-Type"), "json") {
		if err = json.Unmarshal(data, &out); err != nil {
			t.Fatalf("JSON response: %v", err)
		}
	}
	return out
}

func (h *apiHarness) request(t *testing.T, name, method, path, body, token string, want int) map[string]any {
	t.Helper()
	var out map[string]any
	t.Run(name, func(t *testing.T) {
		out = h.raw(t, method, path, body, token, want)
		actual := strings.Split(strings.Trim(path, "/"), "?")[0]
		matched := "unregistered"
		for _, route := range h.routes {
			if route.Method != method {
				continue
			}
			a, b := strings.Split(actual, "/"), strings.Split(strings.Trim(route.Path, "/"), "/")
			if len(a) != len(b) {
				continue
			}
			ok := true
			for i := range a {
				if !strings.HasPrefix(b[i], ":") && a[i] != b[i] {
					ok = false
					break
				}
			}
			if ok {
				matched = route.Path
				break
			}
		}
		t.Logf("BLACKBOX_REQUEST %s %s %s %d", name, method, matched, want)
	})
	return out
}
