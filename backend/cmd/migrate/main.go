// Package main is the entrypoint for running database migrations.
// Usage: go run cmd/migrate/main.go [up|down|status|reset]
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"

	redactlog "github.com/JAS0N-SMITH/redactlog"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"github.com/huchknows/fintech/backend/internal/config"
)

func main() {
    // CLI flags for redactlog behaviour (can also be provided via env vars in config.Load)
    redactEnabled := flag.Bool("redact_enabled", true, "enable redactlog for CLI output")
    redactRequestBody := flag.Bool("redact_request_body", false, "capture and redact request body")
    redactResponseBody := flag.Bool("redact_response_body", false, "capture and redact response body")
    redactQueryParams := flag.String("redact_query_params", "", "comma-separated sensitive query params")
    redactHeaderDenylist := flag.String("redact_header_denylist", "", "comma-separated headers to denylist")
    redactPaths := flag.String("redact_paths", "", "comma-separated redact paths")
    flag.Parse()

    logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
        Level: slog.LevelInfo,
    }))
    slog.SetDefault(logger)

    // Wrap CLI logger with redactlog handler if enabled
    if *redactEnabled {
        opts := []redactlog.Option{redactlog.WithLogger(slog.Default())}
        if *redactRequestBody {
            opts = append(opts, redactlog.WithRequestBody(true))
        }
        if *redactResponseBody {
            opts = append(opts, redactlog.WithResponseBody(true))
        }
        if *redactQueryParams != "" {
            opts = append(opts, redactlog.WithSensitiveQueryParams(strings.Split(*redactQueryParams, ",")...))
        }
        if *redactHeaderDenylist != "" {
            opts = append(opts, redactlog.WithHeaderDenylist(strings.Split(*redactHeaderDenylist, ",")...))
        }
        if *redactPaths != "" {
            opts = append(opts, redactlog.WithRedactPaths(strings.Split(*redactPaths, ",")...))
        }
        redactHandler, err := redactlog.NewPCI(opts...)
        if err == nil && redactHandler != nil {
            slog.SetDefault(slog.New(redactHandler))
        } else if err != nil {
            slog.Warn("redactlog CLI initialization failed, continuing without redaction", "error", err)
        }
    }

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
    defer func() { _ = db.Close() }()

    if err := db.PingContext(context.Background()); err != nil {
        slog.Error("failed to ping database", "error", err)
        os.Exit(1)
    }

    goose.SetBaseFS(nil)
    if err := goose.SetDialect("postgres"); err != nil {
        slog.Error("failed to set dialect", "error", err)
        os.Exit(1)
    }

    // Location of the migration command may be provided as a positional argument
    command := "up"
    if flag.NArg() > 0 {
        command = flag.Arg(0)
    }

    allowedCommands := map[string]struct{}{
        "up":     {},
        "down":   {},
        "status": {},
        "reset":  {},
    }
    if _, ok := allowedCommands[command]; !ok {
        slog.Error("invalid migration command")
        os.Exit(1)
    }

    migrationsDir := "migrations"
    if err := goose.RunContext(context.Background(), command, db, migrationsDir); err != nil {
        slog.Error("migration failed", "error", err)
        os.Exit(1)
    }

    fmt.Printf("migration %q completed successfully\n", command)
}
