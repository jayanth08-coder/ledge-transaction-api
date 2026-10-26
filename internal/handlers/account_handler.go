package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/ledge/ledge-transaction-api/internal/domain"
	"github.com/ledge/ledge-transaction-api/internal/service"
	"github.com/ledge/ledge-transaction-api/templates/components"
	"github.com/ledge/ledge-transaction-api/templates/pages"
)

// AccountHandler serves account JSON and HTML endpoints.
type AccountHandler struct {
	accounts *service.AccountService
	txns     *service.TransactionService
}

func NewAccountHandler(accounts *service.AccountService, txns *service.TransactionService) *AccountHandler {
	return &AccountHandler{accounts: accounts, txns: txns}
}

func (h *AccountHandler) Create(c *gin.Context) {
	var req domain.CreateAccountRequest
	if err := bindJSONOrForm(c, &req); err != nil {
		respondError(c, domain.ErrInvalidRequestBody)
		return
	}

	account, err := h.accounts.Create(req)
	if err != nil {
		respondError(c, err)
		return
	}

	if isHTMX(c) {
		c.Header("HX-Trigger", `{"accountCreated":true}`)
	}
	success(c, http.StatusCreated, account.ToResponse())
}

func (h *AccountHandler) List(c *gin.Context) {
	page, limit, err := parsePageLimit(c)
	if err != nil {
		respondError(c, err)
		return
	}

	result, err := h.accounts.List(page, limit)
	if err != nil {
		respondError(c, err)
		return
	}
	success(c, http.StatusOK, result)
}

func (h *AccountHandler) Get(c *gin.Context) {
	account, err := h.accounts.GetByID(c.Param("id"))
	if err != nil {
		respondError(c, err)
		return
	}
	success(c, http.StatusOK, account.ToResponse())
}

func (h *AccountHandler) GetBalance(c *gin.Context) {
	bal, err := h.accounts.GetBalance(c.Param("id"))
	if err != nil {
		respondError(c, err)
		return
	}
	success(c, http.StatusOK, bal)
}

func (h *AccountHandler) GetTransactions(c *gin.Context) {
	page, limit, err := parsePageLimit(c)
	if err != nil {
		respondError(c, err)
		return
	}
	typ := c.Query("type")

	result, err := h.txns.List(c.Param("id"), typ, page, limit)
	if err != nil {
		respondError(c, err)
		return
	}
	success(c, http.StatusOK, result)
}

func (h *AccountHandler) AccountsPage(c *gin.Context) {
	result, err := h.accounts.List(domain.DefaultPage, domain.MaxPageSize)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to load accounts")
		return
	}
	_ = pages.AccountsPage(result.Data).Render(c.Request.Context(), c.Writer)
}

func (h *AccountHandler) AccountDetailPage(c *gin.Context) {
	id := c.Param("id")
	account, err := h.accounts.GetByID(id)
	if err != nil {
		c.String(http.StatusNotFound, "Account not found")
		return
	}

	page, limit, _ := parsePageLimitOrDefault(c)
	typ := c.Query("type")
	txns, err := h.txns.List(id, typ, page, limit)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to load transactions")
		return
	}

	_ = pages.AccountDetailPage(*account, *txns, typ).Render(c.Request.Context(), c.Writer)
}

func (h *AccountHandler) TransactionsFragment(c *gin.Context) {
	id := c.Param("id")
	page, limit, err := parsePageLimitOrDefault(c)
	if err != nil {
		respondHTMXError(c, domain.ErrInvalidPagination)
		return
	}
	typ := c.Query("type")

	account, err := h.accounts.GetByID(id)
	if err != nil {
		respondHTMXError(c, err)
		return
	}

	txns, err := h.txns.List(id, typ, page, limit)
	if err != nil {
		respondHTMXError(c, err)
		return
	}

	_ = components.TransactionPanel(*account, *txns, typ).Render(c.Request.Context(), c.Writer)
}

func parsePageLimit(c *gin.Context) (int, int, error) {
	page := domain.DefaultPage
	limit := domain.DefaultPageSize

	if raw := c.Query("page"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 {
			return 0, 0, domain.ErrInvalidPagination
		}
		page = n
	}
	if raw := c.Query("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 || n > domain.MaxPageSize {
			return 0, 0, domain.ErrInvalidPagination
		}
		limit = n
	}
	return page, limit, nil
}

func parsePageLimitOrDefault(c *gin.Context) (int, int, error) {
	return parsePageLimit(c)
}

func bindJSONOrForm(c *gin.Context, dest interface{}) error {
	ct := c.ContentType()
	if ct == "application/json" || ct == "application/json; charset=utf-8" {
		return c.ShouldBindJSON(dest)
	}
	if err := c.ShouldBind(dest); err != nil {
		return c.ShouldBindJSON(dest)
	}
	return nil
}

func isHTMX(c *gin.Context) bool {
	return c.GetHeader("HX-Request") == "true"
}
