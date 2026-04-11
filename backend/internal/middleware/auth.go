// Package middleware provides HTTP middleware for the Gin router.
package middleware

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// contextKey is an unexported type for context keys in this package.
type contextKey string

const (
	// ContextKeyUserID is the context key for the authenticated user's UUID.
	ContextKeyUserID contextKey = "user_id"
	// ContextKeyUserRole is the context key for the authenticated user's app role.
	ContextKeyUserRole contextKey = "user_role"
)

// SupabaseClaims represents the JWT claims issued by Supabase Auth.
type SupabaseClaims struct {
	jwt.RegisteredClaims
	Email       string                 `json:"email"`
	Role        string                 `json:"role"` // Supabase DB role ("authenticated"), not app role
	AppMetadata map[string]interface{} `json:"app_metadata"`
}

// appRole extracts the application-level role from app_metadata.
// Falls back to "user" if unset. app_metadata is admin-controlled and safe for RBAC.
func (c *SupabaseClaims) appRole() string {
	if c.AppMetadata == nil {
		return "user"
	}
	if role, ok := c.AppMetadata["role"].(string); ok && role != "" {
		return role
	}
	return "user"
}

// jwksCache holds a fetched EC public key and the time it was last refreshed.
type jwksCache struct {
	mu         sync.RWMutex
	key        *ecdsa.PublicKey
	lastFetch  time.Time
	refreshTTL time.Duration
}

// jwksKey represents a single entry from a JWKS response.
type jwksKey struct {
	Kty string `json:"kty"`
	Alg string `json:"alg"`
	Crv string `json:"crv"`
	X   string `json:"x"`
	Y   string `json:"y"`
}

// RequireAuth returns a Gin middleware that validates Supabase-issued ES256 JWTs.
// The JWKS endpoint is fetched from the Supabase project URL and cached for 1 hour.
// On success, user_id and user_role are stored in the Gin context.
func RequireAuth(supabaseURL string) gin.HandlerFunc {
	cache := &jwksCache{refreshTTL: time.Hour}

	return func(c *gin.Context) {
		token, err := extractToken(c)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing or invalid token"})
			return
		}

		key, err := cache.getKey(c.Request.Context(), supabaseURL)
		if err != nil {
			slog.ErrorContext(c.Request.Context(), "failed to fetch JWKS", "error", err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "authentication unavailable"})
			return
		}

		claims := &SupabaseClaims{}
		parsed, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodECDSA); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return key, nil
		}, jwt.WithExpirationRequired())

		if err != nil || !parsed.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			return
		}

		userID := claims.Subject
		if userID == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "token missing subject claim"})
			return
		}

		c.Set(string(ContextKeyUserID), userID)
		c.Set(string(ContextKeyUserRole), claims.appRole())
		c.Next()
	}
}

// RequireRole returns a Gin middleware that enforces a minimum app role.
// Must be used after RequireAuth which sets the role in context.
func RequireRole(role string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole, exists := c.Get(string(ContextKeyUserRole))
		if !exists || userRole.(string) != role {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
			return
		}
		c.Next()
	}
}

// UserIDFromContext retrieves the authenticated user's ID from a context.
// Returns an empty string if not present.
func UserIDFromContext(ctx context.Context) string {
	if id, ok := ctx.Value(ContextKeyUserID).(string); ok {
		return id
	}
	return ""
}

// extractToken gets the JWT from either the Authorization header or query parameter.
// Authorization header takes precedence. Query parameter is used for WebSocket upgrades.
func extractToken(c *gin.Context) (string, error) {
	// Try Authorization header first (REST requests).
	if authHeader := c.GetHeader("Authorization"); authHeader != "" {
		return extractBearerToken(authHeader)
	}

	// Fall back to query parameter (WebSocket upgrades).
	if token := c.Query("token"); token != "" {
		return token, nil
	}

	return "", fmt.Errorf("no token provided")
}

// extractBearerToken parses "Bearer <token>" from an Authorization header.
func extractBearerToken(header string) (string, error) {
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
		return "", fmt.Errorf("invalid authorization header format")
	}
	return strings.TrimSpace(parts[1]), nil
}

// getKey returns the cached EC public key, refreshing from JWKS if stale.
func (c *jwksCache) getKey(ctx context.Context, supabaseURL string) (*ecdsa.PublicKey, error) {
	c.mu.RLock()
	if c.key != nil && time.Since(c.lastFetch) < c.refreshTTL {
		key := c.key
		c.mu.RUnlock()
		return key, nil
	}
	c.mu.RUnlock()

	c.mu.Lock()
	defer c.mu.Unlock()

	// Re-check after acquiring write lock (another goroutine may have refreshed).
	if c.key != nil && time.Since(c.lastFetch) < c.refreshTTL {
		return c.key, nil
	}

	key, err := fetchECPublicKey(ctx, supabaseURL+"/auth/v1/.well-known/jwks.json")
	if err != nil {
		return nil, err
	}

	c.key = key
	c.lastFetch = time.Now()
	return key, nil
}

// fetchECPublicKey fetches the JWKS endpoint and returns the first ES256 EC public key.
func fetchECPublicKey(ctx context.Context, jwksURL string) (*ecdsa.PublicKey, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, jwksURL, nil)
	if err != nil {
		return nil, fmt.Errorf("building jwks request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching jwks: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Keys []jwksKey `json:"keys"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding jwks: %w", err)
	}

	for _, k := range result.Keys {
		if k.Kty == "EC" && k.Alg == "ES256" && k.Crv == "P-256" {
			return parseP256Key(k.X, k.Y)
		}
	}

	return nil, fmt.Errorf("no ES256 P-256 key found in JWKS")
}

// parseP256Key constructs an ECDSA public key from base64url-encoded X and Y coordinates.
func parseP256Key(xB64, yB64 string) (*ecdsa.PublicKey, error) {
	xBytes, err := base64.RawURLEncoding.DecodeString(xB64)
	if err != nil {
		return nil, fmt.Errorf("decoding x coordinate: %w", err)
	}
	yBytes, err := base64.RawURLEncoding.DecodeString(yB64)
	if err != nil {
		return nil, fmt.Errorf("decoding y coordinate: %w", err)
	}

	return &ecdsa.PublicKey{
		Curve: elliptic.P256(),
		X:     new(big.Int).SetBytes(xBytes),
		Y:     new(big.Int).SetBytes(yBytes),
	}, nil
}
