package service

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/ledge/ledge-transaction-api/internal/domain"
	"github.com/ledge/ledge-transaction-api/internal/repository"
	"gorm.io/gorm"
)

// TransferService implements atomic, idempotent inter-account transfers.
//
// Idempotency policy:
//   - Transfer.ReferenceID is globally unique on the transfers table.
//   - Ledger entries use deterministic per-account references derived from the
//     transfer reference (…:debit / …:credit) so a retry cannot create a second
//     pair of ledger rows even if the transfer insert somehow raced.
//   - A duplicate transfer reference returns DUPLICATE_TRANSFER_REFERENCE (409)
//     without mutating balances.
type TransferService struct {
	db        *gorm.DB
	accounts  *repository.AccountRepository
	txns      *repository.TransactionRepository
	transfers *repository.TransferRepository
}

func NewTransferService(
	db *gorm.DB,
	accounts *repository.AccountRepository,
	txns *repository.TransactionRepository,
	transfers *repository.TransferRepository,
) *TransferService {
	return &TransferService{db: db, accounts: accounts, txns: txns, transfers: transfers}
}

func (s *TransferService) Create(req domain.CreateTransferRequest) (*domain.Transfer, error) {
	fromID := stringsTrim(req.FromAccountID)
	toID := stringsTrim(req.ToAccountID)
	desc := normalizeDescription(req.Description)
	amount := req.Amount

	if fromID == "" || toID == "" {
		return nil, domain.ErrInvalidTransfer
	}
	if fromID == toID {
		return nil, domain.ErrSameAccountTransfer
	}
	if err := validateAmount(amount); err != nil {
		return nil, err
	}
	ref, err := resolveReferenceID(req.ReferenceID)
	if err != nil {
		return nil, err
	}
	if err := validateDescription(desc); err != nil {
		return nil, err
	}

	var created *domain.Transfer

	err = s.db.Transaction(func(tx *gorm.DB) error {
		// Lock accounts in deterministic ID order to avoid deadlocks.
		firstID, secondID := fromID, toID
		if fromID > toID {
			firstID, secondID = toID, fromID
		}
		if _, err := s.accounts.LockForUpdate(tx, firstID); err != nil {
			return err
		}
		if _, err := s.accounts.LockForUpdate(tx, secondID); err != nil {
			return err
		}

		debitRef := fmt.Sprintf("%s:debit", ref)
		creditRef := fmt.Sprintf("%s:credit", ref)
		now := time.Now().UTC()
		transferID := uuid.NewString()
		debitID := uuid.NewString()
		creditID := uuid.NewString()

		debitTxn := &domain.Transaction{
			ID:          debitID,
			AccountID:   fromID,
			Type:        domain.TxnTypeDebit,
			Amount:      amount,
			Description: transferDescription(desc, "Transfer debit"),
			ReferenceID: debitRef,
			CreatedAt:   now,
		}
		creditTxn := &domain.Transaction{
			ID:          creditID,
			AccountID:   toID,
			Type:        domain.TxnTypeCredit,
			Amount:      amount,
			Description: transferDescription(desc, "Transfer credit"),
			ReferenceID: creditRef,
			CreatedAt:   now,
		}

		transfer := &domain.Transfer{
			ID:            transferID,
			FromAccountID: fromID,
			ToAccountID:   toID,
			Amount:        amount,
			ReferenceID:   ref,
			Description:   desc,
			DebitTxnID:    debitID,
			CreditTxnID:   creditID,
			CreatedAt:     now,
		}

		// Insert transfer first — global unique reference_id is the idempotency key.
		if err := s.transfers.Create(tx, transfer); err != nil {
			if repository.IsUniqueViolation(err) {
				return domain.ErrDuplicateTransferRef
			}
			return err
		}

		if err := s.txns.Create(tx, debitTxn); err != nil {
			if repository.IsUniqueViolation(err) {
				return domain.ErrDuplicateTransferRef
			}
			return err
		}
		ok, err := s.accounts.DebitBalance(tx, fromID, amount)
		if err != nil {
			return err
		}
		if !ok {
			return domain.ErrInsufficientBalance
		}

		if err := s.txns.Create(tx, creditTxn); err != nil {
			if repository.IsUniqueViolation(err) {
				return domain.ErrDuplicateTransferRef
			}
			return err
		}
		ok, err = s.accounts.CreditBalance(tx, toID, amount)
		if err != nil {
			return err
		}
		if !ok {
			return domain.ErrAccountNotFound
		}

		created = transfer
		return nil
	})

	if err != nil {
		if repository.IsBusyError(err) {
			return nil, domain.ErrDatabaseUnavailable
		}
		return nil, err
	}
	return created, nil
}

func transferDescription(userDesc, fallback string) string {
	if userDesc == "" {
		return fallback
	}
	return userDesc
}
