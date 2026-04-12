// Package main is the entrypoint for the fintech portfolio dashboard API.
//
// @title           Portfolio Dashboard API
// @version         1.0
// @description     REST API for managing investment portfolios and transactions.
//
// @contact.name    Portfolio Dashboard
// @contact.url     https://github.com/huchknows/fintech
//
// @license.name    MIT
//
// @host            localhost:8080
// @BasePath        /api/v1
//
// @securityDefinitions.apikey  BearerAuth
// @in                          header
// @name                        Authorization
// @description                 Supabase JWT — prefix with "Bearer "
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"golang.org/x/time/rate"

	"github.com/huchknows/fintech/backend/internal/config"
	_ "github.com/huchknows/fintech/backend/docs" // swaggo generated docs
	"github.com/huchknows/fintech/backend/internal/handler"
	"github.com/huchknows/fintech/backend/internal/middleware"
	"github.com/huchknows/fintech/backend/internal/provider"
	"github.com/huchknows/fintech/backend/internal/repository"
	"github.com/huchknows/fintech/backend/internal/service"
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

	// Dependency injection — wire repositories → services → handlers.
	portfolioRepo := repository.NewPortfolioRepository(pool)
	transactionRepo := repository.NewTransactionRepository(pool)
	watchlistRepo := repository.NewWatchlistRepository(pool)

	portfolioSvc := service.NewPortfolioService(portfolioRepo)
	transactionSvc := service.NewTransactionService(transactionRepo, portfolioRepo)
	watchlistSvc := service.NewWatchlistService(watchlistRepo)

	finnhubProvider := provider.NewFinnhubProvider(cfg.FinnhubAPIKey, cfg.FinnhubBaseURL, cfg.FinnhubWSURL)
	marketDataSvc := service.NewMarketDataService(finnhubProvider)

	portfolioHandler := handler.NewPortfolioHandler(portfolioSvc)
	transactionHandler := handler.NewTransactionHandler(transactionSvc)
	watchlistHandler := handler.NewWatchlistHandler(watchlistSvc)
	marketDataHandler := handler.NewMarketDataHandler(marketDataSvc)
	wsHandler := handler.NewWebSocketHandler(finnhubProvider)

	// Build router with explicit middleware — never use gin.Default().
	r := gin.New()

	// Global middleware stack (order matters — see security.md):
	// RequestID → Logger → Recovery → CORS → RateLimit(IP)
	r.Use(
		middleware.RequestID(),
		middleware.Logger(),
		gin.Recovery(),
		middleware.CORS(cfg.AllowedOrigins),
		middleware.RateLimitByIP(rate.Limit(cfg.PublicRateLimit), cfg.PublicRateLimit*2),
	)

	// Swagger UI — development only; in production restrict via NGINX or disable.
	// Apply security headers to Swagger assets.
	swagger := r.Group("/swagger")
	swagger.Use(middleware.SecurityHeaders())
	swagger.GET("/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// API v1 routes with API-specific security headers
	v1 := r.Group("/api/v1")
	v1.Use(middleware.SecurityHeaders())

	// Public routes — no auth, IP-based rate limiting from global stack.
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
	portfolioHandler.RegisterRoutes(authed)
	watchlistHandler.RegisterRoutes(authed)
	marketDataHandler.RegisterRoutes(authed)
	wsHandler.RegisterRoutes(authed)

	// Nested transaction routes under portfolio ID.
	portfolioGroup := authed.Group("/portfolios/:id")
	transactionHandler.RegisterRoutes(portfolioGroup)

	// Admin routes — Auth + role enforcement + per-user rate limiting.
	adminRepo := repository.NewAdminRepository(pool)
	adminSvc := service.NewAdminService(adminRepo, pool, finnhubProvider, wsHandler)

	admin := v1.Group("/admin")
	admin.Use(
		middleware.RequireAuth(cfg.SupabaseURL),
		middleware.RequireRole("admin"),
		middleware.RateLimitByUser(rate.Limit(cfg.AuthRateLimit), cfg.AuthRateLimit*2),
		middleware.AuditAction("user.role_change", "user", adminSvc),
	)
	adminHandler := handler.NewAdminHandler(adminSvc)
	adminHandler.RegisterRoutes(admin)

	addr := fmt.Sprintf(":%s", cfg.Port)
	slog.Info("starting server", "address", addr)
	if err := r.Run(addr); err != nil {
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}
}
