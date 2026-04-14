package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/gin-gonic/gin"
)

// --- test helpers ---

// supabaseMock records calls and returns a fixed response body / status.
type supabaseMock struct {
	server       *httptest.Server
	calls        atomic.Int32
	lastBody     []byte
	lastAPIKey   string
	lastRawQuery string
}

func newSupabaseMock(t *testing.T, status int, body string) *supabaseMock {
	t.Helper()
	m := &supabaseMock{}
	m.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		m.calls.Add(1)
		b, _ := io.ReadAll(r.Body)
		m.lastBody = b
		m.lastAPIKey = r.Header.Get("apikey")
		m.lastRawQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(m.server.Close)
	return m
}

// authRouter builds a test engine with an AuthHandler pointing at the given mock.
func authRouter(supabaseURL string, secure bool) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewAuthHandler(supabaseURL, "test-anon-key", secure)
	h.RegisterRoutes(&r.RouterGroup)
	return r
}

// findCookie returns the named cookie from a response, or nil if absent.
func findCookie(resp *http.Response, name string) *http.Cookie {
	for _, c := range resp.Cookies() {
		if c.Name == name {
			return c
		}
	}
	return nil
}

// --- CreateSession ---

func TestAuthHandler_CreateSession(t *testing.T) {
	const validBody = `{"refresh_token":"rt-value"}`

	tests := []struct {
		name       string
		body       string
		secure     bool
		wantStatus int
		wantCookie bool
	}{
		{"valid_body_sets_cookie", validBody, false, http.StatusNoContent, true},
		{"secure_flag_off_in_dev", validBody, false, http.StatusNoContent, true},
		{"secure_flag_on_in_prod", validBody, true, http.StatusNoContent, true},
		{"missing_refresh_token_field", `{}`, false, http.StatusBadRequest, false},
		{"empty_refresh_token_value", `{"refresh_token":""}`, false, http.StatusBadRequest, false},
		{"malformed_json_body", `{`, false, http.StatusBadRequest, false},
		{"no_body", ``, false, http.StatusBadRequest, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mock := newSupabaseMock(t, http.StatusOK, "")
			r := authRouter(mock.server.URL, tc.secure)

			req := httptest.NewRequest(http.MethodPost, "/auth/session", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tc.wantStatus {
				t.Fatalf("status: got %d, want %d", w.Code, tc.wantStatus)
			}

			c := findCookie(w.Result(), refreshTokenCookie)
			if tc.wantCookie {
				if c == nil {
					t.Fatalf("expected cookie %q, got none", refreshTokenCookie)
				}
				if c.Value != "rt-value" {
					t.Errorf("cookie value: got %q, want %q", c.Value, "rt-value")
				}
				if !c.HttpOnly {
					t.Errorf("cookie not HttpOnly")
				}
				if c.SameSite != http.SameSiteStrictMode {
					t.Errorf("cookie SameSite: got %v, want Strict", c.SameSite)
				}
				if c.Path != cookiePath {
					t.Errorf("cookie Path: got %q, want %q", c.Path, cookiePath)
				}
				if c.MaxAge != cookieMaxAge {
					t.Errorf("cookie MaxAge: got %d, want %d", c.MaxAge, cookieMaxAge)
				}
				if c.Secure != tc.secure {
					t.Errorf("cookie Secure: got %v, want %v", c.Secure, tc.secure)
				}
			} else {
				if c != nil {
					t.Errorf("expected no cookie, got %+v", c)
				}
			}

			if mock.calls.Load() != 0 {
				t.Errorf("CreateSession should never call Supabase, got %d calls", mock.calls.Load())
			}
		})
	}
}

// --- RefreshSession ---

func TestAuthHandler_RefreshSession_HappyPath(t *testing.T) {
	const upstreamBody = `{
		"access_token": "new-access",
		"refresh_token": "new-refresh",
		"expires_in": 900,
		"token_type": "bearer",
		"user": {"id":"user-1","email":"a@b.c"}
	}`
	mock := newSupabaseMock(t, http.StatusOK, upstreamBody)
	r := authRouter(mock.server.URL, false)

	req := httptest.NewRequest(http.MethodGet, "/auth/session", nil)
	req.AddCookie(&http.Cookie{Name: refreshTokenCookie, Value: "old-refresh"})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200", w.Code)
	}

	// Body contains access_token but NOT refresh_token (regression guard).
	var bodyMap map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &bodyMap); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if bodyMap["access_token"] != "new-access" {
		t.Errorf("access_token: got %v, want new-access", bodyMap["access_token"])
	}
	if _, ok := bodyMap["refresh_token"]; ok {
		t.Errorf("response body must not contain refresh_token, got %v", bodyMap)
	}

	// Cookie rotated to the new refresh token.
	c := findCookie(w.Result(), refreshTokenCookie)
	if c == nil {
		t.Fatal("expected rotated cookie, got none")
	}
	if c.Value != "new-refresh" {
		t.Errorf("cookie rotation: got %q, want %q", c.Value, "new-refresh")
	}
	if c.MaxAge != cookieMaxAge {
		t.Errorf("cookie MaxAge: got %d, want %d", c.MaxAge, cookieMaxAge)
	}

	// Upstream assertions.
	if mock.calls.Load() != 1 {
		t.Errorf("expected 1 upstream call, got %d", mock.calls.Load())
	}
	if mock.lastAPIKey != "test-anon-key" {
		t.Errorf("apikey header: got %q, want %q", mock.lastAPIKey, "test-anon-key")
	}
	q, _ := url.ParseQuery(mock.lastRawQuery)
	if q.Get("grant_type") != "refresh_token" {
		t.Errorf("grant_type query: got %q, want refresh_token", q.Get("grant_type"))
	}
	var upBody map[string]string
	if err := json.Unmarshal(mock.lastBody, &upBody); err != nil {
		t.Fatalf("decode upstream body: %v", err)
	}
	if upBody["refresh_token"] != "old-refresh" {
		t.Errorf("upstream body refresh_token: got %q, want old-refresh", upBody["refresh_token"])
	}
}

func TestAuthHandler_RefreshSession_Errors(t *testing.T) {
	tests := []struct {
		name           string
		cookieValue    string // "" → don't send cookie
		sendCookie     bool
		upstreamStatus int
		upstreamBody   string
		closeUpstream  bool
		wantStatus     int
		wantCallCount  int32
	}{
		{
			name:          "no_cookie_returns_401",
			sendCookie:    false,
			wantStatus:    http.StatusUnauthorized,
			wantCallCount: 0,
		},
		{
			name:          "empty_cookie_returns_401",
			sendCookie:    true,
			cookieValue:   "",
			wantStatus:    http.StatusUnauthorized,
			wantCallCount: 0,
		},
		{
			name:           "upstream_401_clears_cookie",
			sendCookie:     true,
			cookieValue:    "stale",
			upstreamStatus: http.StatusUnauthorized,
			upstreamBody:   `{"error":"invalid_grant"}`,
			wantStatus:     http.StatusUnauthorized,
			wantCallCount:  1,
		},
		{
			name:           "upstream_500_clears_cookie",
			sendCookie:     true,
			cookieValue:    "stale",
			upstreamStatus: http.StatusInternalServerError,
			upstreamBody:   `{"error":"boom"}`,
			wantStatus:     http.StatusUnauthorized,
			wantCallCount:  1,
		},
		{
			name:           "upstream_malformed_json_clears_cookie",
			sendCookie:     true,
			cookieValue:    "stale",
			upstreamStatus: http.StatusOK,
			upstreamBody:   `{`,
			wantStatus:     http.StatusUnauthorized,
			wantCallCount:  1,
		},
		{
			name:           "upstream_empty_tokens_clears_cookie",
			sendCookie:     true,
			cookieValue:    "stale",
			upstreamStatus: http.StatusOK,
			upstreamBody:   `{"access_token":"","refresh_token":""}`,
			wantStatus:     http.StatusUnauthorized,
			wantCallCount:  1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mock := newSupabaseMock(t, tc.upstreamStatus, tc.upstreamBody)
			r := authRouter(mock.server.URL, false)

			req := httptest.NewRequest(http.MethodGet, "/auth/session", nil)
			if tc.sendCookie {
				req.AddCookie(&http.Cookie{Name: refreshTokenCookie, Value: tc.cookieValue})
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tc.wantStatus {
				t.Fatalf("status: got %d, want %d", w.Code, tc.wantStatus)
			}
			if mock.calls.Load() != tc.wantCallCount {
				t.Errorf("upstream calls: got %d, want %d", mock.calls.Load(), tc.wantCallCount)
			}

			// When the upstream was called, a clear-cookie header must be set.
			if tc.wantCallCount > 0 {
				c := findCookie(w.Result(), refreshTokenCookie)
				if c == nil {
					t.Fatal("expected clear-cookie, got none")
				}
				if c.MaxAge != -1 {
					t.Errorf("clear cookie MaxAge: got %d, want -1", c.MaxAge)
				}
			}
		})
	}
}

func TestAuthHandler_RefreshSession_UpstreamNetworkError(t *testing.T) {
	// Start a mock then immediately close it so the handler's HTTP call fails.
	mock := newSupabaseMock(t, http.StatusOK, "")
	mock.server.Close()

	r := authRouter(mock.server.URL, false)
	req := httptest.NewRequest(http.MethodGet, "/auth/session", nil)
	req.AddCookie(&http.Cookie{Name: refreshTokenCookie, Value: "stale"})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status: got %d, want 401", w.Code)
	}
	c := findCookie(w.Result(), refreshTokenCookie)
	if c == nil || c.MaxAge != -1 {
		t.Errorf("expected clear-cookie, got %+v", c)
	}
}

// --- DeleteSession ---

func TestAuthHandler_DeleteSession(t *testing.T) {
	tests := []struct {
		name       string
		sendCookie bool
	}{
		{"clears_cookie_when_present", true},
		{"clears_cookie_when_absent", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mock := newSupabaseMock(t, http.StatusOK, "")
			r := authRouter(mock.server.URL, false)

			req := httptest.NewRequest(http.MethodDelete, "/auth/session", nil)
			if tc.sendCookie {
				req.AddCookie(&http.Cookie{Name: refreshTokenCookie, Value: "existing"})
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != http.StatusNoContent {
				t.Fatalf("status: got %d, want 204", w.Code)
			}
			c := findCookie(w.Result(), refreshTokenCookie)
			if c == nil {
				t.Fatal("expected clear-cookie, got none")
			}
			if c.MaxAge != -1 {
				t.Errorf("clear cookie MaxAge: got %d, want -1", c.MaxAge)
			}
			if mock.calls.Load() != 0 {
				t.Errorf("DeleteSession should not call Supabase, got %d calls", mock.calls.Load())
			}
		})
	}
}

// --- Cookie attribute matrix ---

func TestAuthHandler_CookieAttributes(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		secure     bool
		setCookie  bool
		wantMaxAge int
	}{
		{"create_dev", http.MethodPost, false, false, cookieMaxAge},
		{"create_prod", http.MethodPost, true, false, cookieMaxAge},
		{"delete_dev", http.MethodDelete, false, false, -1},
		{"delete_prod", http.MethodDelete, true, false, -1},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mock := newSupabaseMock(t, http.StatusOK, "")
			r := authRouter(mock.server.URL, tc.secure)

			var body io.Reader
			if tc.method == http.MethodPost {
				body = strings.NewReader(`{"refresh_token":"x"}`)
			}
			req := httptest.NewRequest(tc.method, "/auth/session", body)
			if tc.method == http.MethodPost {
				req.Header.Set("Content-Type", "application/json")
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			c := findCookie(w.Result(), refreshTokenCookie)
			if c == nil {
				t.Fatalf("expected cookie, got none")
			}
			if c.HttpOnly != true {
				t.Errorf("HttpOnly: got %v, want true", c.HttpOnly)
			}
			if c.SameSite != http.SameSiteStrictMode {
				t.Errorf("SameSite: got %v, want Strict", c.SameSite)
			}
			if c.Path != cookiePath {
				t.Errorf("Path: got %q, want %q", c.Path, cookiePath)
			}
			if c.Secure != tc.secure {
				t.Errorf("Secure: got %v, want %v", c.Secure, tc.secure)
			}
			if c.MaxAge != tc.wantMaxAge {
				t.Errorf("MaxAge: got %d, want %d", c.MaxAge, tc.wantMaxAge)
			}
		})
	}
}
