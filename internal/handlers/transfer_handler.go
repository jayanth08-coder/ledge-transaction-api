package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ledge/ledge-transaction-api/internal/domain"
	"github.com/ledge/ledge-transaction-api/internal/service"
)

// TransferHandler serves transfer JSON and HTMX form submissions.
type TransferHandler struct {
	transfers *service.TransferService
}

func NewTransferHandler(transfers *service.TransferService) *TransferHandler {
	return &TransferHandler{transfers: transfers}
}

func (h *TransferHandler) Create(c *gin.Context) {
	var req domain.CreateTransferRequest
	if err := bindTransferRequest(c, &req); err != nil {
		respondError(c, domain.ErrInvalidRequestBody)
		return
	}

	transfer, err := h.transfers.Create(req)
	if err != nil {
		respondError(c, err)
		return
	}

	if isHTMX(c) {
		c.Header("HX-Trigger", `{"transferCreated":true}`)
	}

	success(c, http.StatusCreated, transfer.ToResponse())
}

func bindTransferRequest(c *gin.Context, req *domain.CreateTransferRequest) error {
	ct := c.ContentType()
	if ct == "application/json" || ct == "application/json; charset=utf-8" {
		if err := c.ShouldBindJSON(req); err != nil {
			return domain.ErrInvalidRequestBody
		}
		return nil
	}
	type formReq struct {
		FromAccountID string `form:"from_account_id"`
		ToAccountID   string `form:"to_account_id"`
		Amount        int64  `form:"amount"`
		ReferenceID   string `form:"reference_id"`
		Description   string `form:"description"`
	}
	var fr formReq
	if err := c.ShouldBind(&fr); err != nil {
		return err
	}
	req.FromAccountID = fr.FromAccountID
	req.ToAccountID = fr.ToAccountID
	req.Amount = fr.Amount
	req.ReferenceID = fr.ReferenceID
	req.Description = fr.Description
	return nil
}
