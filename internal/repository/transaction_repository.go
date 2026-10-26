package repository

import (
	"github.com/ledge/ledge-transaction-api/internal/domain"
	"gorm.io/gorm"
)

// TransactionRepository encapsulates transaction persistence.
type TransactionRepository struct {
	db *gorm.DB
}

func NewTransactionRepository(db *gorm.DB) *TransactionRepository {
	return &TransactionRepository{db: db}
}

func (r *TransactionRepository) Create(tx *gorm.DB, txn *domain.Transaction) error {
	return tx.Create(txn).Error
}

func (r *TransactionRepository) FindByID(id string) (*domain.Transaction, error) {
	var txn domain.Transaction
	if err := r.db.Where("id = ?", id).First(&txn).Error; err != nil {
		return nil, err
	}
	return &txn, nil
}

func (r *TransactionRepository) CountAll() (int64, error) {
	var total int64
	err := r.db.Model(&domain.Transaction{}).Count(&total).Error
	return total, err
}

// ListByAccount returns filtered, sorted, paginated transactions and the filtered total count.
func (r *TransactionRepository) ListByAccount(q domain.TransactionListQuery) ([]domain.Transaction, int64, error) {
	base := r.db.Model(&domain.Transaction{}).Where("account_id = ?", q.AccountID)
	if q.Type != "" {
		base = base.Where("type = ?", q.Type)
	}

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var items []domain.Transaction
	offset := (q.Page - 1) * q.Limit
	query := r.db.Where("account_id = ?", q.AccountID)
	if q.Type != "" {
		query = query.Where("type = ?", q.Type)
	}
	if err := query.Order("created_at DESC, id DESC").
		Limit(q.Limit).Offset(offset).
		Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}
