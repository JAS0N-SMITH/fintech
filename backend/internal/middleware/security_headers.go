package middleware

import "github.com/gin-gonic/gin"

// SecurityHeaders returns a Gin middleware that sets security-related HTTP
// response headers on every response per the project security rules.
//
// Headers applied:
//   - Strict-Transport-Security: enforces HTTPS for 1 year including subdomains
//   - X-Frame-Options: prevents clickjacking by denying framing
//   - X-Content-Type-Options: prevents MIME-type sniffing
//   - Referrer-Policy: limits referrer information to same origin
//   - Permissions-Policy: disables browser APIs not used by the app
//
// Content-Security-Policy is intentionally omitted here — Angular's autoCsp
// feature (angular.json build option) generates per-request nonces and injects
// the CSP header at the Angular build/serve layer for the frontend. The Go API
// serves only JSON; a strict API-level CSP (default-src 'none') is set below.
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("Permissions-Policy", "camera=(), microphone=(), geolocation=(), payment=()")
		// API responses are JSON only — block everything else at the API layer.
		c.Header("Content-Security-Policy", "default-src 'none'")
		c.Next()
	}
}
