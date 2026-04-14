//go:build integration

package repository

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/huchknows/fintech/backend/internal/model"
)

func TestProfileRepository_GetByID(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewProfileRepository(pool)
	ctx := context.Background()

	userID := insertTestUser(t, pool)

	t.Run("GetByID returns the created profile", func(t *testing.T) {
		// Insert a profile row (migrations create this when auth.users is inserted)
		_, err := pool.Exec(ctx,
			`INSERT INTO public.profiles (id, display_name, role, preferences)
			 VALUES ($1, $2, $3, $4)`,
			userID, "Alice", "user", json.RawMessage(`{}`))
		if err != nil {
			t.Fatalf("insert profile: %v", err)
		}

		profile, err := repo.GetByID(ctx, userID)
		if err != nil {
			t.Fatalf("GetByID: %v", err)
		}
		if profile.ID != userID {
			t.Errorf("ID = %q, want %q", profile.ID, userID)
		}
		if profile.DisplayName != "Alice" {
			t.Errorf("DisplayName = %q, want %q", profile.DisplayName, "Alice")
		}
		if profile.Role != "user" {
			t.Errorf("Role = %q, want %q", profile.Role, "user")
		}
	})

	t.Run("GetByID returns ErrNotFound for missing ID", func(t *testing.T) {
		_, err := repo.GetByID(ctx, "00000000-0000-0000-0000-000000000000")
		if !errors.Is(err, model.ErrNotFound) {
			t.Errorf("err = %v, want ErrNotFound", err)
		}
	})
}

func TestProfileRepository_UpdatePreferences(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewProfileRepository(pool)
	ctx := context.Background()

	userID := insertTestUser(t, pool)

	// Insert a profile
	_, err := pool.Exec(ctx,
		`INSERT INTO public.profiles (id, display_name, role, preferences)
		 VALUES ($1, $2, $3, $4)`,
		userID, "Bob", "user", json.RawMessage(`{"alert_thresholds":[]}`))
	if err != nil {
		t.Fatalf("insert profile: %v", err)
	}

	t.Run("UpdatePreferences merges JSON", func(t *testing.T) {
		newPrefs := json.RawMessage(`{"alert_thresholds":[5,10]}`)
		err := repo.UpdatePreferences(ctx, userID, newPrefs)
		if err != nil {
			t.Fatalf("UpdatePreferences: %v", err)
		}

		// Verify the update
		profile, err := repo.GetByID(ctx, userID)
		if err != nil {
			t.Fatalf("GetByID after update: %v", err)
		}

		var prefs map[string]interface{}
		if err := json.Unmarshal(profile.Preferences, &prefs); err != nil {
			t.Fatalf("unmarshal preferences: %v", err)
		}
		if _, ok := prefs["alert_thresholds"]; !ok {
			t.Error("expected alert_thresholds key in merged preferences")
		}
	})

	t.Run("UpdatePreferences on unknown user silently does nothing", func(t *testing.T) {
		// Update a non-existent user — this is OK (UPDATE affects 0 rows)
		err := repo.UpdatePreferences(ctx, "00000000-0000-0000-0000-000000000000", json.RawMessage(`{}`))
		if err != nil {
			t.Errorf("UpdatePreferences: %v (expected no error)", err)
		}
	})
}
