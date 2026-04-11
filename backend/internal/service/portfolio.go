// Package service implements business logic for the portfolio dashboard.
package service

import (
	"context"
	"errors"

	"github.com/huchknows/fintech/backend/internal/model"
	"github.com/huchknows/fintech/backend/internal/repository"
)

// PortfolioService handles portfolio business logic.
type PortfolioService interface {
	Create(ctx context.Context, userID string, in model.CreatePortfolioInput) (*model.Portfolio, error)
	GetByID(ctx context.Context, callerID, portfolioID string) (*model.Portfolio, error)
	List(ctx context.Context, userID string) ([]*model.Portfolio, error)
	Update(ctx context.Context, callerID, portfolioID string, in model.UpdatePortfolioInput) (*model.Portfolio, error)
	Delete(ctx context.Context, callerID, portfolioID string) error
}

type portfolioService struct {
	repo repository.PortfolioRepository
}

// NewPortfolioService returns a PortfolioService using the given repository.
func NewPortfolioService(repo repository.PortfolioRepository) PortfolioService {
	return &portfolioService{repo: repo}
}

// Create creates a new portfolio owned by userID.
func (s *portfolioService) Create(ctx context.Context, userID string, in model.CreatePortfolioInput) (*model.Portfolio, error) {
	p, err := s.repo.Create(ctx, userID, in)
	if err != nil {
		if errors.Is(err, model.ErrDuplicate) {
			return nil, model.NewValidation("a portfolio with this name already exists")
		}
		return nil, model.NewInternal()
	}
	return p, nil
}

// GetByID returns a portfolio by ID, enforcing that callerID is the owner.
func (s *portfolioService) GetByID(ctx context.Context, callerID, portfolioID string) (*model.Portfolio, error) {
	p, err := s.repo.GetByID(ctx, portfolioID)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			return nil, model.NewNotFound("portfolio")
		}
		return nil, model.NewInternal()
	}
	if p.UserID != callerID {
		return nil, model.NewForbidden()
	}
	return p, nil
}

// List returns all portfolios owned by userID.
func (s *portfolioService) List(ctx context.Context, userID string) ([]*model.Portfolio, error) {
	ps, err := s.repo.ListByUserID(ctx, userID)
	if err != nil {
		return nil, model.NewInternal()
	}
	if ps == nil {
		return []*model.Portfolio{}, nil
	}
	return ps, nil
}

// Update modifies a portfolio's name and description, enforcing ownership.
func (s *portfolioService) Update(ctx context.Context, callerID, portfolioID string, in model.UpdatePortfolioInput) (*model.Portfolio, error) {
	p, err := s.repo.GetByID(ctx, portfolioID)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			return nil, model.NewNotFound("portfolio")
		}
		return nil, model.NewInternal()
	}
	if p.UserID != callerID {
		return nil, model.NewForbidden()
	}
	updated, err := s.repo.Update(ctx, portfolioID, in)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			return nil, model.NewNotFound("portfolio")
		}
		return nil, model.NewInternal()
	}
	return updated, nil
}

// Delete removes a portfolio, enforcing ownership.
func (s *portfolioService) Delete(ctx context.Context, callerID, portfolioID string) error {
	p, err := s.repo.GetByID(ctx, portfolioID)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			return model.NewNotFound("portfolio")
		}
		return model.NewInternal()
	}
	if p.UserID != callerID {
		return model.NewForbidden()
	}
	err = s.repo.Delete(ctx, portfolioID)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			return model.NewNotFound("portfolio")
		}
		return model.NewInternal()
	}
	return nil
}
