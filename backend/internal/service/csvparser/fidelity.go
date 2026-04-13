package csvparser

import (
	"fmt"
	"strings"

	"github.com/shopspring/decimal"

	"github.com/huchknows/fintech/backend/internal/model"
)

// FidelityParser handles Fidelity brokerage CSV exports.
// Expected columns: Run Date, Account, Symbol, Description, Activity Type, Quantity, Price, Amount
type FidelityParser struct{}

// Detect checks if headers match Fidelity format.
func (p *FidelityParser) Detect(headers []string) bool {
	// Fidelity CSVs typically have these key columns
	hasSymbol := false
	hasActivityType := false
	hasQuantity := false
	hasPrice := false
	hasAmount := false

	for _, h := range headers {
		h = strings.ToLower(h)
		if strings.Contains(h, "symbol") {
			hasSymbol = true
		}
		if strings.Contains(h, "activity") && strings.Contains(h, "type") {
			hasActivityType = true
		}
		if strings.Contains(h, "quantity") {
			hasQuantity = true
		}
		if strings.Contains(h, "price") {
			hasPrice = true
		}
		if strings.Contains(h, "amount") {
			hasAmount = true
		}
	}
	return hasSymbol && hasActivityType && hasQuantity && hasPrice && hasAmount
}

// Parse converts a Fidelity CSV row to ImportRow.
func (p *FidelityParser) Parse(record []string, headers map[string]int) (model.ImportRow, error) {
	row := model.ImportRow{}

	// Helper to safely get column value
	get := func(colName string) string {
		idx, ok := headers[colName]
		if !ok || idx >= len(record) {
			return ""
		}
		return strings.TrimSpace(record[idx])
	}

	// Extract fields
	symbol := get("Symbol")
	activityType := get("Activity Type")
	quantity := get("Quantity")
	price := get("Price")
	amount := get("Amount")
	runDate := get("Run Date")

	if symbol == "" {
		return row, fmt.Errorf("missing Symbol")
	}
	if activityType == "" {
		return row, fmt.Errorf("missing Activity Type")
	}
	if amount == "" {
		return row, fmt.Errorf("missing Amount")
	}

	row.Symbol = symbol
	row.TransactionDate = runDate

	// Map Fidelity activity type to transaction type
	switch strings.ToLower(strings.TrimSpace(activityType)) {
	case "buy", "you bought":
		row.TransactionType = model.TransactionTypeBuy
	case "sell", "you sold":
		row.TransactionType = model.TransactionTypeSell
	case "dividend", "cash dividend", "dividend received":
		row.TransactionType = model.TransactionTypeDividend
	case "reinvested dividend", "dividend reinvest":
		row.TransactionType = model.TransactionTypeReinvestedDividend
	default:
		return row, fmt.Errorf("unknown activity type: %s", activityType)
	}

	// Parse amount (remove $ and commas)
	amountStr := strings.ReplaceAll(strings.ReplaceAll(amount, "$", ""), ",", "")
	amountDecimal, err := decimal.NewFromString(amountStr)
	if err != nil {
		return row, fmt.Errorf("invalid amount: %s", amount)
	}
	row.TotalAmount = amountDecimal.Abs() // Fidelity may use negative for sells

	// Parse quantity if present
	if quantity != "" {
		quantityStr := strings.ReplaceAll(quantity, ",", "")
		q, err := decimal.NewFromString(quantityStr)
		if err == nil && !q.IsZero() {
			row.Quantity = &q
		}
	}

	// Parse price if present
	if price != "" {
		priceStr := strings.ReplaceAll(strings.ReplaceAll(price, "$", ""), ",", "")
		p, err := decimal.NewFromString(priceStr)
		if err == nil && !p.IsZero() {
			row.PricePerShare = &p
		}
	}

	// For dividend/reinvested_dividend, derive dividend_per_share from amount and quantity
	if (row.TransactionType == model.TransactionTypeDividend || row.TransactionType == model.TransactionTypeReinvestedDividend) &&
		row.Quantity != nil && !row.Quantity.IsZero() {
		dividend := row.TotalAmount.Div(*row.Quantity)
		row.DividendPerShare = &dividend
	}

	return row, nil
}
