package handler

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/huchknows/fintech/backend/internal/middleware"
	"github.com/huchknows/fintech/backend/internal/model"
	"github.com/huchknows/fintech/backend/internal/service"
)

// PortfolioHandler handles HTTP requests for portfolio resources.
type PortfolioHandler struct {
	svc service.PortfolioService
}

// NewPortfolioHandler returns a PortfolioHandler wired to the given service.
func NewPortfolioHandler(svc service.PortfolioService) *PortfolioHandler {
	return &PortfolioHandler{svc: svc}
}

// RegisterRoutes attaches portfolio endpoints to the given authenticated route group.
func (h *PortfolioHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/portfolios", h.List)
	rg.POST("/portfolios", h.Create)
	rg.GET("/portfolios/:id", h.GetByID)
	rg.PUT("/portfolios/:id", h.Update)
	rg.DELETE("/portfolios/:id", h.Delete)
}

// List godoc
// @Summary     List portfolios
// @Description Returns all portfolios owned by the authenticated user.
// @Tags        portfolios
// @Produce     json
// @Success     200 {array}  model.Portfolio
// @Failure     401 {object} Problem
// @Router      /portfolios [get]
func (h *PortfolioHandler) List(c *gin.Context) {
	userID := c.GetString(string(middleware.ContextKeyUserID))

	ps, err := h.svc.List(c.Request.Context(), userID)
	if err != nil {
		RespondError(c, err)
		return
	}
	c.JSON(http.StatusOK, ps)
}

// Create godoc
// @Summary     Create portfolio
// @Description Creates a new portfolio for the authenticated user.
// @Tags        portfolios
// @Accept      json
// @Produce     json
// @Param       body body model.CreatePortfolioInput true "Portfolio"
// @Success     201 {object} model.Portfolio
// @Failure     400 {object} Problem
// @Failure     401 {object} Problem
// @Failure     422 {object} Problem
// @Router      /portfolios [post]
func (h *PortfolioHandler) Create(c *gin.Context) {
	var in model.CreatePortfolioInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, Problem{
			Status: http.StatusBadRequest,
			Title:  "Bad Request",
			Detail: err.Error(),
		})
		return
	}

	userID := c.GetString(string(middleware.ContextKeyUserID))
	p, err := h.svc.Create(c.Request.Context(), userID, in)
	if err != nil {
		RespondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, p)
}

// GetByID godoc
// @Summary     Get portfolio
// @Description Returns a single portfolio by ID.
// @Tags        portfolios
// @Produce     json
// @Param       id path string true "Portfolio ID"
// @Success     200 {object} model.Portfolio
// @Failure     403 {object} Problem
// @Failure     404 {object} Problem
// @Router      /portfolios/{id} [get]
func (h *PortfolioHandler) GetByID(c *gin.Context) {
	userID := c.GetString(string(middleware.ContextKeyUserID))
	p, err := h.svc.GetByID(c.Request.Context(), userID, c.Param("id"))
	if err != nil {
		RespondError(c, err)
		return
	}
	c.JSON(http.StatusOK, p)
}

// Update godoc
// @Summary     Update portfolio
// @Description Updates the name and description of a portfolio.
// @Tags        portfolios
// @Accept      json
// @Produce     json
// @Param       id   path string                      true "Portfolio ID"
// @Param       body body model.UpdatePortfolioInput  true "Updated fields"
// @Success     200 {object} model.Portfolio
// @Failure     400 {object} Problem
// @Failure     403 {object} Problem
// @Failure     404 {object} Problem
// @Router      /portfolios/{id} [put]
func (h *PortfolioHandler) Update(c *gin.Context) {
	var in model.UpdatePortfolioInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, Problem{
			Status: http.StatusBadRequest,
			Title:  "Bad Request",
			Detail: err.Error(),
		})
		return
	}

	userID := c.GetString(string(middleware.ContextKeyUserID))
	p, err := h.svc.Update(c.Request.Context(), userID, c.Param("id"), in)
	if err != nil {
		RespondError(c, err)
		return
	}
	c.JSON(http.StatusOK, p)
}

// Delete godoc
// @Summary     Delete portfolio
// @Description Deletes a portfolio and all its transactions.
// @Tags        portfolios
// @Produce     json
// @Param       id path string true "Portfolio ID"
// @Success     204
// @Failure     403 {object} Problem
// @Failure     404 {object} Problem
// @Router      /portfolios/{id} [delete]
func (h *PortfolioHandler) Delete(c *gin.Context) {
	userID := c.GetString(string(middleware.ContextKeyUserID))
	if err := h.svc.Delete(c.Request.Context(), userID, c.Param("id")); err != nil {
		RespondError(c, err)
		return
	}
	slog.InfoContext(c.Request.Context(), "portfolio deleted",
		"request_id", middleware.RequestIDFromContext(c),
		"portfolio_id", c.Param("id"),
		"user_id", userID,
	)
	c.Status(http.StatusNoContent)
}
