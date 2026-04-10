// Package repository handles database access using pgx.
// All functions return domain models and sentinel errors defined in model.
package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/huchknows/fintech/backend/internal/model"
)

// PortfolioRepository defines the data access interface for portfolios.
// Services depend on this interface, not the concrete type, for testability.
type PortfolioRepository interface {
	Create(ctx context.Context, userID string, in model.CreatePortfolioInput) (*model.Portfolio, error)
	GetByID(ctx context.Context, id string) (*model.Portfolio, error)
	ListByUserID(ctx context.Context, userID string) ([]*model.Portfolio, error)
	Update(ctx context.Context, id string, in model.UpdatePortfolioInput) (*model.Portfolio, error)
	Delete(ctx context.Context, id string) error
}

// portfolioRepo is the pgx-backed implementation.
type portfolioRepo struct {
	db *pgxpool.Pool
}

// NewPortfolioRepository returns a PortfolioRepository backed by the given pool.
func NewPortfolioRepository(db *pgxpool.Pool) PortfolioRepository {
	return &portfolioRepo{db: db}
}

// Create inserts a new portfolio and returns the created record.
func (r *portfolioRepo) Create(ctx context.Context, userID string, in model.CreatePortfolioInput) (*model.Portfolio, error) {
	const q = `
		INSERT INTO portfolios (user_id, name, description)
		VALUES ($1, $2, $3)
		RETURNING id, user_id, name, COALESCE(description, ''), created_at, updated_at`

	p := &model.Portfolio{}
	err := r.db.QueryRow(ctx, q, userID, in.Name, in.Description).
		Scan(&p.ID, &p.UserID, &p.Name, &p.Description, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return p, nil
}

// GetByID fetches a single portfolio by primary key.
// Returns model.ErrNotFound if no row exists.
func (r *portfolioRepo) GetByID(ctx context.Context, id string) (*model.Portfolio, error) {
	const q = `
		SELECT id, user_id, name, COALESCE(description, ''), created_at, updated_at
		FROM portfolios WHERE id = $1`

	p := &model.Portfolio{}
	err := r.db.QueryRow(ctx, q, id).
		Scan(&p.ID, &p.UserID, &p.Name, &p.Description, &p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return p, nil
}

// ListByUserID returns all portfolios owned by the given user, newest first.
func (r *portfolioRepo) ListByUserID(ctx context.Context, userID string) ([]*model.Portfolio, error) {
	const q = `
		SELECT id, user_id, name, COALESCE(description, ''), created_at, updated_at
		FROM portfolios WHERE user_id = $1
		ORDER BY created_at DESC`

	rows, err := r.db.Query(ctx, q, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var portfolios []*model.Portfolio
	for rows.Next() {
		p := &model.Portfolio{}
		if err := rows.Scan(&p.ID, &p.UserID, &p.Name, &p.Description, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		portfolios = append(portfolios, p)
	}
	return portfolios, rows.Err()
}

// Update modifies the name and description of an existing portfolio.
// Returns model.ErrNotFound if no row matches.
func (r *portfolioRepo) Update(ctx context.Context, id string, in model.UpdatePortfolioInput) (*model.Portfolio, error) {
	const q = `
		UPDATE portfolios
		SET name = $2, description = $3
		WHERE id = $1
		RETURNING id, user_id, name, COALESCE(description, ''), created_at, updated_at`

	p := &model.Portfolio{}
	err := r.db.QueryRow(ctx, q, id, in.Name, in.Description).
		Scan(&p.ID, &p.UserID, &p.Name, &p.Description, &p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return p, nil
}

// Delete removes a portfolio by ID.
// Returns model.ErrNotFound if no row matches.
func (r *portfolioRepo) Delete(ctx context.Context, id string) error {
	const q = `DELETE FROM portfolios WHERE id = $1`

	tag, err := r.db.Exec(ctx, q, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return model.ErrNotFound
	}
	return nil
}
