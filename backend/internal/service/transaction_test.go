package service

import (
	"context"
	"errors"
	"testing"

	"github.com/shopspring/decimal"

	"github.com/huchknows/fintech/backend/internal/model"
)

// ---------------------------------------------------------------------------
// Mock transaction repository
// ---------------------------------------------------------------------------

type mockTransactionRepo struct {
	createFn       func(ctx context.Context, portfolioID string, in model.CreateTransactionInput) (*model.Transaction, error)
	getByIDFn      func(ctx context.Context, id string) (*model.Transaction, error)
	listFn         func(ctx context.Context, portfolioID string) ([]*model.Transaction, error)
	quantityHeldFn func(ctx context.Context, portfolioID, symbol string) (decimal.Decimal, error)
	updateFn       func(ctx context.Context, id string, in model.CreateTransactionInput) (*model.Transaction, error)
	deleteFn       func(ctx context.Context, id string) error
}

func (m *mockTransactionRepo) Create(ctx context.Context, portfolioID string, in model.CreateTransactionInput) (*model.Transaction, error) {
	if m.createFn != nil {
		return m.createFn(ctx, portfolioID, in)
	}
	q := in.Quantity
	p := in.PricePerShare
	return &model.Transaction{
		ID:              "txn-new",
		PortfolioID:     portfolioID,
		TransactionType: in.TransactionType,
		Symbol:          in.Symbol,
		Quantity:        q,
		PricePerShare:   p,
		TotalAmount:     in.TotalAmount,
	}, nil
}

func (m *mockTransactionRepo) GetByID(ctx context.Context, id string) (*model.Transaction, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, model.ErrNotFound
}

func (m *mockTransactionRepo) ListByPortfolioID(ctx context.Context, portfolioID string) ([]*model.Transaction, error) {
	if m.listFn != nil {
		return m.listFn(ctx, portfolioID)
	}
	return nil, nil
}

func (m *mockTransactionRepo) QuantityHeld(ctx context.Context, portfolioID, symbol string) (decimal.Decimal, error) {
	if m.quantityHeldFn != nil {
		return m.quantityHeldFn(ctx, portfolioID, symbol)
	}
	return decimal.Zero, nil
}

func (m *mockTransactionRepo) Update(ctx context.Context, id string, in model.CreateTransactionInput) (*model.Transaction, error) {
	if m.updateFn != nil {
		return m.updateFn(ctx, id, in)
	}
	return nil, model.ErrNotFound
}

func (m *mockTransactionRepo) Delete(ctx context.Context, id string) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id)
	}
	return model.ErrNotFound
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func qty(s string) *decimal.Decimal { d := decimal.RequireFromString(s); return &d }
func price(s string) *decimal.Decimal { d := decimal.RequireFromString(s); return &d }

func buyInput(symbol, quantity, pricePerShare, total string) model.CreateTransactionInput {
	return model.CreateTransactionInput{
		TransactionType: model.TransactionTypeBuy,
		Symbol:          symbol,
		TransactionDate: "2026-01-15",
		Quantity:        qty(quantity),
		PricePerShare:   price(pricePerShare),
		TotalAmount:     decimal.RequireFromString(total),
	}
}

func sellInput(symbol, quantity, pricePerShare, total string) model.CreateTransactionInput {
	return model.CreateTransactionInput{
		TransactionType: model.TransactionTypeSell,
		Symbol:          symbol,
		TransactionDate: "2026-01-20",
		Quantity:        qty(quantity),
		PricePerShare:   price(pricePerShare),
		TotalAmount:     decimal.RequireFromString(total),
	}
}

// ---------------------------------------------------------------------------
// Portfolio ownership mock (transaction service needs portfolio ownership check)
// ---------------------------------------------------------------------------

func ownerPortfolioRepo(portfolioID, ownerID string) *mockPortfolioRepo {
	return &mockPortfolioRepo{
		portfolios: map[string]*model.Portfolio{
			portfolioID: {ID: portfolioID, UserID: ownerID, Name: "Test"},
		},
	}
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestTransactionService_Create_Buy(t *testing.T) {
	ctx := context.Background()

	t.Run("creates buy transaction for portfolio owner", func(t *testing.T) {
		txnRepo := &mockTransactionRepo{}
		portRepo := ownerPortfolioRepo("port-1", "user-a")
		svc := NewTransactionService(txnRepo, portRepo)

		txn, err := svc.Create(ctx, "user-a", "port-1", buyInput("AAPL", "10", "150.00", "1500.00"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if txn.Symbol != "AAPL" {
			t.Errorf("symbol = %q, want AAPL", txn.Symbol)
		}
	})

	t.Run("returns forbidden for non-owner", func(t *testing.T) {
		txnRepo := &mockTransactionRepo{}
		portRepo := ownerPortfolioRepo("port-1", "user-a")
		svc := NewTransactionService(txnRepo, portRepo)

		_, err := svc.Create(ctx, "user-b", "port-1", buyInput("AAPL", "10", "150.00", "1500.00"))
		if !errors.Is(err, model.ErrForbidden) {
			t.Errorf("err = %v, want ErrForbidden", err)
		}
	})

	t.Run("returns not found for unknown portfolio", func(t *testing.T) {
		txnRepo := &mockTransactionRepo{}
		portRepo := &mockPortfolioRepo{portfolios: map[string]*model.Portfolio{}}
		svc := NewTransactionService(txnRepo, portRepo)

		_, err := svc.Create(ctx, "user-a", "ghost", buyInput("AAPL", "10", "150.00", "1500.00"))
		if !errors.Is(err, model.ErrNotFound) {
			t.Errorf("err = %v, want ErrNotFound", err)
		}
	})
}

func TestTransactionService_Create_Sell(t *testing.T) {
	ctx := context.Background()

	t.Run("sell succeeds when shares are held", func(t *testing.T) {
		txnRepo := &mockTransactionRepo{
			quantityHeldFn: func(_ context.Context, _, _ string) (decimal.Decimal, error) {
				return decimal.RequireFromString("100"), nil
			},
		}
		portRepo := ownerPortfolioRepo("port-1", "user-a")
		svc := NewTransactionService(txnRepo, portRepo)

		_, err := svc.Create(ctx, "user-a", "port-1", sellInput("AAPL", "50", "160.00", "8000.00"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("sell exact quantity held succeeds", func(t *testing.T) {
		txnRepo := &mockTransactionRepo{
			quantityHeldFn: func(_ context.Context, _, _ string) (decimal.Decimal, error) {
				return decimal.RequireFromString("50"), nil
			},
		}
		portRepo := ownerPortfolioRepo("port-1", "user-a")
		svc := NewTransactionService(txnRepo, portRepo)

		_, err := svc.Create(ctx, "user-a", "port-1", sellInput("AAPL", "50", "160.00", "8000.00"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("sell more than held returns conflict", func(t *testing.T) {
		txnRepo := &mockTransactionRepo{
			quantityHeldFn: func(_ context.Context, _, _ string) (decimal.Decimal, error) {
				return decimal.RequireFromString("10"), nil
			},
		}
		portRepo := ownerPortfolioRepo("port-1", "user-a")
		svc := NewTransactionService(txnRepo, portRepo)

		_, err := svc.Create(ctx, "user-a", "port-1", sellInput("AAPL", "50", "160.00", "8000.00"))
		if !errors.Is(err, model.ErrConflict) {
			t.Errorf("err = %v, want ErrConflict", err)
		}
	})

	t.Run("sell with zero shares held returns conflict", func(t *testing.T) {
		txnRepo := &mockTransactionRepo{
			quantityHeldFn: func(_ context.Context, _, _ string) (decimal.Decimal, error) {
				return decimal.Zero, nil
			},
		}
		portRepo := ownerPortfolioRepo("port-1", "user-a")
		svc := NewTransactionService(txnRepo, portRepo)

		_, err := svc.Create(ctx, "user-a", "port-1", sellInput("TSLA", "5", "200.00", "1000.00"))
		if !errors.Is(err, model.ErrConflict) {
			t.Errorf("err = %v, want ErrConflict", err)
		}
	})
}

func TestTransactionService_Create_Validation(t *testing.T) {
	ctx := context.Background()
	portRepo := ownerPortfolioRepo("port-1", "user-a")

	tests := []struct {
		name    string
		input   model.CreateTransactionInput
		wantErr error
	}{
		{
			name: "invalid transaction type",
			input: model.CreateTransactionInput{
				TransactionType: "invalid",
				Symbol:          "AAPL",
				TransactionDate: "2026-01-15",
				Quantity:        qty("10"),
				PricePerShare:   price("150"),
				TotalAmount:     decimal.RequireFromString("1500"),
			},
			wantErr: model.ErrValidation,
		},
		{
			name: "buy missing quantity",
			input: model.CreateTransactionInput{
				TransactionType: model.TransactionTypeBuy,
				Symbol:          "AAPL",
				TransactionDate: "2026-01-15",
				PricePerShare:   price("150"),
				TotalAmount:     decimal.RequireFromString("1500"),
			},
			wantErr: model.ErrValidation,
		},
		{
			name: "sell missing price_per_share",
			input: model.CreateTransactionInput{
				TransactionType: model.TransactionTypeSell,
				Symbol:          "AAPL",
				TransactionDate: "2026-01-15",
				Quantity:        qty("10"),
				TotalAmount:     decimal.RequireFromString("1500"),
			},
			wantErr: model.ErrValidation,
		},
		{
			name: "dividend missing dividend_per_share",
			input: model.CreateTransactionInput{
				TransactionType: model.TransactionTypeDividend,
				Symbol:          "AAPL",
				TransactionDate: "2026-01-15",
				TotalAmount:     decimal.RequireFromString("25"),
			},
			wantErr: model.ErrValidation,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			txnRepo := &mockTransactionRepo{}
			svc := NewTransactionService(txnRepo, portRepo)
			_, err := svc.Create(ctx, "user-a", "port-1", tt.input)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("err = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestTransactionService_List(t *testing.T) {
	ctx := context.Background()

	t.Run("returns transactions for portfolio owner", func(t *testing.T) {
		txnRepo := &mockTransactionRepo{
			listFn: func(_ context.Context, _ string) ([]*model.Transaction, error) {
				return []*model.Transaction{{ID: "t1"}, {ID: "t2"}}, nil
			},
		}
		portRepo := ownerPortfolioRepo("port-1", "user-a")
		svc := NewTransactionService(txnRepo, portRepo)

		txns, err := svc.List(ctx, "user-a", "port-1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(txns) != 2 {
			t.Errorf("len = %d, want 2", len(txns))
		}
	})

	t.Run("returns forbidden for non-owner", func(t *testing.T) {
		portRepo := ownerPortfolioRepo("port-1", "user-a")
		svc := NewTransactionService(&mockTransactionRepo{}, portRepo)

		_, err := svc.List(ctx, "user-b", "port-1")
		if !errors.Is(err, model.ErrForbidden) {
			t.Errorf("err = %v, want ErrForbidden", err)
		}
	})
}
