package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/huchknows/fintech/backend/internal/middleware"
	"github.com/huchknows/fintech/backend/internal/model"
)

// --- mock watchlist service ---

type mockWatchlistService struct {
	listFn         func(ctx context.Context, userID string) ([]*model.Watchlist, error)
	createFn       func(ctx context.Context, userID string, in model.CreateWatchlistInput) (*model.Watchlist, error)
	getByIDFn      func(ctx context.Context, userID, id string) (*model.Watchlist, error)
	updateFn       func(ctx context.Context, userID, id string, in model.UpdateWatchlistInput) (*model.Watchlist, error)
	deleteFn       func(ctx context.Context, userID, id string) error
	listItemsFn    func(ctx context.Context, userID, id string) ([]*model.WatchlistItem, error)
	addItemFn      func(ctx context.Context, userID, id string, in model.CreateWatchlistItemInput) (*model.WatchlistItem, error)
	updateItemFn   func(ctx context.Context, userID, id, symbol string, in model.UpdateWatchlistItemInput) (*model.WatchlistItem, error)
	removeItemFn   func(ctx context.Context, userID, id, symbol string) error
}

func (m *mockWatchlistService) List(ctx context.Context, userID string) ([]*model.Watchlist, error) {
	if m.listFn != nil {
		return m.listFn(ctx, userID)
	}
	return []*model.Watchlist{}, nil
}

func (m *mockWatchlistService) Create(ctx context.Context, userID string, in model.CreateWatchlistInput) (*model.Watchlist, error) {
	if m.createFn != nil {
		return m.createFn(ctx, userID, in)
	}
	return &model.Watchlist{ID: "wl-new", UserID: userID, Name: in.Name}, nil
}

func (m *mockWatchlistService) GetByID(ctx context.Context, userID, id string) (*model.Watchlist, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, userID, id)
	}
	return &model.Watchlist{ID: id, UserID: userID}, nil
}

func (m *mockWatchlistService) Update(ctx context.Context, userID, id string, in model.UpdateWatchlistInput) (*model.Watchlist, error) {
	if m.updateFn != nil {
		return m.updateFn(ctx, userID, id, in)
	}
	return &model.Watchlist{ID: id, UserID: userID, Name: in.Name}, nil
}

func (m *mockWatchlistService) Delete(ctx context.Context, userID, id string) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, userID, id)
	}
	return nil
}

func (m *mockWatchlistService) ListItems(ctx context.Context, userID, id string) ([]*model.WatchlistItem, error) {
	if m.listItemsFn != nil {
		return m.listItemsFn(ctx, userID, id)
	}
	return []*model.WatchlistItem{}, nil
}

func (m *mockWatchlistService) AddItem(ctx context.Context, userID, id string, in model.CreateWatchlistItemInput) (*model.WatchlistItem, error) {
	if m.addItemFn != nil {
		return m.addItemFn(ctx, userID, id, in)
	}
	return &model.WatchlistItem{ID: "item-new", WatchlistID: id, Symbol: in.Symbol}, nil
}

func (m *mockWatchlistService) UpdateItem(ctx context.Context, userID, id, symbol string, in model.UpdateWatchlistItemInput) (*model.WatchlistItem, error) {
	if m.updateItemFn != nil {
		return m.updateItemFn(ctx, userID, id, symbol, in)
	}
	return &model.WatchlistItem{WatchlistID: id, Symbol: symbol}, nil
}

func (m *mockWatchlistService) RemoveItem(ctx context.Context, userID, id, symbol string) error {
	if m.removeItemFn != nil {
		return m.removeItemFn(ctx, userID, id, symbol)
	}
	return nil
}

// --- test router helper ---

func watchlistRouter(svc *mockWatchlistService, userID string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	r.Use(func(c *gin.Context) {
		c.Set(string(middleware.ContextKeyUserID), userID)
		c.Next()
	})

	h := NewWatchlistHandler(svc)
	h.RegisterRoutes(&r.RouterGroup)
	return r
}

func fixedWatchlist(id, userID, name string) *model.Watchlist {
	return &model.Watchlist{
		ID:        id,
		UserID:    userID,
		Name:      name,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func fixedWatchlistItem(id, watchlistID, symbol string) *model.WatchlistItem {
	return &model.WatchlistItem{
		ID:          id,
		WatchlistID: watchlistID,
		Symbol:      symbol,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

// --- List ---

func TestWatchlistHandler_List(t *testing.T) {
	tests := []struct {
		name       string
		svcReturn  []*model.Watchlist
		svcErr     error
		wantStatus int
		wantLen    int
	}{
		{
			name: "returns watchlists for user",
			svcReturn: []*model.Watchlist{
				fixedWatchlist("wl1", "u1", "Tech"),
				fixedWatchlist("wl2", "u1", "Healthcare"),
			},
			wantStatus: http.StatusOK,
			wantLen:    2,
		},
		{
			name:       "empty list returns empty array",
			svcReturn:  []*model.Watchlist{},
			wantStatus: http.StatusOK,
			wantLen:    0,
		},
		{
			name:       "service error returns 500",
			svcErr:     errors.New("db down"),
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockWatchlistService{
				listFn: func(_ context.Context, _ string) ([]*model.Watchlist, error) {
					return tt.svcReturn, tt.svcErr
				},
			}

			req := httptest.NewRequest(http.MethodGet, "/watchlists", nil)
			w := httptest.NewRecorder()
			watchlistRouter(svc, "u1").ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}
			if tt.svcErr == nil {
				var ws []*model.Watchlist
				if err := json.NewDecoder(w.Body).Decode(&ws); err != nil {
					t.Fatalf("decode: %v", err)
				}
				if len(ws) != tt.wantLen {
					t.Errorf("len = %d, want %d", len(ws), tt.wantLen)
				}
			}
		})
	}
}

// --- Create ---

func TestWatchlistHandler_Create(t *testing.T) {
	tests := []struct {
		name       string
		body       any
		svcReturn  *model.Watchlist
		svcErr     error
		wantStatus int
		wantName   string
	}{
		{
			name:       "valid input creates watchlist",
			body:       map[string]any{"name": "Tech Stocks"},
			svcReturn:  fixedWatchlist("wl1", "u1", "Tech Stocks"),
			wantStatus: http.StatusCreated,
			wantName:   "Tech Stocks",
		},
		{
			name:       "missing name returns 400",
			body:       map[string]any{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "service validation error returns 422",
			body:       map[string]any{"name": "Valid"},
			svcErr:     model.NewValidation("name too long"),
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:       "malformed JSON returns 400",
			body:       "{bad}",
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockWatchlistService{
				createFn: func(_ context.Context, _ string, _ model.CreateWatchlistInput) (*model.Watchlist, error) {
					return tt.svcReturn, tt.svcErr
				},
			}

			var bodyBytes []byte
			if s, ok := tt.body.(string); ok {
				bodyBytes = []byte(s)
			} else {
				bodyBytes, _ = json.Marshal(tt.body)
			}

			req := httptest.NewRequest(http.MethodPost, "/watchlists", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			watchlistRouter(svc, "u1").ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}
			if tt.wantName != "" {
				var wl model.Watchlist
				if err := json.NewDecoder(w.Body).Decode(&wl); err != nil {
					t.Fatalf("decode: %v", err)
				}
				if wl.Name != tt.wantName {
					t.Errorf("name = %q, want %q", wl.Name, tt.wantName)
				}
			}
		})
	}
}

// --- GetByID ---

func TestWatchlistHandler_GetByID(t *testing.T) {
	tests := []struct {
		name        string
		watchlistID string
		svcReturn   *model.Watchlist
		svcErr      error
		wantStatus  int
	}{
		{
			name:        "owner gets watchlist",
			watchlistID: "wl1",
			svcReturn:   fixedWatchlist("wl1", "u1", "Tech"),
			wantStatus:  http.StatusOK,
		},
		{
			name:        "not found returns 404",
			watchlistID: "missing",
			svcErr:      model.NewNotFound("watchlist"),
			wantStatus:  http.StatusNotFound,
		},
		{
			name:        "non-owner returns 403",
			watchlistID: "wl2",
			svcErr:      model.NewForbidden(),
			wantStatus:  http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockWatchlistService{
				getByIDFn: func(_ context.Context, _, _ string) (*model.Watchlist, error) {
					return tt.svcReturn, tt.svcErr
				},
			}

			req := httptest.NewRequest(http.MethodGet, "/watchlists/"+tt.watchlistID, nil)
			w := httptest.NewRecorder()
			watchlistRouter(svc, "u1").ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}
		})
	}
}

// --- Update ---

func TestWatchlistHandler_Update(t *testing.T) {
	tests := []struct {
		name        string
		watchlistID string
		body        any
		svcReturn   *model.Watchlist
		svcErr      error
		wantStatus  int
	}{
		{
			name:        "owner updates watchlist",
			watchlistID: "wl1",
			body:        map[string]any{"name": "Updated"},
			svcReturn:   fixedWatchlist("wl1", "u1", "Updated"),
			wantStatus:  http.StatusOK,
		},
		{
			name:        "missing name returns 400",
			watchlistID: "wl1",
			body:        map[string]any{},
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "not found returns 404",
			watchlistID: "missing",
			body:        map[string]any{"name": "X"},
			svcErr:      model.NewNotFound("watchlist"),
			wantStatus:  http.StatusNotFound,
		},
		{
			name:        "non-owner returns 403",
			watchlistID: "wl2",
			body:        map[string]any{"name": "X"},
			svcErr:      model.NewForbidden(),
			wantStatus:  http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockWatchlistService{
				updateFn: func(_ context.Context, _, _ string, _ model.UpdateWatchlistInput) (*model.Watchlist, error) {
					return tt.svcReturn, tt.svcErr
				},
			}

			bodyBytes, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPut, "/watchlists/"+tt.watchlistID, bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			watchlistRouter(svc, "u1").ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}
		})
	}
}

// --- Delete ---

func TestWatchlistHandler_Delete(t *testing.T) {
	tests := []struct {
		name        string
		watchlistID string
		svcErr      error
		wantStatus  int
	}{
		{
			name:        "owner deletes watchlist",
			watchlistID: "wl1",
			wantStatus:  http.StatusNoContent,
		},
		{
			name:        "not found returns 404",
			watchlistID: "missing",
			svcErr:      model.NewNotFound("watchlist"),
			wantStatus:  http.StatusNotFound,
		},
		{
			name:        "non-owner returns 403",
			watchlistID: "wl2",
			svcErr:      model.NewForbidden(),
			wantStatus:  http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockWatchlistService{
				deleteFn: func(_ context.Context, _, _ string) error {
					return tt.svcErr
				},
			}

			req := httptest.NewRequest(http.MethodDelete, "/watchlists/"+tt.watchlistID, nil)
			w := httptest.NewRecorder()
			watchlistRouter(svc, "u1").ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}
		})
	}
}

// --- ListItems ---

func TestWatchlistHandler_ListItems(t *testing.T) {
	tests := []struct {
		name        string
		watchlistID string
		svcReturn   []*model.WatchlistItem
		svcErr      error
		wantStatus  int
	}{
		{
			name:        "returns items for watchlist",
			watchlistID: "wl1",
			svcReturn: []*model.WatchlistItem{
				fixedWatchlistItem("item1", "wl1", "AAPL"),
				fixedWatchlistItem("item2", "wl1", "MSFT"),
			},
			wantStatus: http.StatusOK,
		},
		{
			name:        "not found returns 404",
			watchlistID: "missing",
			svcErr:      model.NewNotFound("watchlist"),
			wantStatus:  http.StatusNotFound,
		},
		{
			name:        "non-owner returns 403",
			watchlistID: "wl2",
			svcErr:      model.NewForbidden(),
			wantStatus:  http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockWatchlistService{
				listItemsFn: func(_ context.Context, _, _ string) ([]*model.WatchlistItem, error) {
					return tt.svcReturn, tt.svcErr
				},
			}

			req := httptest.NewRequest(http.MethodGet, "/watchlists/"+tt.watchlistID+"/items", nil)
			w := httptest.NewRecorder()
			watchlistRouter(svc, "u1").ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}
		})
	}
}

// --- AddItem ---

func TestWatchlistHandler_AddItem(t *testing.T) {
	tests := []struct {
		name        string
		watchlistID string
		body        any
		svcReturn   *model.WatchlistItem
		svcErr      error
		wantStatus  int
	}{
		{
			name:        "adds item to watchlist",
			watchlistID: "wl1",
			body:        map[string]any{"symbol": "AAPL"},
			svcReturn:   fixedWatchlistItem("item1", "wl1", "AAPL"),
			wantStatus:  http.StatusCreated,
		},
		{
			name:        "missing symbol returns 400",
			watchlistID: "wl1",
			body:        map[string]any{},
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "watchlist not found returns 404",
			watchlistID: "missing",
			body:        map[string]any{"symbol": "AAPL"},
			svcErr:      model.NewNotFound("watchlist"),
			wantStatus:  http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockWatchlistService{
				addItemFn: func(_ context.Context, _, _ string, _ model.CreateWatchlistItemInput) (*model.WatchlistItem, error) {
					return tt.svcReturn, tt.svcErr
				},
			}

			bodyBytes, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPost, "/watchlists/"+tt.watchlistID+"/items", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			watchlistRouter(svc, "u1").ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}
		})
	}
}

// --- UpdateItem ---

func TestWatchlistHandler_UpdateItem(t *testing.T) {
	tests := []struct {
		name        string
		watchlistID string
		symbol      string
		body        any
		svcReturn   *model.WatchlistItem
		svcErr      error
		wantStatus  int
	}{
		{
			name:        "updates item",
			watchlistID: "wl1",
			symbol:      "AAPL",
			body:        map[string]any{"target_price": 150.50, "notes": "Buy on dip"},
			svcReturn:   fixedWatchlistItem("item1", "wl1", "AAPL"),
			wantStatus:  http.StatusOK,
		},
		{
			name:        "malformed JSON returns 400",
			watchlistID: "wl1",
			symbol:      "AAPL",
			body:        "{bad}",
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "item not found returns 404",
			watchlistID: "wl1",
			symbol:      "INVALID",
			body:        map[string]any{"target_price": 100, "notes": ""},
			svcErr:      model.NewNotFound("watchlist item"),
			wantStatus:  http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockWatchlistService{
				updateItemFn: func(_ context.Context, _, _, _ string, _ model.UpdateWatchlistItemInput) (*model.WatchlistItem, error) {
					return tt.svcReturn, tt.svcErr
				},
			}

			var bodyBytes []byte
			if s, ok := tt.body.(string); ok {
				bodyBytes = []byte(s)
			} else {
				bodyBytes, _ = json.Marshal(tt.body)
			}

			req := httptest.NewRequest(http.MethodPut, "/watchlists/"+tt.watchlistID+"/items/"+tt.symbol, bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			watchlistRouter(svc, "u1").ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}
		})
	}
}

// --- RemoveItem ---

func TestWatchlistHandler_RemoveItem(t *testing.T) {
	tests := []struct {
		name        string
		watchlistID string
		symbol      string
		svcErr      error
		wantStatus  int
	}{
		{
			name:        "removes item",
			watchlistID: "wl1",
			symbol:      "AAPL",
			wantStatus:  http.StatusNoContent,
		},
		{
			name:        "item not found returns 404",
			watchlistID: "wl1",
			symbol:      "INVALID",
			svcErr:      model.NewNotFound("watchlist item"),
			wantStatus:  http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockWatchlistService{
				removeItemFn: func(_ context.Context, _, _, _ string) error {
					return tt.svcErr
				},
			}

			req := httptest.NewRequest(http.MethodDelete, "/watchlists/"+tt.watchlistID+"/items/"+tt.symbol, nil)
			w := httptest.NewRecorder()
			watchlistRouter(svc, "u1").ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}
		})
	}
}
