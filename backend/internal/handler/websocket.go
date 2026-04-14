package handler

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"github.com/huchknows/fintech/backend/internal/middleware"
	"github.com/huchknows/fintech/backend/internal/model"
	"github.com/huchknows/fintech/backend/internal/provider"
)

// wsUpgrader configures the WebSocket upgrade.
// CheckOrigin delegates to CORS middleware — not repeated here.
var wsUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(_ *http.Request) bool {
		// Origin validation is handled by the CORS middleware already applied.
		// Returning true here allows the upgrade; the CORS middleware has
		// already rejected non-allowed origins before this point.
		return true
	},
}

// wsClientMessage is a message sent from the Angular client to the relay.
type wsClientMessage struct {
	Action  string   `json:"action"`  // "subscribe" | "unsubscribe"
	Symbols []string `json:"symbols"` // ticker symbols
}

// wsServerMessage is a message sent from the relay to Angular clients.
type wsServerMessage struct {
	Type string      `json:"type"` // "tick" | "error"
	Data interface{} `json:"data"`
}

// WebSocketHandler relays real-time PriceTick events from the provider to
// authenticated Angular clients. Each client connection manages its own
// set of subscribed symbols; the handler fans in ticks from the provider
// and fans out only to matching subscribers.
type WebSocketHandler struct {
	provider  provider.MarketDataProvider
	connCount atomic.Int64
}

// NewWebSocketHandler returns a WebSocketHandler backed by the given provider.
func NewWebSocketHandler(p provider.MarketDataProvider) *WebSocketHandler {
	return &WebSocketHandler{provider: p}
}

// RegisterRoutes attaches the WebSocket endpoint to the given authenticated route group.
func (h *WebSocketHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/ws/prices", h.Connect)
}

// Connect upgrades an HTTP connection to WebSocket, authenticates the user,
// and begins streaming price ticks for the client's subscribed symbols.
//
// Protocol (client → server):
//
//	{"action":"subscribe","symbols":["AAPL","MSFT"]}
//	{"action":"unsubscribe","symbols":["AAPL"]}
//
// Protocol (server → client):
//
//	{"type":"tick","data":{...PriceTick}}
//	{"type":"error","data":"message"}
func (h *WebSocketHandler) Connect(c *gin.Context) {
	userID := c.GetString(string(middleware.ContextKeyUserID))
	if userID == "" {
		c.JSON(http.StatusUnauthorized, Problem{
			Status: http.StatusUnauthorized,
			Title:  http.StatusText(http.StatusUnauthorized),
			Detail: "authentication required",
		})
		return
	}

	conn, err := wsUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		slog.WarnContext(c.Request.Context(), "websocket upgrade failed",
			"user_id", userID,
			"error", err,
		)
		return
	}
	defer func() {
		_ = conn.Close()
	}()

	// Track active connections
	h.connCount.Add(1)
	defer h.connCount.Add(-1)

	slog.InfoContext(c.Request.Context(), "websocket connected",
		"user_id", userID,
	)

	// Context that cancels when the WebSocket connection closes.
	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()

	// Subscription management: protect symbols map with a mutex.
	var subMu sync.RWMutex
	subscribed := make(map[string]bool)

	// writeMu serialises writes to the WebSocket connection (gorilla requires this).
	var writeMu sync.Mutex

	sendTick := func(tick model.PriceTick) {
		subMu.RLock()
		active := subscribed[tick.Symbol]
		subMu.RUnlock()
		if !active {
			return
		}
		writeMu.Lock()
		defer writeMu.Unlock()
		if err := conn.WriteJSON(wsServerMessage{Type: "tick", Data: tick}); err != nil {
			slog.WarnContext(ctx, "websocket write error",
				"user_id", userID,
				"error", err,
			)
			cancel() // signal the read loop to stop
		}
	}

	// Start streaming in a background goroutine that runs until ctx is cancelled.
	go func() {
		// StreamPrices blocks until ctx is cancelled or connection drops.
		if err := h.provider.StreamPrices(ctx, nil, sendTick); err != nil {
			if ctx.Err() == nil {
				slog.WarnContext(ctx, "price stream error",
					"user_id", userID,
					"error", err,
				)
			}
		}
		cancel()
	}()

	// Read loop: process subscribe/unsubscribe messages from the client.
	for {
		var msg wsClientMessage
		if err := conn.ReadJSON(&msg); err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway, websocket.CloseNoStatusReceived) {
				slog.InfoContext(ctx, "websocket disconnected",
					"user_id", userID,
				)
			} else if ctx.Err() == nil {
				slog.WarnContext(ctx, "websocket read error",
					"user_id", userID,
					"error", err,
				)
			}
			return
		}

		subMu.Lock()
		for _, sym := range msg.Symbols {
			sym = strings.ToUpper(strings.TrimSpace(sym))
			if !symbolPattern.MatchString(sym) {
				continue
			}
			switch msg.Action {
			case "subscribe":
				subscribed[sym] = true
			case "unsubscribe":
				delete(subscribed, sym)
			}
		}
		subMu.Unlock()
	}
}

// Count returns the number of currently active WebSocket connections.
func (h *WebSocketHandler) Count() int {
	return int(h.connCount.Load())
}
