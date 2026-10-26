package tests

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestDuplicateReferenceSameAccount(t *testing.T) {
	app := newTestApp(t)
	acc := createAccount(t, app, "Dup A")
	id := acc["id"].(string)

	payload := map[string]any{
		"account_id": id, "type": "credit", "amount": 100, "reference_id": "PAY-001",
	}
	code, _, _ := doJSON(t, app.Router, http.MethodPost, "/api/transactions", payload)
	if code != http.StatusCreated {
		t.Fatalf("first request expected 201, got %d", code)
	}

	code, env, _ := doJSON(t, app.Router, http.MethodPost, "/api/transactions", payload)
	if code != http.StatusConflict {
		t.Fatalf("duplicate expected 409, got %d", code)
	}
	if env.Error == nil || env.Error.Code != "DUPLICATE_REFERENCE_ID" {
		t.Fatalf("error=%v", env.Error)
	}

	_, env, _ = doJSON(t, app.Router, http.MethodGet, "/api/accounts/"+id+"/balance", nil)
	var bal map[string]any
	_ = json.Unmarshal(env.Data, &bal)
	if bal["balance"].(float64) != 100 {
		t.Fatalf("balance should stay 100, got %v", bal["balance"])
	}

	_, env, _ = doJSON(t, app.Router, http.MethodGet, "/api/accounts/"+id+"/transactions", nil)
	var page map[string]any
	_ = json.Unmarshal(env.Data, &page)
	if page["pagination"].(map[string]any)["total_records"].(float64) != 1 {
		t.Fatalf("expected exactly one transaction")
	}
}

func TestSameReferenceDifferentAccountsAllowed(t *testing.T) {
	app := newTestApp(t)
	a := createAccount(t, app, "A")
	b := createAccount(t, app, "B")

	code, _, _ := doJSON(t, app.Router, http.MethodPost, "/api/transactions", map[string]any{
		"account_id": a["id"], "type": "credit", "amount": 100, "reference_id": "PAY-001",
	})
	if code != http.StatusCreated {
		t.Fatalf("account A failed: %d", code)
	}
	code, _, _ = doJSON(t, app.Router, http.MethodPost, "/api/transactions", map[string]any{
		"account_id": b["id"], "type": "credit", "amount": 200, "reference_id": "PAY-001",
	})
	if code != http.StatusCreated {
		t.Fatalf("account B with same ref should succeed, got %d", code)
	}
}
