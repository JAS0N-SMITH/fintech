// Package config handles application configuration using Viper.
// It reads from .env files and environment variables, with env vars
// taking precedence over file values.
package config

import (
	"fmt"
	"log/slog"

	"github.com/spf13/viper"
)

// Config holds all configuration values for the application.
type Config struct {
	Port        string
	GinMode     string
	DatabaseURL string
	SupabaseURL string
}

// Load reads configuration from .env file and environment variables.
// Environment variables take precedence over .env file values.
func Load() (*Config, error) {
	viper.SetConfigFile(".env")
	viper.SetConfigType("env")

	// Environment variables override file values
	viper.AutomaticEnv()

	// Set defaults
	viper.SetDefault("PORT", "8080")
	viper.SetDefault("GIN_MODE", "debug")

	// Read .env file (optional — env vars alone are sufficient)
	if err := viper.ReadInConfig(); err != nil {
		slog.Warn("no .env file found, using environment variables", "error", err)
	}

	cfg := &Config{
		Port:        viper.GetString("PORT"),
		GinMode:     viper.GetString("GIN_MODE"),
		DatabaseURL: viper.GetString("DATABASE_URL"),
		SupabaseURL: viper.GetString("SUPABASE_URL"),
	}

	if cfg.DatabaseURL == "" {
		return cfg, fmt.Errorf("DATABASE_URL is not set")
	}

	if cfg.SupabaseURL == "" {
		return cfg, fmt.Errorf("SUPABASE_URL is not set")
	}

	return cfg, nil
}
