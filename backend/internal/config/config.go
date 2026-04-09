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

// Config holds all configuration values for the application.
type Config struct {
	Port           string
	GinMode        string
	DatabaseURL    string
	SupabaseURL    string
	AllowedOrigins []string // CORS: exact frontend origins, never wildcard
	// Rate limits (requests per second)
	PublicRateLimit int // per-IP, for unauthenticated endpoints
	AuthRateLimit   int // per-user, for authenticated endpoints
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
		Port:           viper.GetString("PORT"),
		GinMode:        viper.GetString("GIN_MODE"),
		DatabaseURL:    viper.GetString("DATABASE_URL"),
		SupabaseURL:    viper.GetString("SUPABASE_URL"),
		AllowedOrigins: origins,
		PublicRateLimit: viper.GetInt("PUBLIC_RATE_LIMIT"),
		AuthRateLimit:   viper.GetInt("AUTH_RATE_LIMIT"),
	}

	if cfg.DatabaseURL == "" {
		return cfg, fmt.Errorf("DATABASE_URL is not set")
	}
	if cfg.SupabaseURL == "" {
		return cfg, fmt.Errorf("SUPABASE_URL is not set")
	}

	return cfg, nil
}
