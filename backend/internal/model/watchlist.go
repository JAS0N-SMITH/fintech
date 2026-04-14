package model

import "time"

// Watchlist represents a named collection of ticker symbols.
type Watchlist struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// WatchlistItem represents a single ticker symbol on a watchlist.
type WatchlistItem struct {
	ID          string    `json:"id"`
	WatchlistID string    `json:"watchlist_id"`
	Symbol      string    `json:"symbol"`
	TargetPrice *float64  `json:"target_price,omitempty"`
	Notes       string    `json:"notes,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// CreateWatchlistInput is the request body for creating a new watchlist.
type CreateWatchlistInput struct {
	Name string `json:"name" binding:"required,min=1,max=100"`
}

// UpdateWatchlistInput is the request body for updating a watchlist.
type UpdateWatchlistInput struct {
	Name string `json:"name" binding:"required,min=1,max=100"`
}

// CreateWatchlistItemInput is the request body for adding a ticker to a watchlist.
type CreateWatchlistItemInput struct {
	Symbol      string   `json:"symbol" binding:"required,min=1,max=20"`
	TargetPrice *float64 `json:"target_price"`
	Notes       string   `json:"notes" binding:"max=500"`
}

// UpdateWatchlistItemInput is the request body for updating a watchlist item.
type UpdateWatchlistItemInput struct {
	TargetPrice *float64 `json:"target_price"`
	Notes       string   `json:"notes" binding:"max=500"`
}
