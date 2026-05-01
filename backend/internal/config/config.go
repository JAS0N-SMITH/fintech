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

	// SymbolsCacheTTL is how long the exchange symbol list is cached.
	// Symbol lists change infrequently; 24 hours is appropriate.
	SymbolsCacheTTL = 24 * 60 * 60 // seconds (24 hours)
)

// Config holds all configuration values for the application.
type Config struct {
	Port            string
	GinMode         string
	DatabaseURL     string
	SupabaseURL     string
	SupabaseAnonKey string   // Used by the Go auth proxy to call Supabase token refresh.
	AllowedOrigins  []string // CORS: exact frontend origins, never wildcard
	// TrustedProxies is the list of proxy IPs/CIDRs whose X-Forwarded-For headers
	// are trusted for real client IP resolution. Set to nil to trust no proxies
	// (safe for direct connections). In production behind NGINX or a load balancer,
	// set to the proxy's IP (e.g. "127.0.0.1").
	TrustedProxies []string
	// Rate limits (requests per second)
	PublicRateLimit int // per-IP, for unauthenticated endpoints
	AuthRateLimit   int // per-user, for authenticated endpoints
	// Finnhub market data
	FinnhubAPIKey  string
	FinnhubBaseURL string
	FinnhubWSURL   string
	// Polygon.io market data (optional fallback for historical bars)
	PolygonAPIKey string
	// Redaction settings for redactlog
	RedactEnabled           bool
	RedactRequestBody       bool
	RedactResponseBody      bool
	RedactSensitiveQueryParams []string
	RedactHeaderDenylist    []string
	RedactPaths             []string
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
	viper.SetDefault("TRUSTED_PROXIES", "") // empty = trust no proxies (direct connection)
	viper.SetDefault("PUBLIC_RATE_LIMIT", 20)
	viper.SetDefault("AUTH_RATE_LIMIT", 60)
	viper.SetDefault("FINNHUB_BASE_URL", "https://finnhub.io/api/v1")
	viper.SetDefault("FINNHUB_WS_URL", "wss://ws.finnhub.io")
	viper.SetDefault("REDACT_ENABLED", true)
	viper.SetDefault("REDACT_REQUEST_BODY", false)
	viper.SetDefault("REDACT_RESPONSE_BODY", false)
	viper.SetDefault("REDACT_SENSITIVE_QUERY_PARAMS", "access_token,refresh_token,api_key,apikey,token")
	viper.SetDefault("REDACT_HEADER_DENYLIST", "")
	viper.SetDefault("REDACT_PATHS", "")

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

	var trustedProxies []string
	if raw := viper.GetString("TRUSTED_PROXIES"); raw != "" {
		for _, p := range strings.Split(raw, ",") {
			if trimmed := strings.TrimSpace(p); trimmed != "" {
				trustedProxies = append(trustedProxies, trimmed)
			}
		}
	}

	cfg := &Config{
		Port:            viper.GetString("PORT"),
		GinMode:         viper.GetString("GIN_MODE"),
		DatabaseURL:     viper.GetString("DATABASE_URL"),
		SupabaseURL:     viper.GetString("SUPABASE_URL"),
		SupabaseAnonKey: viper.GetString("SUPABASE_ANON_KEY"),
		AllowedOrigins:  origins,
		TrustedProxies:  trustedProxies,
		PublicRateLimit: viper.GetInt("PUBLIC_RATE_LIMIT"),
		AuthRateLimit:   viper.GetInt("AUTH_RATE_LIMIT"),
		FinnhubAPIKey:   viper.GetString("FINNHUB_API_KEY"),
		FinnhubBaseURL:  viper.GetString("FINNHUB_BASE_URL"),
		FinnhubWSURL:    viper.GetString("FINNHUB_WS_URL"),
		PolygonAPIKey:   viper.GetString("POLYGON_API_KEY"),
		RedactEnabled:   viper.GetBool("REDACT_ENABLED"),
		RedactRequestBody: viper.GetBool("REDACT_REQUEST_BODY"),
		RedactResponseBody: viper.GetBool("REDACT_RESPONSE_BODY"),
		RedactSensitiveQueryParams: func() []string {
			raw := viper.GetString("REDACT_SENSITIVE_QUERY_PARAMS")
			if raw == "" {
				return nil
			}
			out := []string{}
			for _, p := range strings.Split(raw, ",") {
				if t := strings.TrimSpace(p); t != "" {
					out = append(out, t)
				}
			}
			return out
		}(),
		RedactHeaderDenylist: func() []string {
			raw := viper.GetString("REDACT_HEADER_DENYLIST")
			if raw == "" {
				return nil
			}
			out := []string{}
			for _, p := range strings.Split(raw, ",") {
				if t := strings.TrimSpace(p); t != "" {
					out = append(out, t)
				}
			}
			return out
		}(),
		RedactPaths: func() []string {
			raw := viper.GetString("REDACT_PATHS")
			if raw == "" {
				return nil
			}
			out := []string{}
			for _, p := range strings.Split(raw, ",") {
				if t := strings.TrimSpace(p); t != "" {
					out = append(out, t)
				}
			}
			return out
		}(),
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
