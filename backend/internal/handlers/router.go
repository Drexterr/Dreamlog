package handlers

import (
	"net/http"

	"github.com/dreamlog/backend/internal/middleware"
	"github.com/dreamlog/backend/internal/repositories"
	"github.com/dreamlog/backend/internal/services"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type Deps struct {
	UserSvc          *services.UserService
	AuthSvc          *services.AuthService
	EntrySvc         *services.EntryService
	StorageSvc       *services.StorageService
	ConvSvc          *services.ConversationService
	SubscriptionSvc  *services.SubscriptionService
	TherapySvc       *services.TherapyService
	EntryRepo        *repositories.EntryRepository
	AnalysisRepo     *repositories.AnalysisRepository
	NudgeRepo        *repositories.NudgeRepository
	UserRepo         *repositories.UserRepository
	WeeklyReviewRepo *repositories.WeeklyReviewRepository
	ShareRepo        *repositories.ShareRepository
	CompanyRepo      *repositories.CompanyRepository
	TherapistRepo    *repositories.TherapistRepository
	InsightShareRepo *repositories.InsightShareRepository
	JourneyRepo      *repositories.JourneyRepository
	AnnualReviewRepo  *repositories.AnnualReviewRepository
	LifeChapterRepo      *repositories.LifeChapterRepository
	RelationshipRepo     *repositories.RelationshipRepository
	ClaudeSvc            *services.ClaudeService
	JWTSecret            string
	AppBaseURL           string
	StripeSecretKey      string
	StripePublishableKey string
	Log                  *zap.Logger
}

func NewRouter(deps Deps) http.Handler {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()

	r.Use(middleware.CORSMiddleware())
	r.Use(middleware.RecoveryHandler(deps.Log))
	r.Use(middleware.RequestLogger(deps.Log))
	r.Use(middleware.ErrorHandler(deps.Log))

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Auth (public — no JWT required)
	authHandler := NewAuthHandler(deps.AuthSvc)
	r.POST("/auth/register", authHandler.Register)
	r.POST("/auth/login", authHandler.Login)

	auth := r.Group("/", middleware.AuthMiddleware(deps.JWTSecret, deps.UserSvc, deps.Log))

	// User
	userHandler := NewUserHandler(deps.UserSvc)
	auth.GET("/me", userHandler.GetMe)
	auth.PUT("/me", userHandler.UpdateMe)
	auth.DELETE("/me", userHandler.DeleteMe)

	// Billing / subscription
	billingHandler := NewBillingHandler(deps.SubscriptionSvc, deps.StripeSecretKey, deps.StripePublishableKey)
	auth.GET("/billing/plan", billingHandler.GetPlan)
	auth.POST("/billing/upgrade", billingHandler.Upgrade)
	auth.POST("/billing/create-payment-intent", billingHandler.CreatePaymentIntent)

	// Upload proxy (no auth — key is unguessable; dev only when STORAGE_PROXY_BASE_URL is set)
	entryHandler := NewEntryHandler(deps.EntrySvc, deps.StorageSvc, deps.SubscriptionSvc)
	r.PUT("/upload", entryHandler.UploadProxy)

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

	// Mood + streak + freeze
	moodHandler := NewMoodHandler(deps.AnalysisRepo, deps.NudgeRepo, deps.UserRepo)
	auth.GET("/mood/weekly", moodHandler.WeeklyMood)
	auth.GET("/mood/streak", moodHandler.Streak)
	auth.GET("/mood/history", moodHandler.MoodHistory)   // Plus+ only — gated in handler
	auth.GET("/mood/patterns", moodHandler.PatternRadar) // all plans — emotion pattern radar
	auth.POST("/streak/freeze", moodHandler.UseFreeze)

	// Device registration (FCM tokens)
	auth.POST("/devices", moodHandler.RegisterDevice)

	// Weekly reviews — Plus+ only (gated in handler)
	reviewHandler := NewWeeklyReviewHandler(deps.WeeklyReviewRepo)
	auth.GET("/reviews/weekly", reviewHandler.List)
	auth.GET("/reviews/weekly/latest", reviewHandler.GetLatest)

	// Annual reviews — Plus+ only (gated in handler)
	annualReviewHandler := NewAnnualReviewHandler(deps.AnnualReviewRepo)
	auth.GET("/reviews/annual", annualReviewHandler.List)
	auth.GET("/reviews/annual/latest", annualReviewHandler.GetLatest)

	// Therapist share links (5a) — Plus+ only (gated in handler)
	shareHandler := NewShareHandler(deps.ShareRepo, deps.SubscriptionSvc, deps.AppBaseURL)
	auth.POST("/share", shareHandler.Create)
	auth.GET("/share", shareHandler.List)
	auth.DELETE("/share/:id", shareHandler.Revoke)
	// Public — no auth middleware; passcode in query param
	r.GET("/share/:token", shareHandler.View)

	// PDF export (5d) — Pro+ only (gated in handler)
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
	auth.GET("/journeys", journeyHandler.ListTemplates)
	auth.POST("/journeys/:journeyID/start", journeyHandler.StartSession)
	auth.GET("/journeys/sessions", journeyHandler.ListSessions)
	auth.GET("/journeys/sessions/:sessionID", journeyHandler.GetSession)
	auth.POST("/journeys/sessions/:sessionID/advance", journeyHandler.AdvanceSession)

	// Life Chapters — user-defined time periods with themes
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

	// Therapist dashboard (5g)
	therapistHandler := NewTherapistHandler(deps.TherapistRepo, deps.AnalysisRepo, deps.ClaudeSvc)
	auth.POST("/therapists/register", therapistHandler.Register)
	auth.POST("/therapists/clients/link", therapistHandler.LinkClient)
	auth.DELETE("/therapists/clients/:clientID", therapistHandler.UnlinkClient)
	auth.GET("/therapists/clients", therapistHandler.ListClients)
	auth.GET("/therapists/clients/:clientID/brief", therapistHandler.ClientBrief)

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
