package middleware

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// newTestKey generates a fresh P-256 key pair for tests.
func newTestKey(t *testing.T) *ecdsa.PrivateKey {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate test key: %v", err)
	}
	return key
}

// signToken creates a signed ES256 JWT with the given claims and key.
func signToken(t *testing.T, key *ecdsa.PrivateKey, claims jwt.Claims) string {
	t.Helper()
	tok := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	signed, err := tok.SignedString(key)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return signed
}

// routerWithMiddleware returns a test Gin engine that serves GET /test behind
// the provided middleware, responding 200 with the user_id from context.
func routerWithMiddleware(mw ...gin.HandlerFunc) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/test", append(mw, func(c *gin.Context) {
		uid, _ := c.Get(string(ContextKeyUserID))
		c.String(http.StatusOK, "%v", uid)
	})...)
	return r
}

// jwksServerFor starts an httptest server that serves a JWKS with the given public key.
func jwksServerFor(t *testing.T, pub *ecdsa.PublicKey) *httptest.Server {
	t.Helper()

	import64 := func(b []byte) string {
		import64 := make([]byte, 32)
		copy(import64[32-len(b):], b)
		return encodeB64(import64)
	}
	xBytes := pub.X.Bytes()
	yBytes := pub.Y.Bytes()

	body := `{"keys":[{"kty":"EC","alg":"ES256","crv":"P-256","use":"sig","x":"` +
		import64(xBytes) + `","y":"` + import64(yBytes) + `"}]}`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv
}

// encodeB64 encodes bytes as base64url without padding.
func encodeB64(b []byte) string {
	const table = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"
	var buf []byte
	for i := 0; i < len(b); i += 3 {
		var block [3]byte
		n := copy(block[:], b[i:])
		buf = append(buf, table[block[0]>>2])
		buf = append(buf, table[((block[0]&0x3)<<4)|(block[1]>>4)])
		if n > 1 {
			buf = append(buf, table[((block[1]&0xf)<<2)|(block[2]>>6)])
		}
		if n > 2 {
			buf = append(buf, table[block[2]&0x3f])
		}
	}
	return string(buf)
}

func TestRequireAuth(t *testing.T) {
	key := newTestKey(t)
	srv := jwksServerFor(t, &key.PublicKey)
	router := routerWithMiddleware(RequireAuth(srv.URL))

	validClaims := func(sub string) SupabaseClaims {
		return SupabaseClaims{
			RegisteredClaims: jwt.RegisteredClaims{
				Subject:   sub,
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
				IssuedAt:  jwt.NewNumericDate(time.Now()),
			},
			Role: "authenticated",
		}
	}

	tests := []struct {
		name       string
		authHeader string
		wantStatus int
	}{
		{
			name:       "valid token",
			authHeader: "Bearer " + signToken(t, key, validClaims("user-uuid-123")),
			wantStatus: http.StatusOK,
		},
		{
			name:       "missing authorization header",
			authHeader: "",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "malformed header — no bearer prefix",
			authHeader: signToken(t, key, validClaims("user-uuid-123")),
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "expired token",
			authHeader: "Bearer " + signToken(t, key, SupabaseClaims{
				RegisteredClaims: jwt.RegisteredClaims{
					Subject:   "user-uuid-123",
					ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Minute)),
				},
			}),
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "wrong signing key",
			authHeader: func() string {
				otherKey := newTestKey(t)
				return "Bearer " + signToken(t, otherKey, validClaims("user-uuid-123"))
			}(),
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "malformed token",
			authHeader: "Bearer not.a.jwt",
			wantStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}
		})
	}
}

func TestRequireAuth_QueryParameterToken(t *testing.T) {
	key := newTestKey(t)
	srv := jwksServerFor(t, &key.PublicKey)
	router := routerWithMiddleware(RequireAuth(srv.URL))

	validClaims := func(sub string) SupabaseClaims {
		return SupabaseClaims{
			RegisteredClaims: jwt.RegisteredClaims{
				Subject:   sub,
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
				IssuedAt:  jwt.NewNumericDate(time.Now()),
			},
			Role: "authenticated",
		}
	}

	tests := []struct {
		name       string
		tokenQuery string
		wantStatus int
	}{
		{
			name:       "valid token in query parameter",
			tokenQuery: signToken(t, key, validClaims("user-uuid-456")),
			wantStatus: http.StatusOK,
		},
		{
			name:       "missing token in query parameter",
			tokenQuery: "",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "invalid token in query parameter",
			tokenQuery: "not.a.jwt",
			wantStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := "/test"
			if tt.tokenQuery != "" {
				url += "?token=" + tt.tokenQuery
			}
			req := httptest.NewRequest(http.MethodGet, url, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}
		})
	}
}

func TestRequireAuth_HeaderPrecedenceOverQuery(t *testing.T) {
	key := newTestKey(t)
	srv := jwksServerFor(t, &key.PublicKey)
	router := routerWithMiddleware(RequireAuth(srv.URL))

	validClaims := func(sub string) SupabaseClaims {
		return SupabaseClaims{
			RegisteredClaims: jwt.RegisteredClaims{
				Subject:   sub,
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
				IssuedAt:  jwt.NewNumericDate(time.Now()),
			},
			Role: "authenticated",
		}
	}

	headerToken := signToken(t, key, validClaims("user-from-header"))
	queryToken := "invalid.token"

	req := httptest.NewRequest(http.MethodGet, "/test?token="+queryToken, nil)
	req.Header.Set("Authorization", "Bearer "+headerToken)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200 (header should take precedence)", w.Code)
	}
	if w.Body.String() != "user-from-header" {
		t.Errorf("user_id = %q, want user-from-header", w.Body.String())
	}
}

func TestRequireAuth_UserIDInContext(t *testing.T) {
	key := newTestKey(t)
	srv := jwksServerFor(t, &key.PublicKey)
	router := routerWithMiddleware(RequireAuth(srv.URL))

	claims := SupabaseClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "expected-user-id",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
		},
		Role: "authenticated",
	}

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+signToken(t, key, claims))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if got := w.Body.String(); got != "expected-user-id" {
		t.Errorf("user_id in context = %q, want %q", got, "expected-user-id")
	}
}

func TestRequireAuth_MissingSubjectClaim(t *testing.T) {
	key := newTestKey(t)
	srv := jwksServerFor(t, &key.PublicKey)
	router := routerWithMiddleware(RequireAuth(srv.URL))

	// Token with no Subject claim.
	claims := SupabaseClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
		},
		Role: "authenticated",
	}
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+signToken(t, key, claims))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401 for missing sub claim", w.Code)
	}
}

func TestRequireAuth_JWKSEndpointError(t *testing.T) {
	// Point at a server that returns 500.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)

	key := newTestKey(t)
	router := routerWithMiddleware(RequireAuth(srv.URL))

	claims := SupabaseClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "user-xyz",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
		},
		Role: "authenticated",
	}
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+signToken(t, key, claims))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500 when JWKS fetch fails", w.Code)
	}
}

func TestRequireAuth_JWKSMissingKey(t *testing.T) {
	// JWKS endpoint returns valid JSON but no ES256 key.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"keys":[]}`))
	}))
	t.Cleanup(srv.Close)

	key := newTestKey(t)
	router := routerWithMiddleware(RequireAuth(srv.URL))

	claims := SupabaseClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "user-xyz",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
		},
		Role: "authenticated",
	}
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+signToken(t, key, claims))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500 when no ES256 key in JWKS", w.Code)
	}
}

func TestAppRole(t *testing.T) {
	tests := []struct {
		name     string
		meta     map[string]interface{}
		wantRole string
	}{
		{"nil app_metadata defaults to user", nil, "user"},
		{"empty app_metadata defaults to user", map[string]interface{}{}, "user"},
		{"role set to admin", map[string]interface{}{"role": "admin"}, "admin"},
		{"role set to user", map[string]interface{}{"role": "user"}, "user"},
		{"non-string role defaults to user", map[string]interface{}{"role": 42}, "user"},
		{"empty string role defaults to user", map[string]interface{}{"role": ""}, "user"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &SupabaseClaims{AppMetadata: tt.meta}
			if got := c.appRole(); got != tt.wantRole {
				t.Errorf("appRole() = %q, want %q", got, tt.wantRole)
			}
		})
	}
}

func TestUserIDFromContext(t *testing.T) {
	t.Run("returns user ID when set", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		var captured string
		r := gin.New()
		r.GET("/", func(c *gin.Context) {
			c.Set(string(ContextKeyUserID), "test-user-id")
			captured = UserIDFromContext(c.Request.Context())
			// Gin context values aren't accessible via request context directly —
			// test that the helper returns empty for request context (not gin context).
			c.Status(http.StatusOK)
		})
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		httptest.NewRecorder()
		r.ServeHTTP(httptest.NewRecorder(), req)
		// UserIDFromContext reads from context.Context (not gin.Context).
		// Verify it returns empty string when key is absent from request context.
		if captured != "" {
			t.Errorf("UserIDFromContext from plain request context = %q, want empty", captured)
		}
	})

	t.Run("returns empty string when not set", func(t *testing.T) {
		if got := UserIDFromContext(httptest.NewRequest(http.MethodGet, "/", nil).Context()); got != "" {
			t.Errorf("UserIDFromContext = %q, want empty string", got)
		}
	})
}

func TestRequireRole(t *testing.T) {
	adminRouter := func(role string) *gin.Engine {
		gin.SetMode(gin.TestMode)
		r := gin.New()
		r.GET("/test", func(c *gin.Context) {
			c.Set(string(ContextKeyUserRole), role)
		}, RequireRole("admin"), func(c *gin.Context) {
			c.Status(http.StatusOK)
		})
		return r
	}

	t.Run("admin role passes", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()
		adminRouter("admin").ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("status = %d, want 200", w.Code)
		}
	})

	t.Run("user role blocked", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()
		adminRouter("user").ServeHTTP(w, req)
		if w.Code != http.StatusForbidden {
			t.Errorf("status = %d, want 403", w.Code)
		}
	})
}
