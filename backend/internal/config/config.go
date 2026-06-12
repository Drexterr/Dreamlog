package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all runtime configuration loaded from environment variables.
// No global state: pass *Config through dependency injection.
type Config struct {
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
	Stripe    StripeConfig
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
	APIKey      string
	BaseURL     string // override for testing
	Model       string
	StubAnalysis bool  // when true, skip API calls and return a fake response (dev only)
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

type StripeConfig struct {
	SecretKey      string
	PublishableKey string
}

type WorkerConfig struct {
	Concurrency   int
	MaxRetries    int
	QueueKey      string
	DLQKey        string
	PollTimeout   time.Duration
}

func Load() (*Config, error) {
	cfg := &Config{
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
			Endpoint:        requireEnv("STORAGE_ENDPOINT"),
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
			AndroidStoreURL:   getEnv("ANDROID_STORE_URL", "https://play.google.com/store/apps/details?id=com.dreamlog.app"),
			IOSStoreURL:       getEnv("IOS_STORE_URL", ""),
		},
		Stripe: StripeConfig{
			SecretKey:      getEnv("STRIPE_SECRET_KEY", ""),
			PublishableKey: getEnv("STRIPE_PUBLISHABLE_KEY", ""),
		},
		Worker: WorkerConfig{
			Concurrency: parseInt("WORKER_CONCURRENCY", 4),
			MaxRetries:  parseInt("WORKER_MAX_RETRIES", 3),
			QueueKey:    getEnv("WORKER_QUEUE_KEY", "dreamlog:transcription:queue"),
			DLQKey:      getEnv("WORKER_DLQ_KEY", "dreamlog:transcription:dlq"),
			PollTimeout: parseDuration("WORKER_POLL_TIMEOUT", 5*time.Second),
		},
	}
	return cfg, nil
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
