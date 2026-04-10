package service

import (
	"context"
	"errors"
	"testing"

	"github.com/huchknows/fintech/backend/internal/model"
)

// ---------------------------------------------------------------------------
// Mock portfolio repository
// ---------------------------------------------------------------------------

type mockPortfolioRepo struct {
	portfolios map[string]*model.Portfolio
	createFn   func(ctx context.Context, userID string, in model.CreatePortfolioInput) (*model.Portfolio, error)
	getByIDFn  func(ctx context.Context, id string) (*model.Portfolio, error)
	listFn     func(ctx context.Context, userID string) ([]*model.Portfolio, error)
	updateFn   func(ctx context.Context, id string, in model.UpdatePortfolioInput) (*model.Portfolio, error)
	deleteFn   func(ctx context.Context, id string) error
}

func (m *mockPortfolioRepo) Create(ctx context.Context, userID string, in model.CreatePortfolioInput) (*model.Portfolio, error) {
	if m.createFn != nil {
		return m.createFn(ctx, userID, in)
	}
	p := &model.Portfolio{ID: "new-id", UserID: userID, Name: in.Name, Description: in.Description}
	return p, nil
}

func (m *mockPortfolioRepo) GetByID(ctx context.Context, id string) (*model.Portfolio, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	if p, ok := m.portfolios[id]; ok {
		return p, nil
	}
	return nil, model.ErrNotFound
}

func (m *mockPortfolioRepo) ListByUserID(ctx context.Context, userID string) ([]*model.Portfolio, error) {
	if m.listFn != nil {
		return m.listFn(ctx, userID)
	}
	var result []*model.Portfolio
	for _, p := range m.portfolios {
		if p.UserID == userID {
			result = append(result, p)
		}
	}
	return result, nil
}

func (m *mockPortfolioRepo) Update(ctx context.Context, id string, in model.UpdatePortfolioInput) (*model.Portfolio, error) {
	if m.updateFn != nil {
		return m.updateFn(ctx, id, in)
	}
	p, ok := m.portfolios[id]
	if !ok {
		return nil, model.ErrNotFound
	}
	p.Name = in.Name
	p.Description = in.Description
	return p, nil
}

func (m *mockPortfolioRepo) Delete(ctx context.Context, id string) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id)
	}
	if _, ok := m.portfolios[id]; !ok {
		return model.ErrNotFound
	}
	delete(m.portfolios, id)
	return nil
}

// fixture returns a repo seeded with one portfolio owned by "user-a".
func portfolioFixture() (*mockPortfolioRepo, *model.Portfolio) {
	p := &model.Portfolio{ID: "port-1", UserID: "user-a", Name: "My Portfolio"}
	return &mockPortfolioRepo{
		portfolios: map[string]*model.Portfolio{"port-1": p},
	}, p
}

// ---------------------------------------------------------------------------
// Tests — written before the service implementation
// ---------------------------------------------------------------------------

func TestPortfolioService_Create(t *testing.T) {
	repo := &mockPortfolioRepo{}
	svc := NewPortfolioService(repo)
	ctx := context.Background()

	t.Run("creates portfolio for user", func(t *testing.T) {
		p, err := svc.Create(ctx, "user-a", model.CreatePortfolioInput{Name: "Roth IRA"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if p.UserID != "user-a" {
			t.Errorf("user_id = %q, want %q", p.UserID, "user-a")
		}
		if p.Name != "Roth IRA" {
			t.Errorf("name = %q, want %q", p.Name, "Roth IRA")
		}
	})

	t.Run("propagates repository error", func(t *testing.T) {
		repo.createFn = func(_ context.Context, _ string, _ model.CreatePortfolioInput) (*model.Portfolio, error) {
			return nil, errors.New("db down")
		}
		defer func() { repo.createFn = nil }()

		_, err := svc.Create(ctx, "user-a", model.CreatePortfolioInput{Name: "X"})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestPortfolioService_GetByID(t *testing.T) {
	repo, fixture := portfolioFixture()
	svc := NewPortfolioService(repo)
	ctx := context.Background()

	t.Run("returns portfolio for owner", func(t *testing.T) {
		p, err := svc.GetByID(ctx, fixture.UserID, fixture.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if p.ID != fixture.ID {
			t.Errorf("id = %q, want %q", p.ID, fixture.ID)
		}
	})

	t.Run("returns forbidden for non-owner", func(t *testing.T) {
		_, err := svc.GetByID(ctx, "user-b", fixture.ID)
		if !errors.Is(err, model.ErrForbidden) {
			t.Errorf("err = %v, want ErrForbidden", err)
		}
	})

	t.Run("returns not found for missing portfolio", func(t *testing.T) {
		_, err := svc.GetByID(ctx, "user-a", "missing-id")
		if !errors.Is(err, model.ErrNotFound) {
			t.Errorf("err = %v, want ErrNotFound", err)
		}
	})
}

func TestPortfolioService_List(t *testing.T) {
	repo, _ := portfolioFixture()
	svc := NewPortfolioService(repo)
	ctx := context.Background()

	t.Run("returns only caller's portfolios", func(t *testing.T) {
		ps, err := svc.List(ctx, "user-a")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(ps) != 1 {
			t.Errorf("len = %d, want 1", len(ps))
		}
	})

	t.Run("returns empty slice for user with no portfolios", func(t *testing.T) {
		ps, err := svc.List(ctx, "user-z")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(ps) != 0 {
			t.Errorf("len = %d, want 0", len(ps))
		}
	})
}

func TestPortfolioService_Update(t *testing.T) {
	repo, fixture := portfolioFixture()
	svc := NewPortfolioService(repo)
	ctx := context.Background()

	t.Run("updates portfolio for owner", func(t *testing.T) {
		p, err := svc.Update(ctx, fixture.UserID, fixture.ID, model.UpdatePortfolioInput{Name: "Updated"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if p.Name != "Updated" {
			t.Errorf("name = %q, want %q", p.Name, "Updated")
		}
	})

	t.Run("returns forbidden for non-owner", func(t *testing.T) {
		_, err := svc.Update(ctx, "user-b", fixture.ID, model.UpdatePortfolioInput{Name: "X"})
		if !errors.Is(err, model.ErrForbidden) {
			t.Errorf("err = %v, want ErrForbidden", err)
		}
	})

	t.Run("returns not found for missing portfolio", func(t *testing.T) {
		_, err := svc.Update(ctx, "user-a", "ghost", model.UpdatePortfolioInput{Name: "X"})
		if !errors.Is(err, model.ErrNotFound) {
			t.Errorf("err = %v, want ErrNotFound", err)
		}
	})
}

func TestPortfolioService_Delete(t *testing.T) {
	repo, fixture := portfolioFixture()
	svc := NewPortfolioService(repo)
	ctx := context.Background()

	t.Run("deletes portfolio for owner", func(t *testing.T) {
		if err := svc.Delete(ctx, fixture.UserID, fixture.ID); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("returns forbidden for non-owner", func(t *testing.T) {
		repo2, fix2 := portfolioFixture()
		svc2 := NewPortfolioService(repo2)
		err := svc2.Delete(ctx, "user-b", fix2.ID)
		if !errors.Is(err, model.ErrForbidden) {
			t.Errorf("err = %v, want ErrForbidden", err)
		}
	})

	t.Run("returns not found for missing portfolio", func(t *testing.T) {
		err := svc.Delete(ctx, "user-a", "ghost")
		if !errors.Is(err, model.ErrNotFound) {
			t.Errorf("err = %v, want ErrNotFound", err)
		}
	})
}
