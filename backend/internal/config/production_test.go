package config

import (
	"testing"
	"time"
)

// M9: production refuses dev stubs and missing safety-critical secrets.
func TestProductionProblems(t *testing.T) {
	dev := &Config{AppEnv: "development", Anthropic: AnthropicConfig{StubAnalysis: true}}
	if len(dev.ProductionProblems(true)) != 0 {
		t.Error("checks must only apply when APP_ENV=production")
	}

	prod := &Config{AppEnv: "production", Anthropic: AnthropicConfig{StubAnalysis: true}}
	if got := len(prod.ProductionProblems(true)); got < 5 {
		t.Errorf("stub analysis + missing keys + missing IAP must all be reported, got %d", got)
	}

	ok := &Config{
		AppEnv:    "production",
		Anthropic: AnthropicConfig{APIKey: "k"},
		Security:  SecurityConfig{MasterEncryptionKey: "m"},
		Supabase:  SupabaseConfig{URL: "https://x.supabase.co"},
		IAP:       IAPConfig{AppleSharedSecret: "a", GooglePlayCredentialsJSON: "{}"},
	}
	if p := ok.ProductionProblems(true); len(p) != 0 {
		t.Errorf("complete production config must pass, got %v", p)
	}
	ok.IAP = IAPConfig{}
	if p := ok.ProductionProblems(false); len(p) != 0 {
		t.Errorf("worker does not need IAP credentials, got %v", p)
	}
}

func TestLoad_CORSDefaultsToLocalhostDevOrigins(t *testing.T) {
	for k, v := range map[string]string{
		"DATABASE_URL": "postgres://x", "STORAGE_ENDPOINT": "http://localhost:9000",
		"STORAGE_ACCESS_KEY_ID": "a", "STORAGE_SECRET_ACCESS_KEY": "b", "STORAGE_BUCKET": "c",
		"SUPABASE_JWT_SECRET": "s", "CORS_ALLOWED_ORIGINS": "",
	} {
		t.Setenv(k, v)
	}
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, o := range cfg.CORSAllowedOrigins {
		if o == "http://localhost:3000" {
			found = true
		}
	}
	if !found {
		t.Errorf("localhost dev origin must be allowed by default, got %v", cfg.CORSAllowedOrigins)
	}
}

// N8: production checks fail closed - only an explicit development/test
// environment relaxes them; unset or unknown values are production.
func TestIsProduction_FailsClosed(t *testing.T) {
	for env, want := range map[string]bool{
		"":            true,
		"production":  true,
		"prod":        true,
		"staging":     true,
		"Production ": true,
		"development": false,
		"dev":         false,
		"local":       false,
		"test":        false,
		" Test ":      false,
	} {
		if got := (&Config{AppEnv: env}).IsProduction(); got != want {
			t.Errorf("APP_ENV=%q: IsProduction=%v, want %v", env, got, want)
		}
	}
}

// N8: forgetting APP_ENV on a service with dev stubs must refuse to start.
func TestProductionProblems_UnsetAppEnvIsChecked(t *testing.T) {
	cfg := &Config{AppEnv: "", Anthropic: AnthropicConfig{StubAnalysis: true}}
	if len(cfg.ProductionProblems(true)) == 0 {
		t.Fatal("an unset APP_ENV must get production checks")
	}
}

func TestLoad_AppEnvHasNoDevDefault(t *testing.T) {
	for k, v := range map[string]string{
		"DATABASE_URL": "postgres://x", "STORAGE_ENDPOINT": "http://localhost:9000",
		"STORAGE_ACCESS_KEY_ID": "a", "STORAGE_SECRET_ACCESS_KEY": "b", "STORAGE_BUCKET": "c",
		"SUPABASE_JWT_SECRET": "s", "APP_ENV": "",
	} {
		t.Setenv(k, v)
	}
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.IsProduction() {
		t.Errorf("Load without APP_ENV must be treated as production, got AppEnv=%q", cfg.AppEnv)
	}
}

// ALLOW_UNVERIFIED_IAP waives only the store-credentials check - every other
// production check stays on - and the startup warning is loud about it.
func TestProductionProblems_AllowUnverifiedIAP(t *testing.T) {
	full := func() *Config {
		return &Config{
			AppEnv:    "production",
			Anthropic: AnthropicConfig{APIKey: "k"},
			Security:  SecurityConfig{MasterEncryptionKey: "m"},
			Supabase:  SupabaseConfig{URL: "https://x.supabase.co"},
		}
	}
	c := full()
	if len(c.ProductionProblems(true)) != 1 {
		t.Fatalf("missing store credentials must be the only problem, got %v", c.ProductionProblems(true))
	}
	c.IAP.AllowUnverified = true
	if p := c.ProductionProblems(true); len(p) != 0 {
		t.Fatalf("the acknowledgement must waive the store-credentials check, got %v", p)
	}
	if c.UnverifiedIAPWarning() == "" {
		t.Error("running unverified in production must warn")
	}

	// Other checks stay on.
	c.Anthropic.StubAnalysis = true
	c.Security.MasterEncryptionKey = ""
	if got := len(c.ProductionProblems(true)); got != 2 {
		t.Errorf("stub analysis and a missing master key must still be reported, got %d", got)
	}

	// No warning once verification is configured, and none outside production.
	ok := full()
	ok.IAP = IAPConfig{AppleSharedSecret: "a", GooglePlayCredentialsJSON: "{}", AllowUnverified: true}
	if ok.UnverifiedIAPWarning() != "" {
		t.Error("configured verification needs no warning")
	}
	dev := &Config{AppEnv: "development", IAP: IAPConfig{AllowUnverified: true}}
	if dev.UnverifiedIAPWarning() != "" {
		t.Error("development needs no warning")
	}
}

func TestLoad_PreviousMasterKeyAndPollTimeoutDefaults(t *testing.T) {
	for k, v := range map[string]string{
		"DATABASE_URL": "postgres://x", "STORAGE_ENDPOINT": "http://localhost:9000",
		"STORAGE_ACCESS_KEY_ID": "a", "STORAGE_SECRET_ACCESS_KEY": "b", "STORAGE_BUCKET": "c",
		"SUPABASE_JWT_SECRET": "s", "MASTER_ENCRYPTION_KEY_PREVIOUS": "", "WORKER_POLL_TIMEOUT": "",
	} {
		t.Setenv(k, v)
	}
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Security.PreviousMasterEncryptionKey != "" {
		t.Error("no previous master key by default")
	}
	// A long blocking pop costs nothing in latency (BRPOP returns on push) but
	// every timeout is a billed Redis request, so the default is not short.
	if cfg.Worker.PollTimeout < 30*time.Second {
		t.Errorf("default queue poll timeout must be at least 30s, got %v", cfg.Worker.PollTimeout)
	}
	t.Setenv("MASTER_ENCRYPTION_KEY_PREVIOUS", "old-key-value")
	t.Setenv("WORKER_POLL_TIMEOUT", "45s")
	cfg, err = Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Security.PreviousMasterEncryptionKey != "old-key-value" || cfg.Worker.PollTimeout != 45*time.Second {
		t.Errorf("settings not read: %q %v", cfg.Security.PreviousMasterEncryptionKey, cfg.Worker.PollTimeout)
	}
}
