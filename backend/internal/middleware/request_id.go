package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"regexp"

	"github.com/gin-gonic/gin"
)

const requestIDHeader = "X-Request-ID"

var requestIDPattern = regexp.MustCompile(`^[a-zA-Z0-9\-]{1,64}$`)

// RequestID generates a unique ID for each request and stores it in the Gin
// context and response header. Downstream handlers and logging middleware
// read it via RequestIDFromContext.
// If a client provides an X-Request-ID header, it is validated; invalid values
// are rejected and a new ID is generated (prevents log injection).
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader(requestIDHeader)
		// Validate client-supplied ID; reject if invalid to prevent log injection
		if id == "" || !requestIDPattern.MatchString(id) {
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
