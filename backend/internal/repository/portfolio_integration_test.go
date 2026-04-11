//go:build integration

package repository

import (
	"context"
	"errors"
	"testing"

	"github.com/huchknows/fintech/backend/internal/model"
)

func TestPortfolioRepository_CRUD(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewPortfolioRepository(pool)
	ctx := context.Background()

	userID := insertTestUser(t, pool)

	t.Run("Create returns portfolio with generated ID", func(t *testing.T) {
		p, err := repo.Create(ctx, userID, model.CreatePortfolioInput{
			Name:        "Brokerage",
			Description: "Main account",
		})
		if err != nil {
			t.Fatalf("Create: %v", err)
		}
		if p.ID == "" {
			t.Error("expected non-empty ID")
		}
		if p.UserID != userID {
			t.Errorf("UserID = %q, want %q", p.UserID, userID)
		}
		if p.Name != "Brokerage" {
			t.Errorf("Name = %q, want %q", p.Name, "Brokerage")
		}
		if p.Description != "Main account" {
			t.Errorf("Description = %q, want %q", p.Description, "Main account")
		}
	})

	t.Run("GetByID returns the created portfolio", func(t *testing.T) {
		created, err := repo.Create(ctx, userID, model.CreatePortfolioInput{Name: "Roth IRA"})
		if err != nil {
			t.Fatalf("Create: %v", err)
		}

		got, err := repo.GetByID(ctx, created.ID)
		if err != nil {
			t.Fatalf("GetByID: %v", err)
		}
		if got.ID != created.ID {
			t.Errorf("ID = %q, want %q", got.ID, created.ID)
		}
		if got.Name != "Roth IRA" {
			t.Errorf("Name = %q, want %q", got.Name, "Roth IRA")
		}
	})

	t.Run("GetByID returns ErrNotFound for missing ID", func(t *testing.T) {
		_, err := repo.GetByID(ctx, "00000000-0000-0000-0000-000000000000")
		if !errors.Is(err, model.ErrNotFound) {
			t.Errorf("err = %v, want ErrNotFound", err)
		}
	})

	t.Run("ListByUserID returns all portfolios for user, newest first", func(t *testing.T) {
		// Create a second user so we can verify isolation.
		otherUserID := insertTestUser(t, pool)

		_, err := repo.Create(ctx, otherUserID, model.CreatePortfolioInput{Name: "Other"})
		if err != nil {
			t.Fatalf("Create other: %v", err)
		}

		// Create two portfolios for the first user.
		_, err = repo.Create(ctx, userID, model.CreatePortfolioInput{Name: "Alpha"})
		if err != nil {
			t.Fatalf("Create Alpha: %v", err)
		}
		_, err = repo.Create(ctx, userID, model.CreatePortfolioInput{Name: "Beta"})
		if err != nil {
			t.Fatalf("Create Beta: %v", err)
		}

		ps, err := repo.ListByUserID(ctx, userID)
		if err != nil {
			t.Fatalf("ListByUserID: %v", err)
		}

		// We created at least 3 portfolios for userID across all sub-tests (Brokerage, Roth IRA, Alpha, Beta).
		// The "Other" portfolio belongs to otherUserID and must not appear.
		for _, p := range ps {
			if p.UserID != userID {
				t.Errorf("returned portfolio user_id = %q, want %q", p.UserID, userID)
			}
		}

		// Newest first: Beta should precede Alpha in the list.
		var idxAlpha, idxBeta int = -1, -1
		for i, p := range ps {
			if p.Name == "Alpha" {
				idxAlpha = i
			}
			if p.Name == "Beta" {
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
		ps, err := repo.ListByUserID(ctx, "00000000-0000-0000-0000-000000000001")
		if err != nil {
			t.Fatalf("ListByUserID: %v", err)
		}
		if ps != nil && len(ps) > 0 {
			t.Errorf("expected empty, got %d portfolios", len(ps))
		}
	})

	t.Run("Update modifies name and description", func(t *testing.T) {
		p, err := repo.Create(ctx, userID, model.CreatePortfolioInput{Name: "Old Name"})
		if err != nil {
			t.Fatalf("Create: %v", err)
		}

		updated, err := repo.Update(ctx, p.ID, model.UpdatePortfolioInput{
			Name:        "New Name",
			Description: "updated desc",
		})
		if err != nil {
			t.Fatalf("Update: %v", err)
		}
		if updated.Name != "New Name" {
			t.Errorf("Name = %q, want %q", updated.Name, "New Name")
		}
		if updated.Description != "updated desc" {
			t.Errorf("Description = %q, want %q", updated.Description, "updated desc")
		}
	})

	t.Run("Update returns ErrNotFound for missing ID", func(t *testing.T) {
		_, err := repo.Update(ctx, "00000000-0000-0000-0000-000000000000", model.UpdatePortfolioInput{Name: "X"})
		if !errors.Is(err, model.ErrNotFound) {
			t.Errorf("err = %v, want ErrNotFound", err)
		}
	})

	t.Run("Delete removes the portfolio", func(t *testing.T) {
		p, err := repo.Create(ctx, userID, model.CreatePortfolioInput{Name: "To Delete"})
		if err != nil {
			t.Fatalf("Create: %v", err)
		}

		if err := repo.Delete(ctx, p.ID); err != nil {
			t.Fatalf("Delete: %v", err)
		}

		_, err = repo.GetByID(ctx, p.ID)
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
