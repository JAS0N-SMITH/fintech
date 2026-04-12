package middleware

import (
	"github.com/gin-gonic/gin"

	"github.com/huchknows/fintech/backend/internal/model"
)

// AdminService interface defines the method needed for audit logging.
// This allows the middleware to record audit events without importing the full admin service.
type AdminService interface {
	RecordAuditEvent(ctx any, entry model.AuditLogEntry) error
}

// AuditAction returns a Gin middleware that records audit events for successful operations.
// It records the action only if the response status is 2xx.
// If recording fails, the error is logged but never returned (fire-and-forget).
func AuditAction(action, targetEntity string, svc AdminService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Let the handler execute first
		c.Next()

		// Only record on success (2xx responses)
		if c.Writer.Status() < 200 || c.Writer.Status() >= 300 {
			return
		}

		userID := c.GetString(string(ContextKeyUserID))
		targetID := c.Param("id")

		entry := model.AuditLogEntry{
			UserID:       userID,
			Action:       action,
			TargetEntity: targetEntity,
			TargetID:     targetID,
			IPAddress:    c.ClientIP(),
			UserAgent:    c.GetHeader("User-Agent"),
		}

		// Record the event (fire-and-forget; errors are logged internally by the service)
		_ = svc.RecordAuditEvent(c.Request.Context(), entry)
	}
}
