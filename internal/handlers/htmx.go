package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ledge/ledge-transaction-api/internal/domain"
	"github.com/ledge/ledge-transaction-api/templates/components"
)

// mapDomainError returns HTTP status, API code, and user-facing message.
func mapDomainError(err error) (int, domain.APIErrorCode, string) {
	switch {
	case errors.Is(err, domain.ErrAccountNotFound):
		return http.StatusNotFound, domain.CodeAccountNotFound, "Account not found."
	case errors.Is(err, domain.ErrMissingAccountName):
		return http.StatusBadRequest, domain.CodeMissingAccountName, "Account name is required."
	case errors.Is(err, domain.ErrNameTooLong):
		return http.StatusBadRequest, domain.CodeNameTooLong, "Account name exceeds maximum length."
	case errors.Is(err, domain.ErrInvalidTransactionType):
		return http.StatusBadRequest, domain.CodeInvalidTransactionType, "Transaction type must be credit or debit."
	case errors.Is(err, domain.ErrInvalidFilterType):
		return http.StatusBadRequest, domain.CodeInvalidFilterType, "Invalid transaction type filter. Use credit or debit."
	case errors.Is(err, domain.ErrInvalidAmount):
		return http.StatusBadRequest, domain.CodeInvalidAmount, "Amount must be a positive whole-rupee amount (e.g. 100 = ₹100)."
	case errors.Is(err, domain.ErrAmountOverflow):
		return http.StatusUnprocessableEntity, domain.CodeAmountOverflow, "Amount or resulting balance exceeds supported integer range."
	case errors.Is(err, domain.ErrInvalidReferenceID):
		return http.StatusBadRequest, domain.CodeInvalidReferenceID, "reference_id is required and must be non-empty."
	case errors.Is(err, domain.ErrReferenceIDTooLong):
		return http.StatusBadRequest, domain.CodeReferenceIDTooLong, "reference_id exceeds maximum length."
	case errors.Is(err, domain.ErrDescriptionTooLong):
		return http.StatusBadRequest, domain.CodeDescriptionTooLong, "Description exceeds maximum length."
	case errors.Is(err, domain.ErrDuplicateReferenceID):
		return http.StatusConflict, domain.CodeDuplicateReferenceID,
			"Duplicate reference ID — this reference already exists for the account."
	case errors.Is(err, domain.ErrInsufficientBalance):
		return http.StatusUnprocessableEntity, domain.CodeInsufficientBalance, "Insufficient balance for this debit."
	case errors.Is(err, domain.ErrInvalidPagination):
		return http.StatusBadRequest, domain.CodeInvalidPagination, "Invalid pagination parameters. page >= 1, 1 <= limit <= 100."
	case errors.Is(err, domain.ErrInvalidRequestBody):
		return http.StatusBadRequest, domain.CodeInvalidRequestBody, "Malformed or invalid request body."
	case errors.Is(err, domain.ErrInvalidTransfer):
		return http.StatusBadRequest, domain.CodeInvalidTransfer, "Invalid transfer request."
	case errors.Is(err, domain.ErrSameAccountTransfer):
		return http.StatusBadRequest, domain.CodeSameAccountTransfer, "Source and destination accounts must be different."
	case errors.Is(err, domain.ErrDuplicateTransferRef):
		return http.StatusConflict, domain.CodeDuplicateTransferRef, "This transfer reference ID already exists."
	case errors.Is(err, domain.ErrDatabaseUnavailable):
		return http.StatusServiceUnavailable, domain.CodeDatabaseUnavailable, "Database temporarily unavailable. Please retry."
	default:
		return http.StatusInternalServerError, domain.CodeInternalError, "An unexpected error occurred."
	}
}

func respondHTMXError(c *gin.Context, err error) {
	status, code, message := mapDomainError(err)
	c.Header("X-Error-Code", string(code))
	c.Status(status)
	_ = components.StatusAlert("error", status, string(code), message).Render(c.Request.Context(), c.Writer)
}

func respondHTMXSuccess(c *gin.Context, status int, message string) {
	c.Status(status)
	_ = components.StatusAlert("success", status, "OK", message).Render(c.Request.Context(), c.Writer)
}
