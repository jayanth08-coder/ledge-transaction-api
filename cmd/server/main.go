package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/ledge/ledge-transaction-api/internal/config"
	"github.com/ledge/ledge-transaction-api/internal/database"
	"github.com/ledge/ledge-transaction-api/internal/handlers"
	"github.com/ledge/ledge-transaction-api/internal/repository"
	"github.com/ledge/ledge-transaction-api/internal/routes"
	"github.com/ledge/ledge-transaction-api/internal/service"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	if cfg.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	db, err := database.Open(cfg.DatabasePath, cfg.LogLevel)
	if err != nil {
		log.Fatalf("database open failed: %v", err)
	}

	if err := database.Migrate(db); err != nil {
		log.Fatalf("database migration failed: %v", err)
	}

	if cfg.SeedDemo {
		if err := database.SeedDemo(db); err != nil {
			log.Fatalf("demo seed failed: %v", err)
		}
		log.Println("demo data seeded (idempotent)")
	}

	accountRepo := repository.NewAccountRepository(db)
	txnRepo := repository.NewTransactionRepository(db)
	transferRepo := repository.NewTransferRepository(db)

	accountSvc := service.NewAccountService(accountRepo)
	txnSvc := service.NewTransactionService(db, accountRepo, txnRepo)
	transferSvc := service.NewTransferService(db, accountRepo, txnRepo, transferRepo)

	accountHandler := handlers.NewAccountHandler(accountSvc, txnSvc)
	txnHandler := handlers.NewTransactionHandler(txnSvc, accountSvc)
	transferHandler := handlers.NewTransferHandler(transferSvc)
	pageHandler := handlers.NewPageHandler(accountSvc, txnSvc)

	r := gin.New()
	routes.Setup(r, routes.Dependencies{
		Accounts:     accountHandler,
		Transactions: txnHandler,
		Transfers:    transferHandler,
		Pages:        pageHandler,
	})

	addr := cfg.Addr()
	log.Printf("Transaction Ledger listening on http://%s (db=%s)", addr, cfg.DatabasePath)
	if err := r.Run(addr); err != nil {
		log.Printf("server stopped: %v", err)
		os.Exit(1)
	}
}
