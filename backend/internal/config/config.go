package config

import (
	"fmt"
	"os"
	"strconv"
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
	URL       string // e.g. https://xxxx.supabase.co — used to build JWKS URL for ES256 tokens
}

type OpenAIConfig struct {
	APIKey  string
	BaseURL string // override for testing
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
	BaseURL string // e.g. "https://dreamlog.app" — used to build share URLs
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
			BaseURL: getEnv("APP_BASE_URL", "https://dreamlog.app"),
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

func requireEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		panic(fmt.Sprintf("required environment variable %q is not set", key))
	}
	return v
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func parseInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

func parseBool(key string, fallback bool) bool {
	if v := os.Getenv(key); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return fallback
}

func parseDuration(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
}
