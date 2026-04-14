package csvparser

import (
	"fmt"
	"strings"
	"time"

	"github.com/huchknows/fintech/backend/internal/model"
)

// BrokerageParser defines the interface for brokerage-specific CSV parsing.
type BrokerageParser interface {
	Detect(headers []string) bool
	Parse(record []string, headers map[string]int) (model.ImportRow, error)
}

// NormalizeRow converts an ImportRow to CreateTransactionInput, applying business logic.
func NormalizeRow(row model.ImportRow) (model.CreateTransactionInput, error) {
	// Validate and uppercase symbol
	symbol := strings.ToUpper(strings.TrimSpace(row.Symbol))
	if err := validateSymbol(symbol); err != nil {
		return model.CreateTransactionInput{}, fmt.Errorf("invalid symbol: %w", err)
	}

	// Validate transaction type
	if !row.TransactionType.IsValid() {
		return model.CreateTransactionInput{}, fmt.Errorf("invalid transaction_type: %s", row.TransactionType)
	}

	// Parse transaction date
	txDate, err := parseTransactionDate(row.TransactionDate)
	if err != nil {
		return model.CreateTransactionInput{}, fmt.Errorf("invalid transaction_date: %w", err)
	}

	// Derive total_amount if missing
	totalAmount := row.TotalAmount
	if totalAmount.IsZero() {
		if row.Quantity == nil || row.PricePerShare == nil {
			return model.CreateTransactionInput{}, fmt.Errorf("total_amount required or cannot be derived from quantity × price_per_share")
		}
		totalAmount = row.Quantity.Mul(*row.PricePerShare)
	}

	// Validate quantity for buy/sell
	if row.TransactionType == model.TransactionTypeBuy || row.TransactionType == model.TransactionTypeSell {
		if row.Quantity == nil || row.Quantity.IsZero() || row.Quantity.IsNegative() {
			return model.CreateTransactionInput{}, fmt.Errorf("%s requires positive quantity", row.TransactionType)
		}
		if row.PricePerShare == nil || row.PricePerShare.IsZero() || row.PricePerShare.IsNegative() {
			return model.CreateTransactionInput{}, fmt.Errorf("%s requires positive price_per_share", row.TransactionType)
		}
	}

	// Validate dividend fields
	if row.TransactionType == model.TransactionTypeDividend || row.TransactionType == model.TransactionTypeReinvestedDividend {
		if row.DividendPerShare == nil || row.DividendPerShare.IsZero() || row.DividendPerShare.IsNegative() {
			return model.CreateTransactionInput{}, fmt.Errorf("%s requires positive dividend_per_share", row.TransactionType)
		}
	}

	// Validate total_amount
	if totalAmount.IsZero() || totalAmount.IsNegative() {
		return model.CreateTransactionInput{}, fmt.Errorf("total_amount must be positive")
	}

	return model.CreateTransactionInput{
		TransactionType:  row.TransactionType,
		Symbol:           symbol,
		TransactionDate:  txDate.Format("2006-01-02"),
		Quantity:         row.Quantity,
		PricePerShare:    row.PricePerShare,
		DividendPerShare: row.DividendPerShare,
		TotalAmount:      totalAmount,
		Notes:            strings.TrimSpace(row.Notes),
	}, nil
}

// validateSymbol checks that the symbol matches the allowed pattern: ^[A-Z0-9.\-]{1,20}$
func validateSymbol(symbol string) error {
	if len(symbol) == 0 || len(symbol) > 20 {
		return fmt.Errorf("symbol must be 1-20 characters")
	}
	for _, ch := range symbol {
		if (ch < 'A' || ch > 'Z') && (ch < '0' || ch > '9') && ch != '.' && ch != '-' {
			return fmt.Errorf("symbol contains invalid characters")
		}
	}
	return nil
}

// parseTransactionDate attempts to parse common date formats: YYYY-MM-DD, MM/DD/YYYY, MM/DD/YY
func parseTransactionDate(dateStr string) (time.Time, error) {
	dateStr = strings.TrimSpace(dateStr)
	formats := []string{
		"2006-01-02",      // YYYY-MM-DD
		"01/02/2006",      // MM/DD/YYYY
		"01/02/06",        // MM/DD/YY
		"January 2, 2006", // Month DD, YYYY
		"Jan 2, 2006",     // Mon DD, YYYY
	}
	for _, fmt := range formats {
		if t, err := time.Parse(fmt, dateStr); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unable to parse date: %s", dateStr)
}

// GetParser returns the appropriate parser for the given brokerage.
// If brokerage is empty, attempts auto-detection using Detect.
func GetParser(headers []string, brokerage string) BrokerageParser {
	if brokerage != "" {
		switch strings.ToLower(brokerage) {
		case "fidelity":
			return &FidelityParser{}
		case "sofi":
			return &SoFiParser{}
		case "generic":
			return &GenericParser{}
		}
	}

	// Auto-detect if no explicit brokerage or unknown brokerage
	parsers := []BrokerageParser{
		&FidelityParser{},
		&SoFiParser{},
		&GenericParser{}, // Generic is always a fallback
	}
	for _, p := range parsers {
		if p.Detect(headers) {
			return p
		}
	}
	return &GenericParser{} // Fallback
}
