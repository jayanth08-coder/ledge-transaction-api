package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ledge/ledge-transaction-api/internal/domain"
)

// APIResponse is the standard envelope for JSON API responses.
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *APIError   `json:"error,omitempty"`
}

// APIError is the standard error payload.
type APIError struct {
	Code    domain.APIErrorCode   `json:"code"`
	Message string                `json:"message"`
	Fields  []domain.FieldError   `json:"fields,omitempty"`
}

func success(c *gin.Context, status int, data interface{}) {
	c.JSON(status, APIResponse{Success: true, Data: data})
}

func fail(c *gin.Context, status int, code domain.APIErrorCode, message string, fields ...domain.FieldError) {
	errPayload := &APIError{Code: code, Message: message}
	if len(fields) > 0 {
		errPayload.Fields = fields
	}
	c.Header("X-Error-Code", string(code))
	c.JSON(status, APIResponse{Success: false, Error: errPayload})
}

// respondError maps domain errors to HTTP status codes and JSON error bodies.
func respondError(c *gin.Context, err error) {
	if err == nil {
		return
	}

	var ve *domain.ValidationError
	if errors.As(err, &ve) {
		fail(c, http.StatusBadRequest, domain.CodeValidationError, ve.Message, ve.Fields...)
		return
	}

	status, code, message := mapDomainError(err)
	fail(c, status, code, message)
}

// wantsJSON reports whether the client expects a JSON response (API or Accept header).
func wantsJSON(c *gin.Context) bool {
	if c.GetHeader("HX-Request") == "true" {
		return false
	}
	accept := c.GetHeader("Accept")
	return accept == "" || accept == "*/*" || containsJSON(accept) || c.ContentType() == "application/json"
}

func containsJSON(accept string) bool {
	return len(accept) >= 16 && (accept == "application/json" ||
		len(accept) > 0 && (contains(accept, "application/json")))
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 ||
		(len(s) > 0 && indexOf(s, sub) >= 0))
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
