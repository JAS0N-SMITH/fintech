//go:build integration

package repository

import (
	"context"
	"errors"
	"testing"

	"github.com/shopspring/decimal"

	"github.com/huchknows/fintech/backend/internal/model"
)

func TestTransactionRepository_CRUD(t *testing.T) {
	pool := setupTestDB(t)
	txRepo := NewTransactionRepository(pool)
	portRepo := NewPortfolioRepository(pool)
	ctx := context.Background()

	userID := insertTestUser(t, pool)

	// Helper to create a portfolio for use across sub-tests.
	newPortfolio := func(t *testing.T) string {
		t.Helper()
		p, err := portRepo.Create(ctx, userID, model.CreatePortfolioInput{Name: "Test Portfolio"})
		if err != nil {
			t.Fatalf("create portfolio: %v", err)
		}
		return p.ID
	}

	qty10 := decimal.NewFromInt(10)
	price150 := decimal.NewFromFloat(150.00)

	buyInput := model.CreateTransactionInput{
		TransactionType: model.TransactionTypeBuy,
		Symbol:          "AAPL",
		TransactionDate: "2024-01-15",
		Quantity:        &qty10,
		PricePerShare:   &price150,
		TotalAmount:     decimal.NewFromFloat(1500.00),
	}

	t.Run("Create buy transaction returns record with generated ID", func(t *testing.T) {
		portID := newPortfolio(t)

		txn, err := txRepo.Create(ctx, portID, buyInput)
		if err != nil {
			t.Fatalf("Create: %v", err)
		}
		if txn.ID == "" {
			t.Error("expected non-empty ID")
		}
		if txn.PortfolioID != portID {
			t.Errorf("PortfolioID = %q, want %q", txn.PortfolioID, portID)
		}
		if txn.Symbol != "AAPL" {
			t.Errorf("Symbol = %q, want %q", txn.Symbol, "AAPL")
		}
		if txn.TransactionType != model.TransactionTypeBuy {
			t.Errorf("TransactionType = %q, want %q", txn.TransactionType, model.TransactionTypeBuy)
		}
		if !txn.Quantity.Equal(qty10) {
			t.Errorf("Quantity = %s, want %s", txn.Quantity, qty10)
		}
	})

	t.Run("Create dividend transaction (no quantity)", func(t *testing.T) {
		portID := newPortfolio(t)
		divPerShare := decimal.NewFromFloat(0.25)
		divInput := model.CreateTransactionInput{
			TransactionType:  model.TransactionTypeDividend,
			Symbol:           "AAPL",
			TransactionDate:  "2024-03-15",
			DividendPerShare: &divPerShare,
			TotalAmount:      decimal.NewFromFloat(25.00),
		}

		txn, err := txRepo.Create(ctx, portID, divInput)
		if err != nil {
			t.Fatalf("Create dividend: %v", err)
		}
		if txn.Quantity != nil {
			t.Errorf("Quantity should be nil for dividend, got %s", txn.Quantity)
		}
		if txn.DividendPerShare == nil || !txn.DividendPerShare.Equal(divPerShare) {
			t.Errorf("DividendPerShare = %v, want %s", txn.DividendPerShare, divPerShare)
		}
	})

	t.Run("Create returns validation error for invalid date format", func(t *testing.T) {
		portID := newPortfolio(t)
		badDate := buyInput
		badDate.TransactionDate = "15-01-2024" // wrong format
		_, err := txRepo.Create(ctx, portID, badDate)
		if err == nil {
			t.Fatal("expected error for invalid date format")
		}
	})

	t.Run("GetByID returns the created transaction", func(t *testing.T) {
		portID := newPortfolio(t)
		created, err := txRepo.Create(ctx, portID, buyInput)
		if err != nil {
			t.Fatalf("Create: %v", err)
		}

		got, err := txRepo.GetByID(ctx, created.ID)
		if err != nil {
			t.Fatalf("GetByID: %v", err)
		}
		if got.ID != created.ID {
			t.Errorf("ID = %q, want %q", got.ID, created.ID)
		}
	})

	t.Run("GetByID returns ErrNotFound for missing ID", func(t *testing.T) {
		_, err := txRepo.GetByID(ctx, "00000000-0000-0000-0000-000000000000")
		if !errors.Is(err, model.ErrNotFound) {
			t.Errorf("err = %v, want ErrNotFound", err)
		}
	})

	t.Run("ListByPortfolioID returns all transactions, newest date first", func(t *testing.T) {
		portID := newPortfolio(t)

		older := buyInput
		older.TransactionDate = "2024-01-01"
		_, err := txRepo.Create(ctx, portID, older)
		if err != nil {
			t.Fatalf("Create older: %v", err)
		}

		newer := buyInput
		newer.TransactionDate = "2024-06-01"
		_, err = txRepo.Create(ctx, portID, newer)
		if err != nil {
			t.Fatalf("Create newer: %v", err)
		}

		txns, err := txRepo.ListByPortfolioID(ctx, portID)
		if err != nil {
			t.Fatalf("ListByPortfolioID: %v", err)
		}
		if len(txns) != 2 {
			t.Fatalf("len = %d, want 2", len(txns))
		}
		// Newest date (2024-06-01) should be first.
		if !txns[0].TransactionDate.After(txns[1].TransactionDate) {
			t.Errorf("expected newest first: [0] %s [1] %s",
				txns[0].TransactionDate.Format("2006-01-02"),
				txns[1].TransactionDate.Format("2006-01-02"),
			)
		}
	})

	t.Run("ListByPortfolioID returns empty slice for empty portfolio", func(t *testing.T) {
		portID := newPortfolio(t)
		txns, err := txRepo.ListByPortfolioID(ctx, portID)
		if err != nil {
			t.Fatalf("ListByPortfolioID: %v", err)
		}
		if len(txns) != 0 {
			t.Errorf("expected empty, got %d", len(txns))
		}
	})

	t.Run("QuantityHeld returns zero with no transactions", func(t *testing.T) {
		portID := newPortfolio(t)
		held, err := txRepo.QuantityHeld(ctx, portID, "AAPL")
		if err != nil {
			t.Fatalf("QuantityHeld: %v", err)
		}
		if !held.IsZero() {
			t.Errorf("held = %s, want 0", held)
		}
	})

	t.Run("QuantityHeld accumulates buys and deducts sells", func(t *testing.T) {
		portID := newPortfolio(t)

		// Buy 10
		_, err := txRepo.Create(ctx, portID, buyInput)
		if err != nil {
			t.Fatalf("Create buy 10: %v", err)
		}

		// Buy 5 more
		qty5 := decimal.NewFromInt(5)
		buy5 := model.CreateTransactionInput{
			TransactionType: model.TransactionTypeBuy,
			Symbol:          "AAPL",
			TransactionDate: "2024-02-01",
			Quantity:        &qty5,
			PricePerShare:   &price150,
			TotalAmount:     decimal.NewFromFloat(750.00),
		}
		_, err = txRepo.Create(ctx, portID, buy5)
		if err != nil {
			t.Fatalf("Create buy 5: %v", err)
		}

		// Sell 3
		qty3 := decimal.NewFromInt(3)
		sell3 := model.CreateTransactionInput{
			TransactionType: model.TransactionTypeSell,
			Symbol:          "AAPL",
			TransactionDate: "2024-03-01",
			Quantity:        &qty3,
			PricePerShare:   &price150,
			TotalAmount:     decimal.NewFromFloat(450.00),
		}
		_, err = txRepo.Create(ctx, portID, sell3)
		if err != nil {
			t.Fatalf("Create sell 3: %v", err)
		}

		held, err := txRepo.QuantityHeld(ctx, portID, "AAPL")
		if err != nil {
			t.Fatalf("QuantityHeld: %v", err)
		}
		want := decimal.NewFromInt(12) // 10 + 5 - 3
		if !held.Equal(want) {
			t.Errorf("held = %s, want %s", held, want)
		}
	})

	t.Run("QuantityHeld includes reinvested_dividend in buy side", func(t *testing.T) {
		portID := newPortfolio(t)

		qty2 := decimal.NewFromInt(2)
		divPerShare := decimal.NewFromFloat(0.25)
		reinvested := model.CreateTransactionInput{
			TransactionType:  model.TransactionTypeReinvestedDividend,
			Symbol:           "AAPL",
			TransactionDate:  "2024-04-01",
			Quantity:         &qty2,
			PricePerShare:    &price150,
			DividendPerShare: &divPerShare,
			TotalAmount:      decimal.NewFromFloat(300.00),
		}
		_, err := txRepo.Create(ctx, portID, reinvested)
		if err != nil {
			t.Fatalf("Create reinvested: %v", err)
		}

		held, err := txRepo.QuantityHeld(ctx, portID, "AAPL")
		if err != nil {
			t.Fatalf("QuantityHeld: %v", err)
		}
		if !held.Equal(qty2) {
			t.Errorf("held = %s, want %s", held, qty2)
		}
	})

	t.Run("QuantityHeld is symbol-scoped (does not bleed across tickers)", func(t *testing.T) {
		portID := newPortfolio(t)

		// Buy 10 AAPL
		_, err := txRepo.Create(ctx, portID, buyInput)
		if err != nil {
			t.Fatalf("Create AAPL: %v", err)
		}

		// Buy 5 MSFT
		qty5 := decimal.NewFromInt(5)
		msft := model.CreateTransactionInput{
			TransactionType: model.TransactionTypeBuy,
			Symbol:          "MSFT",
			TransactionDate: "2024-02-01",
			Quantity:        &qty5,
			PricePerShare:   &price150,
			TotalAmount:     decimal.NewFromFloat(750.00),
		}
		_, err = txRepo.Create(ctx, portID, msft)
		if err != nil {
			t.Fatalf("Create MSFT: %v", err)
		}

		held, err := txRepo.QuantityHeld(ctx, portID, "MSFT")
		if err != nil {
			t.Fatalf("QuantityHeld MSFT: %v", err)
		}
		if !held.Equal(qty5) {
			t.Errorf("MSFT held = %s, want %s", held, qty5)
		}
	})

	t.Run("Delete removes the transaction", func(t *testing.T) {
		portID := newPortfolio(t)
		txn, err := txRepo.Create(ctx, portID, buyInput)
		if err != nil {
			t.Fatalf("Create: %v", err)
		}

		if err := txRepo.Delete(ctx, txn.ID); err != nil {
			t.Fatalf("Delete: %v", err)
		}

		_, err = txRepo.GetByID(ctx, txn.ID)
		if !errors.Is(err, model.ErrNotFound) {
			t.Errorf("after delete: err = %v, want ErrNotFound", err)
		}
	})

	t.Run("Delete returns ErrNotFound for missing ID", func(t *testing.T) {
		err := txRepo.Delete(ctx, "00000000-0000-0000-0000-000000000000")
		if !errors.Is(err, model.ErrNotFound) {
			t.Errorf("err = %v, want ErrNotFound", err)
		}
	})
}
