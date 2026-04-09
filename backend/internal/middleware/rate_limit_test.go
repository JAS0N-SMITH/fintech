package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

func TestRateLimitByIP(t *testing.T) {
	gin.SetMode(gin.TestMode)
	// 1 request per second, burst of 1 — second request must be rejected.
	r := gin.New()
	r.Use(RateLimitByIP(rate.Limit(1), 1))
	r.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

	t.Run("first request allowed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "1.2.3.4:0"
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("status = %d, want 200", w.Code)
		}
	})

	t.Run("second request from same IP immediately rejected", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "1.2.3.4:0"
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusTooManyRequests {
			t.Errorf("status = %d, want 429", w.Code)
		}
	})

	t.Run("different IP not affected", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "5.6.7.8:0"
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("status = %d, want 200", w.Code)
		}
	})
}

func TestRateLimitByUser(t *testing.T) {
	gin.SetMode(gin.TestMode)

	newRouter := func() *gin.Engine {
		r := gin.New()
		r.Use(RateLimitByUser(rate.Limit(1), 1))
		r.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })
		return r
	}

	t.Run("first request allowed", func(t *testing.T) {
		r := newRouter()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()
		// Simulate auth middleware having set user_id
		r.Use(func(c *gin.Context) { c.Set(string(ContextKeyUserID), "user-abc") })
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("status = %d, want 200", w.Code)
		}
	})

	t.Run("falls back to IP when no user ID in context", func(t *testing.T) {
		r := newRouter()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "9.9.9.9:0"
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("status = %d, want 200", w.Code)
		}
	})
}
