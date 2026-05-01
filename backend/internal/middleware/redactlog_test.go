package middleware

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"log/slog"

	redactlog "github.com/JAS0N-SMITH/redactlog"
	redactgin "github.com/JAS0N-SMITH/redactlog/gin"
	"github.com/gin-gonic/gin"
)

func TestRedact_AuthorizationAndPAN(t *testing.T) {
    // Capture logger output
    buf := &bytes.Buffer{}
    baseLogger := slog.New(slog.NewJSONHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

    // Build redact handler (PCI preset)
    h, err := redactlog.NewPCI(redactlog.WithLogger(baseLogger), redactlog.WithRequestBody(true))
    if err != nil {
        t.Fatalf("failed to create redact handler: %v", err)
    }

    // Use the redact handler as the default slog logger so calls to slog.* are wrapped.
    wrappedLogger := slog.New(h)
    slog.SetDefault(wrappedLogger)

    // Gin router with redact middleware
    r := gin.New()
    r.Use(redactgin.New(h))
    r.POST("/pay", func(c *gin.Context) {
        var body map[string]string
        _ = c.BindJSON(&body)
        // Only log the PAN (the middleware will capture headers separately).
        slog.Info("payment", "pan", body["pan"])
        c.JSON(200, gin.H{"status": "ok"})
    })

    // Create request with Authorization header and PAN in body
    req := httptest.NewRequest(http.MethodPost, "/pay", bytes.NewBufferString(`{"pan":"4111111111111111"}`))
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Authorization", "Bearer abc123")

    w := httptest.NewRecorder()
    r.ServeHTTP(w, req)

    if w.Code != http.StatusOK {
        t.Fatalf("unexpected status code: %d", w.Code)
    }

    out := buf.String()

    // Authorization header: redact middleware may either mask the token
    // (e.g. "Bearer ***") or omit the header entirely from captured headers.
    // Ensure the raw token is not present and at least one of those behaviors occurred.
    hasMasked := bytes.Contains([]byte(out), []byte("Bearer ***"))
    hasAuthHeader := bytes.Contains([]byte(out), []byte("Authorization")) || bytes.Contains([]byte(out), []byte("authorization"))
    if !hasMasked && hasAuthHeader {
        t.Fatalf("authorization token not masked or removed in log output: %s", out)
    }

    // Expect PAN masked in first-6/last-4 format (411111******1111)
    if !bytes.Contains([]byte(out), []byte("411111******1111")) {
        t.Fatalf("PAN not masked in log output: %s", out)
    }

    // Ensure raw Authorization token is not present (header should be masked or removed).
    if bytes.Contains([]byte(out), []byte("abc123")) {
        t.Fatalf("raw authorization token appears in logs: %s", out)
    }
}
