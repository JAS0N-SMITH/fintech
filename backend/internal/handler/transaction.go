package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/huchknows/fintech/backend/internal/middleware"
	"github.com/huchknows/fintech/backend/internal/model"
	"github.com/huchknows/fintech/backend/internal/service"
)

// TransactionHandler handles HTTP requests for transaction resources.
type TransactionHandler struct {
	svc service.TransactionService
}

// NewTransactionHandler returns a TransactionHandler wired to the given service.
func NewTransactionHandler(svc service.TransactionService) *TransactionHandler {
	return &TransactionHandler{svc: svc}
}

// RegisterRoutes attaches transaction endpoints under a portfolio route group.
func (h *TransactionHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/portfolios/:portfolioID/transactions", h.List)
	rg.POST("/portfolios/:portfolioID/transactions", h.Create)
	rg.DELETE("/portfolios/:portfolioID/transactions/:id", h.Delete)
}

// List godoc
// @Summary     List transactions
// @Description Returns all transactions for a portfolio.
// @Tags        transactions
// @Produce     json
// @Param       portfolioID path string true "Portfolio ID"
// @Success     200 {array}  model.Transaction
// @Failure     403 {object} Problem
// @Failure     404 {object} Problem
// @Router      /portfolios/{portfolioID}/transactions [get]
func (h *TransactionHandler) List(c *gin.Context) {
	userID := c.GetString(string(middleware.ContextKeyUserID))
	txns, err := h.svc.List(c.Request.Context(), userID, c.Param("portfolioID"))
	if err != nil {
		RespondError(c, err)
		return
	}
	c.JSON(http.StatusOK, txns)
}

// Create godoc
// @Summary     Record transaction
// @Description Records a new financial transaction (buy, sell, dividend, reinvested_dividend).
// @Tags        transactions
// @Accept      json
// @Produce     json
// @Param       portfolioID path string                       true "Portfolio ID"
// @Param       body        body model.CreateTransactionInput true "Transaction"
// @Success     201 {object} model.Transaction
// @Failure     400 {object} Problem
// @Failure     403 {object} Problem
// @Failure     404 {object} Problem
// @Failure     409 {object} Problem
// @Failure     422 {object} Problem
// @Router      /portfolios/{portfolioID}/transactions [post]
func (h *TransactionHandler) Create(c *gin.Context) {
	var in model.CreateTransactionInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, Problem{
			Status: http.StatusBadRequest,
			Title:  "Bad Request",
			Detail: err.Error(),
		})
		return
	}

	userID := c.GetString(string(middleware.ContextKeyUserID))
	txn, err := h.svc.Create(c.Request.Context(), userID, c.Param("portfolioID"), in)
	if err != nil {
		RespondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, txn)
}

// Delete godoc
// @Summary     Delete transaction
// @Description Removes a transaction from a portfolio.
// @Tags        transactions
// @Produce     json
// @Param       portfolioID path string true "Portfolio ID"
// @Param       id          path string true "Transaction ID"
// @Success     204
// @Failure     403 {object} Problem
// @Failure     404 {object} Problem
// @Router      /portfolios/{portfolioID}/transactions/{id} [delete]
func (h *TransactionHandler) Delete(c *gin.Context) {
	userID := c.GetString(string(middleware.ContextKeyUserID))
	if err := h.svc.Delete(c.Request.Context(), userID, c.Param("id")); err != nil {
		RespondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
