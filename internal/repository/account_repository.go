package repository

import (
	"github.com/ledge/ledge-transaction-api/internal/domain"
	"gorm.io/gorm"
)

// AccountRepository encapsulates account persistence.
type AccountRepository struct {
	db *gorm.DB
}

func NewAccountRepository(db *gorm.DB) *AccountRepository {
	return &AccountRepository{db: db}
}

func (r *AccountRepository) Create(account *domain.Account) error {
	return r.db.Create(account).Error
}

func (r *AccountRepository) FindByID(id string) (*domain.Account, error) {
	var account domain.Account
	if err := r.db.Where("id = ?", id).First(&account).Error; err != nil {
		return nil, MapDBError(err)
	}
	return &account, nil
}

func (r *AccountRepository) Exists(id string) (bool, error) {
	var count int64
	if err := r.db.Model(&domain.Account{}).Where("id = ?", id).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *AccountRepository) List(page, limit int) ([]domain.Account, int64, error) {
	var total int64
	if err := r.db.Model(&domain.Account{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var accounts []domain.Account
	offset := (page - 1) * limit
	if err := r.db.Order("created_at DESC, id DESC").
		Limit(limit).Offset(offset).
		Find(&accounts).Error; err != nil {
		return nil, 0, err
	}
	return accounts, total, nil
}

func (r *AccountRepository) Count() (int64, error) {
	var total int64
	err := r.db.Model(&domain.Account{}).Count(&total).Error
	return total, err
}

// LockForUpdate loads the account inside an open DB transaction.
func (r *AccountRepository) LockForUpdate(tx *gorm.DB, id string) (*domain.Account, error) {
	var account domain.Account
	if err := tx.Where("id = ?", id).First(&account).Error; err != nil {
		return nil, MapDBError(err)
	}
	return &account, nil
}

func (r *AccountRepository) CreditBalance(tx *gorm.DB, accountID string, amount int64) (bool, error) {
	res := tx.Model(&domain.Account{}).
		Where("id = ?", accountID).
		Update("balance", gorm.Expr("balance + ?", amount))
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected == 1, nil
}

func (r *AccountRepository) DebitBalance(tx *gorm.DB, accountID string, amount int64) (bool, error) {
	res := tx.Model(&domain.Account{}).
		Where("id = ? AND balance >= ?", accountID, amount).
		Update("balance", gorm.Expr("balance - ?", amount))
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected == 1, nil
}
