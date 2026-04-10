package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/huchknows/fintech/backend/internal/model"
	"github.com/huchknows/fintech/backend/internal/repository"
)

// TransactionService handles transaction business logic.
type TransactionService interface {
	Create(ctx context.Context, callerID, portfolioID string, in model.CreateTransactionInput) (*model.Transaction, error)
	List(ctx context.Context, callerID, portfolioID string) ([]*model.Transaction, error)
	Delete(ctx context.Context, callerID, transactionID string) error
}

type transactionService struct {
	repo     repository.TransactionRepository
	portRepo repository.PortfolioRepository
}

// NewTransactionService returns a TransactionService.
func NewTransactionService(repo repository.TransactionRepository, portRepo repository.PortfolioRepository) TransactionService {
	return &transactionService{repo: repo, portRepo: portRepo}
}

// Create records a new financial transaction after validating business rules.
// Ownership of portfolioID is enforced. Sell transactions are checked for
// sufficient holdings — negative positions are not permitted.
func (s *transactionService) Create(ctx context.Context, callerID, portfolioID string, in model.CreateTransactionInput) (*model.Transaction, error) {
	if err := s.assertOwnership(ctx, callerID, portfolioID); err != nil {
		return nil, err
	}

	if err := validateTransactionInput(in); err != nil {
		return nil, err
	}

	if in.TransactionType == model.TransactionTypeSell {
		if err := s.checkSufficientHoldings(ctx, portfolioID, in); err != nil {
			return nil, err
		}
	}

	return s.repo.Create(ctx, portfolioID, in)
}

// List returns all transactions for a portfolio, enforcing ownership.
func (s *transactionService) List(ctx context.Context, callerID, portfolioID string) ([]*model.Transaction, error) {
	if err := s.assertOwnership(ctx, callerID, portfolioID); err != nil {
		return nil, err
	}
	txns, err := s.repo.ListByPortfolioID(ctx, portfolioID)
	if err != nil {
		return nil, err
	}
	if txns == nil {
		return []*model.Transaction{}, nil
	}
	return txns, nil
}

// Delete removes a transaction after verifying the caller owns the parent portfolio.
func (s *transactionService) Delete(ctx context.Context, callerID, transactionID string) error {
	txn, err := s.repo.GetByID(ctx, transactionID)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			return model.NewNotFound("transaction")
		}
		return err
	}
	if err := s.assertOwnership(ctx, callerID, txn.PortfolioID); err != nil {
		return err
	}
	return s.repo.Delete(ctx, transactionID)
}

// assertOwnership returns ErrForbidden if callerID does not own portfolioID.
func (s *transactionService) assertOwnership(ctx context.Context, callerID, portfolioID string) error {
	p, err := s.portRepo.GetByID(ctx, portfolioID)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			return model.NewNotFound("portfolio")
		}
		return err
	}
	if p.UserID != callerID {
		return model.NewForbidden()
	}
	return nil
}

// checkSufficientHoldings returns ErrConflict when a sell would result in
// a negative position.
func (s *transactionService) checkSufficientHoldings(ctx context.Context, portfolioID string, in model.CreateTransactionInput) error {
	held, err := s.repo.QuantityHeld(ctx, portfolioID, in.Symbol)
	if err != nil {
		return err
	}
	if in.Quantity.GreaterThan(held) {
		return model.NewConflict(fmt.Sprintf(
			"cannot sell %.8s %s: only %.8s held",
			in.Quantity.String(), in.Symbol, held.String(),
		))
	}
	return nil
}

// validateTransactionInput enforces field presence rules per transaction type.
func validateTransactionInput(in model.CreateTransactionInput) error {
	if !in.TransactionType.IsValid() {
		return model.NewValidation(fmt.Sprintf("unknown transaction_type %q", in.TransactionType))
	}

	switch in.TransactionType {
	case model.TransactionTypeBuy, model.TransactionTypeSell, model.TransactionTypeReinvestedDividend:
		if in.Quantity == nil {
			return model.NewValidation("quantity is required for " + string(in.TransactionType))
		}
		if in.PricePerShare == nil {
			return model.NewValidation("price_per_share is required for " + string(in.TransactionType))
		}
	case model.TransactionTypeDividend:
		if in.DividendPerShare == nil {
			return model.NewValidation("dividend_per_share is required for dividend")
		}
	}

	if in.TransactionType == model.TransactionTypeReinvestedDividend && in.DividendPerShare == nil {
		return model.NewValidation("dividend_per_share is required for reinvested_dividend")
	}

	return nil
}
