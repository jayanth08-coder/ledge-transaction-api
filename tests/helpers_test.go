package tests

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/ledge/ledge-transaction-api/internal/database"
	"github.com/ledge/ledge-transaction-api/internal/handlers"
	"github.com/ledge/ledge-transaction-api/internal/repository"
	"github.com/ledge/ledge-transaction-api/internal/routes"
	"github.com/ledge/ledge-transaction-api/internal/service"
	"gorm.io/gorm"
)

func init() {
	gin.SetMode(gin.TestMode)
}

type testApp struct {
	DB     *gorm.DB
	Router *gin.Engine
	Dir    string
}

func newTestApp(t *testing.T) *testApp {
	t.Helper()
	dir, err := os.MkdirTemp("", "ledger-test-*")
	if err != nil {
		t.Fatalf("temp dir: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })

	dbPath := filepath.Join(dir, "test.db")
	db, err := database.Open(dbPath, "silent")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := database.Migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	accountRepo := repository.NewAccountRepository(db)
	txnRepo := repository.NewTransactionRepository(db)
	transferRepo := repository.NewTransferRepository(db)

	accountSvc := service.NewAccountService(accountRepo)
	txnSvc := service.NewTransactionService(db, accountRepo, txnRepo)
	transferSvc := service.NewTransferService(db, accountRepo, txnRepo, transferRepo)

	r := gin.New()
	routes.Setup(r, routes.Dependencies{
		Accounts:     handlers.NewAccountHandler(accountSvc, txnSvc),
		Transactions: handlers.NewTransactionHandler(txnSvc, accountSvc),
		Transfers:    handlers.NewTransferHandler(transferSvc),
		Pages:        handlers.NewPageHandler(accountSvc, txnSvc),
	})

	return &testApp{DB: db, Router: r, Dir: dir}
}

type envelope struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data"`
	Error   *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func doJSON(t *testing.T, r *gin.Engine, method, path string, body any) (int, envelope, []byte) {
	t.Helper()
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		reader = bytes.NewReader(b)
	}
	req := httptest.NewRequest(method, path, reader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	raw := w.Body.Bytes()
	var env envelope
	if len(raw) > 0 && raw[0] == '{' {
		_ = json.Unmarshal(raw, &env)
	}
	return w.Code, env, raw
}

func createAccount(t *testing.T, app *testApp, name string) map[string]any {
	t.Helper()
	code, env, _ := doJSON(t, app.Router, http.MethodPost, "/api/accounts", map[string]any{"name": name})
	if code != http.StatusCreated || !env.Success {
		t.Fatalf("create account: status=%d body=%s", code, string(mustRaw(env)))
	}
	var data map[string]any
	if err := json.Unmarshal(env.Data, &data); err != nil {
		t.Fatalf("decode account: %v", err)
	}
	return data
}

func mustRaw(env envelope) []byte {
	b, _ := json.Marshal(env)
	return b
}
