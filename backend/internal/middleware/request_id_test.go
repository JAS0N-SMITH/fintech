package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRequestID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := func(c *gin.Context) {
		c.String(http.StatusOK, RequestIDFromContext(c))
	}

	t.Run("generates ID when none provided", func(t *testing.T) {
		r := gin.New()
		r.Use(RequestID())
		r.GET("/", handler)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		id := w.Body.String()
		if id == "" || id == "unknown" {
			t.Errorf("expected a generated request ID, got %q", id)
		}
		if w.Header().Get(requestIDHeader) != id {
			t.Errorf("response header %s = %q, want %q", requestIDHeader, w.Header().Get(requestIDHeader), id)
		}
	})

	t.Run("propagates existing ID from request header", func(t *testing.T) {
		r := gin.New()
		r.Use(RequestID())
		r.GET("/", handler)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set(requestIDHeader, "my-trace-id")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if got := w.Body.String(); got != "my-trace-id" {
			t.Errorf("body = %q, want %q", got, "my-trace-id")
		}
		if got := w.Header().Get(requestIDHeader); got != "my-trace-id" {
			t.Errorf("response header = %q, want %q", got, "my-trace-id")
		}
	})

	t.Run("each request gets a unique ID", func(t *testing.T) {
		r := gin.New()
		r.Use(RequestID())
		r.GET("/", handler)

		ids := make(map[string]struct{})
		for range 10 {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			ids[w.Body.String()] = struct{}{}
		}
		if len(ids) != 10 {
			t.Errorf("expected 10 unique IDs, got %d", len(ids))
		}
	})
}
