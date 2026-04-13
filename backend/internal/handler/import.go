package handler

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/huchknows/fintech/backend/internal/middleware"
	"github.com/huchknows/fintech/backend/internal/model"
	"github.com/huchknows/fintech/backend/internal/service"
)

// ImportHandler handles CSV brokerage import operations.
type ImportHandler struct {
	svc service.ImportService
}

// NewImportHandler returns an ImportHandler wired to the given service.
func NewImportHandler(svc service.ImportService) *ImportHandler {
	return &ImportHandler{svc: svc}
}

// RegisterRoutes attaches import endpoints under a portfolio route group.
// Expected to be called on a group already parameterized with :id (portfolio ID).
func (h *ImportHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("/import", h.Preview)
	rg.POST("/import/confirm", h.Confirm)
}

// Preview godoc
// @Summary     Preview CSV import
// @Description Parses and validates a CSV file without persisting transactions. Returns preview of parsed rows and validation errors.
// @Tags        import
// @Accept      multipart/form-data
// @Produce     json
// @Param       id        path   string true "Portfolio ID"
// @Param       file      formData file   true "CSV file"
// @Param       brokerage query  string false "Brokerage (fidelity, sofi, generic). Auto-detected if omitted."
// @Success     200 {object} model.ImportPreview
// @Failure     400 {object} Problem
// @Failure     403 {object} Problem
// @Failure     404 {object} Problem
// @Failure     413 {object} Problem
// @Router      /portfolios/{id}/import [post]
func (h *ImportHandler) Preview(c *gin.Context) {
	userID := c.GetString(string(middleware.ContextKeyUserID))
	portfolioID := c.Param("id")

	// Parse multipart form
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, Problem{
			Status: http.StatusBadRequest,
			Title:  "Bad Request",
			Detail: "missing or invalid 'file' field",
		})
		return
	}

	// Validate file size
	if file.Size > 5*1024*1024 {
		c.JSON(http.StatusRequestEntityTooLarge, Problem{
			Status: http.StatusRequestEntityTooLarge,
			Title:  "Request Entity Too Large",
			Detail: "file exceeds maximum size of 5 MB",
		})
		return
	}

	// Validate file extension (basic check)
	if file.Filename[len(file.Filename)-4:] != ".csv" {
		c.JSON(http.StatusBadRequest, Problem{
			Status: http.StatusBadRequest,
			Title:  "Bad Request",
			Detail: "file must be a CSV (.csv)",
		})
		return
	}

	// Open file
	src, err := file.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, Problem{
			Status: http.StatusBadRequest,
			Title:  "Bad Request",
			Detail: fmt.Sprintf("unable to open file: %v", err),
		})
		return
	}
	defer src.Close()

	// Get brokerage param (optional)
	brokerage := c.DefaultQuery("brokerage", "")

	// Call preview service
	preview, err := h.svc.Preview(c.Request.Context(), userID, portfolioID, src, brokerage)
	if err != nil {
		RespondError(c, err)
		return
	}

	c.JSON(http.StatusOK, preview)
}

// Confirm godoc
// @Summary     Confirm and import transactions
// @Description Persists validated transactions from import preview to the database.
// @Tags        import
// @Accept      json
// @Produce     json
// @Param       id   path string                          true "Portfolio ID"
// @Param       body body model.ImportConfirmRequest true "Transactions to create"
// @Success     200 {object} model.ImportResult
// @Failure     400 {object} Problem
// @Failure     403 {object} Problem
// @Failure     404 {object} Problem
// @Router      /portfolios/{id}/import/confirm [post]
func (h *ImportHandler) Confirm(c *gin.Context) {
	userID := c.GetString(string(middleware.ContextKeyUserID))
	portfolioID := c.Param("id")

	var req model.ImportConfirmRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Problem{
			Status: http.StatusBadRequest,
			Title:  "Bad Request",
			Detail: err.Error(),
		})
		return
	}

	if len(req.Transactions) == 0 {
		c.JSON(http.StatusBadRequest, Problem{
			Status: http.StatusBadRequest,
			Title:  "Bad Request",
			Detail: "transactions list must not be empty",
		})
		return
	}

	// Call confirm service
	result, err := h.svc.Confirm(c.Request.Context(), userID, portfolioID, req)
	if err != nil {
		RespondError(c, err)
		return
	}

	c.JSON(http.StatusOK, result)
}
