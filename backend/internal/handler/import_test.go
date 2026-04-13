package handler

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"

	"github.com/huchknows/fintech/backend/internal/middleware"
	"github.com/huchknows/fintech/backend/internal/model"
	"github.com/huchknows/fintech/backend/internal/service"
)

// MockImportService for testing.
type mockImportService struct {
	previewFunc func(portfolioID, brokerage string) (*model.ImportPreview, error)
	confirmFunc func(portfolioID string, req model.ImportConfirmRequest) (*model.ImportResult, error)
}

func (m *mockImportService) Preview(portfolioID string, file []byte, brokerage string) (*model.ImportPreview, error) {
	if m.previewFunc != nil {
		return m.previewFunc(portfolioID, brokerage)
	}
	return &model.ImportPreview{
		Parsed:       0,
		Valid:        0,
		Errors:       []model.ImportError{},
		Transactions: []model.CreateTransactionInput{},
	}, nil
}

func (m *mockImportService) Confirm(portfolioID string, req model.ImportConfirmRequest) (*model.ImportResult, error) {
	if m.confirmFunc != nil {
		return m.confirmFunc(portfolioID, req)
	}
	return &model.ImportResult{
		Created:  0,
		Failed:   0,
		Errors:   []model.ImportError{},
		Messages: []string{},
	}, nil
}

// Helper to create multipart form with CSV file.
func createMultipartCSV(csv string, brokerage string) (*bytes.Buffer, string) {
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)

	// Add file
	filePart, _ := writer.CreateFormFile("file", "test.csv")
	filePart.Write([]byte(csv))

	// Add brokerage if provided
	if brokerage != "" {
		writer.WriteField("brokerage", brokerage)
	}

	writer.Close()
	return body, writer.FormDataContentType()
}

// Helper to set auth context.
func setAuthContext(c *gin.Context) {
	c.Set(string(middleware.ContextKeyUserID), "user-123")
}

// TestImportHandlerPreview tests the preview endpoint.
func TestImportHandlerPreview(t *testing.T) {
	tests := []struct {
		name           string
		csvContent     string
		contentType    string
		brokerage      string
		mockFunc       func() (*model.ImportPreview, error)
		wantStatus     int
		wantValidCount int
		wantErrorCount int
		wantErr        bool
	}{
		{
			name:        "successful preview",
			csvContent:  "Symbol,Date\nAAPL,2024-01-15\n",
			brokerage:   "fidelity",
			wantStatus:  http.StatusOK,
			wantErr:     false,
			mockFunc: func() (*model.ImportPreview, error) {
				return &model.ImportPreview{
					Parsed:  1,
					Valid:   1,
					Errors:  []model.ImportError{},
					Transactions: []model.CreateTransactionInput{
						{
							TransactionType: model.TransactionTypeBuy,
							Symbol:          "AAPL",
							TransactionDate: "2024-01-15",
							TotalAmount:     decimal.NewFromInt(100),
						},
					},
				}, nil
			},
		},
		{
			name:        "preview with errors",
			csvContent:  "Symbol,Date\nAAPL,2024-01-15\nINVALID,bad\n",
			brokerage:   "fidelity",
			wantStatus:  http.StatusOK,
			wantErr:     false,
			mockFunc: func() (*model.ImportPreview, error) {
				return &model.ImportPreview{
					Parsed: 2,
					Valid:  1,
					Errors: []model.ImportError{
						{Row: 2, Message: "invalid symbol"},
					},
					Transactions: []model.CreateTransactionInput{
						{
							TransactionType: model.TransactionTypeBuy,
							Symbol:          "AAPL",
							TransactionDate: "2024-01-15",
							TotalAmount:     decimal.NewFromInt(100),
						},
					},
				}, nil
			},
		},
		{
			name:       "missing file",
			csvContent: "",
			wantStatus: http.StatusBadRequest,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := &mockImportService{}
			if tt.mockFunc != nil {
				mockSvc.previewFunc = func(portfolioID, brokerage string) (*model.ImportPreview, error) {
					return tt.mockFunc()
				}
			}

			handler := NewImportHandler(mockSvc)

			// Create router
			router := gin.New()
			router.Use(func(c *gin.Context) {
				setAuthContext(c)
			})

			portfolioGroup := router.Group("/portfolios/:id")
			handler.RegisterRoutes(portfolioGroup)

			// Create request
			body, contentType := createMultipartCSV(tt.csvContent, tt.brokerage)
			req, _ := http.NewRequest("POST", "/portfolios/port-456/import", body)
			req.Header.Set("Content-Type", contentType)

			// Execute
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("wantStatus=%d, got %d", tt.wantStatus, w.Code)
			}

			if tt.wantStatus == http.StatusOK {
				var preview model.ImportPreview
				json.Unmarshal(w.Body.Bytes(), &preview)

				if preview.Valid != tt.wantValidCount && tt.wantValidCount > 0 {
					t.Logf("preview: %+v", preview)
				}
			}
		})
	}
}

// TestImportHandlerConfirm tests the confirm endpoint.
func TestImportHandlerConfirm(t *testing.T) {
	tests := []struct {
		name       string
		reqBody    model.ImportConfirmRequest
		mockFunc   func() (*model.ImportResult, error)
		wantStatus int
		wantErr    bool
	}{
		{
			name: "successful confirm",
			reqBody: model.ImportConfirmRequest{
				Transactions: []model.CreateTransactionInput{
					{
						TransactionType: model.TransactionTypeBuy,
						Symbol:          "AAPL",
						TransactionDate: "2024-01-15",
						Quantity:        decPtr(decimal.NewFromInt(10)),
						PricePerShare:   decPtr(decimal.NewFromString("150.00")),
						TotalAmount:     decimal.NewFromString("1500.00"),
					},
				},
			},
			wantStatus: http.StatusOK,
			wantErr:    false,
			mockFunc: func() (*model.ImportResult, error) {
				return &model.ImportResult{
					Created:  1,
					Failed:   0,
					Errors:   []model.ImportError{},
					Messages: []string{"Successfully imported 1 transactions"},
				}, nil
			},
		},
		{
			name: "confirm with failures",
			reqBody: model.ImportConfirmRequest{
				Transactions: []model.CreateTransactionInput{
					{
						TransactionType: model.TransactionTypeBuy,
						Symbol:          "AAPL",
						TransactionDate: "2024-01-15",
						Quantity:        decPtr(decimal.NewFromInt(10)),
						PricePerShare:   decPtr(decimal.NewFromString("150.00")),
						TotalAmount:     decimal.NewFromString("1500.00"),
					},
					{
						TransactionType: model.TransactionTypeSell,
						Symbol:          "AAPL",
						TransactionDate: "2024-02-01",
						Quantity:        decPtr(decimal.NewFromInt(100)),
						PricePerShare:   decPtr(decimal.NewFromString("160.00")),
						TotalAmount:     decimal.NewFromString("16000.00"),
					},
				},
			},
			wantStatus: http.StatusOK,
			wantErr:    false,
			mockFunc: func() (*model.ImportResult, error) {
				return &model.ImportResult{
					Created: 1,
					Failed:  1,
					Errors: []model.ImportError{
						{Row: 2, Message: "insufficient holdings"},
					},
					Messages: []string{
						"Successfully imported 1 transactions",
						"Failed to import 1 transactions (see errors for details)",
					},
				}, nil
			},
		},
		{
			name: "empty transaction list",
			reqBody: model.ImportConfirmRequest{
				Transactions: []model.CreateTransactionInput{},
			},
			wantStatus: http.StatusBadRequest,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := &mockImportService{}
			if tt.mockFunc != nil {
				mockSvc.confirmFunc = func(portfolioID string, req model.ImportConfirmRequest) (*model.ImportResult, error) {
					return tt.mockFunc()
				}
			}

			handler := NewImportHandler(mockSvc)

			// Create router
			router := gin.New()
			router.Use(func(c *gin.Context) {
				setAuthContext(c)
			})

			portfolioGroup := router.Group("/portfolios/:id")
			handler.RegisterRoutes(portfolioGroup)

			// Create request
			bodyBytes, _ := json.Marshal(tt.reqBody)
			req, _ := http.NewRequest("POST", "/portfolios/port-456/import/confirm", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")

			// Execute
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("wantStatus=%d, got %d. body: %s", tt.wantStatus, w.Code, w.Body.String())
			}
		})
	}
}

// TestImportHandlerFileSize tests file size validation.
func TestImportHandlerFileSize(t *testing.T) {
	mockSvc := &mockImportService{}
	handler := NewImportHandler(mockSvc)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		setAuthContext(c)
	})

	portfolioGroup := router.Group("/portfolios/:id")
	handler.RegisterRoutes(portfolioGroup)

	// Create a file larger than 5 MB
	largeCSV := strings.Repeat("A,B,C\n", 1000000) // ~6 MB

	body, contentType := createMultipartCSV(largeCSV, "")
	req, _ := http.NewRequest("POST", "/portfolios/port-456/import", body)
	req.Header.Set("Content-Type", contentType)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("expected 413 for oversized file, got %d", w.Code)
	}
}

// TestImportHandlerInvalidFileType tests file type validation.
func TestImportHandlerInvalidFileType(t *testing.T) {
	mockSvc := &mockImportService{}
	handler := NewImportHandler(mockSvc)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		setAuthContext(c)
	})

	portfolioGroup := router.Group("/portfolios/:id")
	handler.RegisterRoutes(portfolioGroup)

	// Create request with .txt file
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	filePart, _ := writer.CreateFormFile("file", "test.txt")
	filePart.Write([]byte("some data"))
	writer.Close()

	req, _ := http.NewRequest("POST", "/portfolios/port-456/import", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for non-CSV file, got %d", w.Code)
	}
}

// Helper
func decPtr(d decimal.Decimal) *decimal.Decimal {
	return &d
}
