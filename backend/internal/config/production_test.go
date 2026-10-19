package config

import "testing"

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
