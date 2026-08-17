package main

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/dreamlog/backend/internal/config"
	"github.com/dreamlog/backend/internal/handlers"
	"github.com/dreamlog/backend/internal/repositories"
	"github.com/dreamlog/backend/internal/services"
	pkgcrypto "github.com/dreamlog/backend/pkg/crypto"
	"github.com/dreamlog/backend/pkg/monitoring"
	"github.com/dreamlog/backend/pkg/queue"
	pkgstorage "github.com/dreamlog/backend/pkg/storage"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

func main() {
	_ = godotenv.Load()

	log, _ := zap.NewProduction()
	defer log.Sync()

	cfg, err := config.Load()
	if err != nil {
		log.Fatal("config load failed", zap.Error(err))
	}

	// ── Sentry (no-op when SENTRY_DSN unset) ──────────────────────────────────
	defer monitoring.InitSentry(cfg.Sentry.DSN, "api", log)()

	// ── Database ─────────────────────────────────────────────────────────────
	poolCfg, err := pgxpool.ParseConfig(cfg.Database.DSN)
	if err != nil {
		log.Fatal("db parse config", zap.Error(err))
	}
	poolCfg.MaxConns = int32(cfg.Database.MaxOpenConns)
	poolCfg.MinConns = int32(cfg.Database.MaxIdleConns)
	poolCfg.MaxConnLifetime = cfg.Database.ConnMaxLifetime
	// Supabase's pooler doesn't reliably preserve pgx's server-side prepared
	// statement cache across pooled connections - use the simple protocol
	// (Supabase's documented recommendation for pgx behind their pooler).
	poolCfg.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol

	db, err := pgxpool.NewWithConfig(context.Background(), poolCfg)
	if err != nil {
		log.Fatal("db connect", zap.Error(err))
	}
	defer db.Close()

	if err := db.Ping(context.Background()); err != nil {
		log.Fatal("db ping", zap.Error(err))
	}
	log.Info("database connected")

	// ── Migrations ────────────────────────────────────────────────────────────
	m, err := migrate.New("file://migrations", cfg.Database.DSN)
	if err != nil {
		log.Fatal("migrate init", zap.Error(err))
	}
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		log.Fatal("migrate up", zap.Error(err))
	}
	log.Info("migrations applied")

	// ── Redis ─────────────────────────────────────────────────────────────────
	redisOpts := &redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	}
	if cfg.Redis.TLS {
		redisOpts.TLSConfig = &tls.Config{}
	}
	rdb := redis.NewClient(redisOpts)
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		log.Fatal("redis ping", zap.Error(err))
	}

	// ── Storage ───────────────────────────────────────────────────────────────
	storageClient, err := pkgstorage.New(&cfg.Storage)
	if err != nil {
		log.Fatal("storage init", zap.Error(err))
	}

	// ── Repositories ──────────────────────────────────────────────────────────
	userRepo := repositories.NewUserRepository(db)
	entryRepo := repositories.NewEntryRepository(db)
	analysisRepo := repositories.NewAnalysisRepository(db)
	convRepo := repositories.NewConversationRepository(db)
	nudgeRepo := repositories.NewNudgeRepository(db)
	weeklyReviewRepo := repositories.NewWeeklyReviewRepository(db)
	shareRepo := repositories.NewShareRepository(db)
	companyRepo := repositories.NewCompanyRepository(db)
	therapistRepo := repositories.NewTherapistRepository(db)
	insightShareRepo := repositories.NewInsightShareRepository(db)
	journeyRepo := repositories.NewJourneyRepository(db)
	annualReviewRepo := repositories.NewAnnualReviewRepository(db)
	lifeChapterRepo := repositories.NewLifeChapterRepository(db)
	relationshipRepo := repositories.NewRelationshipRepository(db)
	therapyRepo := repositories.NewTherapyRepository(db)
	paymentRepo := repositories.NewPaymentRepository(db)
	analyticsRepo := repositories.NewAnalyticsRepository(db)

	// ── Services ──────────────────────────────────────────────────────────────
	jobQueue := queue.New(rdb, cfg.Worker.QueueKey, cfg.Worker.DLQKey, cfg.Worker.PollTimeout)
	storageSvc := services.NewStorageService(storageClient, &cfg.Storage)
	userSvc := services.NewUserService(userRepo)
	authSvc := services.NewAuthService(userRepo, cfg.Supabase.JWTSecret, log)
	entrySvc := services.NewEntryService(entryRepo, storageSvc, jobQueue)
	claudeSvc := services.NewClaudeService(&cfg.Anthropic)
	convSvc := services.NewConversationService(convRepo, entryRepo, analysisRepo, claudeSvc)
	subscriptionSvc := services.NewSubscriptionService(userRepo, shareRepo)
	transcriptionSvc := services.NewTranscriptionService(&cfg.OpenAI)
	analyticsSvc := services.NewAnalyticsService(analyticsRepo)
	ttsSvc := services.NewTTSService(&cfg.OpenAI, &cfg.AzureTTS, storageClient)
	crisisDetector := services.NewCrisisDetector(claudeSvc, log)
	iapSvc := services.NewIAPService(&cfg.IAP)

	// Therapist notes: per-therapist envelope encryption + OCR job queue.
	masterKey, derivedKey, err := pkgcrypto.ResolveMasterKey(cfg.Security.MasterEncryptionKey, cfg.Supabase.JWTSecret)
	if err != nil {
		log.Fatal("master encryption key invalid", zap.Error(err))
	}
	if derivedKey {
		log.Warn("MASTER_ENCRYPTION_KEY not set - deriving notes encryption key from JWT secret; set an explicit key in production")
	}
	therapistNotesRepo := repositories.NewTherapistNotesRepository(db)
	notesQueue := queue.New(rdb, cfg.Worker.NotesQueueKey, cfg.Worker.NotesDLQKey, cfg.Worker.PollTimeout)
	notesCipher := services.NewNotesCipher(masterKey, therapistNotesRepo)
	therapistNotesSvc := services.NewTherapistNotesService(
		therapistNotesRepo, therapistRepo, notesQueue, storageSvc, claudeSvc, notesCipher,
	)

	therapySvc := services.NewTherapyService(
		therapyRepo, analysisRepo, relationshipRepo, claudeSvc, transcriptionSvc, storageSvc,
		crisisDetector, ttsSvc, cfg.Anthropic.StubAnalysis, log,
	)

	// ── HTTP Server ───────────────────────────────────────────────────────────
	router := handlers.NewRouter(handlers.Deps{
		UserSvc:              userSvc,
		AuthSvc:              authSvc,
		EntrySvc:             entrySvc,
		StorageSvc:           storageSvc,
		ConvSvc:              convSvc,
		SubscriptionSvc:      subscriptionSvc,
		TherapySvc:           therapySvc,
		TherapistNotesSvc:    therapistNotesSvc,
		TherapistNotesRepo:   therapistNotesRepo,
		EntryRepo:            entryRepo,
		AnalysisRepo:         analysisRepo,
		NudgeRepo:            nudgeRepo,
		UserRepo:             userRepo,
		WeeklyReviewRepo:     weeklyReviewRepo,
		ShareRepo:            shareRepo,
		CompanyRepo:          companyRepo,
		TherapistRepo:        therapistRepo,
		InsightShareRepo:     insightShareRepo,
		JourneyRepo:          journeyRepo,
		AnnualReviewRepo:     annualReviewRepo,
		LifeChapterRepo:      lifeChapterRepo,
		RelationshipRepo:     relationshipRepo,
		PaymentRepo:          paymentRepo,
		AnalyticsRepo:        analyticsRepo,
		AnalyticsSvc:         analyticsSvc,
		ClaudeSvc:            claudeSvc,
		IAPSvc:               iapSvc,
		JWTSecret:            cfg.Supabase.JWTSecret,
		SupabaseJWKSURL:      supabaseJWKSURL(cfg.Supabase.URL),
		AppBaseURL:           cfg.App.BaseURL,
		MinimumAppVersion:    cfg.App.MinimumAppVersion,
		AndroidStoreURL:      cfg.App.AndroidStoreURL,
		IOSStoreURL:          cfg.App.IOSStoreURL,
		StorageProxyBaseURL:  cfg.Storage.ProxyBaseURL,
		Log:                  log,
	})

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Info("API server starting", zap.String("addr", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal("server error", zap.Error(err))
		}
	}()

	<-quit
	log.Info("shutdown signal received")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Error("server forced shutdown", zap.Error(err))
	}
	log.Info("server stopped")
}

// supabaseJWKSURL builds the JWKS endpoint from the Supabase project URL.
// Returns empty string when URL is not configured (dev environments using local auth only).
func supabaseJWKSURL(supabaseURL string) string {
	if supabaseURL == "" {
		return ""
	}
	return strings.TrimRight(supabaseURL, "/") + "/auth/v1/.well-known/jwks.json"
}
