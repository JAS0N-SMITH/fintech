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

	_ "github.com/huchknows/fintech/backend/docs" // swaggo generated docs
	"github.com/huchknows/fintech/backend/internal/config"
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
	profileRepo := repository.NewProfileRepository(pool)

	portfolioSvc := service.NewPortfolioService(portfolioRepo)
	transactionSvc := service.NewTransactionService(transactionRepo, portfolioRepo)
	watchlistSvc := service.NewWatchlistService(watchlistRepo)
	profileSvc := service.NewProfileService(profileRepo)
	importSvc := service.NewImportService(transactionSvc, portfolioSvc, logger)

	finnhubProvider := provider.NewFinnhubProvider(cfg.FinnhubAPIKey, cfg.FinnhubBaseURL, cfg.FinnhubWSURL)

	// ADR 015: Polygon is primary for historical bars; Finnhub is realtime + fallback.
	// FallbackProvider routes GetHistoricalBars to Polygon first, then Finnhub.
	// All other methods (quotes, symbols, streaming) always go to Finnhub.
	var polygonProvider provider.MarketDataProvider
	if cfg.PolygonAPIKey != "" {
		polygonProvider = provider.NewPolygonProvider(cfg.PolygonAPIKey)
		slog.Info("polygon.io enabled as primary provider for historical bars")
	} else {
		slog.Warn("POLYGON_API_KEY not set — historical bars will use finnhub only")
	}
	dataProvider := provider.NewFallbackProvider(finnhubProvider, polygonProvider)

	marketDataSvc := service.NewMarketDataService(dataProvider)

	portfolioHandler := handler.NewPortfolioHandler(portfolioSvc)
	transactionHandler := handler.NewTransactionHandler(transactionSvc)
	watchlistHandler := handler.NewWatchlistHandler(watchlistSvc)
	marketDataHandler := handler.NewMarketDataHandler(marketDataSvc)
	wsHandler := handler.NewWebSocketHandler(finnhubProvider)
	profileHandler := handler.NewProfileHandler(profileSvc)
	importHandler := handler.NewImportHandler(importSvc)
	authHandler := handler.NewAuthHandler(cfg.SupabaseURL, cfg.SupabaseAnonKey, cfg.GinMode == "release")

	// Build router with explicit middleware — never use gin.Default().
	r := gin.New()

	// Restrict which proxy IPs are trusted for X-Forwarded-For resolution.
	// nil = trust no proxies (use direct connection IP), which is correct for
	// local dev and any deployment without a reverse proxy in front.
	// Set TRUSTED_PROXIES=127.0.0.1 (or your load balancer CIDR) in production.
	if err := r.SetTrustedProxies(cfg.TrustedProxies); err != nil {
		slog.Error("failed to set trusted proxies", "error", err)
		os.Exit(1)
	}

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
	swagger.Use(middleware.DocsSecurityHeaders())
	swagger.GET("/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// API v1 routes with API-specific security headers
	v1 := r.Group("/api/v1")
	v1.Use(middleware.SecurityHeaders())

	// Public routes — no auth, IP-based rate limiting from global stack.
	public := v1.Group("/")
	public.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})
	authHandler.RegisterRoutes(public)

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
	profileHandler.RegisterRoutes(authed)

	// Nested transaction and import routes under portfolio ID.
	portfolioGroup := authed.Group("/portfolios/:id")
	transactionHandler.RegisterRoutes(portfolioGroup)
	importHandler.RegisterRoutes(portfolioGroup)

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
