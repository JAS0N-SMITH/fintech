package middleware

import (
	"log/slog"
	"net/url"
	"time"

	"github.com/gin-gonic/gin"
)

// Logger returns a Gin middleware that emits a structured slog entry for every
// request. It replaces gin.Logger() to keep log output consistent with the
// application's JSON slog handler.
//
// Each log entry includes: method, path, status, latency, client IP, and the
// request ID set by the RequestID middleware.
// Sensitive query parameters (e.g., ?token=) are stripped before logging.
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		if c.Request.URL.RawQuery != "" {
			// Strip sensitive query parameters (e.g., ?token=<jwt>)
			q := c.Request.URL.Query()
			q.Del("token")
			if len(q) > 0 {
				path += "?" + q.Encode()
			}
		}

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()
		requestID := RequestIDFromContext(c)

		level := slog.LevelInfo
		if status >= 500 {
			level = slog.LevelError
		} else if status >= 400 {
			level = slog.LevelWarn
		}

		slog.LogAttrs(c.Request.Context(), level, "request",
			slog.String("request_id", requestID),
			slog.String("method", c.Request.Method),
			slog.String("path", path),
			slog.Int("status", status),
			slog.Duration("latency", latency),
			slog.String("ip", c.ClientIP()),
		)
	}
}
