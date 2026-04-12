package model

import (
	"time"

	"github.com/shopspring/decimal"
)

// TransactionType enumerates the valid financial event types.
type TransactionType string

const (
	// TransactionTypeBuy records a share purchase.
	TransactionTypeBuy TransactionType = "buy"
	// TransactionTypeSell records a share sale.
	TransactionTypeSell TransactionType = "sell"
	// TransactionTypeDividend records a cash dividend payment (no shares).
	TransactionTypeDividend TransactionType = "dividend"
	// TransactionTypeReinvestedDividend records a dividend reinvested as shares.
	TransactionTypeReinvestedDividend TransactionType = "reinvested_dividend"
)

// IsValid reports whether t is a recognised transaction type.
func (t TransactionType) IsValid() bool {
	switch t {
	case TransactionTypeBuy, TransactionTypeSell,
		TransactionTypeDividend, TransactionTypeReinvestedDividend:
		return true
	}
	return false
}

// Transaction is the single source of truth for a financial event.
// Current holdings and gain/loss are always derived from transaction history.
type Transaction struct {
	ID                string          `json:"id"`
	PortfolioID       string          `json:"portfolio_id"`
	TransactionType   TransactionType `json:"transaction_type"`
	Symbol            string          `json:"symbol"`
	TransactionDate   time.Time       `json:"transaction_date"`
	Quantity          *decimal.Decimal `json:"quantity,omitempty"`
	PricePerShare     *decimal.Decimal `json:"price_per_share,omitempty"`
	DividendPerShare  *decimal.Decimal `json:"dividend_per_share,omitempty"`
	TotalAmount       decimal.Decimal  `json:"total_amount"`
	Notes             string           `json:"notes,omitempty"`
	CreatedAt         time.Time        `json:"created_at"`
	UpdatedAt         time.Time        `json:"updated_at"`
}

// CreateTransactionInput holds validated fields for recording a new transaction.
type CreateTransactionInput struct {
	TransactionType  TransactionType `json:"transaction_type"  binding:"required,oneof=buy sell dividend reinvested_dividend"`
	Symbol           string          `json:"symbol"            binding:"required,min=1,max=20"`
	TransactionDate  string          `json:"transaction_date"  binding:"required"` // YYYY-MM-DD
	Quantity         *decimal.Decimal `json:"quantity"`
	PricePerShare    *decimal.Decimal `json:"price_per_share"`
	DividendPerShare *decimal.Decimal `json:"dividend_per_share"`
	TotalAmount      decimal.Decimal  `json:"total_amount"      binding:"required"`
	Notes            string           `json:"notes"             binding:"max=1000"`
}
