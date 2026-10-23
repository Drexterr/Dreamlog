package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all runtime configuration loaded from environment variables.
// No global state: pass *Config through dependency injection.
type Config struct {
	// AppEnv names the deployment. Anything other than an explicit
	// development/test value - including unset - is production, which refuses
	// to start with dev stubs or missing safety-critical secrets (see
	// ProductionProblems). Fail closed: forgetting APP_ENV must not quietly
	// grant paid plans for free (N8).
	AppEnv string
	// TrustedProxies are the CIDRs whose X-Forwarded-For hops are trusted when
	// resolving the client IP (rate limits, security logs). Defaults to private
	// ranges: the platform edge appends the real client IP after any
	// client-supplied (spoofable) entries.
	TrustedProxies []string
	// TrustedPlatformHeader, when set, is read for the client IP instead
	// (e.g. "CF-Connecting-IP", "Fly-Client-IP").
	TrustedPlatformHeader string
	// CORSAllowedOrigins is the browser-origin allowlist (therapist portal).
	CORSAllowedOrigins []string
	// AdminEmails may use the operator endpoints (/admin/*, e.g. the DLQ
	// tools). Only accounts with a VERIFIED email on this list qualify; empty
	// (the default) disables the admin routes entirely (404).
	AdminEmails []string
	Server    ServerConfig
	Database  DatabaseConfig
	Redis     RedisConfig
	Storage   StorageConfig
	Supabase  SupabaseConfig
	OpenAI    OpenAIConfig
	AzureTTS  AzureTTSConfig
	Anthropic AnthropicConfig
	FCM       FCMConfig
	App       AppConfig
	Worker    WorkerConfig
	IAP       IAPConfig
	Security  SecurityConfig
	Sentry    SentryConfig
}

// SentryConfig configures error reporting. A blank DSN (dev) disables
// Sentry entirely - all monitoring helpers become no-ops.
type SentryConfig struct {
	DSN string
}

// SecurityConfig holds application-layer encryption settings.
type SecurityConfig struct {
	// MasterEncryptionKey encrypts per-therapist data keys (envelope
	// encryption for therapist session notes). 32 bytes as 64 hex chars or
	// base64. When unset, a key is derived from SUPABASE_JWT_SECRET so dev
	// works with no extra setup - set it explicitly in production so key
	// rotation is decoupled from the JWT secret.
	MasterEncryptionKey string
	// RejectLegacyCiphertext refuses legacy (v1, unbound) therapist-note
	// ciphertexts on read (N15). Enable once the startup re-seal reports no
	// legacy rows remain.
	RejectLegacyCiphertext bool
}

type ServerConfig struct {
	Port         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

type DatabaseConfig struct {
	DSN             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

type RedisConfig struct {
	Addr     string
	Password string
	DB       int
	TLS      bool // set true for Upstash and other TLS-only providers
}

type StorageConfig struct {
	Endpoint        string
	AccessKeyID     string
	SecretAccessKey string
	Bucket          string
	Region          string
	UsePathStyle    bool   // true for MinIO, false for R2/S3
	PublicBaseURL   string // optional CDN URL
	ProxyBaseURL    string // when set, upload URLs point to backend proxy instead of storage
	PresignExpiry   time.Duration
}

type SupabaseConfig struct {
	JWTSecret string
	URL       string // e.g. https://xxxx.supabase.co - used to build JWKS URL for ES256 tokens
}

type OpenAIConfig struct {
	APIKey  string
	BaseURL string // override for testing
}

// AzureTTSConfig configures Azure Speech text-to-speech for Therapy Mode voice output.
// When Key+Region are set, Azure is used instead of OpenAI TTS (empathetic SSML styles,
// Hindi voices). When unset, TTS falls back to OpenAI, or is skipped entirely in dev.
type AzureTTSConfig struct {
	Key           string
	Region        string // e.g. "centralindia", "eastus"
	BaseURL       string // override for testing; defaults to https://{region}.tts.speech.microsoft.com
	UseHD         bool   // use per-persona DragonHD multilingual voices (emotion auto-detected, EN+HI+Hinglish in one voice; ~$22/1M chars vs ~$15 standard)
	VoiceOverride string // optional: force one voice for all personas/languages, e.g. "en-IN-Aarti:DragonHDLatestNeural"; wins over UseHD
}

type AnthropicConfig struct {
	APIKey       string
	BaseURL      string // override for testing
	Model        string
	StubAnalysis bool // when true, skip API calls and return a fake response (dev only)
}

type FCMConfig struct {
	CredentialsJSON string // path to service account JSON file
	ProjectID       string
}

type AppConfig struct {
	BaseURL           string // e.g. "https://dreamlog.app" - used to build share URLs
	MinimumAppVersion string // oldest mobile version allowed; older installs see a force-update screen
	AndroidStoreURL   string // Play Store listing URL for the Update Now button
	IOSStoreURL       string // App Store listing URL; empty until the app is live on the App Store
}

// IAPConfig holds credentials for server-side In-App Purchase verification.
// Purchases happen in the store SDKs on-device; the backend only ever verifies
// receipts/tokens against Apple and Google before granting a plan.
type IAPConfig struct {
	AppleSharedSecret         string // App Store Connect shared secret for /verifyReceipt
	GooglePlayCredentialsJSON string // service-account JSON with androidpublisher access
	GooglePlayPackageName     string // Android application ID, e.g. com.talktoode.app
	AppleBundleID             string // iOS bundle ID every receipt must carry (default com.talktoode.app)
	// SandboxAllowedEmails may redeem sandbox / TestFlight / Play test
	// purchases (App Review demo account, internal testers). Everyone else
	// needs a real store purchase.
	SandboxAllowedEmails []string
}

type WorkerConfig struct {
	Concurrency   int
	MaxRetries    int
	QueueKey      string
	DLQKey        string
	NotesQueueKey string // therapist note-photo OCR jobs (separate list from entry jobs)
	NotesDLQKey   string
	PollTimeout   time.Duration
}

func Load() (*Config, error) {
	cfg := &Config{
		AppEnv:                getEnv("APP_ENV", ""),
		TrustedProxies:        splitCommaList(getEnv("TRUSTED_PROXIES", "10.0.0.0/8,172.16.0.0/12,192.168.0.0/16,100.64.0.0/10,127.0.0.0/8,fc00::/7,::1/128")),
		TrustedPlatformHeader: getEnv("TRUSTED_PLATFORM_HEADER", ""),
		CORSAllowedOrigins:    splitCommaList(getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:5173,http://localhost:3000")),
		AdminEmails:           splitCommaList(getEnv("ADMIN_EMAILS", "")),
		Server: ServerConfig{
			Port:         getEnv("PORT", "8080"),
			ReadTimeout:  parseDuration("SERVER_READ_TIMEOUT", 30*time.Second),
			WriteTimeout: parseDuration("SERVER_WRITE_TIMEOUT", 30*time.Second),
			IdleTimeout:  parseDuration("SERVER_IDLE_TIMEOUT", 60*time.Second),
		},
		Database: DatabaseConfig{
			DSN:             requireEnv("DATABASE_URL"),
			MaxOpenConns:    parseInt("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns:    parseInt("DB_MAX_IDLE_CONNS", 5),
			ConnMaxLifetime: parseDuration("DB_CONN_MAX_LIFETIME", 5*time.Minute),
		},
		Redis: RedisConfig{
			Addr:     getEnv("REDIS_ADDR", "localhost:6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       parseInt("REDIS_DB", 0),
			TLS:      parseBool("REDIS_TLS", false),
		},
		Storage: StorageConfig{
			Endpoint:        mustBeValidURL("STORAGE_ENDPOINT", requireEnv("STORAGE_ENDPOINT")),
			AccessKeyID:     requireEnv("STORAGE_ACCESS_KEY_ID"),
			SecretAccessKey: requireEnv("STORAGE_SECRET_ACCESS_KEY"),
			Bucket:          requireEnv("STORAGE_BUCKET"),
			Region:          getEnv("STORAGE_REGION", "auto"),
			UsePathStyle:    parseBool("STORAGE_USE_PATH_STYLE", false),
			PublicBaseURL:   getEnv("STORAGE_PUBLIC_BASE_URL", ""),
			ProxyBaseURL:    getEnv("STORAGE_PROXY_BASE_URL", ""),
			PresignExpiry:   parseDuration("STORAGE_PRESIGN_EXPIRY", 15*time.Minute),
		},
		Supabase: SupabaseConfig{
			JWTSecret: requireEnv("SUPABASE_JWT_SECRET"),
			URL:       getEnv("SUPABASE_URL", ""),
		},
		OpenAI: OpenAIConfig{
			APIKey:  getEnv("OPENAI_API_KEY", ""),
			BaseURL: getEnv("OPENAI_BASE_URL", "https://api.openai.com/v1"),
		},
		AzureTTS: AzureTTSConfig{
			Key:           getEnv("AZURE_TTS_KEY", ""),
			Region:        getEnv("AZURE_TTS_REGION", ""),
			BaseURL:       getEnv("AZURE_TTS_BASE_URL", ""),
			UseHD:         parseBool("AZURE_TTS_USE_HD", false),
			VoiceOverride: getEnv("AZURE_TTS_VOICE_OVERRIDE", ""),
		},
		Anthropic: AnthropicConfig{
			APIKey:       getEnv("ANTHROPIC_API_KEY", ""),
			BaseURL:      getEnv("ANTHROPIC_BASE_URL", "https://api.anthropic.com"),
			Model:        getEnv("ANTHROPIC_MODEL", "claude-sonnet-4-6"),
			StubAnalysis: parseBool("STUB_AI_ANALYSIS", false),
		},
		FCM: FCMConfig{
			CredentialsJSON: getEnv("FCM_CREDENTIALS_JSON", ""),
			ProjectID:       getEnv("FCM_PROJECT_ID", ""),
		},
		App: AppConfig{
			BaseURL:           getEnv("APP_BASE_URL", "https://dreamlog.app"),
			MinimumAppVersion: getEnv("MINIMUM_APP_VERSION", "1.0.0"),
			AndroidStoreURL:   getEnv("ANDROID_STORE_URL", "https://play.google.com/store/apps/details?id=com.talktoode.app"),
			IOSStoreURL:       getEnv("IOS_STORE_URL", ""),
		},
		IAP: IAPConfig{
			AppleSharedSecret:         getEnv("APPLE_SHARED_SECRET", ""),
			GooglePlayCredentialsJSON: getEnv("GOOGLE_PLAY_CREDENTIALS_JSON", ""),
			GooglePlayPackageName:     getEnv("GOOGLE_PLAY_PACKAGE_NAME", "com.talktoode.app"),
			AppleBundleID:             getEnv("APPLE_BUNDLE_ID", "com.talktoode.app"),
			SandboxAllowedEmails:      splitCommaList(getEnv("IAP_SANDBOX_ALLOWED_EMAILS", "")),
		},
		Worker: WorkerConfig{
			Concurrency:   parseInt("WORKER_CONCURRENCY", 4),
			MaxRetries:    parseInt("WORKER_MAX_RETRIES", 3),
			QueueKey:      getEnv("WORKER_QUEUE_KEY", "dreamlog:transcription:queue"),
			DLQKey:        getEnv("WORKER_DLQ_KEY", "dreamlog:transcription:dlq"),
			NotesQueueKey: getEnv("WORKER_NOTES_QUEUE_KEY", "dreamlog:notes:queue"),
			NotesDLQKey:   getEnv("WORKER_NOTES_DLQ_KEY", "dreamlog:notes:dlq"),
			PollTimeout:   parseDuration("WORKER_POLL_TIMEOUT", 5*time.Second),
		},
		Security: SecurityConfig{
			MasterEncryptionKey:    getEnv("MASTER_ENCRYPTION_KEY", ""),
			RejectLegacyCiphertext: parseBool("REJECT_LEGACY_CIPHERTEXT", false),
		},
		Sentry: SentryConfig{
			DSN: getEnv("SENTRY_DSN", ""),
		},
	}
	return cfg, nil
}

// mustBeValidURL parses value as a URL and panics with a clear diagnostic if
// it's malformed - e.g. a stray space pasted into the middle of a hosting
// dashboard field, which TrimSpace alone can't catch since it only trims the
// ends. Fails loudly at startup instead of every request that depends on
// this URL failing silently and surfacing a raw SDK error deep in a worker.
func mustBeValidURL(key, value string) string {
	if _, err := url.Parse(value); err != nil {
		panic(fmt.Sprintf("environment variable %q is not a valid URL: %v", key, err))
	}
	return value
}

// env reads an environment variable with surrounding whitespace stripped.
// Stray spaces/newlines pasted into hosting dashboards (e.g. a trailing space
// in STORAGE_ENDPOINT) otherwise produce hard-to-trace runtime failures like
// `failed to parse endpoint URL: invalid character " " in host name`.
func env(key string) string {
	return strings.TrimSpace(os.Getenv(key))
}

func requireEnv(key string) string {
	v := env(key)
	if v == "" {
		panic(fmt.Sprintf("required environment variable %q is not set", key))
	}
	return v
}

func getEnv(key, fallback string) string {
	if v := env(key); v != "" {
		return v
	}
	return fallback
}

func parseInt(key string, fallback int) int {
	if v := env(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

func parseBool(key string, fallback bool) bool {
	if v := env(key); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return fallback
}

func parseDuration(key string, fallback time.Duration) time.Duration {
	if v := env(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
}

// splitCommaList splits a comma-separated env value, dropping blanks.
func splitCommaList(v string) []string {
	var out []string
	for _, part := range strings.Split(v, ",") {
		if p := strings.TrimSpace(part); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// nonProductionEnvs are the only APP_ENV values that relax production checks.
var nonProductionEnvs = map[string]bool{"development": true, "dev": true, "local": true, "test": true}

// IsProduction reports whether production checks apply: every APP_ENV except
// an explicit development/test value, including an unset one (N8).
func (c *Config) IsProduction() bool {
	return !nonProductionEnvs[strings.ToLower(strings.TrimSpace(c.AppEnv))]
}

// ProductionProblems lists configuration that is unsafe in production: dev
// stubs that grant paid plans or fake AI output, and missing secrets that
// would silently weaken safety or encryption. Empty outside production.
// api=true also checks API-only settings (store verification).
func (c *Config) ProductionProblems(api bool) []string {
	if !c.IsProduction() {
		return nil
	}
	var p []string
	if c.Anthropic.StubAnalysis {
		p = append(p, "STUB_AI_ANALYSIS must be false")
	}
	if c.Anthropic.APIKey == "" {
		p = append(p, "ANTHROPIC_API_KEY is required (crisis Stage 2 and reflections)")
	}
	if c.Security.MasterEncryptionKey == "" {
		p = append(p, "MASTER_ENCRYPTION_KEY is required (no JWT-secret-derived fallback in production)")
	}
	if c.Supabase.URL == "" {
		p = append(p, "SUPABASE_URL is required")
	}
	if c.Storage.ProxyBaseURL != "" {
		p = append(p, "STORAGE_PROXY_BASE_URL must be unset (dev-only unauthenticated upload proxy)")
	}
	if api && (c.IAP.AppleSharedSecret == "" || c.IAP.GooglePlayCredentialsJSON == "") {
		p = append(p, "APPLE_SHARED_SECRET and GOOGLE_PLAY_CREDENTIALS_JSON are required (otherwise /billing/upgrade grants plans without a purchase)")
	}
	return p
}
