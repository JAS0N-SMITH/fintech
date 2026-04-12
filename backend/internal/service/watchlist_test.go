package service

import (
	"context"
	"errors"
	"testing"

	"github.com/huchknows/fintech/backend/internal/model"
)

// ---------------------------------------------------------------------------
// Mock watchlist repository
// ---------------------------------------------------------------------------

type mockWatchlistRepo struct {
	watchlists map[string]*model.Watchlist
	items      map[string][]*model.WatchlistItem // key: watchlistID_symbol
	createFn   func(ctx context.Context, userID string, in model.CreateWatchlistInput) (*model.Watchlist, error)
	getByIDFn  func(ctx context.Context, id string) (*model.Watchlist, error)
	listFn     func(ctx context.Context, userID string) ([]*model.Watchlist, error)
	updateFn   func(ctx context.Context, id string, in model.UpdateWatchlistInput) (*model.Watchlist, error)
	deleteFn   func(ctx context.Context, id string) error
	addItemFn  func(ctx context.Context, watchlistID string, in model.CreateWatchlistItemInput) (*model.WatchlistItem, error)
}

func (m *mockWatchlistRepo) Create(ctx context.Context, userID string, in model.CreateWatchlistInput) (*model.Watchlist, error) {
	if m.createFn != nil {
		return m.createFn(ctx, userID, in)
	}
	w := &model.Watchlist{ID: "wl-new", UserID: userID, Name: in.Name}
	return w, nil
}

func (m *mockWatchlistRepo) GetByID(ctx context.Context, id string) (*model.Watchlist, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	if w, ok := m.watchlists[id]; ok {
		return w, nil
	}
	return nil, model.ErrNotFound
}

func (m *mockWatchlistRepo) ListByUserID(ctx context.Context, userID string) ([]*model.Watchlist, error) {
	if m.listFn != nil {
		return m.listFn(ctx, userID)
	}
	var result []*model.Watchlist
	for _, w := range m.watchlists {
		if w.UserID == userID {
			result = append(result, w)
		}
	}
	return result, nil
}

func (m *mockWatchlistRepo) Update(ctx context.Context, id string, in model.UpdateWatchlistInput) (*model.Watchlist, error) {
	if m.updateFn != nil {
		return m.updateFn(ctx, id, in)
	}
	w, ok := m.watchlists[id]
	if !ok {
		return nil, model.ErrNotFound
	}
	w.Name = in.Name
	return w, nil
}

func (m *mockWatchlistRepo) Delete(ctx context.Context, id string) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id)
	}
	if _, ok := m.watchlists[id]; !ok {
		return model.ErrNotFound
	}
	delete(m.watchlists, id)
	return nil
}

func (m *mockWatchlistRepo) AddItem(ctx context.Context, watchlistID string, in model.CreateWatchlistItemInput) (*model.WatchlistItem, error) {
	if m.addItemFn != nil {
		return m.addItemFn(ctx, watchlistID, in)
	}
	item := &model.WatchlistItem{ID: "item-new", WatchlistID: watchlistID, Symbol: in.Symbol, TargetPrice: in.TargetPrice, Notes: in.Notes}
	return item, nil
}

func (m *mockWatchlistRepo) GetItem(ctx context.Context, watchlistID, symbol string) (*model.WatchlistItem, error) {
	return nil, nil
}

func (m *mockWatchlistRepo) ListItems(ctx context.Context, watchlistID string) ([]*model.WatchlistItem, error) {
	return []*model.WatchlistItem{}, nil
}

func (m *mockWatchlistRepo) UpdateItem(ctx context.Context, watchlistID, symbol string, in model.UpdateWatchlistItemInput) (*model.WatchlistItem, error) {
	return nil, nil
}

func (m *mockWatchlistRepo) RemoveItem(ctx context.Context, watchlistID, symbol string) error {
	return nil
}

// fixture returns a repo seeded with one watchlist owned by "user-a".
func watchlistFixture() (*mockWatchlistRepo, *model.Watchlist) {
	w := &model.Watchlist{ID: "wl-1", UserID: "user-a", Name: "Tech Stocks"}
	return &mockWatchlistRepo{
		watchlists: map[string]*model.Watchlist{"wl-1": w},
		items:      make(map[string][]*model.WatchlistItem),
	}, w
}

// ---------------------------------------------------------------------------
// Tests — written before the service implementation
// ---------------------------------------------------------------------------

func TestWatchlistService_Create(t *testing.T) {
	repo := &mockWatchlistRepo{watchlists: make(map[string]*model.Watchlist)}
	svc := NewWatchlistService(repo)
	ctx := context.Background()

	t.Run("creates watchlist for user", func(t *testing.T) {
		w, err := svc.Create(ctx, "user-a", model.CreateWatchlistInput{Name: "My Watchlist"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if w.UserID != "user-a" {
			t.Errorf("user_id = %q, want %q", w.UserID, "user-a")
		}
		if w.Name != "My Watchlist" {
			t.Errorf("name = %q, want %q", w.Name, "My Watchlist")
		}
	})

	t.Run("propagates repository error", func(t *testing.T) {
		repo.createFn = func(_ context.Context, _ string, _ model.CreateWatchlistInput) (*model.Watchlist, error) {
			return nil, errors.New("db down")
		}
		defer func() { repo.createFn = nil }()

		_, err := svc.Create(ctx, "user-a", model.CreateWatchlistInput{Name: "X"})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestWatchlistService_GetByID(t *testing.T) {
	repo, fixture := watchlistFixture()
	svc := NewWatchlistService(repo)
	ctx := context.Background()

	t.Run("returns watchlist for owner", func(t *testing.T) {
		w, err := svc.GetByID(ctx, fixture.UserID, fixture.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if w.ID != fixture.ID {
			t.Errorf("id = %q, want %q", w.ID, fixture.ID)
		}
	})

	t.Run("returns forbidden for non-owner", func(t *testing.T) {
		_, err := svc.GetByID(ctx, "user-b", fixture.ID)
		if !errors.Is(err, model.ErrForbidden) {
			t.Errorf("err = %v, want ErrForbidden", err)
		}
	})

	t.Run("returns not found for missing watchlist", func(t *testing.T) {
		_, err := svc.GetByID(ctx, "user-a", "missing-id")
		if !errors.Is(err, model.ErrNotFound) {
			t.Errorf("err = %v, want ErrNotFound", err)
		}
	})
}

func TestWatchlistService_List(t *testing.T) {
	repo, _ := watchlistFixture()
	svc := NewWatchlistService(repo)
	ctx := context.Background()

	t.Run("returns only caller's watchlists", func(t *testing.T) {
		ws, err := svc.List(ctx, "user-a")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(ws) != 1 {
			t.Errorf("len = %d, want 1", len(ws))
		}
	})

	t.Run("returns empty slice for user with no watchlists", func(t *testing.T) {
		ws, err := svc.List(ctx, "user-z")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(ws) != 0 {
			t.Errorf("len = %d, want 0", len(ws))
		}
	})
}

func TestWatchlistService_Update(t *testing.T) {
	repo, fixture := watchlistFixture()
	svc := NewWatchlistService(repo)
	ctx := context.Background()

	t.Run("updates watchlist for owner", func(t *testing.T) {
		w, err := svc.Update(ctx, fixture.UserID, fixture.ID, model.UpdateWatchlistInput{Name: "Updated Name"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if w.Name != "Updated Name" {
			t.Errorf("name = %q, want %q", w.Name, "Updated Name")
		}
	})

	t.Run("returns forbidden for non-owner", func(t *testing.T) {
		_, err := svc.Update(ctx, "user-b", fixture.ID, model.UpdateWatchlistInput{Name: "X"})
		if !errors.Is(err, model.ErrForbidden) {
			t.Errorf("err = %v, want ErrForbidden", err)
		}
	})

	t.Run("returns not found for missing watchlist", func(t *testing.T) {
		_, err := svc.Update(ctx, "user-a", "ghost", model.UpdateWatchlistInput{Name: "X"})
		if !errors.Is(err, model.ErrNotFound) {
			t.Errorf("err = %v, want ErrNotFound", err)
		}
	})
}

func TestWatchlistService_Delete(t *testing.T) {
	repo, fixture := watchlistFixture()
	svc := NewWatchlistService(repo)
	ctx := context.Background()

	t.Run("deletes watchlist for owner", func(t *testing.T) {
		if err := svc.Delete(ctx, fixture.UserID, fixture.ID); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("returns forbidden for non-owner", func(t *testing.T) {
		repo2, fix2 := watchlistFixture()
		svc2 := NewWatchlistService(repo2)
		err := svc2.Delete(ctx, "user-b", fix2.ID)
		if !errors.Is(err, model.ErrForbidden) {
			t.Errorf("err = %v, want ErrForbidden", err)
		}
	})

	t.Run("returns not found for missing watchlist", func(t *testing.T) {
		err := svc.Delete(ctx, "user-a", "ghost")
		if !errors.Is(err, model.ErrNotFound) {
			t.Errorf("err = %v, want ErrNotFound", err)
		}
	})
}

func TestWatchlistService_AddItem(t *testing.T) {
	repo, fixture := watchlistFixture()
	svc := NewWatchlistService(repo)
	ctx := context.Background()

	t.Run("adds item to watchlist for owner", func(t *testing.T) {
		item, err := svc.AddItem(ctx, fixture.UserID, fixture.ID, model.CreateWatchlistItemInput{Symbol: "AAPL"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if item.Symbol != "AAPL" {
			t.Errorf("symbol = %q, want %q", item.Symbol, "AAPL")
		}
		if item.WatchlistID != fixture.ID {
			t.Errorf("watchlist_id = %q, want %q", item.WatchlistID, fixture.ID)
		}
	})

	t.Run("returns forbidden for non-owner", func(t *testing.T) {
		_, err := svc.AddItem(ctx, "user-b", fixture.ID, model.CreateWatchlistItemInput{Symbol: "AAPL"})
		if !errors.Is(err, model.ErrForbidden) {
			t.Errorf("err = %v, want ErrForbidden", err)
		}
	})

	t.Run("returns not found for missing watchlist", func(t *testing.T) {
		_, err := svc.AddItem(ctx, "user-a", "ghost", model.CreateWatchlistItemInput{Symbol: "AAPL"})
		if !errors.Is(err, model.ErrNotFound) {
			t.Errorf("err = %v, want ErrNotFound", err)
		}
	})
}

func TestWatchlistService_ListItems(t *testing.T) {
	repo, fixture := watchlistFixture()
	svc := NewWatchlistService(repo)
	ctx := context.Background()

	t.Run("returns items for owner's watchlist", func(t *testing.T) {
		items, err := svc.ListItems(ctx, fixture.UserID, fixture.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if items == nil {
			t.Fatal("expected non-nil slice")
		}
	})

	t.Run("returns forbidden for non-owner", func(t *testing.T) {
		_, err := svc.ListItems(ctx, "user-b", fixture.ID)
		if !errors.Is(err, model.ErrForbidden) {
			t.Errorf("err = %v, want ErrForbidden", err)
		}
	})

	t.Run("returns not found for missing watchlist", func(t *testing.T) {
		_, err := svc.ListItems(ctx, "user-a", "ghost")
		if !errors.Is(err, model.ErrNotFound) {
			t.Errorf("err = %v, want ErrNotFound", err)
		}
	})
}

func TestWatchlistService_RemoveItem(t *testing.T) {
	repo, fixture := watchlistFixture()
	svc := NewWatchlistService(repo)
	ctx := context.Background()

	t.Run("removes item from watchlist for owner", func(t *testing.T) {
		if err := svc.RemoveItem(ctx, fixture.UserID, fixture.ID, "AAPL"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("returns forbidden for non-owner", func(t *testing.T) {
		repo2, fix2 := watchlistFixture()
		svc2 := NewWatchlistService(repo2)
		err := svc2.RemoveItem(ctx, "user-b", fix2.ID, "AAPL")
		if !errors.Is(err, model.ErrForbidden) {
			t.Errorf("err = %v, want ErrForbidden", err)
		}
	})

	t.Run("returns not found for missing watchlist", func(t *testing.T) {
		err := svc.RemoveItem(ctx, "user-a", "ghost", "AAPL")
		if !errors.Is(err, model.ErrNotFound) {
			t.Errorf("err = %v, want ErrNotFound", err)
		}
	})
}
