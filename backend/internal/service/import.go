package service

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"log/slog"

	"github.com/huchknows/fintech/backend/internal/model"
	"github.com/huchknows/fintech/backend/internal/service/csvparser"
)

const (
	maxFileSize      = 5 * 1024 * 1024 // 5 MB
	maxRowsPerImport = 1000
)

// ImportService handles CSV import operations.
type ImportService interface {
	Preview(ctx context.Context, callerID, portfolioID string, csvData io.Reader, brokerage string) (*model.ImportPreview, error)
	Confirm(ctx context.Context, callerID, portfolioID string, req model.ImportConfirmRequest) (*model.ImportResult, error)
}

type importService struct {
	transactionService TransactionService
	portfolioService   PortfolioService
	logger             *slog.Logger
}

// NewImportService returns an ImportService.
func NewImportService(txnSvc TransactionService, portSvc PortfolioService, logger *slog.Logger) ImportService {
	return &importService{
		transactionService: txnSvc,
		portfolioService:   portSvc,
		logger:             logger,
	}
}

// Preview parses and validates a CSV file without persisting transactions.
// Returns a preview of parsed, valid, and error rows.
func (s *importService) Preview(ctx context.Context, callerID, portfolioID string, csvData io.Reader, brokerage string) (*model.ImportPreview, error) {
	// Verify portfolio ownership
	portfolio, err := s.portfolioService.GetByID(ctx, callerID, portfolioID)
	if err != nil {
		return nil, err
	}
	if portfolio == nil {
		return nil, model.NewNotFound("portfolio")
	}

	result := &model.ImportPreview{
		Transactions: make([]model.CreateTransactionInput, 0),
		Errors:       make([]model.ImportError, 0),
	}

	// Parse CSV
	reader := csv.NewReader(csvData)
	headers, err := reader.Read()
	if err != nil {
		return nil, model.NewValidation("invalid CSV: unable to read header row")
	}

	// Build header index (case-insensitive)
	headerIndex := make(map[string]int)
	for i, h := range headers {
		headerIndex[h] = i
	}

	// Select parser
	parser := csvparser.GetParser(headers, brokerage)

	rowNum := 1 // Start at 1 (header is row 0)
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			result.Errors = append(result.Errors, model.ImportError{
				Row:     rowNum + 1,
				Message: fmt.Sprintf("CSV read error: %v", err),
			})
			rowNum++
			continue
		}

		rowNum++
		result.Parsed++

		// Check row limit
		if result.Parsed > maxRowsPerImport {
			result.Errors = append(result.Errors, model.ImportError{
				Row:     rowNum,
				Message: fmt.Sprintf("file exceeds maximum of %d rows", maxRowsPerImport),
			})
			break
		}

		// Parse row with brokerage parser
		importRow, err := parser.Parse(record, headerIndex)
		if err != nil {
			result.Errors = append(result.Errors, model.ImportError{
				Row:     rowNum,
				Message: fmt.Sprintf("parse error: %v", err),
			})
			continue
		}

		// Normalize to CreateTransactionInput
		txnInput, err := csvparser.NormalizeRow(importRow)
		if err != nil {
			result.Errors = append(result.Errors, model.ImportError{
				Row:     rowNum,
				Message: fmt.Sprintf("validation error: %v", err),
			})
			continue
		}

		result.Valid++
		result.Transactions = append(result.Transactions, txnInput)
	}

	return result, nil
}

// Confirm persists a set of validated transactions to the database.
// Each transaction is created via the existing Create flow, which enforces ownership and business rules.
func (s *importService) Confirm(ctx context.Context, callerID, portfolioID string, req model.ImportConfirmRequest) (*model.ImportResult, error) {
	// Verify portfolio ownership
	portfolio, err := s.portfolioService.GetByID(ctx, callerID, portfolioID)
	if err != nil {
		return nil, err
	}
	if portfolio == nil {
		return nil, model.NewNotFound("portfolio")
	}

	result := &model.ImportResult{
		Errors:   make([]model.ImportError, 0),
		Messages: make([]string, 0),
	}

	// Create each transaction
	for i, txnInput := range req.Transactions {
		_, err := s.transactionService.Create(ctx, callerID, portfolioID, txnInput)
		if err != nil {
			// Collect error but continue processing
			result.Failed++
			result.Errors = append(result.Errors, model.ImportError{
				Row:     i + 1,
				Message: fmt.Sprintf("create failed: %v", err),
			})
			s.logger.Warn("import transaction creation failed",
				"portfolio_id", portfolioID,
				"row", i+1,
				"symbol", txnInput.Symbol,
				"error", err,
			)
			continue
		}
		result.Created++
	}

	if result.Created > 0 {
		result.Messages = append(result.Messages,
			fmt.Sprintf("Successfully imported %d transactions", result.Created))
	}
	if result.Failed > 0 {
		result.Messages = append(result.Messages,
			fmt.Sprintf("Failed to import %d transactions (see errors for details)", result.Failed))
	}

	return result, nil
}
