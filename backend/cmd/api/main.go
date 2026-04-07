// Package main is the entrypoint for the fintech portfolio dashboard API.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/huchknows/fintech/backend/internal/config"
)

func main() {
	// Configure structured JSON logging
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		slog.Warn("configuration incomplete", "error", err)
		slog.Info("server will start without database connectivity")
	}

	// Set Gin mode from configuration
	gin.SetMode(cfg.GinMode)

	// Attempt database connection if DATABASE_URL is configured
	if cfg.DatabaseURL != "" {
		pool, poolErr := pgxpool.New(context.Background(), cfg.DatabaseURL)
		if poolErr != nil {
			slog.Error("failed to create connection pool", "error", poolErr)
			os.Exit(1)
		}
		defer pool.Close()

		if pingErr := pool.Ping(context.Background()); pingErr != nil {
			slog.Error("failed to ping database", "error", pingErr)
			os.Exit(1)
		}
		slog.Info("database connection established")
	}

	// Set up Gin router — use gin.New() for explicit middleware control
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	// API v1 routes
	v1 := r.Group("/api/v1")
	v1.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	addr := fmt.Sprintf(":%s", cfg.Port)
	slog.Info("starting server", "address", addr)
	if err := r.Run(addr); err != nil {
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}
}
