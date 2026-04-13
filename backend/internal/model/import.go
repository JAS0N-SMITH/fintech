package model

import "github.com/shopspring/decimal"

// ImportRow represents a single parsed row from a brokerage CSV.
type ImportRow struct {
	Symbol            string
	TransactionType   TransactionType
	TransactionDate   string // YYYY-MM-DD format
	Quantity          *decimal.Decimal
	PricePerShare     *decimal.Decimal
	DividendPerShare  *decimal.Decimal
	TotalAmount       decimal.Decimal
	Notes             string
}

// ImportError represents a parsing or validation error for a single row.
type ImportError struct {
	Row     int    `json:"row"`     // 1-indexed row number in CSV
	Message string `json:"message"` // Human-readable error description
}

// ImportPreview is the response when importing in dry-run mode.
type ImportPreview struct {
	Parsed       int                      `json:"parsed"`       // Total rows parsed (excluding header)
	Valid        int                      `json:"valid"`        // Rows that passed validation
	Errors       []ImportError            `json:"errors"`       // Per-row errors
	Transactions []CreateTransactionInput `json:"transactions"` // Valid transactions ready to create
}

// ImportConfirmRequest holds selected transactions to confirm and create.
type ImportConfirmRequest struct {
	Transactions []CreateTransactionInput `json:"transactions" binding:"required,min=1"`
}

// ImportResult is the response after confirming and persisting transactions.
type ImportResult struct {
	Created  int           `json:"created"`  // Number successfully created
	Failed   int           `json:"failed"`   // Number that failed
	Errors   []ImportError `json:"errors"`   // Per-row failure details (only for confirm, not preview)
	Messages []string      `json:"messages"` // Summary messages
}
