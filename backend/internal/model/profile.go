package model

import "encoding/json"

// UserProfile represents the user's profile data including preferences stored as JSONB.
type UserProfile struct {
	ID          string          `json:"id"`
	Email       string          `json:"email"`
	DisplayName string          `json:"display_name"`
	Role        string          `json:"role"`
	Preferences json.RawMessage `json:"preferences"`
	CreatedAt   string          `json:"created_at"`
	UpdatedAt   string          `json:"updated_at"`
}

// UpdatePreferencesInput is the request payload for PATCH /me/preferences.
type UpdatePreferencesInput struct {
	AlertThresholds json.RawMessage `json:"alert_thresholds"`
}
