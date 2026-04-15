package repository

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/huchknows/fintech/backend/internal/model"
)

// ProfileRepository defines the data access interface for user profile operations.
type ProfileRepository interface {
	GetByID(ctx context.Context, id string) (*model.UserProfile, error)
	UpdatePreferences(ctx context.Context, id string, preferences json.RawMessage) error
}

// profileRepo is the pgx-backed implementation.
type profileRepo struct {
	db *pgxpool.Pool
}

// NewProfileRepository returns a ProfileRepository backed by the given pool.
func NewProfileRepository(db *pgxpool.Pool) ProfileRepository {
	return &profileRepo{db: db}
}

// GetByID retrieves a user profile by ID, including their preferences.
func (r *profileRepo) GetByID(ctx context.Context, id string) (*model.UserProfile, error) {
	q := `
		SELECT id, COALESCE(display_name, ''), role, preferences,
		       created_at::text, updated_at::text
		FROM public.profiles
		WHERE id = $1
	`

	profile := &model.UserProfile{}
	err := r.db.QueryRow(ctx, q, id).Scan(
		&profile.ID,
		&profile.DisplayName,
		&profile.Role,
		&profile.Preferences,
		&profile.CreatedAt,
		&profile.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	return profile, nil
}

// UpdatePreferences merges the given preferences JSON into the user's existing preferences using JSONB merge operator.
// This preserves other preference keys and only updates/adds the provided keys.
func (r *profileRepo) UpdatePreferences(ctx context.Context, id string, preferences json.RawMessage) error {
	q := `
		UPDATE public.profiles
		SET preferences = preferences || $1::jsonb, updated_at = now()
		WHERE id = $2
	`

	_, err := r.db.Exec(ctx, q, preferences, id)
	return err
}
