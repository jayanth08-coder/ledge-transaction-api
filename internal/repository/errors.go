package repository

import (
	"errors"
	"strings"

	"github.com/ledge/ledge-transaction-api/internal/domain"
	"gorm.io/gorm"
)

// IsUniqueViolation reports whether err is a SQLite unique constraint violation.
func IsUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "unique constraint failed") ||
		strings.Contains(msg, "constraint failed") && strings.Contains(msg, "unique")
}

// IsBusyError reports whether err is a SQLite busy/locked error.
func IsBusyError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "database is locked") ||
		strings.Contains(msg, "sqlite_busy") ||
		strings.Contains(msg, "busy")
}

// MapDBError converts low-level DB errors into domain errors where possible.
func MapDBError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return domain.ErrAccountNotFound
	}
	if IsBusyError(err) {
		return domain.ErrDatabaseUnavailable
	}
	return err
}

// UniqueField indicates which unique constraint was violated based on the error text.
func UniqueField(err error) string {
	if err == nil {
		return ""
	}
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "idx_account_reference") || strings.Contains(msg, "reference_id"):
		if strings.Contains(msg, "transfers") {
			return "transfer_reference"
		}
		return "account_reference"
	case strings.Contains(msg, "transfers") && strings.Contains(msg, "reference"):
		return "transfer_reference"
	default:
		return "unknown"
	}
}
