package config

import (
	"strings"
	"testing"
)

func TestLoad_AllRequiredKeysSet(t *testing.T) {
	// Set all required env vars; prevent .env from being read
	t.Setenv("DATABASE_URL", "postgres://test")
	t.Setenv("SUPABASE_URL", "https://test.supabase.co")
	t.Setenv("SUPABASE_ANON_KEY", "test-anon-key")
	t.Setenv("FINNHUB_API_KEY", "test-api-key")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.DatabaseURL != "postgres://test" {
		t.Errorf("DatabaseURL = %q, want %q", cfg.DatabaseURL, "postgres://test")
	}
	if cfg.SupabaseURL != "https://test.supabase.co" {
		t.Errorf("SupabaseURL = %q, want %q", cfg.SupabaseURL, "https://test.supabase.co")
	}
	if cfg.SupabaseAnonKey != "test-anon-key" {
		t.Errorf("SupabaseAnonKey = %q, want %q", cfg.SupabaseAnonKey, "test-anon-key")
	}
	if cfg.FinnhubAPIKey != "test-api-key" {
		t.Errorf("FinnhubAPIKey = %q, want %q", cfg.FinnhubAPIKey, "test-api-key")
	}
}

func TestLoad_MissingDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("SUPABASE_URL", "https://test.supabase.co")
	t.Setenv("SUPABASE_ANON_KEY", "test-anon-key")
	t.Setenv("FINNHUB_API_KEY", "test-api-key")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "DATABASE_URL") {
		t.Errorf("Load: got %v, want error containing 'DATABASE_URL'", err)
	}
}

func TestLoad_MissingSupabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://test")
	t.Setenv("SUPABASE_URL", "")
	t.Setenv("SUPABASE_ANON_KEY", "test-anon-key")
	t.Setenv("FINNHUB_API_KEY", "test-api-key")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "SUPABASE_URL") {
		t.Errorf("Load: got %v, want error containing 'SUPABASE_URL'", err)
	}
}

func TestLoad_MissingSupabaseAnonKey(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://test")
	t.Setenv("SUPABASE_URL", "https://test.supabase.co")
	t.Setenv("SUPABASE_ANON_KEY", "")
	t.Setenv("FINNHUB_API_KEY", "test-api-key")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "SUPABASE_ANON_KEY") {
		t.Errorf("Load: got %v, want error containing 'SUPABASE_ANON_KEY'", err)
	}
}

func TestLoad_MissingFinnhubAPIKey(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://test")
	t.Setenv("SUPABASE_URL", "https://test.supabase.co")
	t.Setenv("SUPABASE_ANON_KEY", "test-anon-key")
	t.Setenv("FINNHUB_API_KEY", "")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "FINNHUB_API_KEY") {
		t.Errorf("Load: got %v, want error containing 'FINNHUB_API_KEY'", err)
	}
}

func TestLoad_DefaultPort(t *testing.T) {
	t.Setenv("PORT", "")
	t.Setenv("DATABASE_URL", "postgres://test")
	t.Setenv("SUPABASE_URL", "https://test.supabase.co")
	t.Setenv("SUPABASE_ANON_KEY", "test-anon-key")
	t.Setenv("FINNHUB_API_KEY", "test-api-key")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.Port != "8080" {
		t.Errorf("Port = %q, want %q", cfg.Port, "8080")
	}
}

func TestLoad_DefaultGinMode(t *testing.T) {
	t.Setenv("GIN_MODE", "")
	t.Setenv("DATABASE_URL", "postgres://test")
	t.Setenv("SUPABASE_URL", "https://test.supabase.co")
	t.Setenv("SUPABASE_ANON_KEY", "test-anon-key")
	t.Setenv("FINNHUB_API_KEY", "test-api-key")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.GinMode != "debug" {
		t.Errorf("GinMode = %q, want %q", cfg.GinMode, "debug")
	}
}

func TestLoad_DefaultRateLimits(t *testing.T) {
	t.Setenv("PUBLIC_RATE_LIMIT", "")
	t.Setenv("AUTH_RATE_LIMIT", "")
	t.Setenv("DATABASE_URL", "postgres://test")
	t.Setenv("SUPABASE_URL", "https://test.supabase.co")
	t.Setenv("SUPABASE_ANON_KEY", "test-anon-key")
	t.Setenv("FINNHUB_API_KEY", "test-api-key")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.PublicRateLimit != 20 {
		t.Errorf("PublicRateLimit = %d, want 20", cfg.PublicRateLimit)
	}
	if cfg.AuthRateLimit != 60 {
		t.Errorf("AuthRateLimit = %d, want 60", cfg.AuthRateLimit)
	}
}

func TestLoad_AllowedOriginsCommaSeparated(t *testing.T) {
	t.Setenv("ALLOWED_ORIGINS", "http://localhost:4200, https://app.example.com , https://admin.example.com")
	t.Setenv("DATABASE_URL", "postgres://test")
	t.Setenv("SUPABASE_URL", "https://test.supabase.co")
	t.Setenv("SUPABASE_ANON_KEY", "test-anon-key")
	t.Setenv("FINNHUB_API_KEY", "test-api-key")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(cfg.AllowedOrigins) != 3 {
		t.Errorf("AllowedOrigins len = %d, want 3", len(cfg.AllowedOrigins))
	}
	if cfg.AllowedOrigins[0] != "http://localhost:4200" {
		t.Errorf("AllowedOrigins[0] = %q, want %q", cfg.AllowedOrigins[0], "http://localhost:4200")
	}
	if cfg.AllowedOrigins[1] != "https://app.example.com" {
		t.Errorf("AllowedOrigins[1] = %q, want %q (should be trimmed)", cfg.AllowedOrigins[1], "https://app.example.com")
	}
	if cfg.AllowedOrigins[2] != "https://admin.example.com" {
		t.Errorf("AllowedOrigins[2] = %q, want %q (should be trimmed)", cfg.AllowedOrigins[2], "https://admin.example.com")
	}
}

func TestLoad_AllowedOriginsSingleValue(t *testing.T) {
	t.Setenv("ALLOWED_ORIGINS", "http://localhost:3000")
	t.Setenv("DATABASE_URL", "postgres://test")
	t.Setenv("SUPABASE_URL", "https://test.supabase.co")
	t.Setenv("SUPABASE_ANON_KEY", "test-anon-key")
	t.Setenv("FINNHUB_API_KEY", "test-api-key")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(cfg.AllowedOrigins) != 1 {
		t.Errorf("AllowedOrigins len = %d, want 1", len(cfg.AllowedOrigins))
	}
	if cfg.AllowedOrigins[0] != "http://localhost:3000" {
		t.Errorf("AllowedOrigins[0] = %q, want %q", cfg.AllowedOrigins[0], "http://localhost:3000")
	}
}

func TestLoad_FinnhubDefaults(t *testing.T) {
	t.Setenv("FINNHUB_BASE_URL", "")
	t.Setenv("FINNHUB_WS_URL", "")
	t.Setenv("DATABASE_URL", "postgres://test")
	t.Setenv("SUPABASE_URL", "https://test.supabase.co")
	t.Setenv("SUPABASE_ANON_KEY", "test-anon-key")
	t.Setenv("FINNHUB_API_KEY", "test-api-key")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.FinnhubBaseURL != "https://finnhub.io/api/v1" {
		t.Errorf("FinnhubBaseURL = %q, want %q", cfg.FinnhubBaseURL, "https://finnhub.io/api/v1")
	}
	if cfg.FinnhubWSURL != "wss://ws.finnhub.io" {
		t.Errorf("FinnhubWSURL = %q, want %q", cfg.FinnhubWSURL, "wss://ws.finnhub.io")
	}
}
