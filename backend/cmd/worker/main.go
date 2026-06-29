package main

import (
	"context"
	"crypto/tls"
	"os"
	"os/signal"
	"syscall"

	"github.com/dreamlog/backend/internal/config"
	"github.com/dreamlog/backend/internal/repositories"
	"github.com/dreamlog/backend/internal/services"
	"github.com/dreamlog/backend/internal/workers"
	pkgstorage "github.com/dreamlog/backend/pkg/storage"
	"github.com/dreamlog/backend/pkg/queue"
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
	poolCfg.MaxConns = 5
	poolCfg.MinConns = 1
	poolCfg.MaxConnLifetime = cfg.Database.ConnMaxLifetime

	db, err := pgxpool.NewWithConfig(context.Background(), poolCfg)
	if err != nil {
		log.Fatal("db connect", zap.Error(err))
	}
	defer db.Close()

	if err := db.Ping(context.Background()); err != nil {
		log.Fatal("db ping", zap.Error(err))
	}

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
	nudgeRepo := repositories.NewNudgeRepository(db)
	weeklyReviewRepo := repositories.NewWeeklyReviewRepository(db)
	annualReviewRepo := repositories.NewAnnualReviewRepository(db)
	relationshipRepo := repositories.NewRelationshipRepository(db)

	// ── Services ──────────────────────────────────────────────────────────────
	jobQueue := queue.New(rdb, cfg.Worker.QueueKey, cfg.Worker.DLQKey, cfg.Worker.PollTimeout)
	transcriber := services.NewTranscriptionService(&cfg.OpenAI)
	claudeSvc := services.NewClaudeService(&cfg.Anthropic)
	crisisDetector := services.NewCrisisDetector(claudeSvc)
	contextBuilder := services.NewContextBuilder(entryRepo, userRepo, analysisRepo)
	nudgeSvc := services.NewNudgeService(nudgeRepo, userRepo)
	fcmSvc := services.NewFCMService(&cfg.FCM)

	// ── Worker ────────────────────────────────────────────────────────────────
	worker := workers.NewTranscriptionWorker(workers.TranscriptionWorkerDeps{
		Queue:          jobQueue,
		EntryRepo:      entryRepo,
		AnalysisRepo:   analysisRepo,
		NudgeRepo:      nudgeRepo,
		Transcriber:    transcriber,
		CrisisDetector: crisisDetector,
		ContextBuilder: contextBuilder,
		Claude:         claudeSvc,
		NudgeSvc:        nudgeSvc,
		Storage:         storageClient,
		PersonExtractor: claudeSvc,
		PersonRepo:      relationshipRepo,
		Log:             log,
		MaxRetries:      cfg.Worker.MaxRetries,
		Concurrency:     cfg.Worker.Concurrency,
	})

	nudgeScheduler := workers.NewNudgeScheduler(nudgeRepo, fcmSvc, log)
	reengagementScheduler := workers.NewReengagementScheduler(nudgeRepo, fcmSvc, log)

	weeklyReviewScheduler := workers.NewWeeklyReviewScheduler(workers.WeeklyReviewSchedulerDeps{
		ReviewRepo:    weeklyReviewRepo,
		UserRepo:      userRepo,
		AnalysisRepo:  analysisRepo,
		Claude:        claudeSvc,
		NudgeRepo:     nudgeRepo,
		FCM:           fcmSvc,
		FreezeGranter: userRepo,
		Log:           log,
	})

	yearInReviewScheduler := workers.NewYearInReviewScheduler(workers.YearInReviewSchedulerDeps{
		ReviewRepo:   annualReviewRepo,
		UserRepo:     userRepo,
		AnalysisRepo: analysisRepo,
		Claude:       claudeSvc,
		NudgeRepo:    nudgeRepo,
		FCM:          fcmSvc,
		Log:          log,
	})

	// ── Graceful Shutdown ─────────────────────────────────────────────────────
	ctx, cancel := context.WithCancel(context.Background())
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		log.Info("worker shutdown signal received")
		cancel()
	}()

	// Run schedulers in background goroutines.
	go nudgeScheduler.Run(ctx)
	go reengagementScheduler.Run(ctx)
	go weeklyReviewScheduler.Run(ctx)
	go yearInReviewScheduler.Run(ctx)

	log.Info("starting transcription worker")
	worker.Run(ctx)
	log.Info("worker exited")
}
