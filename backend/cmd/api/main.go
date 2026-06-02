package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dreamlog/backend/internal/config"
	"github.com/dreamlog/backend/internal/handlers"
	"github.com/dreamlog/backend/internal/repositories"
	"github.com/dreamlog/backend/internal/services"
	pkgstorage "github.com/dreamlog/backend/pkg/storage"
	"github.com/dreamlog/backend/pkg/queue"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
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

	// ── Database ─────────────────────────────────────────────────────────────
	poolCfg, err := pgxpool.ParseConfig(cfg.Database.DSN)
	if err != nil {
		log.Fatal("db parse config", zap.Error(err))
	}
	poolCfg.MaxConns = int32(cfg.Database.MaxOpenConns)
	poolCfg.MinConns = int32(cfg.Database.MaxIdleConns)
	poolCfg.MaxConnLifetime = cfg.Database.ConnMaxLifetime

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
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
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

	// ── Services ──────────────────────────────────────────────────────────────
	jobQueue := queue.New(rdb, cfg.Worker.QueueKey, cfg.Worker.DLQKey, cfg.Worker.PollTimeout)
	storageSvc := services.NewStorageService(storageClient, &cfg.Storage)
	userSvc := services.NewUserService(userRepo)
	authSvc := services.NewAuthService(userRepo, cfg.Supabase.JWTSecret)
	entrySvc := services.NewEntryService(entryRepo, storageSvc, jobQueue)
	claudeSvc := services.NewClaudeService(&cfg.Anthropic)
	convSvc := services.NewConversationService(convRepo, entryRepo, analysisRepo, claudeSvc)
	subscriptionSvc := services.NewSubscriptionService(userRepo, shareRepo)
	transcriptionSvc := services.NewTranscriptionService(&cfg.OpenAI)
	crisisDetector := services.NewCrisisDetector(claudeSvc)
	therapySvc := services.NewTherapyService(
		therapyRepo, analysisRepo, claudeSvc, transcriptionSvc, storageSvc,
		crisisDetector, cfg.Anthropic.StubAnalysis,
	)

	// ── HTTP Server ───────────────────────────────────────────────────────────
	router := handlers.NewRouter(handlers.Deps{
		UserSvc:          userSvc,
		AuthSvc:          authSvc,
		EntrySvc:         entrySvc,
		StorageSvc:       storageSvc,
		ConvSvc:          convSvc,
		SubscriptionSvc:  subscriptionSvc,
		TherapySvc:       therapySvc,
		EntryRepo:        entryRepo,
		AnalysisRepo:     analysisRepo,
		NudgeRepo:        nudgeRepo,
		UserRepo:         userRepo,
		WeeklyReviewRepo: weeklyReviewRepo,
		ShareRepo:        shareRepo,
		CompanyRepo:      companyRepo,
		TherapistRepo:    therapistRepo,
		InsightShareRepo: insightShareRepo,
		JourneyRepo:      journeyRepo,
		AnnualReviewRepo: annualReviewRepo,
		LifeChapterRepo:  lifeChapterRepo,
		RelationshipRepo: relationshipRepo,
		ClaudeSvc:        claudeSvc,
		JWTSecret:            cfg.Supabase.JWTSecret,
		AppBaseURL:           cfg.App.BaseURL,
		StripeSecretKey:      cfg.Stripe.SecretKey,
		StripePublishableKey: cfg.Stripe.PublishableKey,
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
