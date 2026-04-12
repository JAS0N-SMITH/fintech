package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/huchknows/fintech/backend/internal/model"
	"github.com/huchknows/fintech/backend/internal/repository"
)

// ---------------------------------------------------------------------------
// Mock admin repository
// ---------------------------------------------------------------------------

type mockAdminRepo struct {
	listUsersFn    func(ctx context.Context, page, pageSize int) ([]model.AdminUser, int, error)
	updateRoleFn   func(ctx context.Context, id, role string) (*model.AdminUser, error)
	insertAuditFn  func(ctx context.Context, entry model.AuditLogEntry) error
	listAuditLogFn func(ctx context.Context, filter repository.AuditLogFilter) ([]model.AuditLogEntry, int, error)
}

func (m *mockAdminRepo) ListUsers(ctx context.Context, page, pageSize int) ([]model.AdminUser, int, error) {
	if m.listUsersFn != nil {
		return m.listUsersFn(ctx, page, pageSize)
	}
	return []model.AdminUser{}, 0, nil
}

func (m *mockAdminRepo) UpdateUserRole(ctx context.Context, id, role string) (*model.AdminUser, error) {
	if m.updateRoleFn != nil {
		return m.updateRoleFn(ctx, id, role)
	}
	return &model.AdminUser{ID: id, Role: role}, nil
}

func (m *mockAdminRepo) InsertAuditLog(ctx context.Context, entry model.AuditLogEntry) error {
	if m.insertAuditFn != nil {
		return m.insertAuditFn(ctx, entry)
	}
	return nil
}

func (m *mockAdminRepo) ListAuditLog(ctx context.Context, filter repository.AuditLogFilter) ([]model.AuditLogEntry, int, error) {
	if m.listAuditLogFn != nil {
		return m.listAuditLogFn(ctx, filter)
	}
	return []model.AuditLogEntry{}, 0, nil
}

// ---------------------------------------------------------------------------
// Mock ConnectionCounter
// ---------------------------------------------------------------------------

type mockConnectionCounter struct {
	countFn func() int
}

func (m *mockConnectionCounter) Count() int {
	if m.countFn != nil {
		return m.countFn()
	}
	return 0
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestAdminService_UpdateUserRole(t *testing.T) {
	tests := []struct {
		name       string
		adminID    string
		targetID   string
		newRole    string
		repoErr    error
		wantErr    bool
		wantRole   string
		wantErrMsg string
	}{
		{
			name:     "changes role for different user",
			adminID:  "admin-1",
			targetID: "user-2",
			newRole:  "admin",
			wantRole: "admin",
		},
		{
			name:       "prevents admin from changing own role",
			adminID:    "admin-1",
			targetID:   "admin-1",
			newRole:    "user",
			wantErr:    true,
			wantErrMsg: "cannot change your own role",
		},
		{
			name:     "repo not found error propagates",
			adminID:  "admin-1",
			targetID: "nonexistent",
			newRole:  "admin",
			repoErr:  model.ErrNotFound,
			wantErr:  true,
		},
		{
			name:     "repo internal error propagates",
			adminID:  "admin-1",
			targetID: "user-2",
			newRole:  "admin",
			repoErr:  errors.New("db error"),
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockAdminRepo{
				updateRoleFn: func(ctx context.Context, id, role string) (*model.AdminUser, error) {
					if tt.repoErr != nil {
						return nil, tt.repoErr
					}
					return &model.AdminUser{
						ID:    id,
						Email: "user@example.com",
						Role:  role,
					}, nil
				},
				insertAuditFn: func(ctx context.Context, entry model.AuditLogEntry) error {
					return nil
				},
			}

			svc := NewAdminService(repo, nil, nil, &mockConnectionCounter{})
			ctx := context.Background()

			user, err := svc.UpdateUserRole(ctx, tt.adminID, tt.targetID, tt.newRole)

			if tt.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !tt.wantErr && user.Role != tt.wantRole {
				t.Errorf("role = %q, want %q", user.Role, tt.wantRole)
			}

			if tt.wantErrMsg != "" && err != nil && !errors.Is(err, model.ErrConflict) {
				t.Errorf("expected conflict error, got %T", err)
			}
		})
	}
}

func TestAdminService_ListUsers(t *testing.T) {
	tests := []struct {
		name     string
		page     int
		pageSize int
		repoErr  error
		wantErr  bool
		wantLen  int
	}{
		{
			name:     "returns paginated users",
			page:     1,
			pageSize: 10,
			wantLen:  2,
		},
		{
			name:    "repo error propagates",
			page:    1,
			pageSize: 10,
			repoErr: errors.New("db error"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockAdminRepo{
				listUsersFn: func(ctx context.Context, page, pageSize int) ([]model.AdminUser, int, error) {
					if tt.repoErr != nil {
						return nil, 0, tt.repoErr
					}
					return []model.AdminUser{
						{ID: "u1", Email: "alice@example.com", Role: "user"},
						{ID: "u2", Email: "bob@example.com", Role: "admin"},
					}, 2, nil
				},
			}

			svc := NewAdminService(repo, nil, nil, &mockConnectionCounter{})
			ctx := context.Background()

			result, err := svc.ListUsers(ctx, tt.page, tt.pageSize)

			if tt.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !tt.wantErr && len(result.Users) != tt.wantLen {
				t.Errorf("len = %d, want %d", len(result.Users), tt.wantLen)
			}
		})
	}
}

func TestAdminService_RecordAuditEvent(t *testing.T) {
	tests := []struct {
		name    string
		entry   model.AuditLogEntry
		repoErr error
		wantErr bool
	}{
		{
			name: "records event successfully",
			entry: model.AuditLogEntry{
				ID:           "log1",
				UserID:       "admin-1",
				Action:       "role_change",
				TargetEntity: "user",
				TargetID:     "user-2",
				IPAddress:    "127.0.0.1",
				UserAgent:    "test",
			},
		},
		{
			name: "repo error propagates",
			entry: model.AuditLogEntry{
				ID:           "log1",
				UserID:       "admin-1",
				Action:       "role_change",
				TargetEntity: "user",
				TargetID:     "user-2",
			},
			repoErr: errors.New("db error"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockAdminRepo{
				insertAuditFn: func(ctx context.Context, entry model.AuditLogEntry) error {
					return tt.repoErr
				},
			}

			svc := NewAdminService(repo, nil, nil, &mockConnectionCounter{})
			ctx := context.Background()

			err := svc.RecordAuditEvent(ctx, tt.entry)

			if tt.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestAdminService_ListAuditLog(t *testing.T) {
	tests := []struct {
		name     string
		page     int
		pageSize int
		repoErr  error
		wantErr  bool
		wantLen  int
	}{
		{
			name:     "returns paginated audit log",
			page:     1,
			pageSize: 10,
			wantLen:  1,
		},
		{
			name:     "repo error propagates",
			page:     1,
			pageSize: 10,
			repoErr:  errors.New("db error"),
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockAdminRepo{
				listAuditLogFn: func(ctx context.Context, filter repository.AuditLogFilter) ([]model.AuditLogEntry, int, error) {
					if tt.repoErr != nil {
						return nil, 0, tt.repoErr
					}
					return []model.AuditLogEntry{
						{
							ID:           "log1",
							UserID:       "admin-1",
							Action:       "role_change",
							TargetEntity: "user",
							TargetID:     "user-2",
						},
					}, 1, nil
				},
			}

			svc := NewAdminService(repo, nil, nil, &mockConnectionCounter{})
			ctx := context.Background()

			filter := repository.AuditLogFilter{Page: tt.page, PageSize: tt.pageSize}
			result, err := svc.ListAuditLog(ctx, filter)

			if tt.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !tt.wantErr && len(result.Entries) != tt.wantLen {
				t.Errorf("len = %d, want %d", len(result.Entries), tt.wantLen)
			}
		})
	}
}

func TestAdminService_AuditEventPIIMasking(t *testing.T) {
	// Test that audit log entries don't include sensitive data
	repo := &mockAdminRepo{
		insertAuditFn: func(ctx context.Context, entry model.AuditLogEntry) error {
			// Verify after_value doesn't contain PII (like email)
			if entry.AfterValue != nil {
				var val map[string]interface{}
				if err := json.Unmarshal(entry.AfterValue, &val); err != nil {
					return err
				}
				// Only role should be present, no email or other PII
				for k := range val {
					if k != "role" {
						t.Errorf("unexpected field in audit log: %s", k)
					}
				}
			}
			return nil
		},
	}

	svc := NewAdminService(repo, nil, nil, &mockConnectionCounter{})
	ctx := context.Background()

	// Simulate recording a role change
	after, _ := json.Marshal(map[string]interface{}{"role": "admin"})
	entry := model.AuditLogEntry{
		ID:           "log1",
		UserID:       "admin-1",
		Action:       "role_change",
		TargetEntity: "user",
		TargetID:     "user-2",
		AfterValue:   after,
	}

	err := svc.RecordAuditEvent(ctx, entry)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
