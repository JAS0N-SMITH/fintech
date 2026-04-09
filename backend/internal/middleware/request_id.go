package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"

	"github.com/gin-gonic/gin"
)

const requestIDHeader = "X-Request-ID"

// RequestID generates a unique ID for each request and stores it in the Gin
// context and response header. Downstream handlers and logging middleware
// read it via RequestIDFromContext.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader(requestIDHeader)
		if id == "" {
			id = newRequestID()
		}
		c.Set(requestIDHeader, id)
		c.Header(requestIDHeader, id)
		c.Next()
	}
}

// RequestIDFromContext returns the request ID stored by the RequestID middleware.
// Returns an empty string if not present.
func RequestIDFromContext(c *gin.Context) string {
	id, _ := c.Get(requestIDHeader)
	if s, ok := id.(string); ok {
		return s
	}
	return ""
}

// newRequestID generates a 16-byte random hex string.
func newRequestID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "unknown"
	}
	return hex.EncodeToString(b)
}

// statusRecorder wraps gin.ResponseWriter to capture the status code after
// the handler writes it, for use in logging middleware.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}
