// Package config handles application configuration using Viper.
// It reads from .env files and environment variables, with env vars
// taking precedence over file values.
package config

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/spf13/viper"
)

// Market data cache TTLs.
const (
	// QuoteCacheTTL is how long a real-time quote is considered fresh.
	QuoteCacheTTL = 10 // seconds

	// HistoricalCacheTTL is how long historical bar data is cached.
	// Historical bars are immutable once the market closes for that period.
	HistoricalCacheTTL = 24 * 60 * 60 // seconds (24 hours)
)

// Config holds all configuration values for the application.
type Config struct {
	Port           string
	GinMode        string
	DatabaseURL     string
	SupabaseURL     string
	SupabaseAnonKey string // Used by the Go auth proxy to call Supabase token refresh.
	AllowedOrigins  []string // CORS: exact frontend origins, never wildcard
	// Rate limits (requests per second)
	PublicRateLimit int // per-IP, for unauthenticated endpoints
	AuthRateLimit   int // per-user, for authenticated endpoints
	// Finnhub market data
	FinnhubAPIKey string
	FinnhubBaseURL string
	FinnhubWSURL   string
}

// Load reads configuration from .env file and environment variables.
// Environment variables take precedence over .env file values.
func Load() (*Config, error) {
	viper.SetConfigFile(".env")
	viper.SetConfigType("env")

	viper.AutomaticEnv()

	viper.SetDefault("PORT", "8080")
	viper.SetDefault("GIN_MODE", "debug")
	viper.SetDefault("ALLOWED_ORIGINS", "http://localhost:4200")
	viper.SetDefault("PUBLIC_RATE_LIMIT", 20)
	viper.SetDefault("AUTH_RATE_LIMIT", 60)
	viper.SetDefault("FINNHUB_BASE_URL", "https://finnhub.io/api/v1")
	viper.SetDefault("FINNHUB_WS_URL", "wss://ws.finnhub.io")

	if err := viper.ReadInConfig(); err != nil {
		slog.Warn("no .env file found, using environment variables", "error", err)
	}

	rawOrigins := viper.GetString("ALLOWED_ORIGINS")
	origins := []string{}
	for _, o := range strings.Split(rawOrigins, ",") {
		if trimmed := strings.TrimSpace(o); trimmed != "" {
			origins = append(origins, trimmed)
		}
	}

	cfg := &Config{
		Port:            viper.GetString("PORT"),
		GinMode:         viper.GetString("GIN_MODE"),
		DatabaseURL:     viper.GetString("DATABASE_URL"),
		SupabaseURL:     viper.GetString("SUPABASE_URL"),
		SupabaseAnonKey: viper.GetString("SUPABASE_ANON_KEY"),
		AllowedOrigins:  origins,
		PublicRateLimit: viper.GetInt("PUBLIC_RATE_LIMIT"),
		AuthRateLimit:   viper.GetInt("AUTH_RATE_LIMIT"),
		FinnhubAPIKey:   viper.GetString("FINNHUB_API_KEY"),
		FinnhubBaseURL:  viper.GetString("FINNHUB_BASE_URL"),
		FinnhubWSURL:    viper.GetString("FINNHUB_WS_URL"),
	}

	if cfg.DatabaseURL == "" {
		return cfg, fmt.Errorf("DATABASE_URL is not set")
	}
	if cfg.SupabaseURL == "" {
		return cfg, fmt.Errorf("SUPABASE_URL is not set")
	}
	if cfg.SupabaseAnonKey == "" {
		return cfg, fmt.Errorf("SUPABASE_ANON_KEY is not set")
	}
	if cfg.FinnhubAPIKey == "" {
		return cfg, fmt.Errorf("FINNHUB_API_KEY is not set")
	}

	return cfg, nil
}
