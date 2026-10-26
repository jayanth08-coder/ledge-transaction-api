package domain

import "errors"

// Sentinel / typed domain errors mapped to HTTP responses in the handler layer.
var (
	ErrAccountNotFound         = errors.New("account not found")
	ErrMissingAccountName      = errors.New("account name is required")
	ErrInvalidTransactionType  = errors.New("transaction type must be credit or debit")
	ErrInvalidAmount           = errors.New("amount must be a positive whole-rupee integer")
	ErrAmountOverflow          = errors.New("amount or resulting balance exceeds supported integer range")
	ErrInvalidReferenceID      = errors.New("reference_id is required")
	ErrDuplicateReferenceID    = errors.New("reference ID already exists for the account")
	ErrInsufficientBalance     = errors.New("insufficient balance")
	ErrInvalidPagination       = errors.New("invalid pagination parameters")
	ErrInvalidRequestBody      = errors.New("invalid request body")
	ErrInvalidTransfer         = errors.New("invalid transfer request")
	ErrSameAccountTransfer     = errors.New("source and destination accounts must be different")
	ErrDuplicateTransferRef    = errors.New("transfer reference ID already exists")
	ErrDescriptionTooLong      = errors.New("description exceeds maximum length")
	ErrNameTooLong             = errors.New("account name exceeds maximum length")
	ErrReferenceIDTooLong      = errors.New("reference_id exceeds maximum length")
	ErrInvalidFilterType       = errors.New("invalid transaction type filter")
	ErrDatabaseUnavailable     = errors.New("database temporarily unavailable")
	ErrDatabaseTransactionFail = errors.New("database transaction failed")
)

// APIErrorCode is a stable machine-readable error code returned to clients.
type APIErrorCode string

const (
	CodeAccountNotFound        APIErrorCode = "ACCOUNT_NOT_FOUND"
	CodeMissingAccountName     APIErrorCode = "MISSING_ACCOUNT_NAME"
	CodeInvalidTransactionType APIErrorCode = "INVALID_TRANSACTION_TYPE"
	CodeInvalidAmount          APIErrorCode = "INVALID_AMOUNT"
	CodeAmountOverflow         APIErrorCode = "AMOUNT_OVERFLOW"
	CodeInvalidReferenceID     APIErrorCode = "INVALID_REFERENCE_ID"
	CodeDuplicateReferenceID   APIErrorCode = "DUPLICATE_REFERENCE_ID"
	CodeInsufficientBalance    APIErrorCode = "INSUFFICIENT_BALANCE"
	CodeInvalidPagination      APIErrorCode = "INVALID_PAGINATION"
	CodeInvalidRequestBody     APIErrorCode = "INVALID_REQUEST_BODY"
	CodeInvalidTransfer        APIErrorCode = "INVALID_TRANSFER"
	CodeSameAccountTransfer    APIErrorCode = "SAME_ACCOUNT_TRANSFER"
	CodeDuplicateTransferRef   APIErrorCode = "DUPLICATE_TRANSFER_REFERENCE"
	CodeDescriptionTooLong     APIErrorCode = "DESCRIPTION_TOO_LONG"
	CodeNameTooLong            APIErrorCode = "NAME_TOO_LONG"
	CodeReferenceIDTooLong     APIErrorCode = "REFERENCE_ID_TOO_LONG"
	CodeInvalidFilterType      APIErrorCode = "INVALID_FILTER_TYPE"
	CodeDatabaseUnavailable    APIErrorCode = "DATABASE_UNAVAILABLE"
	CodeInternalError          APIErrorCode = "INTERNAL_ERROR"
	CodeValidationError APIErrorCode = "VALIDATION_ERROR"
)

// FieldError describes a single field-level validation failure.
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ValidationError aggregates field-level validation failures.
type ValidationError struct {
	Message string
	Fields  []FieldError
}

func (e *ValidationError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return "validation failed"
}

// NewValidationError constructs a ValidationError with optional field details.
func NewValidationError(message string, fields ...FieldError) *ValidationError {
	return &ValidationError{Message: message, Fields: fields}
}
