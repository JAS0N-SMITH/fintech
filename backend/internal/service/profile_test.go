package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/huchknows/fintech/backend/internal/model"
	"github.com/huchknows/fintech/backend/internal/repository"
)

// --- mock profile repository ---

type mockProfileRepo struct {
	getByIDFn         func(ctx context.Context, id string) (*model.UserProfile, error)
	updatePreferenceFn func(ctx context.Context, id string, preferences json.RawMessage) error
}

func (m *mockProfileRepo) GetByID(ctx context.Context, id string) (*model.UserProfile, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return &model.UserProfile{ID: id}, nil
}

func (m *mockProfileRepo) UpdatePreferences(ctx context.Context, id string, preferences json.RawMessage) error {
	if m.updatePreferenceFn != nil {
		return m.updatePreferenceFn(ctx, id, preferences)
	}
	return nil
}

var _ repository.ProfileRepository = (*mockProfileRepo)(nil)

// --- tests ---

func TestProfileService_GetByID(t *testing.T) {
	tests := []struct {
		name      string
		userID    string
		repoRtn   *model.UserProfile
		repoErr   error
		wantErr   bool
		wantID    string
	}{
		{
			name:    "returns profile from repo",
			userID:  "u1",
			repoRtn: &model.UserProfile{ID: "u1", DisplayName: "Alice"},
			wantID:  "u1",
		},
		{
			name:    "repo ErrNotFound propagates",
			userID:  "missing",
			repoErr: model.ErrNotFound,
			wantErr: true,
		},
		{
			name:    "repo internal error propagates",
			userID:  "u1",
			repoErr: errors.New("db down"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockProfileRepo{
				getByIDFn: func(_ context.Context, id string) (*model.UserProfile, error) {
					return tt.repoRtn, tt.repoErr
				},
			}

			svc := NewProfileService(repo)
			profile, err := svc.GetByID(context.Background(), tt.userID)

			if (err != nil) != tt.wantErr {
				t.Errorf("err = %v, wantErr = %v", err, tt.wantErr)
			}
			if !tt.wantErr && profile.ID != tt.wantID {
				t.Errorf("profile.ID = %q, want %q", profile.ID, tt.wantID)
			}
		})
	}
}

func TestProfileService_UpdatePreferences(t *testing.T) {
	tests := []struct {
		name      string
		userID    string
		prefs     json.RawMessage
		repoErr   error
		wantErr   bool
	}{
		{
			name:    "updates preferences successfully",
			userID:  "u1",
			prefs:   json.RawMessage(`{"alert_thresholds":[5,10]}`),
			wantErr: false,
		},
		{
			name:    "repo error propagates",
			userID:  "u1",
			prefs:   json.RawMessage(`{"alert_thresholds":[]}`),
			repoErr: errors.New("update failed"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockProfileRepo{
				updatePreferenceFn: func(_ context.Context, _ string, _ json.RawMessage) error {
					return tt.repoErr
				},
			}

			svc := NewProfileService(repo)
			err := svc.UpdatePreferences(context.Background(), tt.userID, tt.prefs)

			if (err != nil) != tt.wantErr {
				t.Errorf("err = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}
