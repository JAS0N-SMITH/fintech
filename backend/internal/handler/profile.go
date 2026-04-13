package handler

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/huchknows/fintech/backend/internal/middleware"
	"github.com/huchknows/fintech/backend/internal/model"
	"github.com/huchknows/fintech/backend/internal/service"
)

// ProfileHandler handles HTTP requests for user profile resources.
type ProfileHandler struct {
	svc service.ProfileService
}

// NewProfileHandler returns a ProfileHandler wired to the given service.
func NewProfileHandler(svc service.ProfileService) *ProfileHandler {
	return &ProfileHandler{svc: svc}
}

// RegisterRoutes attaches profile endpoints to the given authenticated route group.
func (h *ProfileHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/me", h.GetMe)
	rg.PATCH("/me/preferences", h.UpdatePreferences)
}

// GetMe godoc
// @Summary     Get user profile
// @Description Returns the authenticated user's profile including preferences.
// @Tags        profile
// @Produce     json
// @Success     200 {object} model.UserProfile
// @Failure     401 {object} Problem
// @Failure     500 {object} Problem
// @Router      /me [get]
func (h *ProfileHandler) GetMe(c *gin.Context) {
	userID := c.GetString(string(middleware.ContextKeyUserID))

	profile, err := h.svc.GetByID(c.Request.Context(), userID)
	if err != nil {
		RespondError(c, err)
		return
	}

	c.JSON(http.StatusOK, profile)
}

// UpdatePreferences godoc
// @Summary     Update user preferences
// @Description Updates the authenticated user's preferences (alert_thresholds, etc.) using JSONB merge.
// @Tags        profile
// @Accept      json
// @Produce     json
// @Param       body body model.UpdatePreferencesInput true "Preferences update"
// @Success     204
// @Failure     400 {object} Problem
// @Failure     401 {object} Problem
// @Failure     500 {object} Problem
// @Router      /me/preferences [patch]
func (h *ProfileHandler) UpdatePreferences(c *gin.Context) {
	var input model.UpdatePreferencesInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, Problem{
			Status: http.StatusBadRequest,
			Title:  "Bad Request",
			Detail: err.Error(),
		})
		return
	}

	userID := c.GetString(string(middleware.ContextKeyUserID))

	// Build the JSONB merge payload: { "alert_thresholds": [...] }
	payload, err := json.Marshal(map[string]json.RawMessage{
		"alert_thresholds": input.AlertThresholds,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, Problem{
			Status: http.StatusBadRequest,
			Title:  "Bad Request",
			Detail: "Failed to marshal preferences",
		})
		return
	}

	if err := h.svc.UpdatePreferences(c.Request.Context(), userID, payload); err != nil {
		RespondError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}
