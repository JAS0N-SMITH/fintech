package model

import (
	"encoding/json"
	"time"
)

// AdminUser represents a user from the admin perspective, joined from profiles and auth.users.
type AdminUser struct {
	ID          string    `json:"id"`
	Email       string    `json:"email"`
	DisplayName string    `json:"display_name"`
	Role        string    `json:"role"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// AuditLogEntry represents a security-relevant event recorded to the audit log.
type AuditLogEntry struct {
	ID           string          `json:"id"`
	UserID       string          `json:"user_id"`
	Action       string          `json:"action"`
	TargetEntity string          `json:"target_entity"`
	TargetID     string          `json:"target_id"`
	BeforeValue  json.RawMessage `json:"before_value"`
	AfterValue   json.RawMessage `json:"after_value"`
	IPAddress    string          `json:"ip_address"`
	UserAgent    string          `json:"user_agent"`
	CreatedAt    time.Time       `json:"created_at"`
}

// PatchRoleInput is the request body for changing a user's role.
type PatchRoleInput struct {
	Role string `json:"role" binding:"required,oneof=user admin"`
}

// AdminUserList wraps a paginated list of admin users.
type AdminUserList struct {
	Users []AdminUser `json:"users"`
	Total int         `json:"total"`
}

// AuditLogList wraps a paginated list of audit log entries.
type AuditLogList struct {
	Entries  []AuditLogEntry `json:"entries"`
	Total    int             `json:"total"`
	Page     int             `json:"page"`
	PageSize int             `json:"page_size"`
}

// HealthStatus represents the health of system components.
type HealthStatus struct {
	DB             string    `json:"db"`
	FinnhubAPI     string    `json:"finnhub_api"`
	WebSocketCount int       `json:"websocket_count"`
	Timestamp      time.Time `json:"timestamp"`
}
