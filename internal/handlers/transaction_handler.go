package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ledge/ledge-transaction-api/internal/domain"
	"github.com/ledge/ledge-transaction-api/internal/service"
)

// TransactionHandler serves transaction JSON and HTMX form submissions.
type TransactionHandler struct {
	txns     *service.TransactionService
	accounts *service.AccountService
}

func NewTransactionHandler(txns *service.TransactionService, accounts *service.AccountService) *TransactionHandler {
	return &TransactionHandler{txns: txns, accounts: accounts}
}

func (h *TransactionHandler) Create(c *gin.Context) {
	var req domain.CreateTransactionRequest
	if err := bindTransactionRequest(c, &req); err != nil {
		respondError(c, domain.ErrInvalidRequestBody)
		return
	}

	txn, err := h.txns.Create(req)
	if err != nil {
		respondError(c, err)
		return
	}

	if isHTMX(c) {
		account, aerr := h.accounts.GetByID(txn.AccountID)
		trigger := map[string]interface{}{
			"transactionCreated": true,
			"accountId":          txn.AccountID,
		}
		if aerr == nil {
			trigger["balance"] = account.Balance
		}
		b, _ := json.Marshal(trigger)
		c.Header("HX-Trigger", string(b))
	}

	success(c, http.StatusCreated, txn.ToResponse())
}

func bindTransactionRequest(c *gin.Context, req *domain.CreateTransactionRequest) error {
	ct := c.ContentType()
	if ct == "application/json" || ct == "application/json; charset=utf-8" {
		if err := c.ShouldBindJSON(req); err != nil {
			return domain.ErrInvalidRequestBody
		}
		return nil
	}

	type formReq struct {
		AccountID   string `form:"account_id"`
		Type        string `form:"type"`
		Amount      int64  `form:"amount"`
		Description string `form:"description"`
		ReferenceID string `form:"reference_id"`
	}
	var fr formReq
	if err := c.ShouldBind(&fr); err != nil {
		return err
	}
	req.AccountID = fr.AccountID
	req.Type = fr.Type
	req.Amount = fr.Amount
	req.Description = fr.Description
	req.ReferenceID = fr.ReferenceID
	return nil
}
