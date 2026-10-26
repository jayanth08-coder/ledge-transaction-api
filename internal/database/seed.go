package database

import (
	"fmt"
	"time"

	"github.com/ledge/ledge-transaction-api/internal/domain"
	"gorm.io/gorm"
)

// Deterministic demo account IDs (stable for evaluators / docs).
const (
	demoAccountAliceID   = "demo-account-alice"
	demoAccountBobID     = "demo-account-bob"
	demoAccountCaraID    = "demo-account-cara"
	demoAccountDevID     = "demo-account-dev"
	demoAccountEvaID     = "demo-account-eva"
	demoAccountFinanceID = "demo-account-finance"
)

// SeedDemo inserts predictable demo accounts and transactions when SEED_DEMO=true.
// Seeding is idempotent: repeated starts do not duplicate transactions or alter balances.
//
// Demo snapshot (whole rupees):
//
//	Alice    120  — credit 100, credit 50, debit 30
//	Bob       80  — credit 50, credit 40, debit 10
//	Cara      25  — credit 25
//	Dev        0  — empty ledger (zero balance)
//	Eva      200  — credit 200
//	Finance  150  — credit 100, credit 75, debit 25
func SeedDemo(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		now := time.Now().UTC()

		accounts := []struct {
			id, name string
		}{
			{demoAccountAliceID, "Alice Sharma"},
			{demoAccountBobID, "Bob Kumar"},
			{demoAccountCaraID, "Cara Patel"},
			{demoAccountDevID, "Dev Ops Ledger"},
			{demoAccountEvaID, "Eva Nair"},
			{demoAccountFinanceID, "Finance Pool"},
		}
		for _, a := range accounts {
			if err := ensureDemoAccount(tx, a.id, a.name, now); err != nil {
				return err
			}
		}

		var count int64
		if err := tx.Model(&domain.Transaction{}).Where("account_id = ?", demoAccountAliceID).Count(&count).Error; err != nil {
			return fmt.Errorf("count demo transactions: %w", err)
		}
		if count > 0 {
			return nil
		}

		seeds := []struct {
			id, accountID, typ, desc, ref string
			amount                        int64
			offset                        time.Duration
		}{
			{"demo-txn-alice-1", demoAccountAliceID, domain.TxnTypeCredit, "Initial deposit", "DEMO-ALICE-001", 100, -6 * time.Hour},
			{"demo-txn-alice-2", demoAccountAliceID, domain.TxnTypeCredit, "Salary credit", "DEMO-ALICE-002", 50, -5 * time.Hour},
			{"demo-txn-alice-3", demoAccountAliceID, domain.TxnTypeDebit, "Grocery purchase", "DEMO-ALICE-003", 30, -4 * time.Hour},

			{"demo-txn-bob-1", demoAccountBobID, domain.TxnTypeCredit, "Opening deposit", "DEMO-BOB-001", 50, -5*time.Hour - 30*time.Minute},
			{"demo-txn-bob-2", demoAccountBobID, domain.TxnTypeCredit, "Freelance payment", "DEMO-BOB-002", 40, -3 * time.Hour},
			{"demo-txn-bob-3", demoAccountBobID, domain.TxnTypeDebit, "Utility bill", "DEMO-BOB-003", 10, -2 * time.Hour},

			{"demo-txn-cara-1", demoAccountCaraID, domain.TxnTypeCredit, "Pocket money", "DEMO-CARA-001", 25, -90 * time.Minute},

			{"demo-txn-eva-1", demoAccountEvaID, domain.TxnTypeCredit, "Bonus credit", "DEMO-EVA-001", 200, -4*time.Hour - 15*time.Minute},

			{"demo-txn-fin-1", demoAccountFinanceID, domain.TxnTypeCredit, "Budget allocation", "DEMO-FIN-001", 100, -7 * time.Hour},
			{"demo-txn-fin-2", demoAccountFinanceID, domain.TxnTypeCredit, "Vendor refund", "DEMO-FIN-002", 75, -3*time.Hour - 20*time.Minute},
			{"demo-txn-fin-3", demoAccountFinanceID, domain.TxnTypeDebit, "Office supplies", "DEMO-FIN-003", 25, -45 * time.Minute},
		}

		for _, s := range seeds {
			txn := domain.Transaction{
				ID:          s.id,
				AccountID:   s.accountID,
				Type:        s.typ,
				Amount:      s.amount,
				Description: s.desc,
				ReferenceID: s.ref,
				CreatedAt:   now.Add(s.offset),
			}
			if err := tx.Create(&txn).Error; err != nil {
				return fmt.Errorf("seed transaction %s: %w", s.id, err)
			}

			var expr string
			if s.typ == domain.TxnTypeCredit {
				expr = "balance + ?"
			} else {
				expr = "balance - ?"
			}
			res := tx.Model(&domain.Account{}).Where("id = ?", s.accountID).
				Update("balance", gorm.Expr(expr, s.amount))
			if res.Error != nil {
				return fmt.Errorf("update demo balance for %s: %w", s.accountID, res.Error)
			}
		}

		return nil
	})
}

func ensureDemoAccount(tx *gorm.DB, id, name string, now time.Time) error {
	var existing domain.Account
	err := tx.Where("id = ?", id).First(&existing).Error
	if err == nil {
		return nil
	}
	if err != gorm.ErrRecordNotFound {
		return fmt.Errorf("lookup demo account %s: %w", id, err)
	}
	acc := domain.Account{
		ID:        id,
		Name:      name,
		Balance:   0,
		CreatedAt: now,
	}
	if err := tx.Create(&acc).Error; err != nil {
		return fmt.Errorf("create demo account %s: %w", id, err)
	}
	return nil
}

// DemoAccountIDs returns primary demo account identifiers for documentation/tests.
func DemoAccountIDs() (alice, bob string) {
	return demoAccountAliceID, demoAccountBobID
}
