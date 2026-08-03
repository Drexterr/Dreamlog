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
	pkgcrypto "github.com/dreamlog/backend/pkg/crypto"
	"github.com/dreamlog/backend/pkg/monitoring"
	"github.com/dreamlog/backend/pkg/queue"
	pkgstorage "github.com/dreamlog/backend/pkg/storage"
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
	defer monitoring.InitSentry(cfg.Sentry.DSN, "worker", log)()
	// Report a main-goroutine panic before the process dies; restart
	// semantics are unchanged (RecoverRepanic re-panics after flushing).
	defer monitoring.RecoverRepanic()

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
	crisisDetector := services.NewCrisisDetector(claudeSvc, log)
	contextBuilder := services.NewContextBuilder(entryRepo, userRepo, analysisRepo)
	nudgeSvc := services.NewNudgeService(nudgeRepo, userRepo)
	fcmSvc := services.NewFCMService(&cfg.FCM)

	// ── Worker ────────────────────────────────────────────────────────────────
	worker := workers.NewTranscriptionWorker(workers.TranscriptionWorkerDeps{
		Queue:           jobQueue,
		EntryRepo:       entryRepo,
		AnalysisRepo:    analysisRepo,
		NudgeRepo:       nudgeRepo,
		Transcriber:     transcriber,
		CrisisDetector:  crisisDetector,
		ContextBuilder:  contextBuilder,
		Claude:          claudeSvc,
		NudgeSvc:        nudgeSvc,
		Storage:         storageClient,
		PersonExtractor: claudeSvc,
		PersonRepo:      relationshipRepo,
		InsightBuilder:  services.NewConnectionInsightService(analysisRepo),
		Log:             log,
		MaxRetries:      cfg.Worker.MaxRetries,
		Concurrency:     cfg.Worker.Concurrency,
	})

	// ── Note OCR worker (therapist session notes) ────────────────────────────
	masterKey, derivedKey, err := pkgcrypto.ResolveMasterKey(cfg.Security.MasterEncryptionKey, cfg.Supabase.JWTSecret)
	if err != nil {
		log.Fatal("master encryption key invalid", zap.Error(err))
	}
	if derivedKey {
		log.Warn("MASTER_ENCRYPTION_KEY not set - deriving notes encryption key from JWT secret; set an explicit key in production")
	}
	therapistNotesRepo := repositories.NewTherapistNotesRepository(db)
	notesQueue := queue.New(rdb, cfg.Worker.NotesQueueKey, cfg.Worker.NotesDLQKey, cfg.Worker.PollTimeout)
	noteOCRWorker := workers.NewNoteOCRWorker(workers.NoteOCRWorkerDeps{
		Queue:      notesQueue,
		Repo:       therapistNotesRepo,
		Storage:    storageClient,
		OCR:        claudeSvc,
		Cipher:     services.NewNotesCipher(masterKey, therapistNotesRepo),
		Log:        log,
		MaxRetries: cfg.Worker.MaxRetries,
	})

	nudgeScheduler := workers.NewNudgeScheduler(nudgeRepo, fcmSvc, log)
	reengagementScheduler := workers.NewReengagementScheduler(nudgeRepo, fcmSvc, log)
	streakRiskScheduler := workers.NewStreakRiskScheduler(nudgeRepo, analysisRepo, fcmSvc, log)
	planExpiryScheduler := workers.NewPlanExpiryScheduler(nudgeRepo, fcmSvc, log)

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

	// Run schedulers in background goroutines. Each is wrapped so a panic is
	// reported to Sentry before it crashes the process (goroutine panics
	// bypass main's deferred recovery).
	runReported := func(run func(context.Context)) {
		go func() {
			defer monitoring.RecoverRepanic()
			run(ctx)
		}()
	}
	runReported(nudgeScheduler.Run)
	runReported(reengagementScheduler.Run)
	runReported(streakRiskScheduler.Run)
	runReported(planExpiryScheduler.Run)
	runReported(weeklyReviewScheduler.Run)
	runReported(yearInReviewScheduler.Run)
	runReported(noteOCRWorker.Run)

	log.Info("starting transcription worker")
	worker.Run(ctx)
	log.Info("worker exited")
}
