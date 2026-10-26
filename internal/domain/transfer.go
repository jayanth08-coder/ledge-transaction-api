package domain

import "time"

// Transfer records an atomic money movement between two accounts.
// ReferenceID is globally unique so a repeated transfer submission cannot
// re-apply the transfer under a different combination of per-account refs.
type Transfer struct {
	ID            string    `gorm:"primaryKey;size:36" json:"id"`
	FromAccountID string    `gorm:"not null;size:36;index" json:"from_account_id"`
	ToAccountID   string    `gorm:"not null;size:36;index" json:"to_account_id"`
	Amount        int64     `gorm:"not null" json:"amount"`
	ReferenceID   string    `gorm:"not null;size:128;uniqueIndex" json:"reference_id"`
	Description   string    `gorm:"size:500" json:"description"`
	DebitTxnID    string    `gorm:"not null;size:36" json:"debit_txn_id"`
	CreditTxnID   string    `gorm:"not null;size:36" json:"credit_txn_id"`
	CreatedAt     time.Time `gorm:"not null;autoCreateTime:false" json:"created_at"`
}

func (Transfer) TableName() string { return "transfers" }

// CreateTransferRequest is the client payload for transfers.
type CreateTransferRequest struct {
	FromAccountID string `json:"from_account_id" form:"from_account_id"`
	ToAccountID   string `json:"to_account_id" form:"to_account_id"`
	Amount        int64  `json:"amount" form:"amount"`
	ReferenceID   string `json:"reference_id" form:"reference_id"`
	Description   string `json:"description" form:"description"`
}

// TransferResponse is the public transfer representation.
type TransferResponse struct {
	ID            string    `json:"id"`
	FromAccountID string    `json:"from_account_id"`
	ToAccountID   string    `json:"to_account_id"`
	Amount        int64     `json:"amount"`
	ReferenceID   string    `json:"reference_id"`
	Description   string    `json:"description"`
	DebitTxnID    string    `json:"debit_txn_id"`
	CreditTxnID   string    `json:"credit_txn_id"`
	CreatedAt     time.Time `json:"created_at"`
}

// ToResponse maps a Transfer to its API response shape.
func (t Transfer) ToResponse() TransferResponse {
	return TransferResponse{
		ID:            t.ID,
		FromAccountID: t.FromAccountID,
		ToAccountID:   t.ToAccountID,
		Amount:        t.Amount,
		ReferenceID:   t.ReferenceID,
		Description:   t.Description,
		DebitTxnID:    t.DebitTxnID,
		CreditTxnID:   t.CreditTxnID,
		CreatedAt:     t.CreatedAt.UTC(),
	}
}
