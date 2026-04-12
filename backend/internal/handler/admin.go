package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/huchknows/fintech/backend/internal/middleware"
	"github.com/huchknows/fintech/backend/internal/model"
	"github.com/huchknows/fintech/backend/internal/repository"
	"github.com/huchknows/fintech/backend/internal/service"
)

// AdminHandler handles HTTP requests for admin operations.
type AdminHandler struct {
	svc service.AdminService
}

// NewAdminHandler creates a new AdminHandler with the given service.
func NewAdminHandler(svc service.AdminService) *AdminHandler {
	return &AdminHandler{svc: svc}
}

// RegisterRoutes registers all admin routes on the given router group.
// Note: Audit middleware is applied separately in main.go when wiring the handler.
func (h *AdminHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/users", h.ListUsers)
	rg.PATCH("/users/:id/role", h.PatchRole)
	rg.GET("/audit-log", h.ListAuditLog)
	rg.GET("/health", h.Health)
}

// ListUsers returns a paginated list of users.
// Query params: page (default 1), page_size (default 25).
func (h *AdminHandler) ListUsers(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "25"))
	if pageSize < 1 || pageSize > 100 {
		pageSize = 25
	}

	result, err := h.svc.ListUsers(c.Request.Context(), page, pageSize)
	if err != nil {
		RespondError(c, err)
		return
	}

	c.JSON(http.StatusOK, result)
}

// PatchRole changes a user's role.
// Path param: id (user ID).
// Body: { "role": "user" | "admin" }.
func (h *AdminHandler) PatchRole(c *gin.Context) {
	adminID := c.GetString(string(middleware.ContextKeyUserID))
	targetID := c.Param("id")

	var in model.PatchRoleInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	user, err := h.svc.UpdateUserRole(c.Request.Context(), adminID, targetID, in.Role)
	if err != nil {
		RespondError(c, err)
		return
	}

	c.JSON(http.StatusOK, user)
}

// ListAuditLog returns a paginated list of audit log entries.
// Query params: page (default 1), page_size (default 25).
func (h *AdminHandler) ListAuditLog(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "25"))
	if pageSize < 1 || pageSize > 100 {
		pageSize = 25
	}

	filter := repository.AuditLogFilter{
		Page:     page,
		PageSize: pageSize,
	}

	result, err := h.svc.ListAuditLog(c.Request.Context(), filter)
	if err != nil {
		RespondError(c, err)
		return
	}

	c.JSON(http.StatusOK, result)
}

// Health returns the health status of system components.
func (h *AdminHandler) Health(c *gin.Context) {
	status, err := h.svc.GetSystemHealth(c.Request.Context())
	if err != nil {
		RespondError(c, err)
		return
	}

	c.JSON(http.StatusOK, status)
}
