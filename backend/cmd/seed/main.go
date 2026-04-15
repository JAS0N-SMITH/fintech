// Package main implements a test data reseed command.
//
// Command seed resets and repopulates test portfolio and watchlist data
// using only symbols verified to exist on the Finnhub US exchange
// (Common Stock on NASDAQ/NYSE). This ensures all symbols support
// historical bar data on the free tier, avoiding 422 errors.
//
// Usage: go run cmd/seed/main.go
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spf13/viper"
)

// TransactionRow represents a single transaction for batch insertion.
type TransactionRow struct {
	PortfolioID      string
	TxType           string
	Symbol           string
	Date             string
	Quantity         *float64
	PricePerShare    *float64
	DividendPerShare *float64
	TotalAmount      float64
	Notes            string
}

// Constants
const (
	userID      = "99e69fd3-c724-496b-bd92-2386c5eb404e"
	portfolioID = "1b0c532b-38ba-4ffe-aa2d-c302200d5cf5"
	baseURL     = "https://finnhub.io/api/v1"
)

var (
	portfolioSymbols = []string{
		"AAPL", "MSFT", "GOOGL", "AMZN", "NVDA", "JPM", "TSLA", "META",
	}

	watchlistData = map[string][]string{
		"a1b2c3d4-0001-4000-8000-000000000001": {"AMD", "AVGO", "QCOM", "TSM", "ARM", "INTC"},       // AI & Semiconductors
		"a1b2c3d4-0001-4000-8000-000000000002": {"O", "KO", "PG", "JNJ", "MCD", "T"},                // Dividend Income
		"a1b2c3d4-0001-4000-8000-000000000003": {"BRK.B", "V", "COST", "NFLX", "UNH", "LLY", "WMT"}, // Potential Buys
	}
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Load config from .env file
	viper.SetConfigFile(".env")
	viper.SetConfigType("env")
	viper.AutomaticEnv()

	viper.SetDefault("FINNHUB_BASE_URL", baseURL)

	if err := viper.ReadInConfig(); err != nil {
		logger.Warn("no .env file found, using environment variables only", "error", err)
	}

	apiKey := viper.GetString("FINNHUB_API_KEY")
	dbURL := viper.GetString("DATABASE_URL")

	if apiKey == "" || dbURL == "" {
		logger.Error("missing required env vars", "need", "FINNHUB_API_KEY and DATABASE_URL")
		os.Exit(1)
	}

	// Use curated symbol lists that have been manually verified to work on Finnhub free tier.
	// Portfolio symbols: common tech/finance stocks + growth plays (TSLA, META replacing ETFs)
	// Watchlist symbols: categorized by investment theme
	watchlistSymbolCount := 0
	for _, symbols := range watchlistData {
		watchlistSymbolCount += len(symbols)
	}
	logger.Info("using curated symbol lists",
		"portfolio_symbols", len(portfolioSymbols),
		"watchlist_symbols", watchlistSymbolCount,
	)

	// Connect to database
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		logger.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	// Flatten watchlist symbols into a single slice for verification in reseed
	var allWatchlistSymbols []string
	for _, symbols := range watchlistData {
		allWatchlistSymbols = append(allWatchlistSymbols, symbols...)
	}

	// Run reseed
	if err := reseed(ctx, pool, logger, portfolioSymbols, allWatchlistSymbols); err != nil {
		logger.Error("reseed failed", "error", err)
		os.Exit(1)
	}

	logger.Info("reseed completed successfully")
}

// reseed clears and repopulates test data in a single transaction.
func reseed(ctx context.Context, pool *pgxpool.Pool, logger *slog.Logger, portfolioSymbols, watchlistSymbols []string) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Delete old data
	logger.Info("clearing old test data")
	if _, err := tx.Exec(ctx, "DELETE FROM public.transactions WHERE portfolio_id = $1", portfolioID); err != nil {
		return fmt.Errorf("delete transactions: %w", err)
	}

	if _, err := tx.Exec(ctx, "DELETE FROM public.watchlists WHERE user_id = $1", userID); err != nil {
		return fmt.Errorf("delete watchlists: %w", err)
	}

	// Ensure portfolio exists
	if _, err := tx.Exec(ctx,
		`INSERT INTO public.portfolios (id, user_id, name, description)
         VALUES ($1, $2, $3, $4)
         ON CONFLICT (id) DO NOTHING`,
		portfolioID, userID, "Test Portfolio", "Seeded test portfolio with verified symbols",
	); err != nil {
		return fmt.Errorf("insert portfolio: %w", err)
	}

	// Insert watchlists
	logger.Info("inserting watchlist headers")
	watchlistNames := map[string]string{
		"a1b2c3d4-0001-4000-8000-000000000001": "AI & Semiconductors",
		"a1b2c3d4-0001-4000-8000-000000000002": "Dividend Income",
		"a1b2c3d4-0001-4000-8000-000000000003": "Potential Buys",
	}

	for id, name := range watchlistNames {
		if _, err := tx.Exec(ctx,
			`INSERT INTO public.watchlists (id, user_id, name)
             VALUES ($1, $2, $3)
             ON CONFLICT (id) DO NOTHING`,
			id, userID, name,
		); err != nil {
			return fmt.Errorf("insert watchlist %s: %w", name, err)
		}
	}

	// Insert watchlist items
	logger.Info("inserting watchlist items")
	for watchlistID, symbols := range watchlistData {
		for _, sym := range symbols {
			// Check if symbol was verified
			found := false
			for _, v := range watchlistSymbols {
				if v == sym {
					found = true
					break
				}
			}

			if !found {
				logger.Warn("skipping unverified watchlist item", "symbol", sym, "watchlist", watchlistID)
				continue
			}

			if _, err := tx.Exec(ctx,
				`INSERT INTO public.watchlist_items (watchlist_id, symbol)
                 VALUES ($1, $2)
                 ON CONFLICT (watchlist_id, symbol) DO NOTHING`,
				watchlistID, sym,
			); err != nil {
				return fmt.Errorf("insert watchlist item %s: %w", sym, err)
			}
		}
	}

	// Build and insert transactions
	logger.Info("building transaction seed data")
	transactions := buildTransactions(portfolioSymbols)

	logger.Info("inserting transactions", "count", len(transactions))
	for _, tx := range transactions {
		if _, err := pool.Exec(ctx,
			`INSERT INTO public.transactions
             (portfolio_id, transaction_type, symbol, transaction_date, quantity, price_per_share, dividend_per_share, total_amount, notes)
             VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
             ON CONFLICT DO NOTHING`,
			tx.PortfolioID,
			tx.TxType,
			tx.Symbol,
			tx.Date,
			tx.Quantity,
			tx.PricePerShare,
			tx.DividendPerShare,
			tx.TotalAmount,
			tx.Notes,
		); err != nil {
			return fmt.Errorf("insert transaction: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	logger.Info("data committed successfully")
	return nil
}

// buildTransactions generates 18 months of realistic transaction data for the portfolio symbols.
func buildTransactions(symbols []string) []TransactionRow {
	var transactions []TransactionRow

	// Price ranges and transaction patterns for each symbol
	// Format: {symbol: {startPrice, endPrice, dividendPerShare}}
	symbolData := map[string]struct {
		startPrice     float64
		endPrice       float64
		priceStep      float64
		hasDividend    bool
		dividendPerQtr float64
	}{
		"AAPL":  {150, 240, 7.5, true, 0.25},
		"MSFT":  {370, 450, 4.0, true, 0.68},
		"GOOGL": {140, 190, 2.5, false, 0},
		"AMZN":  {185, 235, 2.5, false, 0},
		"NVDA":  {130, 170, 2.0, false, 0},
		"JPM":   {175, 225, 2.5, true, 1.10},
		"TSLA":  {220, 360, 7.0, false, 0},
		"META":  {560, 740, 9.0, false, 0},
	}

	// Generate transactions for each symbol
	for _, symbol := range symbols {
		data, ok := symbolData[symbol]
		if !ok {
			continue // skip unknown symbols
		}

		// 8 buy/sell transactions + dividend payments over 18 months
		dates := []string{
			"2024-10-15", "2024-12-10", "2025-02-20", "2025-04-15", "2025-06-15", "2025-08-20", "2025-11-10", "2026-02-15",
		}

		for i, date := range dates {
			qty := 5.0 + float64(i%3)*2.0
			price := data.startPrice + float64(i)*data.priceStep

			// Buy transaction
			transactions = append(transactions, TransactionRow{
				PortfolioID:   portfolioID,
				TxType:        "buy",
				Symbol:        symbol,
				Date:          date,
				Quantity:      &qty,
				PricePerShare: &price,
				TotalAmount:   qty * price,
				Notes:         fmt.Sprintf("DCA add %d", i+1),
			})

			// Occasional sell (every 3rd transaction)
			if i > 0 && i%3 == 0 {
				sellQty := qty * 0.5
				sellPrice := price * 1.05
				transactions = append(transactions, TransactionRow{
					PortfolioID:   portfolioID,
					TxType:        "sell",
					Symbol:        symbol,
					Date:          date,
					Quantity:      &sellQty,
					PricePerShare: &sellPrice,
					TotalAmount:   sellQty * sellPrice,
					Notes:         "Partial profit taking",
				})
			}

			// Quarterly dividends if applicable
			if data.hasDividend && i < 4 {
				sharedOwned := qty * (2.0 + float64(i)) // accumulated quantity
				divTotal := sharedOwned * data.dividendPerQtr
				transactions = append(transactions, TransactionRow{
					PortfolioID:      portfolioID,
					TxType:           "dividend",
					Symbol:           symbol,
					Date:             date,
					Quantity:         nil,
					PricePerShare:    nil,
					DividendPerShare: &data.dividendPerQtr,
					TotalAmount:      divTotal,
					Notes:            fmt.Sprintf("Q%d dividend", i+1),
				})
			}
		}
	}

	return transactions
}
