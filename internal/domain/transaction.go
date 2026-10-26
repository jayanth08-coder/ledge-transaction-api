package domain

import "time"

// Transaction is an immutable ledger entry.
type Transaction struct {
	ID          string    `gorm:"primaryKey;size:36" json:"id"`
	AccountID   string    `gorm:"not null;size:36;index:idx_txn_account_created,priority:1;uniqueIndex:idx_account_reference,priority:1" json:"account_id"`
	Type        string    `gorm:"not null;size:10;index" json:"type"`
	Amount      int64     `gorm:"not null" json:"amount"`
	Description string    `gorm:"size:500" json:"description"`
	ReferenceID string    `gorm:"not null;size:128;uniqueIndex:idx_account_reference,priority:2" json:"reference_id"`
	CreatedAt   time.Time `gorm:"not null;index:idx_txn_account_created,priority:2;autoCreateTime:false" json:"created_at"`
	Account     *Account  `gorm:"constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;foreignKey:AccountID;references:ID" json:"-"`
}

func (Transaction) TableName() string { return "transactions" }

// CreateTransactionRequest is the client payload for credit/debit.
// ID and created_at cannot be set by the client.
type CreateTransactionRequest struct {
	AccountID   string `json:"account_id" form:"account_id"`
	Type        string `json:"type" form:"type"`
	Amount      int64  `json:"amount" form:"amount"`
	Description string `json:"description" form:"description"`
	ReferenceID string `json:"reference_id" form:"reference_id"`
}

// TransactionResponse is the public transaction representation.
type TransactionResponse struct {
	ID          string    `json:"id"`
	AccountID   string    `json:"account_id"`
	Type        string    `json:"type"`
	Amount      int64     `json:"amount"`
	Description string    `json:"description"`
	ReferenceID string    `json:"reference_id"`
	CreatedAt   time.Time `json:"created_at"`
}

// ToResponse maps a Transaction to its API response shape.
func (t Transaction) ToResponse() TransactionResponse {
	return TransactionResponse{
		ID:          t.ID,
		AccountID:   t.AccountID,
		Type:        t.Type,
		Amount:      t.Amount,
		Description: t.Description,
		ReferenceID: t.ReferenceID,
		CreatedAt:   t.CreatedAt.UTC(),
	}
}

// TransactionListQuery holds validated filter/pagination parameters.
type TransactionListQuery struct {
	AccountID string
	Type      string // empty, "credit", or "debit"
	Page      int
	Limit     int
}

// PaginatedTransactions wraps a filtered, paginated transaction list.
type PaginatedTransactions struct {
	Data       []TransactionResponse `json:"data"`
	Pagination PaginationMeta        `json:"pagination"`
}
