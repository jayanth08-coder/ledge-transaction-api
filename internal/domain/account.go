package domain

import "time"

// Money is stored as whole rupees (₹) using int64 — never floating point.
// Example: amount 100 means ₹100.
const (
	MaxAccountNameLength = 255
	MaxDescriptionLength = 500
	MaxReferenceIDLength = 128

	DefaultPage     = 1
	DefaultPageSize = 3
	MaxPageSize     = 100

	TxnTypeCredit = "credit"
	TxnTypeDebit  = "debit"
)

// Account is the persisted account model.
type Account struct {
	ID        string    `gorm:"primaryKey;size:36" json:"id"`
	Name      string    `gorm:"not null;size:255" json:"name"`
	Balance   int64     `gorm:"not null;default:0" json:"balance"`
	CreatedAt time.Time `gorm:"not null;autoCreateTime:false" json:"created_at"`
}

func (Account) TableName() string { return "accounts" }

// CreateAccountRequest is the client payload for account creation.
// Protected fields (id, balance, created_at) are intentionally absent.
type CreateAccountRequest struct {
	Name string `json:"name" form:"name"`
}

// AccountResponse is the public account representation.
type AccountResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Balance   int64     `json:"balance"`
	CreatedAt time.Time `json:"created_at"`
}

// BalanceResponse is returned by GET /api/accounts/:id/balance.
type BalanceResponse struct {
	AccountID string `json:"account_id"`
	Balance   int64  `json:"balance"`
}

// ToResponse maps an Account to its API response shape.
func (a Account) ToResponse() AccountResponse {
	return AccountResponse{
		ID:        a.ID,
		Name:      a.Name,
		Balance:   a.Balance,
		CreatedAt: a.CreatedAt.UTC(),
	}
}

// PaginationMeta describes pagination state for list endpoints.
type PaginationMeta struct {
	CurrentPage  int   `json:"current_page"`
	PageSize     int   `json:"page_size"`
	TotalRecords int64 `json:"total_records"`
	TotalPages   int   `json:"total_pages"`
}

// PaginatedAccounts wraps a paginated account list.
type PaginatedAccounts struct {
	Data       []AccountResponse `json:"data"`
	Pagination PaginationMeta    `json:"pagination"`
}
