// Package main is the entrypoint for the fintech portfolio dashboard API.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/time/rate"

	"github.com/huchknows/fintech/backend/internal/config"
	"github.com/huchknows/fintech/backend/internal/middleware"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	gin.SetMode(cfg.GinMode)

	pool, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		slog.Error("failed to create connection pool", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := pool.Ping(context.Background()); err != nil {
		slog.Error("failed to ping database", "error", err)
		os.Exit(1)
	}
	slog.Info("database connection established")

	// Build router with explicit middleware — never use gin.Default().
	r := gin.New()

	// Global middleware stack (order matters — see security.md):
	// RequestID → Logger → Recovery → SecurityHeaders → CORS → RateLimit(IP)
	r.Use(
		middleware.RequestID(),
		middleware.Logger(),
		gin.Recovery(),
		middleware.SecurityHeaders(),
		middleware.CORS(cfg.AllowedOrigins),
		middleware.RateLimitByIP(rate.Limit(cfg.PublicRateLimit), cfg.PublicRateLimit*2),
	)

	// API v1 routes
	v1 := r.Group("/api/v1")

	// Public routes — no auth required, IP-based rate limiting from global stack.
	public := v1.Group("/")
	public.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// Authenticated routes — Auth middleware + per-user rate limiting.
	authed := v1.Group("/")
	authed.Use(
		middleware.RequireAuth(cfg.SupabaseURL),
		middleware.RateLimitByUser(rate.Limit(cfg.AuthRateLimit), cfg.AuthRateLimit*2),
	)
	// authed.GET("/portfolios", ...) — added in Phase 3

	// Admin routes — Auth + role enforcement + per-user rate limiting.
	admin := v1.Group("/admin")
	admin.Use(
		middleware.RequireAuth(cfg.SupabaseURL),
		middleware.RequireRole("admin"),
		middleware.RateLimitByUser(rate.Limit(cfg.AuthRateLimit), cfg.AuthRateLimit*2),
	)
	// admin.GET("/users", ...) — added in Phase 5
	_ = admin

	addr := fmt.Sprintf(":%s", cfg.Port)
	slog.Info("starting server", "address", addr)
	if err := r.Run(addr); err != nil {
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}
}
