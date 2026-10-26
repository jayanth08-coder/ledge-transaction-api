package database

import (
	"fmt"

	"github.com/ledge/ledge-transaction-api/internal/domain"
	"gorm.io/gorm"
)

// Migrate applies schema initialization via GORM AutoMigrate and supplemental
// CHECK constraints that AutoMigrate cannot express reliably for SQLite.
//
// Limitations of AutoMigrate:
//   - It adds missing tables/columns/indexes but does not drop or rename columns.
//   - It will not safely transform arbitrary existing schemas.
//   - Complex CHECK constraints may need explicit SQL (documented below).
//
// For this assessment AutoMigrate is sufficient for greenfield and additive changes.
// Existing data is preserved across restarts; balances are never reset.
func Migrate(db *gorm.DB) error {
	if err := db.AutoMigrate(
		&domain.Account{},
		&domain.Transaction{},
		&domain.Transfer{},
	); err != nil {
		return fmt.Errorf("automigrate failed: %w", err)
	}

	// Enforce positive amounts and valid transaction types at the DB level.
	// SQLite allows adding CHECK via table recreation; for assessment we use
	// triggers as a portable supplement when the table already exists.
	checks := []string{
		`CREATE TRIGGER IF NOT EXISTS trg_txn_amount_positive
			BEFORE INSERT ON transactions
			FOR EACH ROW
			WHEN NEW.amount <= 0
			BEGIN
				SELECT RAISE(ABORT, 'amount must be positive');
			END`,
		`CREATE TRIGGER IF NOT EXISTS trg_txn_type_valid
			BEFORE INSERT ON transactions
			FOR EACH ROW
			WHEN NEW.type NOT IN ('credit', 'debit')
			BEGIN
				SELECT RAISE(ABORT, 'invalid transaction type');
			END`,
		`CREATE TRIGGER IF NOT EXISTS trg_account_balance_nonneg
			BEFORE UPDATE OF balance ON accounts
			FOR EACH ROW
			WHEN NEW.balance < 0
			BEGIN
				SELECT RAISE(ABORT, 'balance cannot be negative');
			END`,
		`CREATE TRIGGER IF NOT EXISTS trg_transfer_amount_positive
			BEFORE INSERT ON transfers
			FOR EACH ROW
			WHEN NEW.amount <= 0
			BEGIN
				SELECT RAISE(ABORT, 'transfer amount must be positive');
			END`,
	}

	for _, stmt := range checks {
		if err := db.Exec(stmt).Error; err != nil {
			return fmt.Errorf("apply schema trigger: %w", err)
		}
	}

	return nil
}
