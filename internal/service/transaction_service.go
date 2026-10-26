package service

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/ledge/ledge-transaction-api/internal/domain"
	"github.com/ledge/ledge-transaction-api/internal/repository"
	"gorm.io/gorm"
)

// TransactionService implements credit/debit business rules with atomic updates.
type TransactionService struct {
	db       *gorm.DB
	accounts *repository.AccountRepository
	txns     *repository.TransactionRepository
}

func NewTransactionService(
	db *gorm.DB,
	accounts *repository.AccountRepository,
	txns *repository.TransactionRepository,
) *TransactionService {
	return &TransactionService{db: db, accounts: accounts, txns: txns}
}

func (s *TransactionService) Create(req domain.CreateTransactionRequest) (*domain.Transaction, error) {
	accountID := stringsTrim(req.AccountID)
	typ := stringsTrim(req.Type)
	desc := normalizeDescription(req.Description)
	amount := req.Amount

	if accountID == "" {
		return nil, domain.ErrAccountNotFound
	}
	if err := validateTransactionType(typ); err != nil {
		return nil, err
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

	var created *domain.Transaction

	err = s.db.Transaction(func(tx *gorm.DB) error {
		account, err := s.accounts.LockForUpdate(tx, accountID)
		if err != nil {
			return err
		}

		if typ == domain.TxnTypeCredit && wouldOverflow(account.Balance, amount) {
			return domain.ErrAmountOverflow
		}

		txn := &domain.Transaction{
			ID:          uuid.NewString(),
			AccountID:   accountID,
			Type:        typ,
			Amount:      amount,
			Description: desc,
			ReferenceID: ref,
			CreatedAt:   time.Now().UTC(),
		}

		// Insert first so unique (account_id, reference_id) is enforced by the DB.
		if err := s.txns.Create(tx, txn); err != nil {
			if repository.IsUniqueViolation(err) {
				return domain.ErrDuplicateReferenceID
			}
			return err
		}

		var ok bool
		switch typ {
		case domain.TxnTypeCredit:
			ok, err = s.accounts.CreditBalance(tx, accountID, amount)
		case domain.TxnTypeDebit:
			ok, err = s.accounts.DebitBalance(tx, accountID, amount)
		}
		if err != nil {
			return err
		}
		if !ok {
			// Account vanished or insufficient funds for debit.
			if typ == domain.TxnTypeDebit {
				return domain.ErrInsufficientBalance
			}
			return domain.ErrAccountNotFound
		}

		created = txn
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

func (s *TransactionService) List(accountID string, typ string, page, limit int) (*domain.PaginatedTransactions, error) {
	accountID = stringsTrim(accountID)
	if accountID == "" {
		return nil, domain.ErrAccountNotFound
	}

	exists, err := s.accounts.Exists(accountID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, domain.ErrAccountNotFound
	}

	typ = stringsTrim(typ)
	if typ != "" {
		if err := validateTransactionType(typ); err != nil {
			return nil, domain.ErrInvalidFilterType
		}
	}

	page, limit, err = ParsePagination(page, limit)
	if err != nil {
		return nil, err
	}

	items, total, err := s.txns.ListByAccount(domain.TransactionListQuery{
		AccountID: accountID,
		Type:      typ,
		Page:      page,
		Limit:     limit,
	})
	if err != nil {
		return nil, err
	}

	data := make([]domain.TransactionResponse, 0, len(items))
	for _, t := range items {
		data = append(data, t.ToResponse())
	}

	return &domain.PaginatedTransactions{
		Data: data,
		Pagination: domain.PaginationMeta{
			CurrentPage:  page,
			PageSize:     limit,
			TotalRecords: total,
			TotalPages:   TotalPages(total, limit),
		},
	}, nil
}

func (s *TransactionService) CountAll() (int64, error) {
	return s.txns.CountAll()
}

func stringsTrim(s string) string {
	return strings.TrimSpace(s)
}
