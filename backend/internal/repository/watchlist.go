package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/huchknows/fintech/backend/internal/model"
)

// WatchlistRepository defines the data access interface for watchlists and watchlist items.
// Services depend on this interface, not the concrete type, for testability.
type WatchlistRepository interface {
	// Watchlist operations
	Create(ctx context.Context, userID string, in model.CreateWatchlistInput) (*model.Watchlist, error)
	GetByID(ctx context.Context, id string) (*model.Watchlist, error)
	ListByUserID(ctx context.Context, userID string) ([]*model.Watchlist, error)
	Update(ctx context.Context, id string, in model.UpdateWatchlistInput) (*model.Watchlist, error)
	Delete(ctx context.Context, id string) error

	// Watchlist item operations
	AddItem(ctx context.Context, watchlistID string, in model.CreateWatchlistItemInput) (*model.WatchlistItem, error)
	GetItem(ctx context.Context, watchlistID, symbol string) (*model.WatchlistItem, error)
	ListItems(ctx context.Context, watchlistID string) ([]*model.WatchlistItem, error)
	UpdateItem(ctx context.Context, watchlistID, symbol string, in model.UpdateWatchlistItemInput) (*model.WatchlistItem, error)
	RemoveItem(ctx context.Context, watchlistID, symbol string) error
}

// watchlistRepo is the pgx-backed implementation.
type watchlistRepo struct {
	db *pgxpool.Pool
}

// NewWatchlistRepository returns a WatchlistRepository backed by the given pool.
func NewWatchlistRepository(db *pgxpool.Pool) WatchlistRepository {
	return &watchlistRepo{db: db}
}

// Create inserts a new watchlist and returns the created record.
func (r *watchlistRepo) Create(ctx context.Context, userID string, in model.CreateWatchlistInput) (*model.Watchlist, error) {
	const q = `
		INSERT INTO watchlists (user_id, name)
		VALUES ($1, $2)
		RETURNING id, user_id, name, created_at, updated_at`

	w := &model.Watchlist{}
	err := r.db.QueryRow(ctx, q, userID, in.Name).
		Scan(&w.ID, &w.UserID, &w.Name, &w.CreatedAt, &w.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return w, nil
}

// GetByID fetches a single watchlist by primary key.
// Returns model.ErrNotFound if no row exists.
func (r *watchlistRepo) GetByID(ctx context.Context, id string) (*model.Watchlist, error) {
	const q = `
		SELECT id, user_id, name, created_at, updated_at
		FROM watchlists WHERE id = $1`

	w := &model.Watchlist{}
	err := r.db.QueryRow(ctx, q, id).
		Scan(&w.ID, &w.UserID, &w.Name, &w.CreatedAt, &w.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return w, nil
}

// ListByUserID returns all watchlists owned by the given user, newest first.
func (r *watchlistRepo) ListByUserID(ctx context.Context, userID string) ([]*model.Watchlist, error) {
	const q = `
		SELECT id, user_id, name, created_at, updated_at
		FROM watchlists WHERE user_id = $1
		ORDER BY created_at DESC`

	rows, err := r.db.Query(ctx, q, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var watchlists []*model.Watchlist
	for rows.Next() {
		w := &model.Watchlist{}
		if err := rows.Scan(&w.ID, &w.UserID, &w.Name, &w.CreatedAt, &w.UpdatedAt); err != nil {
			return nil, err
		}
		watchlists = append(watchlists, w)
	}
	if watchlists == nil {
		watchlists = []*model.Watchlist{}
	}
	return watchlists, rows.Err()
}

// Update modifies the name of an existing watchlist.
// Returns model.ErrNotFound if no row matches.
func (r *watchlistRepo) Update(ctx context.Context, id string, in model.UpdateWatchlistInput) (*model.Watchlist, error) {
	const q = `
		UPDATE watchlists
		SET name = $2
		WHERE id = $1
		RETURNING id, user_id, name, created_at, updated_at`

	w := &model.Watchlist{}
	err := r.db.QueryRow(ctx, q, id, in.Name).
		Scan(&w.ID, &w.UserID, &w.Name, &w.CreatedAt, &w.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return w, nil
}

// Delete removes a watchlist by ID.
// Returns model.ErrNotFound if no row matches.
func (r *watchlistRepo) Delete(ctx context.Context, id string) error {
	const q = `DELETE FROM watchlists WHERE id = $1`

	tag, err := r.db.Exec(ctx, q, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return model.ErrNotFound
	}
	return nil
}

// AddItem inserts a new ticker symbol to a watchlist and returns the created record.
func (r *watchlistRepo) AddItem(ctx context.Context, watchlistID string, in model.CreateWatchlistItemInput) (*model.WatchlistItem, error) {
	const q = `
		INSERT INTO watchlist_items (watchlist_id, symbol, target_price, notes)
		VALUES ($1, $2, $3, $4)
		RETURNING id, watchlist_id, symbol, target_price, COALESCE(notes, ''), created_at, updated_at`

	item := &model.WatchlistItem{}
	err := r.db.QueryRow(ctx, q, watchlistID, in.Symbol, in.TargetPrice, in.Notes).
		Scan(&item.ID, &item.WatchlistID, &item.Symbol, &item.TargetPrice, &item.Notes, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		// Map duplicate constraint to model.ErrDuplicate
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrNotFound
		}
		return nil, err
	}
	return item, nil
}

// GetItem fetches a single watchlist item by watchlist ID and symbol.
// Returns model.ErrNotFound if no row exists.
func (r *watchlistRepo) GetItem(ctx context.Context, watchlistID, symbol string) (*model.WatchlistItem, error) {
	const q = `
		SELECT id, watchlist_id, symbol, target_price, COALESCE(notes, ''), created_at, updated_at
		FROM watchlist_items WHERE watchlist_id = $1 AND symbol = $2`

	item := &model.WatchlistItem{}
	err := r.db.QueryRow(ctx, q, watchlistID, symbol).
		Scan(&item.ID, &item.WatchlistID, &item.Symbol, &item.TargetPrice, &item.Notes, &item.CreatedAt, &item.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return item, nil
}

// ListItems returns all items in a watchlist, ordered by creation time.
func (r *watchlistRepo) ListItems(ctx context.Context, watchlistID string) ([]*model.WatchlistItem, error) {
	const q = `
		SELECT id, watchlist_id, symbol, target_price, COALESCE(notes, ''), created_at, updated_at
		FROM watchlist_items WHERE watchlist_id = $1
		ORDER BY created_at ASC`

	rows, err := r.db.Query(ctx, q, watchlistID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*model.WatchlistItem
	for rows.Next() {
		item := &model.WatchlistItem{}
		if err := rows.Scan(&item.ID, &item.WatchlistID, &item.Symbol, &item.TargetPrice, &item.Notes, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if items == nil {
		items = []*model.WatchlistItem{}
	}
	return items, rows.Err()
}

// UpdateItem modifies the target_price and notes of an existing watchlist item.
// Returns model.ErrNotFound if no row matches.
func (r *watchlistRepo) UpdateItem(ctx context.Context, watchlistID, symbol string, in model.UpdateWatchlistItemInput) (*model.WatchlistItem, error) {
	const q = `
		UPDATE watchlist_items
		SET target_price = $3, notes = $4
		WHERE watchlist_id = $1 AND symbol = $2
		RETURNING id, watchlist_id, symbol, target_price, COALESCE(notes, ''), created_at, updated_at`

	item := &model.WatchlistItem{}
	err := r.db.QueryRow(ctx, q, watchlistID, symbol, in.TargetPrice, in.Notes).
		Scan(&item.ID, &item.WatchlistID, &item.Symbol, &item.TargetPrice, &item.Notes, &item.CreatedAt, &item.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return item, nil
}

// RemoveItem deletes a watchlist item by watchlist ID and symbol.
// Returns model.ErrNotFound if no row matches.
func (r *watchlistRepo) RemoveItem(ctx context.Context, watchlistID, symbol string) error {
	const q = `DELETE FROM watchlist_items WHERE watchlist_id = $1 AND symbol = $2`

	tag, err := r.db.Exec(ctx, q, watchlistID, symbol)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return model.ErrNotFound
	}
	return nil
}
