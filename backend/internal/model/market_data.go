// Package model contains domain types for the fintech application.
package model

import "time"

// Quote represents a full market data snapshot for a single symbol.
// Used for the initial REST fetch on dashboard load (snapshot-plus-deltas pattern).
type Quote struct {
	Symbol        string    `json:"symbol"`
	Price         float64   `json:"price"`
	DayHigh       float64   `json:"day_high"`
	DayLow        float64   `json:"day_low"`
	Open          float64   `json:"open"`
	PreviousClose float64   `json:"previous_close"`
	Volume        int64     `json:"volume"`
	Timestamp     time.Time `json:"timestamp"`
}

// PriceTick is a lightweight real-time price update pushed over WebSocket.
// It contains only the fields that change tick-by-tick; other Quote fields
// are preserved from the last full snapshot.
type PriceTick struct {
	Symbol    string    `json:"symbol"`
	Price     float64   `json:"price"`
	Volume    int64     `json:"volume"`
	Timestamp time.Time `json:"timestamp"`
}

// Bar represents a single OHLCV candlestick for historical chart data.
type Bar struct {
	Symbol    string    `json:"symbol"`
	Open      float64   `json:"open"`
	High      float64   `json:"high"`
	Low       float64   `json:"low"`
	Close     float64   `json:"close"`
	Volume    int64     `json:"volume"`
	Timestamp time.Time `json:"timestamp"`
}

// Symbol represents a listed stock on an exchange, as returned by Finnhub.
type Symbol struct {
	Symbol        string `json:"symbol"`
	Description   string `json:"description"`
	DisplaySymbol string `json:"display_symbol"`
	Type          string `json:"type"`
	MIC           string `json:"mic"`
	Currency      string `json:"currency"`
}

// Timeframe represents the candle resolution for historical bar data.
type Timeframe string

const (
	// Timeframe1D represents 1-day candles (intraday 1-minute bars typically).
	Timeframe1D Timeframe = "1D"
	// Timeframe1W represents 1-week candles.
	Timeframe1W Timeframe = "1W"
	// Timeframe1M represents 1-month candles.
	Timeframe1M Timeframe = "1M"
	// Timeframe3M represents 3-month candles.
	Timeframe3M Timeframe = "3M"
	// Timeframe1Y represents 1-year candles.
	Timeframe1Y Timeframe = "1Y"
	// TimeframeAll represents all available history.
	TimeframeAll Timeframe = "ALL"
)

// ValidTimeframes is the set of accepted Timeframe values for input validation.
var ValidTimeframes = map[Timeframe]bool{
	Timeframe1D:  true,
	Timeframe1W:  true,
	Timeframe1M:  true,
	Timeframe3M:  true,
	Timeframe1Y:  true,
	TimeframeAll: true,
}
