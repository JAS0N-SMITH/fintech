package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/huchknows/fintech/backend/internal/model"
)

// AuditLogFilter specifies criteria for filtering audit log entries.
type AuditLogFilter struct {
	UserID   string
	Action   string
	From     time.Time
	To       time.Time
	Page     int
	PageSize int
}

// AdminRepository defines the data access interface for admin operations.
type AdminRepository interface {
	ListUsers(ctx context.Context, page, pageSize int) ([]model.AdminUser, int, error)
	UpdateUserRole(ctx context.Context, id, role string) (*model.AdminUser, error)
	InsertAuditLog(ctx context.Context, entry model.AuditLogEntry) error
	ListAuditLog(ctx context.Context, filter AuditLogFilter) ([]model.AuditLogEntry, int, error)
}

// adminRepo is the pgx-backed implementation.
type adminRepo struct {
	db *pgxpool.Pool
}

// NewAdminRepository returns an AdminRepository backed by the given pool.
func NewAdminRepository(db *pgxpool.Pool) AdminRepository {
	return &adminRepo{db: db}
}

// ListUsers returns a paginated list of users joined with their auth email.
func (r *adminRepo) ListUsers(ctx context.Context, page, pageSize int) ([]model.AdminUser, int, error) {
	offset := (page - 1) * pageSize

	// Fetch count
	var count int
	countQ := `SELECT COUNT(*) FROM public.profiles`
	if err := r.db.QueryRow(ctx, countQ).Scan(&count); err != nil {
		return nil, 0, err
	}

	// Fetch paginated users
	q := `
		SELECT
			p.id,
			COALESCE(u.email, ''),
			COALESCE(p.display_name, ''),
			p.role,
			p.created_at,
			p.updated_at
		FROM public.profiles p
		LEFT JOIN auth.users u ON u.id = p.id
		ORDER BY p.created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.db.Query(ctx, q, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var users []model.AdminUser
	for rows.Next() {
		var u model.AdminUser
		if err := rows.Scan(&u.ID, &u.Email, &u.DisplayName, &u.Role, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, 0, err
		}
		users = append(users, u)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return users, count, nil
}

// UpdateUserRole updates a user's role and returns the updated record.
func (r *adminRepo) UpdateUserRole(ctx context.Context, id, role string) (*model.AdminUser, error) {
	q := `
		UPDATE public.profiles
		SET role = $1, updated_at = now()
		WHERE id = $2
		RETURNING id, COALESCE(display_name, ''), role, created_at, updated_at
	`

	user := &model.AdminUser{}
	err := r.db.QueryRow(ctx, q, role, id).
		Scan(&user.ID, &user.DisplayName, &user.Role, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrNotFound
		}
		return nil, err
	}

	// Fetch email from auth.users
	emailQ := `SELECT email FROM auth.users WHERE id = $1`
	_ = r.db.QueryRow(ctx, emailQ, id).Scan(&user.Email)

	return user, nil
}

// InsertAuditLog records an audit event. This table is append-only.
func (r *adminRepo) InsertAuditLog(ctx context.Context, entry model.AuditLogEntry) error {
	q := `
		INSERT INTO public.audit_log
		(user_id, action, target_entity, target_id, before_value, after_value, ip_address, user_agent)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err := r.db.Exec(
		ctx, q,
		entry.UserID,
		entry.Action,
		entry.TargetEntity,
		entry.TargetID,
		entry.BeforeValue,
		entry.AfterValue,
		entry.IPAddress,
		entry.UserAgent,
	)
	return err
}

// ListAuditLog returns a paginated list of audit log entries with optional filtering.
func (r *adminRepo) ListAuditLog(ctx context.Context, filter AuditLogFilter) ([]model.AuditLogEntry, int, error) {
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PageSize < 1 {
		filter.PageSize = 25
	}
	offset := (filter.Page - 1) * filter.PageSize

	// Build WHERE clause dynamically
	whereClause := ""
	args := []interface{}{}
	argIndex := 1

	if filter.UserID != "" {
		whereClause += fmt.Sprintf("AND user_id = $%d ", argIndex)
		args = append(args, filter.UserID)
		argIndex++
	}

	if filter.Action != "" {
		whereClause += fmt.Sprintf("AND action = $%d ", argIndex)
		args = append(args, filter.Action)
		argIndex++
	}

	if !filter.From.IsZero() {
		whereClause += fmt.Sprintf("AND created_at >= $%d ", argIndex)
		args = append(args, filter.From)
		argIndex++
	}

	if !filter.To.IsZero() {
		whereClause += fmt.Sprintf("AND created_at <= $%d ", argIndex)
		args = append(args, filter.To)
		argIndex++
	}

	// Count with filters
	countQ := fmt.Sprintf(`SELECT COUNT(*) FROM public.audit_log WHERE 1=1 %s`, whereClause)
	var count int
	countArgs := args
	if err := r.db.QueryRow(ctx, countQ, countArgs...).Scan(&count); err != nil {
		return nil, 0, err
	}

	// Fetch paginated results
	q := fmt.Sprintf(`
		SELECT
			id,
			user_id,
			action,
			target_entity,
			target_id,
			COALESCE(before_value, 'null'::jsonb),
			COALESCE(after_value, 'null'::jsonb),
			COALESCE(ip_address, ''),
			COALESCE(user_agent, ''),
			created_at
		FROM public.audit_log
		WHERE 1=1 %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIndex, argIndex+1)

	// Append pagination args
	args = append(args, filter.PageSize, offset)

	rows, err := r.db.Query(ctx, q, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var entries []model.AuditLogEntry
	for rows.Next() {
		var e model.AuditLogEntry
		if err := rows.Scan(
			&e.ID,
			&e.UserID,
			&e.Action,
			&e.TargetEntity,
			&e.TargetID,
			&e.BeforeValue,
			&e.AfterValue,
			&e.IPAddress,
			&e.UserAgent,
			&e.CreatedAt,
		); err != nil {
			return nil, 0, err
		}
		entries = append(entries, e)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return entries, count, nil
}
