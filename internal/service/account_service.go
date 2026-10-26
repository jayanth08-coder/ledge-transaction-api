package service

import (
	"time"

	"github.com/google/uuid"
	"github.com/ledge/ledge-transaction-api/internal/domain"
	"github.com/ledge/ledge-transaction-api/internal/repository"
)

// AccountService implements account business rules.
type AccountService struct {
	accounts *repository.AccountRepository
}

func NewAccountService(accounts *repository.AccountRepository) *AccountService {
	return &AccountService{accounts: accounts}
}

func (s *AccountService) Create(req domain.CreateAccountRequest) (*domain.Account, error) {
	name := normalizeName(req.Name)
	if err := validateAccountName(name); err != nil {
		return nil, err
	}

	account := &domain.Account{
		ID:        uuid.NewString(),
		Name:      name,
		Balance:   0,
		CreatedAt: time.Now().UTC(),
	}

	if err := s.accounts.Create(account); err != nil {
		return nil, err
	}
	return account, nil
}

func (s *AccountService) GetByID(id string) (*domain.Account, error) {
	if id == "" {
		return nil, domain.ErrAccountNotFound
	}
	return s.accounts.FindByID(id)
}

func (s *AccountService) GetBalance(id string) (*domain.BalanceResponse, error) {
	account, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}
	return &domain.BalanceResponse{
		AccountID: account.ID,
		Balance:   account.Balance,
	}, nil
}

func (s *AccountService) List(page, limit int) (*domain.PaginatedAccounts, error) {
	page, limit, err := ParsePagination(page, limit)
	if err != nil {
		return nil, err
	}

	accounts, total, err := s.accounts.List(page, limit)
	if err != nil {
		return nil, err
	}

	data := make([]domain.AccountResponse, 0, len(accounts))
	for _, a := range accounts {
		data = append(data, a.ToResponse())
	}

	return &domain.PaginatedAccounts{
		Data: data,
		Pagination: domain.PaginationMeta{
			CurrentPage:  page,
			PageSize:     limit,
			TotalRecords: total,
			TotalPages:   TotalPages(total, limit),
		},
	}, nil
}

func (s *AccountService) Count() (int64, error) {
	return s.accounts.Count()
}
