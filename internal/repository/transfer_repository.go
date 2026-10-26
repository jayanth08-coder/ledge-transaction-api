package repository

import (
	"github.com/ledge/ledge-transaction-api/internal/domain"
	"gorm.io/gorm"
)

// TransferRepository encapsulates transfer persistence.
type TransferRepository struct {
	db *gorm.DB
}

func NewTransferRepository(db *gorm.DB) *TransferRepository {
	return &TransferRepository{db: db}
}

func (r *TransferRepository) Create(tx *gorm.DB, transfer *domain.Transfer) error {
	return tx.Create(transfer).Error
}

func (r *TransferRepository) FindByReferenceID(ref string) (*domain.Transfer, error) {
	var t domain.Transfer
	if err := r.db.Where("reference_id = ?", ref).First(&t).Error; err != nil {
		return nil, err
	}
	return &t, nil
}
