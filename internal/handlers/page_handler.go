package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ledge/ledge-transaction-api/internal/domain"
	"github.com/ledge/ledge-transaction-api/internal/service"
	"github.com/ledge/ledge-transaction-api/templates/pages"
)

// PageHandler serves server-rendered pages and health checks.
type PageHandler struct {
	accounts *service.AccountService
	txns     *service.TransactionService
}

func NewPageHandler(accounts *service.AccountService, txns *service.TransactionService) *PageHandler {
	return &PageHandler{accounts: accounts, txns: txns}
}

func (h *PageHandler) Dashboard(c *gin.Context) {
	accountCount, _ := h.accounts.Count()
	txnCount, _ := h.txns.CountAll()

	list, err := h.accounts.List(1, 5)
	recent := []domain.AccountResponse{}
	if err == nil {
		recent = list.Data
	}

	_ = pages.DashboardPage(accountCount, txnCount, recent).Render(c.Request.Context(), c.Writer)
}

func (h *PageHandler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "ok",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}
