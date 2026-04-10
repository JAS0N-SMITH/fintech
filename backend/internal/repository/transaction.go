package repository

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	"github.com/huchknows/fintech/backend/internal/model"
)

// TransactionRepository defines the data access interface for transactions.
type TransactionRepository interface {
	Create(ctx context.Context, portfolioID string, in model.CreateTransactionInput) (*model.Transaction, error)
	GetByID(ctx context.Context, id string) (*model.Transaction, error)
	ListByPortfolioID(ctx context.Context, portfolioID string) ([]*model.Transaction, error)
	// QuantityHeld returns the net quantity of symbol held in portfolioID.
	// Used by the service layer to enforce no-negative-holdings on sell.
	QuantityHeld(ctx context.Context, portfolioID, symbol string) (decimal.Decimal, error)
	Update(ctx context.Context, id string, in model.CreateTransactionInput) (*model.Transaction, error)
	Delete(ctx context.Context, id string) error
}

type transactionRepo struct {
	db *pgxpool.Pool
}

// NewTransactionRepository returns a TransactionRepository backed by the given pool.
func NewTransactionRepository(db *pgxpool.Pool) TransactionRepository {
	return &transactionRepo{db: db}
}

// Create inserts a transaction record and returns the created row.
func (r *transactionRepo) Create(ctx context.Context, portfolioID string, in model.CreateTransactionInput) (*model.Transaction, error) {
	const q = `
		INSERT INTO transactions
			(portfolio_id, transaction_type, symbol, transaction_date,
			 quantity, price_per_share, dividend_per_share, total_amount, notes)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		RETURNING id, portfolio_id, transaction_type, symbol, transaction_date,
		          quantity, price_per_share, dividend_per_share, total_amount,
		          COALESCE(notes,''), created_at, updated_at`

	date, err := time.Parse("2006-01-02", in.TransactionDate)
	if err != nil {
		return nil, model.NewValidation("transaction_date must be in YYYY-MM-DD format")
	}

	t := &model.Transaction{}
	err = r.db.QueryRow(ctx, q,
		portfolioID,
		in.TransactionType,
		in.Symbol,
		date,
		in.Quantity,
		in.PricePerShare,
		in.DividendPerShare,
		in.TotalAmount,
		in.Notes,
	).Scan(
		&t.ID, &t.PortfolioID, &t.TransactionType, &t.Symbol, &t.TransactionDate,
		&t.Quantity, &t.PricePerShare, &t.DividendPerShare, &t.TotalAmount,
		&t.Notes, &t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return t, nil
}

// GetByID fetches a single transaction by primary key.
func (r *transactionRepo) GetByID(ctx context.Context, id string) (*model.Transaction, error) {
	const q = `
		SELECT id, portfolio_id, transaction_type, symbol, transaction_date,
		       quantity, price_per_share, dividend_per_share, total_amount,
		       COALESCE(notes,''), created_at, updated_at
		FROM transactions WHERE id = $1`

	t := &model.Transaction{}
	err := r.db.QueryRow(ctx, q, id).Scan(
		&t.ID, &t.PortfolioID, &t.TransactionType, &t.Symbol, &t.TransactionDate,
		&t.Quantity, &t.PricePerShare, &t.DividendPerShare, &t.TotalAmount,
		&t.Notes, &t.CreatedAt, &t.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return t, nil
}

// ListByPortfolioID returns all transactions for a portfolio, newest date first.
func (r *transactionRepo) ListByPortfolioID(ctx context.Context, portfolioID string) ([]*model.Transaction, error) {
	const q = `
		SELECT id, portfolio_id, transaction_type, symbol, transaction_date,
		       quantity, price_per_share, dividend_per_share, total_amount,
		       COALESCE(notes,''), created_at, updated_at
		FROM transactions WHERE portfolio_id = $1
		ORDER BY transaction_date DESC, created_at DESC`

	rows, err := r.db.Query(ctx, q, portfolioID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var txns []*model.Transaction
	for rows.Next() {
		t := &model.Transaction{}
		if err := rows.Scan(
			&t.ID, &t.PortfolioID, &t.TransactionType, &t.Symbol, &t.TransactionDate,
			&t.Quantity, &t.PricePerShare, &t.DividendPerShare, &t.TotalAmount,
			&t.Notes, &t.CreatedAt, &t.UpdatedAt,
		); err != nil {
			return nil, err
		}
		txns = append(txns, t)
	}
	return txns, rows.Err()
}

// QuantityHeld returns SUM(buy+reinvested) - SUM(sell) for a symbol in a portfolio.
// Returns zero if there are no transactions.
func (r *transactionRepo) QuantityHeld(ctx context.Context, portfolioID, symbol string) (decimal.Decimal, error) {
	const q = `
		SELECT COALESCE(SUM(
			CASE
				WHEN transaction_type IN ('buy','reinvested_dividend') THEN quantity
				WHEN transaction_type = 'sell' THEN -quantity
				ELSE 0
			END
		), 0)
		FROM transactions
		WHERE portfolio_id = $1 AND symbol = $2 AND quantity IS NOT NULL`

	var held decimal.Decimal
	if err := r.db.QueryRow(ctx, q, portfolioID, symbol).Scan(&held); err != nil {
		return decimal.Zero, err
	}
	return held, nil
}

// Update replaces a transaction's fields. Returns model.ErrNotFound if missing.
func (r *transactionRepo) Update(ctx context.Context, id string, in model.CreateTransactionInput) (*model.Transaction, error) {
	const q = `
		UPDATE transactions
		SET transaction_type=$2, symbol=$3, transaction_date=$4,
		    quantity=$5, price_per_share=$6, dividend_per_share=$7,
		    total_amount=$8, notes=$9
		WHERE id=$1
		RETURNING id, portfolio_id, transaction_type, symbol, transaction_date,
		          quantity, price_per_share, dividend_per_share, total_amount,
		          COALESCE(notes,''), created_at, updated_at`

	date, err := time.Parse("2006-01-02", in.TransactionDate)
	if err != nil {
		return nil, model.NewValidation("transaction_date must be in YYYY-MM-DD format")
	}

	t := &model.Transaction{}
	err = r.db.QueryRow(ctx, q,
		id,
		in.TransactionType,
		in.Symbol,
		date,
		in.Quantity,
		in.PricePerShare,
		in.DividendPerShare,
		in.TotalAmount,
		in.Notes,
	).Scan(
		&t.ID, &t.PortfolioID, &t.TransactionType, &t.Symbol, &t.TransactionDate,
		&t.Quantity, &t.PricePerShare, &t.DividendPerShare, &t.TotalAmount,
		&t.Notes, &t.CreatedAt, &t.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return t, nil
}

// Delete removes a transaction by ID.
func (r *transactionRepo) Delete(ctx context.Context, id string) error {
	const q = `DELETE FROM transactions WHERE id = $1`
	tag, err := r.db.Exec(ctx, q, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return model.ErrNotFound
	}
	return nil
}
