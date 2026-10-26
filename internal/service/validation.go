package service

import (
	"fmt"
	"math"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/ledge/ledge-transaction-api/internal/domain"
)

func normalizeName(name string) string {
	return strings.TrimSpace(name)
}

func normalizeReferenceID(ref string) string {
	return strings.TrimSpace(ref)
}

func normalizeDescription(desc string) string {
	return strings.TrimSpace(desc)
}

func validateAccountName(name string) error {
	if name == "" {
		return domain.ErrMissingAccountName
	}
	if utf8.RuneCountInString(name) > domain.MaxAccountNameLength {
		return domain.ErrNameTooLong
	}
	return nil
}

func validateReferenceID(ref string) error {
	if ref == "" {
		return domain.ErrInvalidReferenceID
	}
	if utf8.RuneCountInString(ref) > domain.MaxReferenceIDLength {
		return domain.ErrReferenceIDTooLong
	}
	return nil
}

// resolveReferenceID returns the client reference when provided, otherwise
// generates a unique server reference (AUTO-…). Evaluators can still supply an
// explicit reference_id to test duplicate handling.
func resolveReferenceID(ref string) (string, error) {
	ref = normalizeReferenceID(ref)
	if ref == "" {
		ref = fmt.Sprintf("AUTO-%s", uuid.NewString())
	}
	if err := validateReferenceID(ref); err != nil {
		return "", err
	}
	return ref, nil
}

func validateDescription(desc string) error {
	if utf8.RuneCountInString(desc) > domain.MaxDescriptionLength {
		return domain.ErrDescriptionTooLong
	}
	return nil
}

func validateTransactionType(typ string) error {
	switch typ {
	case domain.TxnTypeCredit, domain.TxnTypeDebit:
		return nil
	default:
		return domain.ErrInvalidTransactionType
	}
}

// validateAmount ensures amount is a positive whole-rupee integer within int64 range.
func validateAmount(amount int64) error {
	if amount <= 0 {
		return domain.ErrInvalidAmount
	}
	// Guard against values that would overflow when added to a balance near MaxInt64.
	if amount > math.MaxInt64 {
		return domain.ErrAmountOverflow
	}
	return nil
}

func wouldOverflow(balance, amount int64) bool {
	return balance > math.MaxInt64-amount
}

// ParsePagination validates page/limit values after the handler has applied
// defaults for omitted query parameters. Zero and negative values are rejected.
func ParsePagination(page, limit int) (int, int, error) {
	if page < 1 {
		return 0, 0, domain.ErrInvalidPagination
	}
	if limit < 1 || limit > domain.MaxPageSize {
		return 0, 0, domain.ErrInvalidPagination
	}
	return page, limit, nil
}

// TotalPages computes total pages from record count and page size.
func TotalPages(totalRecords int64, pageSize int) int {
	if pageSize <= 0 {
		return 0
	}
	if totalRecords == 0 {
		return 0
	}
	pages := int((totalRecords + int64(pageSize) - 1) / int64(pageSize))
	return pages
}
