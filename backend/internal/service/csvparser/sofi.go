package csvparser

import (
	"fmt"
	"strings"

	"github.com/shopspring/decimal"

	"github.com/huchknows/fintech/backend/internal/model"
)

// SoFiParser handles SoFi (Invest) brokerage CSV exports.
// Expected columns: Date, Type, Ticker, Shares, Price, Amount, ...
type SoFiParser struct{}

// Detect checks if headers match SoFi format.
func (p *SoFiParser) Detect(headers []string) bool {
	hasDate := false
	hasType := false
	hasTicker := false
	hasShares := false

	for _, h := range headers {
		h = strings.ToLower(h)
		if h == "date" || strings.Contains(h, "date") {
			hasDate = true
		}
		if h == "type" || strings.Contains(h, "type") {
			hasType = true
		}
		if h == "ticker" || strings.Contains(h, "ticker") {
			hasTicker = true
		}
		if h == "shares" || strings.Contains(h, "shares") || h == "quantity" {
			hasShares = true
		}
	}
	return hasDate && hasType && hasTicker && hasShares
}

// Parse converts a SoFi CSV row to ImportRow.
func (p *SoFiParser) Parse(record []string, headers map[string]int) (model.ImportRow, error) {
	row := model.ImportRow{}

	// Helper to safely get column value
	get := func(colName string) string {
		// Try exact match first, then case-insensitive
		if idx, ok := headers[colName]; ok && idx < len(record) {
			return strings.TrimSpace(record[idx])
		}
		for key, idx := range headers {
			if strings.EqualFold(key, colName) && idx < len(record) {
				return strings.TrimSpace(record[idx])
			}
		}
		return ""
	}

	// Extract fields
	date := get("Date")
	txType := get("Type")
	ticker := get("Ticker")
	shares := get("Shares")
	price := get("Price")
	amount := get("Amount")

	if ticker == "" {
		return row, fmt.Errorf("missing Ticker")
	}
	if txType == "" {
		return row, fmt.Errorf("missing Type")
	}
	if amount == "" {
		return row, fmt.Errorf("missing Amount")
	}

	row.Symbol = ticker
	row.TransactionDate = date

	// Map SoFi transaction type
	switch strings.ToLower(strings.TrimSpace(txType)) {
	case "buy", "purchase":
		row.TransactionType = model.TransactionTypeBuy
	case "sell":
		row.TransactionType = model.TransactionTypeSell
	case "dividend":
		row.TransactionType = model.TransactionTypeDividend
	case "reinvested dividend", "dividend reinvestment":
		row.TransactionType = model.TransactionTypeReinvestedDividend
	default:
		return row, fmt.Errorf("unknown transaction type: %s", txType)
	}

	// Parse amount (remove $ and commas, handle parentheses for negative)
	amountStr := strings.TrimSpace(amount)
	isNegative := strings.HasPrefix(amountStr, "(") && strings.HasSuffix(amountStr, ")")
	amountStr = strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(amountStr, "$", ""), ",", ""), "(", "")
	amountStr = strings.ReplaceAll(amountStr, ")", "")

	amountDecimal, err := decimal.NewFromString(amountStr)
	if err != nil {
		return row, fmt.Errorf("invalid amount: %s", amount)
	}
	if isNegative {
		amountDecimal = amountDecimal.Neg()
	}
	row.TotalAmount = amountDecimal.Abs() // Normalize to positive

	// Parse shares/quantity
	if shares != "" {
		sharesStr := strings.ReplaceAll(strings.ReplaceAll(shares, ",", ""), " ", "")
		s, err := decimal.NewFromString(sharesStr)
		if err == nil && !s.IsZero() {
			row.Quantity = &s
		}
	}

	// Parse price
	if price != "" {
		priceStr := strings.ReplaceAll(strings.ReplaceAll(price, "$", ""), ",", "")
		p, err := decimal.NewFromString(priceStr)
		if err == nil && !p.IsZero() {
			row.PricePerShare = &p
		}
	}

	// For dividend/reinvested_dividend, derive dividend_per_share
	if (row.TransactionType == model.TransactionTypeDividend || row.TransactionType == model.TransactionTypeReinvestedDividend) &&
		row.Quantity != nil && !row.Quantity.IsZero() {
		dividend := row.TotalAmount.Div(*row.Quantity)
		row.DividendPerShare = &dividend
	}

	return row, nil
}
