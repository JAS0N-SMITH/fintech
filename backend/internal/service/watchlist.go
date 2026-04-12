package service

import (
	"context"
	"errors"
	"regexp"
	"strings"

	"github.com/huchknows/fintech/backend/internal/model"
	"github.com/huchknows/fintech/backend/internal/repository"
)

var symbolPattern = regexp.MustCompile(`^[A-Z0-9.\-]{1,20}$`)

// WatchlistService handles watchlist business logic.
type WatchlistService interface {
	Create(ctx context.Context, userID string, in model.CreateWatchlistInput) (*model.Watchlist, error)
	GetByID(ctx context.Context, callerID, watchlistID string) (*model.Watchlist, error)
	List(ctx context.Context, userID string) ([]*model.Watchlist, error)
	Update(ctx context.Context, callerID, watchlistID string, in model.UpdateWatchlistInput) (*model.Watchlist, error)
	Delete(ctx context.Context, callerID, watchlistID string) error

	AddItem(ctx context.Context, callerID, watchlistID string, in model.CreateWatchlistItemInput) (*model.WatchlistItem, error)
	ListItems(ctx context.Context, callerID, watchlistID string) ([]*model.WatchlistItem, error)
	UpdateItem(ctx context.Context, callerID, watchlistID, symbol string, in model.UpdateWatchlistItemInput) (*model.WatchlistItem, error)
	RemoveItem(ctx context.Context, callerID, watchlistID, symbol string) error
}

type watchlistService struct {
	repo repository.WatchlistRepository
}

// NewWatchlistService returns a WatchlistService using the given repository.
func NewWatchlistService(repo repository.WatchlistRepository) WatchlistService {
	return &watchlistService{repo: repo}
}

// Create creates a new watchlist owned by userID.
func (s *watchlistService) Create(ctx context.Context, userID string, in model.CreateWatchlistInput) (*model.Watchlist, error) {
	w, err := s.repo.Create(ctx, userID, in)
	if err != nil {
		if errors.Is(err, model.ErrDuplicate) {
			return nil, model.NewValidation("a watchlist with this name already exists")
		}
		return nil, model.NewInternal()
	}
	return w, nil
}

// GetByID returns a watchlist by ID, enforcing that callerID is the owner.
func (s *watchlistService) GetByID(ctx context.Context, callerID, watchlistID string) (*model.Watchlist, error) {
	w, err := s.repo.GetByID(ctx, watchlistID)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			return nil, model.NewNotFound("watchlist")
		}
		return nil, model.NewInternal()
	}
	if w.UserID != callerID {
		return nil, model.NewForbidden()
	}
	return w, nil
}

// List returns all watchlists owned by userID.
func (s *watchlistService) List(ctx context.Context, userID string) ([]*model.Watchlist, error) {
	ws, err := s.repo.ListByUserID(ctx, userID)
	if err != nil {
		return nil, model.NewInternal()
	}
	if ws == nil {
		return []*model.Watchlist{}, nil
	}
	return ws, nil
}

// Update modifies a watchlist's name, enforcing ownership.
func (s *watchlistService) Update(ctx context.Context, callerID, watchlistID string, in model.UpdateWatchlistInput) (*model.Watchlist, error) {
	w, err := s.repo.GetByID(ctx, watchlistID)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			return nil, model.NewNotFound("watchlist")
		}
		return nil, model.NewInternal()
	}
	if w.UserID != callerID {
		return nil, model.NewForbidden()
	}
	updated, err := s.repo.Update(ctx, watchlistID, in)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			return nil, model.NewNotFound("watchlist")
		}
		return nil, model.NewInternal()
	}
	return updated, nil
}

// Delete removes a watchlist, enforcing ownership.
func (s *watchlistService) Delete(ctx context.Context, callerID, watchlistID string) error {
	w, err := s.repo.GetByID(ctx, watchlistID)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			return model.NewNotFound("watchlist")
		}
		return model.NewInternal()
	}
	if w.UserID != callerID {
		return model.NewForbidden()
	}
	err = s.repo.Delete(ctx, watchlistID)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			return model.NewNotFound("watchlist")
		}
		return model.NewInternal()
	}
	return nil
}

// AddItem adds a ticker symbol to a watchlist, enforcing ownership.
func (s *watchlistService) AddItem(ctx context.Context, callerID, watchlistID string, in model.CreateWatchlistItemInput) (*model.WatchlistItem, error) {
	// Validate symbol format (alphanumeric, dots, hyphens, max 20 chars)
	symbol := strings.ToUpper(in.Symbol)
	if !symbolPattern.MatchString(symbol) {
		return nil, model.NewValidation("invalid symbol format")
	}

	w, err := s.repo.GetByID(ctx, watchlistID)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			return nil, model.NewNotFound("watchlist")
		}
		return nil, model.NewInternal()
	}
	if w.UserID != callerID {
		return nil, model.NewForbidden()
	}

	// Normalize the symbol to uppercase before storing
	in.Symbol = symbol
	item, err := s.repo.AddItem(ctx, watchlistID, in)
	if err != nil {
		if errors.Is(err, model.ErrDuplicate) {
			return nil, model.NewValidation("this symbol is already in the watchlist")
		}
		return nil, model.NewInternal()
	}
	return item, nil
}

// ListItems returns all items in a watchlist, enforcing ownership.
func (s *watchlistService) ListItems(ctx context.Context, callerID, watchlistID string) ([]*model.WatchlistItem, error) {
	w, err := s.repo.GetByID(ctx, watchlistID)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			return nil, model.NewNotFound("watchlist")
		}
		return nil, model.NewInternal()
	}
	if w.UserID != callerID {
		return nil, model.NewForbidden()
	}
	items, err := s.repo.ListItems(ctx, watchlistID)
	if err != nil {
		return nil, model.NewInternal()
	}
	if items == nil {
		return []*model.WatchlistItem{}, nil
	}
	return items, nil
}

// UpdateItem modifies a watchlist item's target price and notes, enforcing ownership.
func (s *watchlistService) UpdateItem(ctx context.Context, callerID, watchlistID, symbol string, in model.UpdateWatchlistItemInput) (*model.WatchlistItem, error) {
	w, err := s.repo.GetByID(ctx, watchlistID)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			return nil, model.NewNotFound("watchlist")
		}
		return nil, model.NewInternal()
	}
	if w.UserID != callerID {
		return nil, model.NewForbidden()
	}
	item, err := s.repo.UpdateItem(ctx, watchlistID, symbol, in)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			return nil, model.NewNotFound("watchlist item")
		}
		return nil, model.NewInternal()
	}
	return item, nil
}

// RemoveItem deletes a watchlist item, enforcing ownership.
func (s *watchlistService) RemoveItem(ctx context.Context, callerID, watchlistID, symbol string) error {
	w, err := s.repo.GetByID(ctx, watchlistID)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			return model.NewNotFound("watchlist")
		}
		return model.NewInternal()
	}
	if w.UserID != callerID {
		return model.NewForbidden()
	}
	err = s.repo.RemoveItem(ctx, watchlistID, symbol)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			return model.NewNotFound("watchlist item")
		}
		return model.NewInternal()
	}
	return nil
}
