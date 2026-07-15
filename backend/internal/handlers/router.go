package handlers

import (
	"net/http"

	"github.com/dreamlog/backend/internal/middleware"
	"github.com/dreamlog/backend/internal/repositories"
	"github.com/dreamlog/backend/internal/services"
	sentrygin "github.com/getsentry/sentry-go/gin"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ensure AnalyticsService satisfies the analyticsTracker interface at compile time.
var _ analyticsTracker = (*services.AnalyticsService)(nil)

type Deps struct {
	UserSvc              *services.UserService
	AuthSvc              *services.AuthService
	EntrySvc             *services.EntryService
	StorageSvc           *services.StorageService
	ConvSvc              *services.ConversationService
	SubscriptionSvc      *services.SubscriptionService
	TherapySvc           *services.TherapyService
	TherapistNotesSvc    *services.TherapistNotesService
	TherapistNotesRepo   *repositories.TherapistNotesRepository
	EntryRepo            *repositories.EntryRepository
	AnalysisRepo         *repositories.AnalysisRepository
	NudgeRepo            *repositories.NudgeRepository
	UserRepo             *repositories.UserRepository
	WeeklyReviewRepo     *repositories.WeeklyReviewRepository
	ShareRepo            *repositories.ShareRepository
	CompanyRepo          *repositories.CompanyRepository
	TherapistRepo        *repositories.TherapistRepository
	InsightShareRepo     *repositories.InsightShareRepository
	JourneyRepo          *repositories.JourneyRepository
	AnnualReviewRepo     *repositories.AnnualReviewRepository
	LifeChapterRepo      *repositories.LifeChapterRepository
	RelationshipRepo     *repositories.RelationshipRepository
	PaymentRepo          *repositories.PaymentRepository
	AnalyticsRepo        *repositories.AnalyticsRepository
	AnalyticsSvc         *services.AnalyticsService
	ClaudeSvc            *services.ClaudeService
	IAPSvc               *services.IAPService
	JWTSecret            string
	SupabaseJWKSURL      string
	AppBaseURL           string
	MinimumAppVersion    string
	AndroidStoreURL      string
	IOSStoreURL          string
	StorageProxyBaseURL  string
	Log                  *zap.Logger
}

func NewRouter(deps Deps) http.Handler {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()

	r.Use(middleware.CORSMiddleware())
	r.Use(middleware.RecoveryHandler(deps.Log))
	// Registered after RecoveryHandler so it sees a panic first (deferred
	// recovery runs innermost-first): Sentry captures it, re-panics, and
	// RecoveryHandler still produces the 500 response. No-op when Sentry is
	// not initialized (blank SENTRY_DSN in dev).
	r.Use(sentrygin.New(sentrygin.Options{Repanic: true}))
	r.Use(middleware.RequestLogger(deps.Log))
	r.Use(middleware.ErrorHandler(deps.Log))

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Force-update gate (public - the app checks this before auth)
	versionHandler := NewVersionHandler(deps.MinimumAppVersion, deps.AndroidStoreURL, deps.IOSStoreURL)
	r.GET("/version", versionHandler.Get)

	// Rate limiters for abuse-sensitive public endpoints. Tight bucket on
	// credential endpoints (login/register) to blunt credential stuffing;
	// a looser bucket on the public share-view passcode path (the per-link
	// DB lockout is the cross-instance backstop there).
	authLimiter := middleware.NewRateLimiter(0.2, 5)   // ~1 req / 5s, burst 5
	shareLimiter := middleware.NewRateLimiter(0.5, 10) // ~1 req / 2s, burst 10

	// Auth (public - no JWT required)
	authHandler := NewAuthHandler(deps.AuthSvc)
	r.POST("/auth/register", authLimiter.Middleware(), authHandler.Register)
	r.POST("/auth/login", authLimiter.Middleware(), authHandler.Login)

	auth := r.Group("/", middleware.AuthMiddleware(deps.JWTSecret, deps.SupabaseJWKSURL, deps.UserSvc, deps.Log))

	// User
	userHandler := NewUserHandler(deps.UserSvc)
	auth.GET("/me", userHandler.GetMe)
	auth.PUT("/me", userHandler.UpdateMe)
	auth.DELETE("/me", userHandler.DeleteMe)

	// Billing / subscription (purchases happen in the store SDKs on-device;
	// the backend verifies receipts server-side before granting a plan)
	billingHandler := NewBillingHandler(deps.SubscriptionSvc, deps.PaymentRepo, deps.AnalyticsSvc, deps.IAPSvc)
	auth.GET("/billing/plan", billingHandler.GetPlan)
	auth.POST("/billing/upgrade", billingHandler.Upgrade)

	entryHandler := NewEntryHandler(deps.EntrySvc, deps.StorageSvc, deps.SubscriptionSvc)
	// Upload proxy: only registered when STORAGE_PROXY_BASE_URL is set (local MinIO dev only)
	if deps.StorageProxyBaseURL != "" {
		r.PUT("/upload", entryHandler.UploadProxy)
	}

	// Entries + presign
	entries := auth.Group("/entries")
	{
		entries.POST("/presign", entryHandler.Presign)
		entries.POST("", entryHandler.Create)
		entries.GET("", entryHandler.List)
		entries.GET("/:id", entryHandler.Get)
	}

	// Analysis + timeline + search
	analysisHandler := NewAnalysisHandler(deps.EntryRepo, deps.AnalysisRepo, deps.ConvSvc)
	auth.GET("/entries/:id/analysis", analysisHandler.GetAnalysis)
	auth.GET("/timeline", analysisHandler.GetTimeline)
	auth.GET("/entries/search", analysisHandler.Search)

	// Follow-up conversations
	convHandler := NewConversationHandler(deps.ConvSvc)
	auth.POST("/entries/:id/conversation", convHandler.GetOrCreate)
	auth.POST("/conversations/:id/messages", convHandler.SendMessage)

	// Habit loop: flashback time capsule + self-set check-in nudges
	hookHandler := NewHookHandler(deps.EntryRepo, deps.AnalysisRepo, deps.AnalysisRepo,
		services.NewNudgeService(deps.NudgeRepo, deps.UserRepo))
	auth.GET("/entries/flashback", hookHandler.GetFlashback)
	auth.POST("/entries/:id/checkin", hookHandler.CreateCheckin)

	// Mood + streak + freeze
	moodHandler := NewMoodHandler(deps.AnalysisRepo, deps.NudgeRepo, deps.UserRepo)
	auth.GET("/mood/weekly", moodHandler.WeeklyMood)
	auth.GET("/mood/streak", moodHandler.Streak)
	auth.GET("/mood/history", moodHandler.MoodHistory)   // Plus+ only - gated in handler
	auth.GET("/mood/patterns", moodHandler.PatternRadar) // all plans - emotion pattern radar
	auth.POST("/streak/freeze", moodHandler.UseFreeze)

	// Device registration (FCM tokens)
	auth.POST("/devices", moodHandler.RegisterDevice)

	// Weekly reviews - Plus+ only (gated in handler)
	reviewHandler := NewWeeklyReviewHandler(deps.WeeklyReviewRepo)
	auth.GET("/reviews/weekly", reviewHandler.List)
	auth.GET("/reviews/weekly/latest", reviewHandler.GetLatest)

	// Annual reviews - Plus+ only (gated in handler)
	annualReviewHandler := NewAnnualReviewHandler(deps.AnnualReviewRepo)
	auth.GET("/reviews/annual", annualReviewHandler.List)
	auth.GET("/reviews/annual/latest", annualReviewHandler.GetLatest)

	// Therapist share links (5a) - Plus+ only (gated in handler)
	shareHandler := NewShareHandler(deps.ShareRepo, deps.SubscriptionSvc, deps.AppBaseURL)
	auth.POST("/share", shareHandler.Create)
	auth.GET("/share", shareHandler.List)
	auth.DELETE("/share/:id", shareHandler.Revoke)
	// Public - no auth middleware; passcode in query param
	r.GET("/share/:token", shareLimiter.Middleware(), shareHandler.View)

	// PDF export (5d) - Pro+ only (gated in handler)
	exportHandler := NewExportHandler(deps.AnalysisRepo, deps.UserRepo)
	auth.GET("/export/pdf", exportHandler.ExportPDF)

	// Shareable insight cards (4d)
	insightHandler := NewInsightHandler(deps.InsightShareRepo, deps.AnalysisRepo)
	auth.GET("/insights/card", insightHandler.GetCard)
	auth.POST("/insights/share", insightHandler.RecordShare)

	// B2B corporate wellness (5c)
	b2bHandler := NewB2BHandler(deps.CompanyRepo)
	auth.POST("/b2b/companies/:slug/join", b2bHandler.Join)
	auth.GET("/b2b/companies/:slug/mood", b2bHandler.TeamMood)

	// Guided Journeys
	journeyHandler := NewJourneyHandler(services.NewJourneyService(deps.JourneyRepo))
	// Public - the template catalogue is static seeded content (no user data), so
	// guests can browse journeys before signing in. Starting one still requires auth.
	r.GET("/journeys", journeyHandler.ListTemplates)
	auth.POST("/journeys/:journeyID/start", journeyHandler.StartSession)
	auth.GET("/journeys/sessions", journeyHandler.ListSessions)
	auth.GET("/journeys/sessions/:sessionID", journeyHandler.GetSession)
	auth.POST("/journeys/sessions/:sessionID/advance", journeyHandler.AdvanceSession)

	// Life Chapters - user-defined time periods with themes
	chapterHandler := NewLifeChapterHandler(deps.LifeChapterRepo, deps.ClaudeSvc)
	auth.GET("/chapters", chapterHandler.List)
	auth.POST("/chapters", chapterHandler.Create)
	auth.GET("/chapters/:id", chapterHandler.GetByID)
	auth.PUT("/chapters/:id", chapterHandler.Update)
	auth.DELETE("/chapters/:id", chapterHandler.Delete)
	auth.GET("/chapters/:id/detail", chapterHandler.GetDetail)
	auth.POST("/chapters/:id/summarize", chapterHandler.Summarize)

	// Relationship Map
	relationshipHandler := NewRelationshipHandler(deps.RelationshipRepo)
	auth.GET("/relationships", relationshipHandler.GetMap)
	auth.GET("/relationships/:id", relationshipHandler.GetPersonDetail)
	auth.PATCH("/relationships/:id", relationshipHandler.UpdatePerson)
	auth.POST("/relationships/:id/merge", relationshipHandler.MergePerson)

	// Therapist dashboard (5g)
	therapistHandler := NewTherapistHandler(deps.TherapistRepo, deps.AnalysisRepo, deps.ClaudeSvc)
	auth.POST("/therapists/register", therapistHandler.Register)
	auth.POST("/therapists/clients/link", therapistHandler.LinkClient)
	auth.DELETE("/therapists/clients/:clientID", therapistHandler.UnlinkClient)
	auth.GET("/therapists/clients", therapistHandler.ListClients)
	auth.GET("/therapists/clients/:clientID/brief", therapistHandler.ClientBrief)
	// Client-facing consent endpoints: a therapist link stays 'pending' and
	// exposes no data until the client approves it here.
	auth.GET("/therapists/requests", therapistHandler.ListLinkRequests)
	auth.POST("/therapists/requests/:therapistID/approve", therapistHandler.ApproveLinkRequest)
	auth.POST("/therapists/requests/:therapistID/decline", therapistHandler.DeclineLinkRequest)

	// Therapist workspace: in-app dashboard, external clients, session notes
	// (photo OCR → bullets), AI summaries. Encrypted at rest per-therapist.
	notesHandler := NewTherapistNotesHandler(deps.TherapistNotesSvc, deps.TherapistRepo, deps.TherapistNotesRepo)
	auth.GET("/therapists/me", notesHandler.GetMe)
	auth.POST("/therapists/consent", notesHandler.AcceptClientConsent)
	auth.GET("/therapists/overview", notesHandler.Overview)
	auth.POST("/therapists/external-clients", notesHandler.CreateExternalClient)
	auth.GET("/therapists/external-clients", notesHandler.ListExternalClients)
	auth.GET("/therapists/external-clients/:id", notesHandler.GetExternalClient)
	auth.PATCH("/therapists/external-clients/:id", notesHandler.UpdateExternalClient)
	auth.DELETE("/therapists/external-clients/:id", notesHandler.DeleteExternalClient)
	auth.POST("/therapists/sessions/presign", notesHandler.PresignNote)
	auth.POST("/therapists/sessions", notesHandler.CreateSession)
	auth.GET("/therapists/sessions", notesHandler.ListSessions)
	auth.GET("/therapists/sessions/:id", notesHandler.GetSession)
	auth.PATCH("/therapists/sessions/:id", notesHandler.UpdateSessionBullets)
	auth.POST("/therapists/sessions/:id/summarize", notesHandler.SummarizeSession)
	auth.DELETE("/therapists/sessions/:id", notesHandler.DeleteSession)
	// ToS / privacy acceptance (all users)
	auth.POST("/me/accept-terms", notesHandler.AcceptUserTerms)
	auth.GET("/me/terms", notesHandler.GetUserTerms)

	// Therapy Mode (Phase 6)
	therapyHandler := NewTherapyHandler(deps.TherapySvc, deps.StorageSvc, deps.UserRepo)
	therapy := auth.Group("/therapy")
	{
		therapy.POST("/sessions", therapyHandler.StartSession)
		therapy.GET("/sessions", therapyHandler.ListSessions)
		therapy.GET("/sessions/:id", therapyHandler.GetSession)
		therapy.POST("/sessions/:id/presign", therapyHandler.PresignAudio)
		therapy.POST("/sessions/:id/messages", therapyHandler.SendMessage)
		therapy.POST("/sessions/:id/end", therapyHandler.EndSession)
	}

	return r
}
