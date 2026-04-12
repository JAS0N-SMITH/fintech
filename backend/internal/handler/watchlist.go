package handler

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/huchknows/fintech/backend/internal/middleware"
	"github.com/huchknows/fintech/backend/internal/model"
	"github.com/huchknows/fintech/backend/internal/service"
)

// WatchlistHandler handles HTTP requests for watchlist resources.
type WatchlistHandler struct {
	svc service.WatchlistService
}

// NewWatchlistHandler returns a WatchlistHandler wired to the given service.
func NewWatchlistHandler(svc service.WatchlistService) *WatchlistHandler {
	return &WatchlistHandler{svc: svc}
}

// RegisterRoutes attaches watchlist endpoints to the given authenticated route group.
func (h *WatchlistHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/watchlists", h.List)
	rg.POST("/watchlists", h.Create)
	rg.GET("/watchlists/:id", h.GetByID)
	rg.PUT("/watchlists/:id", h.Update)
	rg.DELETE("/watchlists/:id", h.Delete)

	// Watchlist items
	rg.GET("/watchlists/:id/items", h.ListItems)
	rg.POST("/watchlists/:id/items", h.AddItem)
	rg.PUT("/watchlists/:id/items/:symbol", h.UpdateItem)
	rg.DELETE("/watchlists/:id/items/:symbol", h.RemoveItem)
}

// List godoc
// @Summary     List watchlists
// @Description Returns all watchlists owned by the authenticated user.
// @Tags        watchlists
// @Produce     json
// @Success     200 {array}  model.Watchlist
// @Failure     401 {object} Problem
// @Router      /watchlists [get]
func (h *WatchlistHandler) List(c *gin.Context) {
	userID := c.GetString(string(middleware.ContextKeyUserID))

	ws, err := h.svc.List(c.Request.Context(), userID)
	if err != nil {
		RespondError(c, err)
		return
	}
	c.JSON(http.StatusOK, ws)
}

// Create godoc
// @Summary     Create watchlist
// @Description Creates a new watchlist for the authenticated user.
// @Tags        watchlists
// @Accept      json
// @Produce     json
// @Param       body body model.CreateWatchlistInput true "Watchlist"
// @Success     201 {object} model.Watchlist
// @Failure     400 {object} Problem
// @Failure     401 {object} Problem
// @Failure     422 {object} Problem
// @Router      /watchlists [post]
func (h *WatchlistHandler) Create(c *gin.Context) {
	var in model.CreateWatchlistInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, Problem{
			Status: http.StatusBadRequest,
			Title:  "Bad Request",
			Detail: err.Error(),
		})
		return
	}

	userID := c.GetString(string(middleware.ContextKeyUserID))
	w, err := h.svc.Create(c.Request.Context(), userID, in)
	if err != nil {
		RespondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, w)
}

// GetByID godoc
// @Summary     Get watchlist
// @Description Returns a single watchlist by ID.
// @Tags        watchlists
// @Produce     json
// @Param       id path string true "Watchlist ID"
// @Success     200 {object} model.Watchlist
// @Failure     403 {object} Problem
// @Failure     404 {object} Problem
// @Router      /watchlists/{id} [get]
func (h *WatchlistHandler) GetByID(c *gin.Context) {
	userID := c.GetString(string(middleware.ContextKeyUserID))
	w, err := h.svc.GetByID(c.Request.Context(), userID, c.Param("id"))
	if err != nil {
		RespondError(c, err)
		return
	}
	c.JSON(http.StatusOK, w)
}

// Update godoc
// @Summary     Update watchlist
// @Description Updates the name of a watchlist.
// @Tags        watchlists
// @Accept      json
// @Produce     json
// @Param       id   path string                      true "Watchlist ID"
// @Param       body body model.UpdateWatchlistInput  true "Updated fields"
// @Success     200 {object} model.Watchlist
// @Failure     400 {object} Problem
// @Failure     403 {object} Problem
// @Failure     404 {object} Problem
// @Router      /watchlists/{id} [put]
func (h *WatchlistHandler) Update(c *gin.Context) {
	var in model.UpdateWatchlistInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, Problem{
			Status: http.StatusBadRequest,
			Title:  "Bad Request",
			Detail: err.Error(),
		})
		return
	}

	userID := c.GetString(string(middleware.ContextKeyUserID))
	w, err := h.svc.Update(c.Request.Context(), userID, c.Param("id"), in)
	if err != nil {
		RespondError(c, err)
		return
	}
	c.JSON(http.StatusOK, w)
}

// Delete godoc
// @Summary     Delete watchlist
// @Description Deletes a watchlist.
// @Tags        watchlists
// @Produce     json
// @Param       id path string true "Watchlist ID"
// @Success     204
// @Failure     403 {object} Problem
// @Failure     404 {object} Problem
// @Router      /watchlists/{id} [delete]
func (h *WatchlistHandler) Delete(c *gin.Context) {
	userID := c.GetString(string(middleware.ContextKeyUserID))
	if err := h.svc.Delete(c.Request.Context(), userID, c.Param("id")); err != nil {
		RespondError(c, err)
		return
	}
	slog.InfoContext(c.Request.Context(), "watchlist deleted",
		"request_id", middleware.RequestIDFromContext(c),
		"watchlist_id", c.Param("id"),
		"user_id", userID,
	)
	c.Status(http.StatusNoContent)
}

// ListItems godoc
// @Summary     List watchlist items
// @Description Returns all items in a watchlist.
// @Tags        watchlist-items
// @Produce     json
// @Param       id path string true "Watchlist ID"
// @Success     200 {array}  model.WatchlistItem
// @Failure     403 {object} Problem
// @Failure     404 {object} Problem
// @Router      /watchlists/{id}/items [get]
func (h *WatchlistHandler) ListItems(c *gin.Context) {
	userID := c.GetString(string(middleware.ContextKeyUserID))
	items, err := h.svc.ListItems(c.Request.Context(), userID, c.Param("id"))
	if err != nil {
		RespondError(c, err)
		return
	}
	c.JSON(http.StatusOK, items)
}

// AddItem godoc
// @Summary     Add item to watchlist
// @Description Adds a ticker symbol to a watchlist.
// @Tags        watchlist-items
// @Accept      json
// @Produce     json
// @Param       id   path string                              true "Watchlist ID"
// @Param       body body model.CreateWatchlistItemInput      true "Item"
// @Success     201 {object} model.WatchlistItem
// @Failure     400 {object} Problem
// @Failure     403 {object} Problem
// @Failure     404 {object} Problem
// @Failure     422 {object} Problem
// @Router      /watchlists/{id}/items [post]
func (h *WatchlistHandler) AddItem(c *gin.Context) {
	var in model.CreateWatchlistItemInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, Problem{
			Status: http.StatusBadRequest,
			Title:  "Bad Request",
			Detail: err.Error(),
		})
		return
	}

	userID := c.GetString(string(middleware.ContextKeyUserID))
	item, err := h.svc.AddItem(c.Request.Context(), userID, c.Param("id"), in)
	if err != nil {
		RespondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, item)
}

// UpdateItem godoc
// @Summary     Update watchlist item
// @Description Updates the target price and notes of a watchlist item.
// @Tags        watchlist-items
// @Accept      json
// @Produce     json
// @Param       id     path string                              true "Watchlist ID"
// @Param       symbol path string                              true "Symbol"
// @Param       body   body model.UpdateWatchlistItemInput      true "Updated fields"
// @Success     200 {object} model.WatchlistItem
// @Failure     400 {object} Problem
// @Failure     403 {object} Problem
// @Failure     404 {object} Problem
// @Router      /watchlists/{id}/items/{symbol} [put]
func (h *WatchlistHandler) UpdateItem(c *gin.Context) {
	var in model.UpdateWatchlistItemInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, Problem{
			Status: http.StatusBadRequest,
			Title:  "Bad Request",
			Detail: err.Error(),
		})
		return
	}

	userID := c.GetString(string(middleware.ContextKeyUserID))
	item, err := h.svc.UpdateItem(c.Request.Context(), userID, c.Param("id"), c.Param("symbol"), in)
	if err != nil {
		RespondError(c, err)
		return
	}
	c.JSON(http.StatusOK, item)
}

// RemoveItem godoc
// @Summary     Remove item from watchlist
// @Description Removes a ticker symbol from a watchlist.
// @Tags        watchlist-items
// @Produce     json
// @Param       id     path string true "Watchlist ID"
// @Param       symbol path string true "Symbol"
// @Success     204
// @Failure     403 {object} Problem
// @Failure     404 {object} Problem
// @Router      /watchlists/{id}/items/{symbol} [delete]
func (h *WatchlistHandler) RemoveItem(c *gin.Context) {
	userID := c.GetString(string(middleware.ContextKeyUserID))
	if err := h.svc.RemoveItem(c.Request.Context(), userID, c.Param("id"), c.Param("symbol")); err != nil {
		RespondError(c, err)
		return
	}
	slog.InfoContext(c.Request.Context(), "watchlist item removed",
		"request_id", middleware.RequestIDFromContext(c),
		"watchlist_id", c.Param("id"),
		"symbol", c.Param("symbol"),
		"user_id", userID,
	)
	c.Status(http.StatusNoContent)
}
