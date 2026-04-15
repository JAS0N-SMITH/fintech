package handler

import (
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/huchknows/fintech/backend/internal/model"
	"github.com/huchknows/fintech/backend/internal/service"
)

// symbolPattern restricts ticker symbols to alphanumeric characters, dots, and hyphens.
// This matches NYSE, NASDAQ, and international exchange formats.
var symbolPattern = regexp.MustCompile(`^[A-Z0-9.\-]{1,20}$`)

// MarketDataHandler handles HTTP requests for market data (quotes and bars).
type MarketDataHandler struct {
	svc service.MarketDataService
}

// NewMarketDataHandler returns a MarketDataHandler wired to the given service.
func NewMarketDataHandler(svc service.MarketDataService) *MarketDataHandler {
	return &MarketDataHandler{svc: svc}
}

// RegisterRoutes attaches market data endpoints to the given authenticated route group.
func (h *MarketDataHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/quotes/:symbol", h.GetQuote)
	rg.GET("/quotes", h.GetQuotesBatch)
	rg.GET("/bars/:symbol", h.GetBars)
	rg.GET("/symbols", h.GetSymbols)
}

// GetQuote godoc
// @Summary     Get quote
// @Description Returns a real-time or recently cached quote for a single symbol.
// @Tags        market-data
// @Produce     json
// @Param       symbol path string true "Ticker symbol (e.g. AAPL)"
// @Success     200 {object} model.Quote
// @Failure     401 {object} Problem
// @Failure     422 {object} Problem
// @Router      /quotes/{symbol} [get]
func (h *MarketDataHandler) GetQuote(c *gin.Context) {
	symbol := strings.ToUpper(c.Param("symbol"))
	if !symbolPattern.MatchString(symbol) {
		RespondError(c, &model.AppError{
			Code:       model.ErrValidation,
			Message:    "symbol must be 1-20 uppercase alphanumeric characters, dots, or hyphens",
			HTTPStatus: http.StatusUnprocessableEntity,
		})
		return
	}

	quote, err := h.svc.GetQuote(c.Request.Context(), symbol)
	if err != nil {
		RespondError(c, err)
		return
	}
	c.JSON(http.StatusOK, quote)
}

// GetQuotesBatch godoc
// @Summary     Get quotes (batch)
// @Description Returns quotes for multiple symbols in a single request.
// @Tags        market-data
// @Produce     json
// @Param       symbols query string true "Comma-separated ticker symbols (e.g. AAPL,MSFT)"
// @Success     200 {object} map[string]model.Quote
// @Failure     400 {object} Problem
// @Failure     401 {object} Problem
// @Router      /quotes [get]
func (h *MarketDataHandler) GetQuotesBatch(c *gin.Context) {
	raw := c.Query("symbols")
	if raw == "" {
		RespondError(c, &model.AppError{
			Code:       model.ErrValidation,
			Message:    "symbols query parameter is required",
			HTTPStatus: http.StatusBadRequest,
		})
		return
	}

	parts := strings.Split(raw, ",")
	if len(parts) > 50 {
		RespondError(c, &model.AppError{
			Code:       model.ErrValidation,
			Message:    "maximum 50 symbols per batch request",
			HTTPStatus: http.StatusUnprocessableEntity,
		})
		return
	}

	results := make(map[string]*model.Quote, len(parts))
	for _, sym := range parts {
		sym = strings.TrimSpace(strings.ToUpper(sym))
		if !symbolPattern.MatchString(sym) {
			continue // skip invalid symbols silently; valid ones are returned
		}
		quote, err := h.svc.GetQuote(c.Request.Context(), sym)
		if err != nil {
			continue // partial results: skip failed symbols
		}
		results[sym] = quote
	}
	c.JSON(http.StatusOK, results)
}

// GetBars godoc
// @Summary     Get historical bars
// @Description Returns OHLCV candle data for a symbol over the requested timeframe.
// @Tags        market-data
// @Produce     json
// @Param       symbol    path  string true  "Ticker symbol (e.g. AAPL)"
// @Param       timeframe query string false "Candle resolution: 1D, 1W, 1M, 3M, 1Y, ALL (default: 1M)"
// @Param       start     query string false "Start date ISO 8601 (default: 30 days ago)"
// @Param       end       query string false "End date ISO 8601 (default: now)"
// @Success     200 {array}  model.Bar
// @Failure     401 {object} Problem
// @Failure     422 {object} Problem
// @Router      /bars/{symbol} [get]
func (h *MarketDataHandler) GetBars(c *gin.Context) {
	symbol := strings.ToUpper(c.Param("symbol"))
	if !symbolPattern.MatchString(symbol) {
		RespondError(c, &model.AppError{
			Code:       model.ErrValidation,
			Message:    "symbol must be 1-20 uppercase alphanumeric characters, dots, or hyphens",
			HTTPStatus: http.StatusUnprocessableEntity,
		})
		return
	}

	// Parse timeframe with default.
	tfStr := strings.ToUpper(c.DefaultQuery("timeframe", string(model.Timeframe1M)))
	tf := model.Timeframe(tfStr)
	if !model.ValidTimeframes[tf] {
		RespondError(c, &model.AppError{
			Code:       model.ErrValidation,
			Message:    "timeframe must be one of: 1D, 1W, 1M, 3M, 1Y, ALL",
			HTTPStatus: http.StatusUnprocessableEntity,
		})
		return
	}

	// Parse date range with defaults.
	now := time.Now().UTC()
	end := now
	start := now.AddDate(0, -1, 0) // 1 month ago

	if s := c.Query("start"); s != "" {
		t, err := time.Parse(time.RFC3339, s)
		if err != nil {
			RespondError(c, &model.AppError{
				Code:       model.ErrValidation,
				Message:    "start must be a valid ISO 8601 date-time",
				HTTPStatus: http.StatusUnprocessableEntity,
			})
			return
		}
		start = t.UTC()
	}
	if e := c.Query("end"); e != "" {
		t, err := time.Parse(time.RFC3339, e)
		if err != nil {
			RespondError(c, &model.AppError{
				Code:       model.ErrValidation,
				Message:    "end must be a valid ISO 8601 date-time",
				HTTPStatus: http.StatusUnprocessableEntity,
			})
			return
		}
		end = t.UTC()
	}

	if !end.After(start) {
		RespondError(c, &model.AppError{
			Code:       model.ErrValidation,
			Message:    "end must be after start",
			HTTPStatus: http.StatusUnprocessableEntity,
		})
		return
	}

	bars, err := h.svc.GetHistoricalBars(c.Request.Context(), symbol, tf, start, end)
	if err != nil {
		RespondError(c, err)
		return
	}
	c.JSON(http.StatusOK, bars)
}

// GetSymbols godoc
// @Summary     Search symbols
// @Description Returns a list of supported stock symbols matching the query.
// @Tags        market-data
// @Produce     json
// @Param       q     query string false "Symbol search query (prefix match on symbol, substring on description)"
// @Param       limit query int    false "Maximum number of results (1-50, default 20)"
// @Success     200   {array}  model.Symbol
// @Failure     401   {object} Problem
// @Failure     422   {object} Problem
// @Router      /symbols [get]
func (h *MarketDataHandler) GetSymbols(c *gin.Context) {
	q := strings.TrimSpace(c.Query("q"))
	limitStr := c.DefaultQuery("limit", "20")

	limit := 20
	if l, err := strconv.Atoi(limitStr); err == nil {
		limit = l
	}

	// Validate limit (service will cap it, but validate here for early feedback)
	if limit < 1 || limit > 50 {
		RespondError(c, &model.AppError{
			Code:       model.ErrValidation,
			Message:    "limit must be between 1 and 50",
			HTTPStatus: http.StatusUnprocessableEntity,
		})
		return
	}

	symbols, err := h.svc.SearchSymbols(c.Request.Context(), q, limit)
	if err != nil {
		RespondError(c, err)
		return
	}

	// Return empty array instead of nil for consistency
	if symbols == nil {
		symbols = []model.Symbol{}
	}
	c.JSON(http.StatusOK, symbols)
}
