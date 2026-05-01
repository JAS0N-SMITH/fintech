package integration

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"log/slog"

	redactlog "github.com/JAS0N-SMITH/redactlog"
	redactgin "github.com/JAS0N-SMITH/redactlog/gin"
	"github.com/gin-gonic/gin"

	mw "github.com/huchknows/fintech/backend/internal/middleware"
)

func TestAPI_Redaction_MiddlewareLogsAreMasked(t *testing.T) {
    // Capture logs to buffer
    buf := &bytes.Buffer{}
    baseLogger := slog.New(slog.NewJSONHandler(buf, &slog.HandlerOptions{Level: slog.LevelInfo}))

    // Build redact handler and wrap default slog
    h, err := redactlog.NewPCI(redactlog.WithLogger(baseLogger), redactlog.WithRequestBody(true))
    if err != nil {
        t.Fatalf("failed to create redact handler: %v", err)
    }
    wrapped := slog.New(h)
    slog.SetDefault(wrapped)

    // Build router similar to main.go middleware stack
    r := gin.New()
    r.Use(redactgin.New(h))
    r.Use(
        mw.RequestID(),
        mw.Logger(),
        gin.Recovery(),
    )

    r.POST("/pay", func(c *gin.Context) {
        var body map[string]string
        _ = c.BindJSON(&body)
        // Log PAN via slog so the redact handler will redact it in structured attrs.
        slog.Info("payment", "pan", body["pan"])
        c.JSON(200, gin.H{"status": "ok"})
    })

    srv := httptest.NewServer(r)
    defer srv.Close()

    reqBody := bytes.NewBufferString(`{"pan":"4111111111111111"}`)
    req, _ := http.NewRequest(http.MethodPost, srv.URL+"/pay", reqBody)
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Authorization", "Bearer abc123")

    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        t.Fatalf("request failed: %v", err)
    }
    defer resp.Body.Close()

    out := buf.String()

    if !bytes.Contains([]byte(out), []byte("411111******1111")) {
        t.Fatalf("PAN not masked in logs: %s", out)
    }

    if bytes.Contains([]byte(out), []byte("abc123")) {
        t.Fatalf("raw authorization token appears in logs: %s", out)
    }
}
