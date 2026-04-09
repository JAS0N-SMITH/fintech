// Package main is the entrypoint for running database migrations.
// Usage: go run cmd/migrate/main.go [up|down|status|reset]
package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"github.com/huchknows/fintech/backend/internal/config"
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

	db, err := sql.Open("pgx", cfg.DatabaseURL)
	if err != nil {
		slog.Error("failed to open database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := db.PingContext(context.Background()); err != nil {
		slog.Error("failed to ping database", "error", err)
		os.Exit(1)
	}

	goose.SetBaseFS(nil)
	if err := goose.SetDialect("postgres"); err != nil {
		slog.Error("failed to set dialect", "error", err)
		os.Exit(1)
	}

	command := "up"
	if len(os.Args) > 1 {
		command = os.Args[1]
	}

	migrationsDir := "migrations"
	if err := goose.RunContext(context.Background(), command, db, migrationsDir); err != nil {
		slog.Error("migration failed", "command", command, "error", err)
		os.Exit(1)
	}

	fmt.Printf("migration %q completed successfully\n", command)
}
