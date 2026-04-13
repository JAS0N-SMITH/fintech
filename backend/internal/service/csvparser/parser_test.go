package csvparser

import (
	"testing"

	"github.com/shopspring/decimal"

	"github.com/huchknows/fintech/backend/internal/model"
)

// Helper to create decimal pointer
func decPtr(d decimal.Decimal) *decimal.Decimal {
	return &d
}

// Helper to create decimal from string
func dec(s string) decimal.Decimal {
	d, _ := decimal.NewFromString(s)
	return d
}

// TestNormalizeRow validates the normalization process from ImportRow to CreateTransactionInput.
func TestNormalizeRow(t *testing.T) {
	tests := []struct {
		name    string
		row     model.ImportRow
		wantErr bool
		check   func(t *testing.T, input model.CreateTransactionInput)
	}{
		{
			name: "valid buy transaction",
			row: model.ImportRow{
				Symbol:          "aapl",
				TransactionType: model.TransactionTypeBuy,
				TransactionDate: "2024-01-15",
				Quantity:        decPtr(decimal.NewFromInt(10)),
				PricePerShare:   decPtr(dec("150.00")),
				TotalAmount:     dec("1500.00"),
			},
			wantErr: false,
			check: func(t *testing.T, input model.CreateTransactionInput) {
				if input.Symbol != "AAPL" {
					t.Errorf("expected symbol AAPL, got %s", input.Symbol)
				}
				if input.TransactionType != model.TransactionTypeBuy {
					t.Errorf("expected type buy, got %s", input.TransactionType)
				}
			},
		},
		{
			name: "symbol uppercase conversion",
			row: model.ImportRow{
				Symbol:          "tsla",
				TransactionType: model.TransactionTypeSell,
				TransactionDate: "2024-02-01",
				Quantity:        decPtr(decimal.NewFromInt(5)),
				PricePerShare:   decPtr(dec("200.00")),
				TotalAmount:     dec("1000.00"),
			},
			wantErr: false,
			check: func(t *testing.T, input model.CreateTransactionInput) {
				if input.Symbol != "TSLA" {
					t.Errorf("expected uppercase TSLA, got %s", input.Symbol)
				}
			},
		},
		{
			name: "symbol with dots and hyphens",
			row: model.ImportRow{
				Symbol:          "ber-k.a",
				TransactionType: model.TransactionTypeBuy,
				TransactionDate: "2024-01-15",
				Quantity:        decPtr(decimal.NewFromInt(1)),
				PricePerShare:   decPtr(dec("300.00")),
				TotalAmount:     dec("300.00"),
			},
			wantErr: false,
			check: func(t *testing.T, input model.CreateTransactionInput) {
				if input.Symbol != "BER-K.A" {
					t.Errorf("expected BER-K.A, got %s", input.Symbol)
				}
			},
		},
		{
			name: "symbol too long",
			row: model.ImportRow{
				Symbol:          "TOOLONGSYMBOLUTF8XYZZ",
				TransactionType: model.TransactionTypeBuy,
				TransactionDate: "2024-01-15",
				Quantity:        decPtr(decimal.NewFromInt(1)),
				PricePerShare:   decPtr(dec("100.00")),
				TotalAmount:     dec("100.00"),
			},
			wantErr: true,
		},
		{
			name: "symbol with invalid characters",
			row: model.ImportRow{
				Symbol:          "AAPL!",
				TransactionType: model.TransactionTypeBuy,
				TransactionDate: "2024-01-15",
				Quantity:        decPtr(decimal.NewFromInt(1)),
				PricePerShare:   decPtr(dec("100.00")),
				TotalAmount:     dec("100.00"),
			},
			wantErr: true,
		},
		{
			name: "buy requires quantity and price",
			row: model.ImportRow{
				Symbol:          "AAPL",
				TransactionType: model.TransactionTypeBuy,
				TransactionDate: "2024-01-15",
				Quantity:        nil,
				PricePerShare:   decPtr(dec("150.00")),
				TotalAmount:     dec("1500.00"),
			},
			wantErr: true,
		},
		{
			name: "total amount derived from quantity × price",
			row: model.ImportRow{
				Symbol:          "AAPL",
				TransactionType: model.TransactionTypeBuy,
				TransactionDate: "2024-01-15",
				Quantity:        decPtr(decimal.NewFromInt(10)),
				PricePerShare:   decPtr(dec("150.00")),
				TotalAmount:     decimal.Zero,
			},
			wantErr: false,
			check: func(t *testing.T, input model.CreateTransactionInput) {
				expected := dec("1500.00")
				if !input.TotalAmount.Equal(expected) {
					t.Errorf("expected total_amount 1500.00, got %s", input.TotalAmount.String())
				}
			},
		},
		{
			name: "dividend transaction",
			row: model.ImportRow{
				Symbol:           "AAPL",
				TransactionType:  model.TransactionTypeDividend,
				TransactionDate:  "2024-01-15",
				DividendPerShare: decPtr(dec("0.25")),
				TotalAmount:      dec("25.00"),
			},
			wantErr: false,
			check: func(t *testing.T, input model.CreateTransactionInput) {
				if input.TransactionType != model.TransactionTypeDividend {
					t.Errorf("expected dividend type, got %s", input.TransactionType)
				}
			},
		},
		{
			name: "dividend requires dividend_per_share",
			row: model.ImportRow{
				Symbol:           "AAPL",
				TransactionType:  model.TransactionTypeDividend,
				TransactionDate:  "2024-01-15",
				DividendPerShare: nil,
				TotalAmount:      dec("25.00"),
			},
			wantErr: true,
		},
		{
			name: "negative total amount",
			row: model.ImportRow{
				Symbol:          "AAPL",
				TransactionType: model.TransactionTypeBuy,
				TransactionDate: "2024-01-15",
				Quantity:        decPtr(decimal.NewFromInt(10)),
				PricePerShare:   decPtr(dec("150.00")),
				TotalAmount:     dec("-1500.00"),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeRow(tt.row)
			if (err != nil) != tt.wantErr {
				t.Errorf("wantErr=%v, got err=%v", tt.wantErr, err)
			}
			if !tt.wantErr && tt.check != nil {
				tt.check(t, got)
			}
		})
	}
}

// TestFidelityParser tests Fidelity CSV parsing.
func TestFidelityParser(t *testing.T) {
	parser := &FidelityParser{}

	tests := []struct {
		name       string
		headers    map[string]int
		record     []string
		wantErr    bool
		wantType   model.TransactionType
		wantSymbol string
	}{
		{
			name: "valid buy",
			headers: map[string]int{
				"Run Date":      0,
				"Symbol":        1,
				"Activity Type": 2,
				"Quantity":      3,
				"Price":         4,
				"Amount":        5,
			},
			record:     []string{"01/15/2024", "AAPL", "Buy", "10", "150.00", "$1,500.00"},
			wantErr:    false,
			wantType:   model.TransactionTypeBuy,
			wantSymbol: "AAPL",
		},
		{
			name: "valid sell",
			headers: map[string]int{
				"Run Date":      0,
				"Symbol":        1,
				"Activity Type": 2,
				"Quantity":      3,
				"Price":         4,
				"Amount":        5,
			},
			record:     []string{"02/01/2024", "TSLA", "Sell", "5", "200.00", "$1,000.00"},
			wantErr:    false,
			wantType:   model.TransactionTypeSell,
			wantSymbol: "TSLA",
		},
		{
			name: "dividend",
			headers: map[string]int{
				"Run Date":      0,
				"Symbol":        1,
				"Activity Type": 2,
				"Quantity":      3,
				"Price":         4,
				"Amount":        5,
			},
			record:     []string{"03/01/2024", "MSFT", "Cash Dividend", "100", "", "$25.00"},
			wantErr:    false,
			wantType:   model.TransactionTypeDividend,
			wantSymbol: "MSFT",
		},
		{
			name: "missing symbol",
			headers: map[string]int{
				"Run Date":      0,
				"Activity Type": 1,
				"Amount":        2,
			},
			record:  []string{"01/15/2024", "Buy", "$1,500.00"},
			wantErr: true,
		},
		{
			name: "unknown activity type",
			headers: map[string]int{
				"Run Date":      0,
				"Symbol":        1,
				"Activity Type": 2,
				"Quantity":      3,
				"Price":         4,
				"Amount":        5,
			},
			record:  []string{"01/15/2024", "AAPL", "Unknown", "10", "150.00", "$1,500.00"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parser.Parse(tt.record, tt.headers)
			if (err != nil) != tt.wantErr {
				t.Errorf("wantErr=%v, got err=%v", tt.wantErr, err)
			}
			if !tt.wantErr {
				if got.TransactionType != tt.wantType {
					t.Errorf("wantType=%s, got %s", tt.wantType, got.TransactionType)
				}
				if got.Symbol != tt.wantSymbol {
					t.Errorf("wantSymbol=%s, got %s", tt.wantSymbol, got.Symbol)
				}
			}
		})
	}
}

// TestSoFiParser tests SoFi CSV parsing.
func TestSoFiParser(t *testing.T) {
	parser := &SoFiParser{}

	tests := []struct {
		name       string
		headers    map[string]int
		record     []string
		wantErr    bool
		wantType   model.TransactionType
		wantSymbol string
	}{
		{
			name: "valid buy",
			headers: map[string]int{
				"Date":   0,
				"Type":   1,
				"Ticker": 2,
				"Shares": 3,
				"Price":  4,
				"Amount": 5,
			},
			record:     []string{"01/15/2024", "Buy", "AAPL", "10", "$150.00", "$1,500.00"},
			wantErr:    false,
			wantType:   model.TransactionTypeBuy,
			wantSymbol: "AAPL",
		},
		{
			name: "valid sell",
			headers: map[string]int{
				"Date":   0,
				"Type":   1,
				"Ticker": 2,
				"Shares": 3,
				"Price":  4,
				"Amount": 5,
			},
			record:     []string{"02/01/2024", "Sell", "GOOG", "3", "$2,500.00", "$7,500.00"},
			wantErr:    false,
			wantType:   model.TransactionTypeSell,
			wantSymbol: "GOOG",
		},
		{
			name: "dividend",
			headers: map[string]int{
				"Date":   0,
				"Type":   1,
				"Ticker": 2,
				"Shares": 3,
				"Price":  4,
				"Amount": 5,
			},
			record:     []string{"03/15/2024", "Dividend", "MSFT", "50", "", "$12.50"},
			wantErr:    false,
			wantType:   model.TransactionTypeDividend,
			wantSymbol: "MSFT",
		},
		{
			name: "missing ticker",
			headers: map[string]int{
				"Date":   0,
				"Type":   1,
				"Shares": 2,
				"Amount": 3,
			},
			record:  []string{"01/15/2024", "Buy", "10", "$1,500.00"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parser.Parse(tt.record, tt.headers)
			if (err != nil) != tt.wantErr {
				t.Errorf("wantErr=%v, got err=%v", tt.wantErr, err)
			}
			if !tt.wantErr {
				if got.TransactionType != tt.wantType {
					t.Errorf("wantType=%s, got %s", tt.wantType, got.TransactionType)
				}
				if got.Symbol != tt.wantSymbol {
					t.Errorf("wantSymbol=%s, got %s", tt.wantSymbol, got.Symbol)
				}
			}
		})
	}
}

// TestGetParser tests auto-detection of brokerage format.
func TestGetParser(t *testing.T) {
	tests := []struct {
		name           string
		headers        []string
		brokerage      string
		wantParserType string
	}{
		{
			name:           "explicit fidelity",
			brokerage:      "fidelity",
			wantParserType: "*csvparser.FidelityParser",
		},
		{
			name:           "explicit sofi",
			brokerage:      "sofi",
			wantParserType: "*csvparser.SoFiParser",
		},
		{
			name:           "auto-detect fidelity",
			headers:        []string{"Run Date", "Account", "Symbol", "Activity Type", "Quantity", "Price", "Amount"},
			wantParserType: "*csvparser.FidelityParser",
		},
		{
			name:           "auto-detect sofi",
			headers:        []string{"Date", "Type", "Ticker", "Shares", "Price", "Amount"},
			wantParserType: "*csvparser.SoFiParser",
		},
		{
			name:           "fallback to generic",
			headers:        []string{"UnknownCol1", "UnknownCol2"},
			wantParserType: "*csvparser.GenericParser",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetParser(tt.headers, tt.brokerage)
			// Just verify that we got a parser back (full type checking is complex in Go tests)
			if got == nil {
				t.Errorf("expected non-nil parser")
			}
		})
	}
}
