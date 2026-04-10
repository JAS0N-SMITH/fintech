package model

import "time"

// Portfolio represents a named grouping of transactions — typically one brokerage account.
// No financial values are stored; all are derived from transactions at query time.
type Portfolio struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// CreatePortfolioInput holds validated fields for creating a new portfolio.
type CreatePortfolioInput struct {
	Name        string `json:"name"        binding:"required,min=1,max=100"`
	Description string `json:"description" binding:"max=500"`
}

// UpdatePortfolioInput holds validated fields for updating an existing portfolio.
// Only Name and Description are mutable; UserID is immutable after creation.
type UpdatePortfolioInput struct {
	Name        string `json:"name"        binding:"required,min=1,max=100"`
	Description string `json:"description" binding:"max=500"`
}
