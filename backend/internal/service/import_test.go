package service

import (
	"bytes"
	"context"
	"encoding/csv"
	"io"
	"log/slog"
	"testing"

	"github.com/shopspring/decimal"

	"github.com/huchknows/fintech/backend/internal/model"
)

// MockTransactionService for testing.
type mockTransactionService struct {
	createFunc func(ctx context.Context, callerID, portfolioID string, in model.CreateTransactionInput) (*model.Transaction, error)
	callCount  int
}

func (m *mockTransactionService) Create(ctx context.Context, callerID, portfolioID string, in model.CreateTransactionInput) (*model.Transaction, error) {
	m.callCount++
	if m.createFunc != nil {
		return m.createFunc(ctx, callerID, portfolioID, in)
	}
	// Default: success
	return &model.Transaction{
		ID:              "txn-" + in.Symbol,
		PortfolioID:     portfolioID,
		TransactionType: in.TransactionType,
		Symbol:          in.Symbol,
	}, nil
}

func (m *mockTransactionService) List(ctx context.Context, callerID, portfolioID string) ([]*model.Transaction, error) {
	return nil, nil
}

func (m *mockTransactionService) Delete(ctx context.Context, callerID, transactionID string) error {
	return nil
}

// MockPortfolioService for testing.
type mockPortfolioService struct {
	getByIDFunc func(ctx context.Context, callerID, portfolioID string) (*model.Portfolio, error)
}

func (m *mockPortfolioService) GetByID(ctx context.Context, callerID, portfolioID string) (*model.Portfolio, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, callerID, portfolioID)
	}
	return &model.Portfolio{
		ID:     portfolioID,
		UserID: callerID,
		Name:   "Test Portfolio",
	}, nil
}

func (m *mockPortfolioService) Create(ctx context.Context, userID string, in model.CreatePortfolioInput) (*model.Portfolio, error) {
	return nil, nil
}

func (m *mockPortfolioService) List(ctx context.Context, userID string) ([]*model.Portfolio, error) {
	return nil, nil
}

func (m *mockPortfolioService) Update(ctx context.Context, callerID, portfolioID string, in model.UpdatePortfolioInput) (*model.Portfolio, error) {
	return nil, nil
}

func (m *mockPortfolioService) Delete(ctx context.Context, callerID, portfolioID string) error {
	return nil
}

// Helper to create CSV data in memory.
func createCSV(headers []string, rows [][]string) io.Reader {
	buf := new(bytes.Buffer)
	w := csv.NewWriter(buf)
	w.Write(headers)
	for _, row := range rows {
		w.Write(row)
	}
	w.Flush()
	return buf
}

// TestImportServicePreview tests the preview (dry-run) flow.
func TestImportServicePreview(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))

	tests := []struct {
		name           string
		csv            io.Reader
		brokerage      string
		wantValidCount int
		wantErrorCount int
		wantErr        bool
		wantErrMsg     string
	}{
		{
			name: "valid fidelity csv",
			csv: createCSV(
				[]string{"Run Date", "Symbol", "Activity Type", "Quantity", "Price", "Amount"},
				[][]string{
					{"01/15/2024", "AAPL", "Buy", "10", "150.00", "$1,500.00"},
					{"02/01/2024", "TSLA", "Sell", "5", "200.00", "$1,000.00"},
				},
			),
			brokerage:      "fidelity",
			wantValidCount: 2,
			wantErrorCount: 0,
			wantErr:        false,
		},
		{
			name: "fidelity csv with parse error",
			csv: createCSV(
				[]string{"Run Date", "Symbol", "Activity Type", "Quantity", "Price", "Amount"},
				[][]string{
					{"01/15/2024", "AAPL", "Buy", "10", "150.00", "$1,500.00"},
					{"02/01/2024", "TSLA", "InvalidType", "5", "200.00", "$1,000.00"},
				},
			),
			brokerage:      "fidelity",
			wantValidCount: 1,
			wantErrorCount: 1,
			wantErr:        false,
		},
		{
			name: "csv with validation error",
			csv: createCSV(
				[]string{"Run Date", "Symbol", "Activity Type", "Quantity", "Price", "Amount"},
				[][]string{
					{"01/15/2024", "AAPL", "Buy", "0", "150.00", "$0.00"}, // Invalid: quantity and amount are zero
				},
			),
			brokerage:      "fidelity",
			wantValidCount: 0,
			wantErrorCount: 1,
			wantErr:        false,
		},
		{
			name: "portfolio not found",
			csv: createCSV(
				[]string{"Run Date", "Symbol", "Activity Type", "Quantity", "Price", "Amount"},
				[][]string{
					{"01/15/2024", "AAPL", "Buy", "10", "150.00", "$1,500.00"},
				},
			),
			brokerage: "fidelity",
			wantErr:   true,
			wantErrMsg: "portfolio",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockTxnSvc := &mockTransactionService{}
			mockPortSvc := &mockPortfolioService{}

			// For portfolio not found case
			if tt.wantErr && tt.wantErrMsg == "portfolio" {
				mockPortSvc.getByIDFunc = func(ctx context.Context, callerID, portfolioID string) (*model.Portfolio, error) {
					return nil, model.NewNotFound("portfolio")
				}
			}

			svc := NewImportService(mockTxnSvc, mockPortSvc, logger)
			preview, err := svc.Preview(context.Background(), "user-123", "port-456", tt.csv, tt.brokerage)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if preview.Valid != tt.wantValidCount {
				t.Errorf("wantValid=%d, got %d", tt.wantValidCount, preview.Valid)
			}

			if len(preview.Errors) != tt.wantErrorCount {
				t.Errorf("wantErrorCount=%d, got %d", tt.wantErrorCount, len(preview.Errors))
			}

			if preview.Parsed != preview.Valid+len(preview.Errors) {
				t.Errorf("parsed count mismatch: %d != %d + %d", preview.Parsed, preview.Valid, len(preview.Errors))
			}

			if len(preview.Transactions) != preview.Valid {
				t.Errorf("transactions count should equal valid: %d != %d", len(preview.Transactions), preview.Valid)
			}
		})
	}
}

// TestImportServiceConfirm tests the confirm (persist) flow.
func TestImportServiceConfirm(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))

	tests := []struct {
		name          string
		transactions  []model.CreateTransactionInput
		createFunc    func(ctx context.Context, callerID, portfolioID string, in model.CreateTransactionInput) (*model.Transaction, error)
		wantCreated   int
		wantFailed    int
		wantErr       bool
	}{
		{
			name: "all transactions succeed",
			transactions: []model.CreateTransactionInput{
				{
					TransactionType: model.TransactionTypeBuy,
					Symbol:          "AAPL",
					TransactionDate: "2024-01-15",
					Quantity:        decPtr(decimal.NewFromInt(10)),
					PricePerShare:   decPtr(dec("150.00")),
					TotalAmount:     dec("1500.00"),
				},
				{
					TransactionType: model.TransactionTypeBuy,
					Symbol:          "TSLA",
					TransactionDate: "2024-01-15",
					Quantity:        decPtr(decimal.NewFromInt(5)),
					PricePerShare:   decPtr(dec("200.00")),
					TotalAmount:     dec("1000.00"),
				},
			},
			createFunc:  nil, // Use default (success)
			wantCreated: 2,
			wantFailed:  0,
		},
		{
			name: "partial failures",
			transactions: []model.CreateTransactionInput{
				{
					TransactionType: model.TransactionTypeBuy,
					Symbol:          "AAPL",
					TransactionDate: "2024-01-15",
					Quantity:        decPtr(decimal.NewFromInt(10)),
					PricePerShare:   decPtr(dec("150.00")),
					TotalAmount:     dec("1500.00"),
				},
				{
					TransactionType: model.TransactionTypeSell,
					Symbol:          "AAPL",
					TransactionDate: "2024-02-01",
					Quantity:        decPtr(decimal.NewFromInt(100)), // Will fail: insufficient holdings
					PricePerShare:   decPtr(dec("160.00")),
					TotalAmount:     dec("16000.00"),
				},
			},
			createFunc: func(ctx context.Context, callerID, portfolioID string, in model.CreateTransactionInput) (*model.Transaction, error) {
				// First succeeds, second fails
				if in.TransactionType == model.TransactionTypeSell {
					return nil, model.NewConflict("insufficient holdings")
				}
				return &model.Transaction{ID: "txn-1"}, nil
			},
			wantCreated: 1,
			wantFailed:  1,
		},
		{
			name:          "empty list",
			transactions:  []model.CreateTransactionInput{},
			wantCreated:   0,
			wantFailed:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockTxnSvc := &mockTransactionService{createFunc: tt.createFunc}
			mockPortSvc := &mockPortfolioService{}

			svc := NewImportService(mockTxnSvc, mockPortSvc, logger)
			result, err := svc.Confirm(
				context.Background(),
				"user-123",
				"port-456",
				model.ImportConfirmRequest{Transactions: tt.transactions},
			)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if result.Created != tt.wantCreated {
				t.Errorf("wantCreated=%d, got %d", tt.wantCreated, result.Created)
			}

			if result.Failed != tt.wantFailed {
				t.Errorf("wantFailed=%d, got %d", tt.wantFailed, result.Failed)
			}

			// Verify service was called for each transaction
			expectedCalls := len(tt.transactions)
			if mockTxnSvc.callCount != expectedCalls {
				t.Errorf("expected %d Create calls, got %d", expectedCalls, mockTxnSvc.callCount)
			}

			// Verify error list has correct count
			if len(result.Errors) != tt.wantFailed {
				t.Errorf("expected %d errors, got %d", tt.wantFailed, len(result.Errors))
			}
		})
	}
}

// TestImportServiceConfirmPortfolioNotFound tests when portfolio access is denied.
func TestImportServiceConfirmPortfolioNotFound(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))

	mockTxnSvc := &mockTransactionService{}
	mockPortSvc := &mockPortfolioService{
		getByIDFunc: func(ctx context.Context, callerID, portfolioID string) (*model.Portfolio, error) {
			return nil, model.NewNotFound("portfolio")
		},
	}

	svc := NewImportService(mockTxnSvc, mockPortSvc, logger)
	req := model.ImportConfirmRequest{
		Transactions: []model.CreateTransactionInput{
			{
				TransactionType: model.TransactionTypeBuy,
				Symbol:          "AAPL",
				TransactionDate: "2024-01-15",
				Quantity:        decPtr(decimal.NewFromInt(10)),
				PricePerShare:   decPtr(dec("150.00")),
				TotalAmount:     dec("1500.00"),
			},
		},
	}

	_, err := svc.Confirm(context.Background(), "user-123", "nonexistent-port", req)
	if err == nil {
		t.Errorf("expected error for nonexistent portfolio")
	}
}

// Helper functions
func decPtr(d decimal.Decimal) *decimal.Decimal {
	return &d
}

func dec(s string) decimal.Decimal {
	d, _ := decimal.NewFromString(s)
	return d
}
