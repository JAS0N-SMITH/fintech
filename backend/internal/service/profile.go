package service

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/huchknows/fintech/backend/internal/model"
	"github.com/huchknows/fintech/backend/internal/repository"
)

// ProfileService defines the business logic interface for user profile operations.
type ProfileService interface {
	GetByID(ctx context.Context, id string) (*model.UserProfile, error)
	UpdatePreferences(ctx context.Context, id string, preferences json.RawMessage) error
}

// profileService is the concrete implementation.
type profileService struct {
	repo repository.ProfileRepository
}

// NewProfileService returns a ProfileService wired to the given repository.
func NewProfileService(repo repository.ProfileRepository) ProfileService {
	return &profileService{repo: repo}
}

// GetByID retrieves a user profile by ID.
func (s *profileService) GetByID(ctx context.Context, id string) (*model.UserProfile, error) {
	profile, err := s.repo.GetByID(ctx, id)
	if errors.Is(err, model.ErrNotFound) {
		return nil, model.NewNotFound("profile")
	}
	if err != nil {
		return nil, model.NewInternal()
	}
	return profile, nil
}

// UpdatePreferences updates the user's preferences in the database.
func (s *profileService) UpdatePreferences(ctx context.Context, id string, preferences json.RawMessage) error {
	err := s.repo.UpdatePreferences(ctx, id, preferences)
	if err != nil {
		return model.NewInternal()
	}
	return nil
}
