package tests

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestCreditAndDebitSuccess(t *testing.T) {
	app := newTestApp(t)
	acc := createAccount(t, app, "Txn User")
	id := acc["id"].(string)

	code, _, _ := doJSON(t, app.Router, http.MethodPost, "/api/transactions", map[string]any{
		"account_id": id, "type": "credit", "amount": 100, "reference_id": "C1", "description": "credit",
	})
	if code != http.StatusCreated {
		t.Fatalf("credit status=%d", code)
	}

	code, _, _ = doJSON(t, app.Router, http.MethodPost, "/api/transactions", map[string]any{
		"account_id": id, "type": "debit", "amount": 30, "reference_id": "D1", "description": "debit",
	})
	if code != http.StatusCreated {
		t.Fatalf("debit status=%d", code)
	}

	code, env, _ := doJSON(t, app.Router, http.MethodGet, "/api/accounts/"+id+"/balance", nil)
	if code != http.StatusOK {
		t.Fatalf("balance status=%d", code)
	}
	var bal map[string]any
	_ = json.Unmarshal(env.Data, &bal)
	if bal["balance"].(float64) != 70 {
		t.Fatalf("expected 70, got %v", bal["balance"])
	}
}

func TestInsufficientBalance(t *testing.T) {
	app := newTestApp(t)
	acc := createAccount(t, app, "Poor")
	id := acc["id"].(string)

	_, _, _ = doJSON(t, app.Router, http.MethodPost, "/api/transactions", map[string]any{
		"account_id": id, "type": "credit", "amount": 100, "reference_id": "C1",
	})

	code, env, _ := doJSON(t, app.Router, http.MethodPost, "/api/transactions", map[string]any{
		"account_id": id, "type": "debit", "amount": 150, "reference_id": "D1",
	})
	if code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", code)
	}
	if env.Error == nil || env.Error.Code != "INSUFFICIENT_BALANCE" {
		t.Fatalf("error=%v", env.Error)
	}

	code, env, _ = doJSON(t, app.Router, http.MethodGet, "/api/accounts/"+id+"/balance", nil)
	var bal map[string]any
	_ = json.Unmarshal(env.Data, &bal)
	if bal["balance"].(float64) != 100 {
		t.Fatalf("balance should remain 100, got %v", bal["balance"])
	}

	code, env, _ = doJSON(t, app.Router, http.MethodGet, "/api/accounts/"+id+"/transactions", nil)
	var page map[string]any
	_ = json.Unmarshal(env.Data, &page)
	pag := page["pagination"].(map[string]any)
	if pag["total_records"].(float64) != 1 {
		t.Fatalf("failed debit must not create ledger entry, total=%v", pag["total_records"])
	}
}

func TestInvalidTransactionType(t *testing.T) {
	app := newTestApp(t)
	acc := createAccount(t, app, "Type Test")
	code, env, _ := doJSON(t, app.Router, http.MethodPost, "/api/transactions", map[string]any{
		"account_id": acc["id"], "type": "transfer", "amount": 10, "reference_id": "X1",
	})
	if code != http.StatusBadRequest || env.Error.Code != "INVALID_TRANSACTION_TYPE" {
		t.Fatalf("got %d %v", code, env.Error)
	}
}

func TestZeroAndNegativeAmounts(t *testing.T) {
	app := newTestApp(t)
	acc := createAccount(t, app, "Amt")
	for _, amount := range []int64{0, -5} {
		code, env, _ := doJSON(t, app.Router, http.MethodPost, "/api/transactions", map[string]any{
			"account_id": acc["id"], "type": "credit", "amount": amount, "reference_id": "A",
		})
		if code != http.StatusBadRequest || env.Error.Code != "INVALID_AMOUNT" {
			t.Fatalf("amount=%d => %d %v", amount, code, env.Error)
		}
	}
}

func TestMissingAccountAndAutoReference(t *testing.T) {
	app := newTestApp(t)
	code, _, _ := doJSON(t, app.Router, http.MethodPost, "/api/transactions", map[string]any{
		"account_id": "nope", "type": "credit", "amount": 10, "reference_id": "R1",
	})
	if code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", code)
	}

	acc := createAccount(t, app, "Ref")
	code, env, _ := doJSON(t, app.Router, http.MethodPost, "/api/transactions", map[string]any{
		"account_id": acc["id"], "type": "credit", "amount": 10, "reference_id": "  ",
	})
	if code != http.StatusCreated || !env.Success {
		t.Fatalf("blank reference should auto-generate, got %d %v", code, env.Error)
	}
	var txn map[string]any
	_ = json.Unmarshal(env.Data, &txn)
	ref, _ := txn["reference_id"].(string)
	if ref == "" || !strings.HasPrefix(ref, "AUTO-") {
		t.Fatalf("expected AUTO- reference, got %q", ref)
	}
}

func TestTransactionImmutabilityNoUpdateDeleteRoutes(t *testing.T) {
	app := newTestApp(t)
	acc := createAccount(t, app, "Imm")
	code, env, _ := doJSON(t, app.Router, http.MethodPost, "/api/transactions", map[string]any{
		"account_id": acc["id"], "type": "credit", "amount": 50, "reference_id": "IMM-1",
	})
	if code != http.StatusCreated {
		t.Fatalf("create failed")
	}
	var txn map[string]any
	_ = json.Unmarshal(env.Data, &txn)
	id := txn["id"].(string)

	for _, method := range []string{http.MethodPut, http.MethodPatch, http.MethodDelete} {
		reqCode, _, raw := doJSON(t, app.Router, method, "/api/transactions/"+id, map[string]any{"amount": 1})
		if reqCode != http.StatusNotFound && reqCode != http.StatusMethodNotAllowed {
			t.Fatalf("%s should not be supported, got %d %s", method, reqCode, string(raw))
		}
	}
}

func TestLedgerMathExample(t *testing.T) {
	app := newTestApp(t)
	acc := createAccount(t, app, "Math")
	id := acc["id"].(string)
	steps := []struct {
		typ, ref string
		amount   int64
	}{
		{"credit", "M1", 100},
		{"credit", "M2", 50},
		{"debit", "M3", 30},
		{"debit", "M4", 20},
	}
	for _, s := range steps {
		code, _, _ := doJSON(t, app.Router, http.MethodPost, "/api/transactions", map[string]any{
			"account_id": id, "type": s.typ, "amount": s.amount, "reference_id": s.ref,
		})
		if code != http.StatusCreated {
			t.Fatalf("%s %s failed: %d", s.typ, s.ref, code)
		}
	}
	_, env, _ := doJSON(t, app.Router, http.MethodGet, "/api/accounts/"+id+"/balance", nil)
	var bal map[string]any
	_ = json.Unmarshal(env.Data, &bal)
	if bal["balance"].(float64) != 100 {
		t.Fatalf("expected final 100, got %v", bal["balance"])
	}
}
