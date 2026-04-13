package service

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/huchknows/fintech/backend/internal/model"
	"github.com/huchknows/fintech/backend/internal/provider"
	"github.com/huchknows/fintech/backend/internal/repository"
)

// AdminService defines operations for admin functionality.
type AdminService interface {
	ListUsers(ctx context.Context, page, pageSize int) (*model.AdminUserList, error)
	UpdateUserRole(ctx context.Context, adminID, targetID, newRole string) (*model.AdminUser, error)
	ListAuditLog(ctx context.Context, filter repository.AuditLogFilter) (*model.AuditLogList, error)
	RecordAuditEvent(ctx context.Context, entry model.AuditLogEntry) error
	GetSystemHealth(ctx context.Context) (*model.HealthStatus, error)
}

// ConnectionCounter provides the count of active WebSocket connections.
type ConnectionCounter interface {
	Count() int
}

// adminService is the concrete implementation.
type adminService struct {
	repo            repository.AdminRepository
	db              *pgxpool.Pool
	finnhubProvider provider.MarketDataProvider
	wsCounter       ConnectionCounter
}

// NewAdminService creates a new admin service.
func NewAdminService(
	repo repository.AdminRepository,
	db *pgxpool.Pool,
	finnhubProvider provider.MarketDataProvider,
	wsCounter ConnectionCounter,
) AdminService {
	return &adminService{
		repo:            repo,
		db:              db,
		finnhubProvider: finnhubProvider,
		wsCounter:       wsCounter,
	}
}

// ListUsers returns a paginated list of users.
func (s *adminService) ListUsers(ctx context.Context, page, pageSize int) (*model.AdminUserList, error) {
	users, total, err := s.repo.ListUsers(ctx, page, pageSize)
	if err != nil {
		return nil, model.NewInternal()
	}

	return &model.AdminUserList{
		Users: users,
		Total: total,
	}, nil
}

// UpdateUserRole changes a user's role. Prevents self-role changes.
// Captures before/after state and records to audit log.
func (s *adminService) UpdateUserRole(ctx context.Context, adminID, targetID, newRole string) (*model.AdminUser, error) {
	// Prevent self-role changes
	if adminID == targetID {
		return nil, model.NewConflict("cannot change your own role")
	}

	// Perform the update
	user, err := s.repo.UpdateUserRole(ctx, targetID, newRole)
	if err != nil {
		return nil, s.wrapRepoError(err)
	}

	// Capture before/after for audit log (synchronously, not in middleware)
	afterValue, _ := json.Marshal(map[string]interface{}{"role": newRole})

	// Record the audit event (fire-and-forget; errors are logged)
	auditEntry := model.AuditLogEntry{
		UserID:       adminID,
		Action:       "role_change",
		TargetEntity: "user",
		TargetID:     targetID,
		AfterValue:   afterValue,
		// IPAddress and UserAgent are set by middleware
	}

	_ = s.RecordAuditEvent(ctx, auditEntry)

	return user, nil
}

// ListAuditLog returns a paginated list of audit log entries.
func (s *adminService) ListAuditLog(ctx context.Context, filter repository.AuditLogFilter) (*model.AuditLogList, error) {
	entries, total, err := s.repo.ListAuditLog(ctx, filter)
	if err != nil {
		return nil, model.NewInternal()
	}

	return &model.AuditLogList{
		Entries:  entries,
		Total:    total,
		Page:     filter.Page,
		PageSize: filter.PageSize,
	}, nil
}

// RecordAuditEvent writes an audit log entry. Errors are logged but never returned.
func (s *adminService) RecordAuditEvent(ctx context.Context, entry model.AuditLogEntry) error {
	if err := s.repo.InsertAuditLog(ctx, entry); err != nil {
		// Log the error but don't return it — audit failures must not degrade the primary operation
		slog.Error("failed to record audit event", "error", err, "action", entry.Action)
		return nil // swallow the error
	}
	return nil
}

// GetSystemHealth returns the health status of system components.
// Returns a 200 status even if components are degraded.
func (s *adminService) GetSystemHealth(ctx context.Context) (*model.HealthStatus, error) {
	status := &model.HealthStatus{
		Timestamp: time.Now(),
	}

	// Check database connectivity
	if err := s.db.Ping(ctx); err != nil {
		status.DB = "unhealthy"
		slog.Warn("database health check failed", "error", err)
	} else {
		status.DB = "healthy"
	}

	// Check Finnhub API status (via cached health check on provider)
	if s.finnhubProvider != nil {
		if err := s.finnhubProvider.HealthCheck(ctx); err != nil {
			status.FinnhubAPI = "unhealthy"
			slog.Warn("finnhub health check failed", "error", err)
		} else {
			status.FinnhubAPI = "healthy"
		}
	} else {
		status.FinnhubAPI = "unavailable"
	}

	// Count active WebSocket connections
	if s.wsCounter != nil {
		status.WebSocketCount = s.wsCounter.Count()
	}

	return status, nil
}

// wrapRepoError translates repository errors to service errors.
func (s *adminService) wrapRepoError(err error) error {
	if err == model.ErrNotFound {
		return model.NewNotFound("user")
	}
	return model.NewInternal()
}
