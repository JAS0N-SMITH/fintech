package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/huchknows/fintech/backend/internal/middleware"
)

// Refresh token cookie attributes. The cookie is scoped to /api/v1/auth so the
// browser only includes it on session endpoints — minimising exposure.
const (
	refreshTokenCookie = "rt"
	cookiePath         = "/api/v1/auth"
	cookieMaxAge       = 60 * 60 * 24 * 30 // 30 days
)

// sessionRequest is the body Angular POSTs after a successful Supabase login.
type sessionRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// supabaseTokenResponse mirrors fields returned by Supabase /auth/v1/token.
type supabaseTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
	User         any    `json:"user"`
}

// sessionResponse is returned to Angular on GET /auth/session.
// The refresh token is intentionally omitted — it must stay in the HTTP-only cookie.
type sessionResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	TokenType   string `json:"token_type"`
	User        any    `json:"user"`
}

// AuthHandler is a public auth proxy that owns the refresh token cookie lifecycle.
//
// The browser never exposes the refresh token to JavaScript. Instead:
//   - POST /auth/session stores a fresh refresh token in an HttpOnly cookie.
//   - GET  /auth/session exchanges the cookie for a new Supabase session and
//     rotates the cookie with the newly-issued refresh token.
//   - DELETE /auth/session clears the cookie on logout.
type AuthHandler struct {
	supabaseURL     string
	supabaseAnonKey string
	httpClient      *http.Client
	secure          bool
}

// NewAuthHandler constructs an AuthHandler. The secure flag drives the cookie
// Secure attribute — set true in production (HTTPS), false in local HTTP dev.
func NewAuthHandler(supabaseURL, supabaseAnonKey string, secure bool) *AuthHandler {
	return &AuthHandler{
		supabaseURL:     supabaseURL,
		supabaseAnonKey: supabaseAnonKey,
		httpClient:      &http.Client{Timeout: 10 * time.Second},
		secure:          secure,
	}
}

// RegisterRoutes attaches /auth/session endpoints to the given route group.
// These routes must be registered under a public group — callers do not yet
// have an access token when they hit GET on page reload.
func (h *AuthHandler) RegisterRoutes(rg *gin.RouterGroup) {
	auth := rg.Group("/auth")
	auth.POST("/session", h.CreateSession)
	auth.GET("/session", h.RefreshSession)
	auth.DELETE("/session", h.DeleteSession)
}

// CreateSession stores the caller-supplied refresh token in an HttpOnly cookie.
// Called by Angular immediately after a successful Supabase sign-in.
//
//	@Summary     Persist refresh token in HTTP-only cookie
//	@Tags        auth
//	@Accept      json
//	@Produce     json
//	@Param       body body sessionRequest true "Refresh token"
//	@Success     204
//	@Failure     400 {object} Problem
//	@Router      /auth/session [post]
func (h *AuthHandler) CreateSession(c *gin.Context) {
	var req sessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Problem{
			Status: http.StatusBadRequest,
			Title:  http.StatusText(http.StatusBadRequest),
			Detail: "refresh_token is required",
		})
		return
	}

	h.setRefreshCookie(c, req.RefreshToken)
	c.Status(http.StatusNoContent)
}

// RefreshSession exchanges the refresh token in the rt cookie for a fresh
// Supabase session. The new refresh token is written back to the cookie; the
// new access token is returned in the JSON body.
//
//	@Summary     Exchange refresh cookie for new access token
//	@Tags        auth
//	@Produce     json
//	@Success     200 {object} sessionResponse
//	@Failure     401 {object} Problem
//	@Router      /auth/session [get]
func (h *AuthHandler) RefreshSession(c *gin.Context) {
	refreshToken, err := c.Cookie(refreshTokenCookie)
	if err != nil || refreshToken == "" {
		c.JSON(http.StatusUnauthorized, Problem{
			Status: http.StatusUnauthorized,
			Title:  http.StatusText(http.StatusUnauthorized),
			Detail: "no session",
		})
		return
	}

	supaResp, err := h.callSupabaseRefresh(c.Request.Context(), refreshToken)
	if err != nil {
		slog.WarnContext(c.Request.Context(), "supabase refresh failed",
			"request_id", middleware.RequestIDFromContext(c),
			"error", err,
		)
		h.clearRefreshCookie(c)
		c.JSON(http.StatusUnauthorized, Problem{
			Status: http.StatusUnauthorized,
			Title:  http.StatusText(http.StatusUnauthorized),
			Detail: "session expired",
		})
		return
	}

	h.setRefreshCookie(c, supaResp.RefreshToken)
	c.JSON(http.StatusOK, sessionResponse{
		AccessToken: supaResp.AccessToken,
		ExpiresIn:   supaResp.ExpiresIn,
		TokenType:   supaResp.TokenType,
		User:        supaResp.User,
	})
}

// DeleteSession clears the refresh token cookie. Called on logout.
//
//	@Summary     Clear the refresh token cookie
//	@Tags        auth
//	@Success     204
//	@Router      /auth/session [delete]
func (h *AuthHandler) DeleteSession(c *gin.Context) {
	h.clearRefreshCookie(c)
	c.Status(http.StatusNoContent)
}

// callSupabaseRefresh POSTs the refresh token to Supabase /auth/v1/token and
// returns the rotated session. Any non-200 or malformed response becomes a
// generic error — upstream details are not exposed to the caller.
func (h *AuthHandler) callSupabaseRefresh(ctx context.Context, refreshToken string) (*supabaseTokenResponse, error) {
	body, err := json.Marshal(map[string]string{"refresh_token": refreshToken})
	if err != nil {
		return nil, fmt.Errorf("marshal supabase request: %w", err)
	}

	url := h.supabaseURL + "/auth/v1/token?grant_type=refresh_token"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build supabase request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("apikey", h.supabaseAnonKey)

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call supabase: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("supabase returned %d", resp.StatusCode)
	}

	var result supabaseTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode supabase response: %w", err)
	}
	if result.AccessToken == "" || result.RefreshToken == "" {
		return nil, errors.New("supabase response missing tokens")
	}
	return &result, nil
}

// setRefreshCookie writes the refresh token as an HttpOnly, SameSite=Strict
// cookie. Call SetSameSite before SetCookie — Gin applies the SameSite mode
// to whichever SetCookie call follows.
func (h *AuthHandler) setRefreshCookie(c *gin.Context, token string) {
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(
		refreshTokenCookie,
		token,
		cookieMaxAge,
		cookiePath,
		"", // domain — empty means the responding host
		h.secure,
		true, // httpOnly
	)
}

// clearRefreshCookie writes a tombstone cookie with MaxAge=-1 so the browser
// removes it. Safe to call when no cookie exists.
func (h *AuthHandler) clearRefreshCookie(c *gin.Context) {
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(
		refreshTokenCookie,
		"",
		-1,
		cookiePath,
		"",
		h.secure,
		true,
	)
}
