//go:build integration

package repository

import (
	"context"
	"errors"
	"testing"

	"github.com/huchknows/fintech/backend/internal/model"
)

func TestWatchlistRepository_CRUD(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewWatchlistRepository(pool)
	ctx := context.Background()

	userID := insertTestUser(t, pool)

	t.Run("Create returns watchlist with generated ID", func(t *testing.T) {
		w, err := repo.Create(ctx, userID, model.CreateWatchlistInput{
			Name: "Tech Stocks",
		})
		if err != nil {
			t.Fatalf("Create: %v", err)
		}
		if w.ID == "" {
			t.Error("expected non-empty ID")
		}
		if w.UserID != userID {
			t.Errorf("UserID = %q, want %q", w.UserID, userID)
		}
		if w.Name != "Tech Stocks" {
			t.Errorf("Name = %q, want %q", w.Name, "Tech Stocks")
		}
	})

	t.Run("GetByID returns the created watchlist", func(t *testing.T) {
		created, err := repo.Create(ctx, userID, model.CreateWatchlistInput{Name: "Healthcare"})
		if err != nil {
			t.Fatalf("Create: %v", err)
		}

		w, err := repo.GetByID(ctx, created.ID)
		if err != nil {
			t.Fatalf("GetByID: %v", err)
		}
		if w.ID != created.ID {
			t.Errorf("ID = %q, want %q", w.ID, created.ID)
		}
		if w.Name != "Healthcare" {
			t.Errorf("Name = %q, want %q", w.Name, "Healthcare")
		}
	})

	t.Run("GetByID returns ErrNotFound for missing ID", func(t *testing.T) {
		_, err := repo.GetByID(ctx, "00000000-0000-0000-0000-000000000000")
		if !errors.Is(err, model.ErrNotFound) {
			t.Errorf("err = %v, want ErrNotFound", err)
		}
	})

	t.Run("ListByUserID returns all watchlists for user, newest first", func(t *testing.T) {
		// Create a second user so we can verify isolation.
		otherUserID := insertTestUser(t, pool)

		_, err := repo.Create(ctx, otherUserID, model.CreateWatchlistInput{Name: "Other"})
		if err != nil {
			t.Fatalf("Create other: %v", err)
		}

		// Create two watchlists for the first user.
		w1, err := repo.Create(ctx, userID, model.CreateWatchlistInput{Name: "Alpha"})
		if err != nil {
			t.Fatalf("Create Alpha: %v", err)
		}
		w2, err := repo.Create(ctx, userID, model.CreateWatchlistInput{Name: "Beta"})
		if err != nil {
			t.Fatalf("Create Beta: %v", err)
		}

		ws, err := repo.ListByUserID(ctx, userID)
		if err != nil {
			t.Fatalf("ListByUserID: %v", err)
		}

		// Verify only this user's watchlists are returned
		for _, w := range ws {
			if w.UserID != userID {
				t.Errorf("returned watchlist user_id = %q, want %q", w.UserID, userID)
			}
		}

		// Newest first: Beta should come before Alpha
		var idxAlpha, idxBeta int = -1, -1
		for i, w := range ws {
			if w.ID == w1.ID {
				idxAlpha = i
			}
			if w.ID == w2.ID {
				idxBeta = i
			}
		}
		if idxAlpha == -1 || idxBeta == -1 {
			t.Fatal("Alpha or Beta not found in list")
		}
		if idxBeta > idxAlpha {
			t.Errorf("Beta (idx %d) should come before Alpha (idx %d) — newest first", idxBeta, idxAlpha)
		}
	})

	t.Run("ListByUserID returns empty slice for unknown user", func(t *testing.T) {
		ws, err := repo.ListByUserID(ctx, "00000000-0000-0000-0000-000000000000")
		if err != nil {
			t.Fatalf("ListByUserID: %v", err)
		}
		if ws != nil && len(ws) > 0 {
			t.Errorf("expected empty, got %d watchlists", len(ws))
		}
	})

	t.Run("Update modifies name", func(t *testing.T) {
		w, err := repo.Create(ctx, userID, model.CreateWatchlistInput{Name: "Old Name"})
		if err != nil {
			t.Fatalf("Create: %v", err)
		}

		updated, err := repo.Update(ctx, w.ID, model.UpdateWatchlistInput{
			Name: "New Name",
		})
		if err != nil {
			t.Fatalf("Update: %v", err)
		}
		if updated.Name != "New Name" {
			t.Errorf("Name = %q, want %q", updated.Name, "New Name")
		}
	})

	t.Run("Update returns ErrNotFound for missing ID", func(t *testing.T) {
		_, err := repo.Update(ctx, "00000000-0000-0000-0000-000000000000", model.UpdateWatchlistInput{Name: "X"})
		if !errors.Is(err, model.ErrNotFound) {
			t.Errorf("err = %v, want ErrNotFound", err)
		}
	})

	t.Run("Delete removes the watchlist", func(t *testing.T) {
		w, err := repo.Create(ctx, userID, model.CreateWatchlistInput{Name: "To Delete"})
		if err != nil {
			t.Fatalf("Create: %v", err)
		}

		if err := repo.Delete(ctx, w.ID); err != nil {
			t.Fatalf("Delete: %v", err)
		}

		_, err = repo.GetByID(ctx, w.ID)
		if !errors.Is(err, model.ErrNotFound) {
			t.Errorf("after delete: err = %v, want ErrNotFound", err)
		}
	})

	t.Run("Delete returns ErrNotFound for missing ID", func(t *testing.T) {
		err := repo.Delete(ctx, "00000000-0000-0000-0000-000000000000")
		if !errors.Is(err, model.ErrNotFound) {
			t.Errorf("err = %v, want ErrNotFound", err)
		}
	})
}

func TestWatchlistRepository_Items(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewWatchlistRepository(pool)
	ctx := context.Background()

	userID := insertTestUser(t, pool)
	w, err := repo.Create(ctx, userID, model.CreateWatchlistInput{Name: "Stocks"})
	if err != nil {
		t.Fatalf("Create watchlist: %v", err)
	}

	t.Run("AddItem inserts item with symbol", func(t *testing.T) {
		price := 150.0
		item, err := repo.AddItem(ctx, w.ID, model.CreateWatchlistItemInput{
			Symbol:      "AAPL",
			TargetPrice: &price,
			Notes:       "Buy on dip",
		})
		if err != nil {
			t.Fatalf("AddItem: %v", err)
		}
		if item.ID == "" {
			t.Error("expected non-empty ID")
		}
		if item.WatchlistID != w.ID {
			t.Errorf("WatchlistID = %q, want %q", item.WatchlistID, w.ID)
		}
		if item.Symbol != "AAPL" {
			t.Errorf("Symbol = %q, want %q", item.Symbol, "AAPL")
		}
		if item.Notes != "Buy on dip" {
			t.Errorf("Notes = %q, want %q", item.Notes, "Buy on dip")
		}
	})

	t.Run("GetItem retrieves by watchlistID and symbol", func(t *testing.T) {
		_, err := repo.AddItem(ctx, w.ID, model.CreateWatchlistItemInput{
			Symbol: "MSFT",
		})
		if err != nil {
			t.Fatalf("AddItem: %v", err)
		}

		item, err := repo.GetItem(ctx, w.ID, "MSFT")
		if err != nil {
			t.Fatalf("GetItem: %v", err)
		}
		if item.Symbol != "MSFT" {
			t.Errorf("Symbol = %q, want %q", item.Symbol, "MSFT")
		}
	})

	t.Run("GetItem returns ErrNotFound for missing symbol", func(t *testing.T) {
		_, err := repo.GetItem(ctx, w.ID, "INVALID")
		if !errors.Is(err, model.ErrNotFound) {
			t.Errorf("err = %v, want ErrNotFound", err)
		}
	})

	t.Run("ListItems returns all items ordered by created_at ASC", func(t *testing.T) {
		// Add multiple items
		for _, sym := range []string{"GOOG", "TSLA", "META"} {
			_, err := repo.AddItem(ctx, w.ID, model.CreateWatchlistItemInput{Symbol: sym})
			if err != nil {
				t.Fatalf("AddItem %q: %v", sym, err)
			}
		}

		items, err := repo.ListItems(ctx, w.ID)
		if err != nil {
			t.Fatalf("ListItems: %v", err)
		}

		if len(items) < 3 {
			t.Errorf("expected at least 3 items, got %d", len(items))
		}

		// Verify they're sorted by created_at (ascending)
		for i := 1; i < len(items); i++ {
			if items[i].CreatedAt.Before(items[i-1].CreatedAt) {
				t.Errorf("items not sorted ASC by created_at")
			}
		}
	})

	t.Run("ListItems returns empty slice for empty watchlist", func(t *testing.T) {
		w2, err := repo.Create(ctx, userID, model.CreateWatchlistInput{Name: "Empty"})
		if err != nil {
			t.Fatalf("Create watchlist: %v", err)
		}

		items, err := repo.ListItems(ctx, w2.ID)
		if err != nil {
			t.Fatalf("ListItems: %v", err)
		}
		if items != nil && len(items) > 0 {
			t.Errorf("expected empty, got %d items", len(items))
		}
	})

	t.Run("UpdateItem modifies target_price and notes", func(t *testing.T) {
		price100 := 100.0
		price120 := 120.0
		_, err := repo.AddItem(ctx, w.ID, model.CreateWatchlistItemInput{
			Symbol:      "AMZN",
			TargetPrice: &price100,
			Notes:       "Monitor",
		})
		if err != nil {
			t.Fatalf("AddItem: %v", err)
		}

		updated, err := repo.UpdateItem(ctx, w.ID, "AMZN", model.UpdateWatchlistItemInput{
			TargetPrice: &price120,
			Notes:       "Updated",
		})
		if err != nil {
			t.Fatalf("UpdateItem: %v", err)
		}
		if updated.TargetPrice == nil || *updated.TargetPrice != 120.0 {
			t.Errorf("TargetPrice = %v, want 120", updated.TargetPrice)
		}
		if updated.Notes != "Updated" {
			t.Errorf("Notes = %q, want %q", updated.Notes, "Updated")
		}
	})

	t.Run("UpdateItem returns ErrNotFound for missing item", func(t *testing.T) {
		price := 100.0
		_, err := repo.UpdateItem(ctx, w.ID, "INVALID", model.UpdateWatchlistItemInput{
			TargetPrice: &price,
			Notes:       "",
		})
		if !errors.Is(err, model.ErrNotFound) {
			t.Errorf("err = %v, want ErrNotFound", err)
		}
	})

	t.Run("RemoveItem deletes item by watchlistID and symbol", func(t *testing.T) {
		_, err := repo.AddItem(ctx, w.ID, model.CreateWatchlistItemInput{
			Symbol: "IBM",
		})
		if err != nil {
			t.Fatalf("AddItem: %v", err)
		}

		if err := repo.RemoveItem(ctx, w.ID, "IBM"); err != nil {
			t.Fatalf("RemoveItem: %v", err)
		}

		_, err = repo.GetItem(ctx, w.ID, "IBM")
		if !errors.Is(err, model.ErrNotFound) {
			t.Errorf("after delete: err = %v, want ErrNotFound", err)
		}
	})

	t.Run("RemoveItem returns ErrNotFound for missing item", func(t *testing.T) {
		err := repo.RemoveItem(ctx, w.ID, "INVALID")
		if !errors.Is(err, model.ErrNotFound) {
			t.Errorf("err = %v, want ErrNotFound", err)
		}
	})
}
