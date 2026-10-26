package routes

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ledge/ledge-transaction-api/internal/handlers"
	"github.com/ledge/ledge-transaction-api/internal/middleware"
)

// Dependencies groups HTTP handlers for route registration.
type Dependencies struct {
	Accounts     *handlers.AccountHandler
	Transactions *handlers.TransactionHandler
	Transfers    *handlers.TransferHandler
	Pages        *handlers.PageHandler
}

// Setup registers middleware, static assets, web pages, and API routes.
func Setup(r *gin.Engine, deps Dependencies) {
	r.Use(gin.Logger())
	r.Use(gin.Recovery())
	r.Use(middleware.RequestID())
	r.Use(func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 1<<20) // 1 MiB
		c.Next()
	})

	r.Static("/static", "./static")

	// Web UI
	r.GET("/", deps.Pages.Dashboard)
	r.GET("/health", deps.Pages.Health)
	r.GET("/accounts", deps.Accounts.AccountsPage)
	r.GET("/accounts/:id", deps.Accounts.AccountDetailPage)
	r.GET("/accounts/:id/transactions", deps.Accounts.TransactionsFragment)

	// REST API
	api := r.Group("/api")
	{
		api.POST("/accounts", deps.Accounts.Create)
		api.GET("/accounts", deps.Accounts.List)
		api.GET("/accounts/:id", deps.Accounts.Get)
		api.GET("/accounts/:id/balance", deps.Accounts.GetBalance)
		api.GET("/accounts/:id/transactions", deps.Accounts.GetTransactions)

		api.POST("/transactions", deps.Transactions.Create)
		api.POST("/transfers", deps.Transfers.Create)
	}
}
