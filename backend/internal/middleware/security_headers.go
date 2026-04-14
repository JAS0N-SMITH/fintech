package middleware

import "github.com/gin-gonic/gin"

const (
	strictAPICSP   = "default-src 'none'"
	swaggerDocsCSP = "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; font-src 'self' data:"
)

func setCommonSecurityHeaders(c *gin.Context) {
	c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
	c.Header("X-Frame-Options", "DENY")
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
	c.Header("Permissions-Policy", "camera=(), microphone=(), geolocation=(), payment=()")
	c.Header("Cross-Origin-Opener-Policy", "same-origin")
	c.Header("Cross-Origin-Resource-Policy", "same-origin")
	c.Header("Cross-Origin-Embedder-Policy", "require-corp")
	c.Header("X-Permitted-Cross-Domain-Policies", "none")
}

// SecurityHeaders returns a Gin middleware that sets security-related HTTP
// response headers on every response per the project security rules.
//
// Headers applied:
//   - Strict-Transport-Security: enforces HTTPS for 1 year including subdomains, preload for HSTS Preload List
//   - X-Frame-Options: prevents clickjacking by denying framing
//   - X-Content-Type-Options: prevents MIME-type sniffing
//   - Referrer-Policy: limits referrer information to same origin
//   - Permissions-Policy: disables browser APIs not used by the app
//   - Cross-Origin-Opener-Policy: isolates the window context
//   - Cross-Origin-Resource-Policy: prevents cross-origin fetching
//   - Cross-Origin-Embedder-Policy: enables cross-origin resource sharing in same-origin fashion
//   - X-Permitted-Cross-Domain-Policies: disables Adobe cross-domain policies
//
// Content-Security-Policy is intentionally omitted here — Angular's autoCsp
// feature (angular.json build option) generates per-request nonces and injects
// the CSP header at the Angular build/serve layer for the frontend. The Go API
// serves only JSON; a strict API-level CSP (default-src 'none') is set below.
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		setCommonSecurityHeaders(c)
		// API responses are JSON only — block everything else at the API layer.
		c.Header("Content-Security-Policy", strictAPICSP)
		c.Next()
	}
}

// DocsSecurityHeaders returns a Gin middleware that sets security headers for
// Swagger documentation assets served by the API process.
func DocsSecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		setCommonSecurityHeaders(c)
		// Swagger UI requires same-origin scripts/styles and a small inline style.
		c.Header("Content-Security-Policy", swaggerDocsCSP)
		c.Next()
	}
}
