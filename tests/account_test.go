package tests

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestCreateAccountSuccess(t *testing.T) {
	app := newTestApp(t)
	code, env, _ := doJSON(t, app.Router, http.MethodPost, "/api/accounts", map[string]any{"name": "Alice"})
	if code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", code)
	}
	if !env.Success {
		t.Fatalf("expected success")
	}
	var data map[string]any
	_ = json.Unmarshal(env.Data, &data)
	if data["name"] != "Alice" {
		t.Fatalf("name=%v", data["name"])
	}
	if data["balance"].(float64) != 0 {
		t.Fatalf("balance should be 0, got %v", data["balance"])
	}
	if data["id"] == nil || data["id"] == "" {
		t.Fatal("id should be generated")
	}
	if data["created_at"] == nil {
		t.Fatal("created_at should be set")
	}
}

func TestCreateAccountMissingName(t *testing.T) {
	app := newTestApp(t)
	code, env, _ := doJSON(t, app.Router, http.MethodPost, "/api/accounts", map[string]any{"name": "   "})
	if code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", code)
	}
	if env.Error == nil || env.Error.Code != "MISSING_ACCOUNT_NAME" {
		t.Fatalf("error=%v", env.Error)
	}
}

func TestCreateAccountIgnoresClientBalanceAndID(t *testing.T) {
	app := newTestApp(t)
	code, env, _ := doJSON(t, app.Router, http.MethodPost, "/api/accounts", map[string]any{
		"name":       "Bob",		"balance":    99999,
		"id":         "client-supplied-id",
		"created_at": "2000-01-01T00:00:00Z",
	})
	if code != http.StatusCreated {
		t.Fatalf("expected 201, got %d env=%v", code, env)
	}
	var data map[string]any
	_ = json.Unmarshal(env.Data, &data)
	if data["balance"].(float64) != 0 {
		t.Fatalf("client must not set balance, got %v", data["balance"])
	}
	if data["id"] == "client-supplied-id" {
		t.Fatal("client must not set id")
	}
}

func TestGetAccountBalanceNotFound(t *testing.T) {
	app := newTestApp(t)
	code, env, _ := doJSON(t, app.Router, http.MethodGet, "/api/accounts/missing/balance", nil)
	if code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", code)
	}
	if env.Error == nil || env.Error.Code != "ACCOUNT_NOT_FOUND" {
		t.Fatalf("error=%v", env.Error)
	}
}

func TestGetAccountBalance(t *testing.T) {
	app := newTestApp(t)
	acc := createAccount(t, app, "Carol")
	id := acc["id"].(string)

	code, env, _ := doJSON(t, app.Router, http.MethodPost, "/api/transactions", map[string]any{
		"account_id":   id,
		"type":         "credit",
		"amount":       150,
		"reference_id": "BAL-1",
		"description":  "top up",
	})
	if code != http.StatusCreated {
		t.Fatalf("credit failed: %d %v", code, env)
	}

	code, env, _ = doJSON(t, app.Router, http.MethodGet, "/api/accounts/"+id+"/balance", nil)
	if code != http.StatusOK {
		t.Fatalf("expected 200, got %d", code)
	}
	var bal map[string]any
	_ = json.Unmarshal(env.Data, &bal)
	if bal["balance"].(float64) != 150 {
		t.Fatalf("balance=%v", bal["balance"])
	}
	if bal["account_id"] != id {
		t.Fatalf("account_id=%v", bal["account_id"])
	}
}
