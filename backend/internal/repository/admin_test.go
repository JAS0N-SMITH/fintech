//go:build integration

package repository

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/huchknows/fintech/backend/internal/model"
)

func TestAdminRepository_ListUsers(t *testing.T) {
	db := setupTestDB(t)
	repo := NewAdminRepository(db)
	ctx := context.Background()

	// Insert test users (auth trigger creates profiles automatically)
	u1 := insertTestUser(t, db)
	u2 := insertTestUser(t, db)
	u3 := insertTestUser(t, db)

	// Update profiles with display_name and role
	db.Exec(ctx, "UPDATE public.profiles SET display_name = $1, role = $2 WHERE id = $3", "Alice", "user", u1)
	db.Exec(ctx, "UPDATE public.profiles SET display_name = $1, role = $2 WHERE id = $3", "Bob", "admin", u2)
	db.Exec(ctx, "UPDATE public.profiles SET display_name = $1, role = $2 WHERE id = $3", "Charlie", "user", u3)

	tests := []struct {
		name       string
		page       int
		pageSize   int
		wantLen    int
		wantTotal  int
	}{
		{
			name:      "returns paginated users",
			page:      1,
			pageSize:  2,
			wantLen:   2,
			wantTotal: 3,
		},
		{
			name:      "returns second page",
			page:      2,
			pageSize:  2,
			wantLen:   1,
			wantTotal: 3,
		},
		{
			name:      "empty page returns empty list",
			page:      10,
			pageSize:  2,
			wantLen:   0,
			wantTotal: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			users, total, err := repo.ListUsers(ctx, tt.page, tt.pageSize)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(users) != tt.wantLen {
				t.Errorf("len = %d, want %d", len(users), tt.wantLen)
			}
			if total != tt.wantTotal {
				t.Errorf("total = %d, want %d", total, tt.wantTotal)
			}

			// Verify all users have required fields
			for _, u := range users {
				if u.ID == "" {
					t.Error("user ID is empty")
				}
				if u.Email == "" {
					t.Error("user email is empty")
				}
				if u.Role == "" {
					t.Error("user role is empty")
				}
			}
		})
	}
}

func TestAdminRepository_UpdateUserRole(t *testing.T) {
	db := setupTestDB(t)
	repo := NewAdminRepository(db)
	ctx := context.Background()

	userID := insertTestUser(t, db)
	db.Exec(ctx, "UPDATE public.profiles SET display_name = $1, role = $2 WHERE id = $3", "Test User", "user", userID)

	tests := []struct {
		name      string
		userID    string
		newRole   string
		wantErr   bool
		wantRole  string
	}{
		{
			name:     "updates user role to admin",
			userID:   userID,
			newRole:  "admin",
			wantRole: "admin",
		},
		{
			name:     "updates user role back to user",
			userID:   userID,
			newRole:  "user",
			wantRole: "user",
		},
		{
			name:    "nonexistent user returns not found",
			userID:  "nonexistent",
			newRole: "admin",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user, err := repo.UpdateUserRole(ctx, tt.userID, tt.newRole)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if err != model.ErrNotFound {
					t.Errorf("error = %v, want ErrNotFound", err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if user.Role != tt.wantRole {
				t.Errorf("role = %q, want %q", user.Role, tt.wantRole)
			}

			// Verify updated_at changed
			if user.UpdatedAt.IsZero() {
				t.Error("updated_at is zero")
			}
		})
	}
}

func TestAdminRepository_InsertAndListAuditLog(t *testing.T) {
	db := setupTestDB(t)
	repo := NewAdminRepository(db)
	ctx := context.Background()

	userID := insertTestUser(t, db)
	targetID := insertTestUser(t, db)

	// Insert test audit entries
	before := json.RawMessage(`{"role":"user"}`)
	after := json.RawMessage(`{"role":"admin"}`)

	entries := []model.AuditLogEntry{
		{
			UserID:       userID,
			Action:       "role_change",
			TargetEntity: "user",
			TargetID:     targetID,
			BeforeValue:  before,
			AfterValue:   after,
			IPAddress:    "192.168.1.1",
			UserAgent:    "test-client",
		},
		{
			UserID:       userID,
			Action:       "role_change",
			TargetEntity: "user",
			TargetID:     "another-user",
			BeforeValue:  before,
			AfterValue:   after,
			IPAddress:    "192.168.1.2",
			UserAgent:    "test-client",
		},
		{
			UserID:       "other-admin",
			Action:       "role_change",
			TargetEntity: "user",
			TargetID:     targetID,
			BeforeValue:  before,
			AfterValue:   after,
			IPAddress:    "192.168.1.3",
			UserAgent:    "test-client",
		},
	}

	for _, entry := range entries {
		if err := repo.InsertAuditLog(ctx, entry); err != nil {
			t.Fatalf("insert failed: %v", err)
		}
	}

	tests := []struct {
		name      string
		filter    AuditLogFilter
		wantLen   int
		wantTotal int
	}{
		{
			name:      "lists all entries",
			filter:    AuditLogFilter{Page: 1, PageSize: 10},
			wantLen:   3,
			wantTotal: 3,
		},
		{
			name:      "filters by user_id",
			filter:    AuditLogFilter{UserID: userID, Page: 1, PageSize: 10},
			wantLen:   2,
			wantTotal: 2,
		},
		{
			name:      "filters by action",
			filter:    AuditLogFilter{Action: "role_change", Page: 1, PageSize: 10},
			wantLen:   3,
			wantTotal: 3,
		},
		{
			name:      "filters by user and action",
			filter:    AuditLogFilter{UserID: userID, Action: "role_change", Page: 1, PageSize: 10},
			wantLen:   2,
			wantTotal: 2,
		},
		{
			name:      "respects pagination",
			filter:    AuditLogFilter{Page: 1, PageSize: 2},
			wantLen:   2,
			wantTotal: 3,
		},
		{
			name:      "empty page returns empty list",
			filter:    AuditLogFilter{Page: 10, PageSize: 10},
			wantLen:   0,
			wantTotal: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, total, err := repo.ListAuditLog(ctx, tt.filter)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(result) != tt.wantLen {
				t.Errorf("len = %d, want %d", len(result), tt.wantLen)
			}
			if total != tt.wantTotal {
				t.Errorf("total = %d, want %d", total, tt.wantTotal)
			}

			// Verify all entries have required fields
			for _, entry := range result {
				if entry.ID == "" {
					t.Error("entry ID is empty")
				}
				if entry.Action == "" {
					t.Error("entry action is empty")
				}
				if entry.CreatedAt.IsZero() {
					t.Error("entry created_at is zero")
				}
			}
		})
	}
}

func TestAdminRepository_AuditLogDateFilter(t *testing.T) {
	db := setupTestDB(t)
	repo := NewAdminRepository(db)
	ctx := context.Background()

	userID := insertTestUser(t, db)
	targetID := insertTestUser(t, db)

	// Insert entries at different times
	now := time.Now().UTC()
	past := now.Add(-24 * time.Hour)
	future := now.Add(24 * time.Hour)

	// We'll insert an entry now; filter tests check if date range logic works
	entry := model.AuditLogEntry{
		UserID:       userID,
		Action:       "role_change",
		TargetEntity: "user",
		TargetID:     targetID,
		IPAddress:    "127.0.0.1",
		UserAgent:    "test",
	}

	if err := repo.InsertAuditLog(ctx, entry); err != nil {
		t.Fatalf("insert failed: %v", err)
	}

	tests := []struct {
		name      string
		from      time.Time
		to        time.Time
		wantLen   int
	}{
		{
			name:    "filters by date range including now",
			from:    past,
			to:      future,
			wantLen: 1,
		},
		{
			name:    "filters by date range excluding now",
			from:    past,
			to:      now.Add(-1 * time.Hour),
			wantLen: 0,
		},
		{
			name:    "future date range returns empty",
			from:    now.Add(1 * time.Hour),
			to:      future,
			wantLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := AuditLogFilter{
				From:     tt.from,
				To:       tt.to,
				Page:     1,
				PageSize: 10,
			}

			result, _, err := repo.ListAuditLog(ctx, filter)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(result) != tt.wantLen {
				t.Errorf("len = %d, want %d", len(result), tt.wantLen)
			}
		})
	}
}

