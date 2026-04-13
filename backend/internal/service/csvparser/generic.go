package csvparser

import (
	"fmt"
	"strings"

	"github.com/shopspring/decimal"

	"github.com/huchknows/fintech/backend/internal/model"
)

// GenericParser is a flexible fallback parser that detects columns by header name.
// It looks for: Date, Symbol/Ticker, Type/Action, Quantity/Shares, Price, Amount
type GenericParser struct{}

// Detect always returns true (it's a catch-all fallback).
func (p *GenericParser) Detect(headers []string) bool {
	return true
}

// Parse attempts to map generic columns to ImportRow.
func (p *GenericParser) Parse(record []string, headers map[string]int) (model.ImportRow, error) {
	row := model.ImportRow{}

	// Find columns by header (case-insensitive, substring match)
	findCol := func(keywords ...string) string {
		for headerName, idx := range headers {
			hLower := strings.ToLower(headerName)
			for _, kw := range keywords {
				if strings.Contains(hLower, strings.ToLower(kw)) {
					if idx < len(record) {
						return strings.TrimSpace(record[idx])
					}
				}
			}
		}
		return ""
	}

	// Try to find each required field
	symbol := findCol("symbol", "ticker")
	if symbol == "" {
		return row, fmt.Errorf("missing Symbol or Ticker column")
	}
	row.Symbol = symbol

	txType := findCol("type", "action", "activity")
	if txType == "" {
		return row, fmt.Errorf("missing Type/Action column")
	}

	// Map transaction type (generic keywords)
	txTypeLower := strings.ToLower(txType)
	switch {
	case strings.Contains(txTypeLower, "buy") || strings.Contains(txTypeLower, "purchase"):
		row.TransactionType = model.TransactionTypeBuy
	case strings.Contains(txTypeLower, "sell"):
		row.TransactionType = model.TransactionTypeSell
	case strings.Contains(txTypeLower, "dividend"):
		if strings.Contains(txTypeLower, "reinvest") {
			row.TransactionType = model.TransactionTypeReinvestedDividend
		} else {
			row.TransactionType = model.TransactionTypeDividend
		}
	default:
		return row, fmt.Errorf("unknown transaction type: %s", txType)
	}

	// Date
	date := findCol("date")
	if date == "" {
		return row, fmt.Errorf("missing Date column")
	}
	row.TransactionDate = date

	// Amount (required)
	amount := findCol("amount")
	if amount == "" {
		return row, fmt.Errorf("missing Amount column")
	}
	amountStr := strings.ReplaceAll(strings.ReplaceAll(amount, "$", ""), ",", "")
	amountDecimal, err := decimal.NewFromString(amountStr)
	if err != nil {
		return row, fmt.Errorf("invalid amount: %s", amount)
	}
	row.TotalAmount = amountDecimal.Abs()

	// Quantity (optional)
	quantity := findCol("quantity", "shares")
	if quantity != "" {
		quantityStr := strings.ReplaceAll(quantity, ",", "")
		q, err := decimal.NewFromString(quantityStr)
		if err == nil && !q.IsZero() {
			row.Quantity = &q
		}
	}

	// Price (optional)
	price := findCol("price")
	if price != "" {
		priceStr := strings.ReplaceAll(strings.ReplaceAll(price, "$", ""), ",", "")
		p, err := decimal.NewFromString(priceStr)
		if err == nil && !p.IsZero() {
			row.PricePerShare = &p
		}
	}

	// For dividend types, try to parse dividend per share
	if row.TransactionType == model.TransactionTypeDividend || row.TransactionType == model.TransactionTypeReinvestedDividend {
		divPerShare := findCol("dividend", "dividend_per_share")
		if divPerShare != "" {
			divStr := strings.ReplaceAll(strings.ReplaceAll(divPerShare, "$", ""), ",", "")
			d, err := decimal.NewFromString(divStr)
			if err == nil && !d.IsZero() {
				row.DividendPerShare = &d
			}
		}
		// If not found and quantity exists, derive from amount
		if row.DividendPerShare == nil && row.Quantity != nil && !row.Quantity.IsZero() {
			dividend := row.TotalAmount.Div(*row.Quantity)
			row.DividendPerShare = &dividend
		}
	}

	return row, nil
}
