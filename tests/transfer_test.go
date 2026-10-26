package tests

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
)

func TestTransferSuccess(t *testing.T) {
	app := newTestApp(t)
	a := createAccount(t, app, "From")
	b := createAccount(t, app, "To")
	aID := a["id"].(string)
	bID := b["id"].(string)

	_, _, _ = doJSON(t, app.Router, http.MethodPost, "/api/transactions", map[string]any{
		"account_id": aID, "type": "credit", "amount": 100, "reference_id": "OPEN-A",
	})
	_, _, _ = doJSON(t, app.Router, http.MethodPost, "/api/transactions", map[string]any{
		"account_id": bID, "type": "credit", "amount": 50, "reference_id": "OPEN-B",
	})

	code, env, _ := doJSON(t, app.Router, http.MethodPost, "/api/transfers", map[string]any{
		"from_account_id": aID,
		"to_account_id":   bID,
		"amount":          30,
		"reference_id":    "XFER-001",
		"description":     "rent share",
	})
	if code != http.StatusCreated || !env.Success {
		t.Fatalf("transfer failed: %d %v", code, env)
	}

	assertBalance(t, app, aID, 70)
	assertBalance(t, app, bID, 80)

	// Both sides appear in histories
	_, env, _ = doJSON(t, app.Router, http.MethodGet, "/api/accounts/"+aID+"/transactions?limit=100", nil)
	var page map[string]any
	_ = json.Unmarshal(env.Data, &page)
	foundDebit := false
	for _, row := range page["data"].([]any) {
		m := row.(map[string]any)
		if m["reference_id"] == "XFER-001:debit" && m["type"] == "debit" {
			foundDebit = true
		}
	}
	if !foundDebit {
		t.Fatal("missing debit ledger entry for transfer")
	}

	_, env, _ = doJSON(t, app.Router, http.MethodGet, "/api/accounts/"+bID+"/transactions?limit=100", nil)
	_ = json.Unmarshal(env.Data, &page)
	foundCredit := false
	for _, row := range page["data"].([]any) {
		m := row.(map[string]any)
		if m["reference_id"] == "XFER-001:credit" && m["type"] == "credit" {
			foundCredit = true
		}
	}
	if !foundCredit {
		t.Fatal("missing credit ledger entry for transfer")
	}
}

func TestTransferValidation(t *testing.T) {
	app := newTestApp(t)
	a := createAccount(t, app, "A")
	b := createAccount(t, app, "B")
	aID := a["id"].(string)
	bID := b["id"].(string)

	_, _, _ = doJSON(t, app.Router, http.MethodPost, "/api/transactions", map[string]any{
		"account_id": aID, "type": "credit", "amount": 100, "reference_id": "OPEN",
	})

	// Insufficient
	code, env, _ := doJSON(t, app.Router, http.MethodPost, "/api/transfers", map[string]any{
		"from_account_id": aID, "to_account_id": bID, "amount": 500, "reference_id": "T1",
	})
	if code != http.StatusUnprocessableEntity || env.Error.Code != "INSUFFICIENT_BALANCE" {
		t.Fatalf("insufficient => %d %v", code, env.Error)
	}
	assertBalance(t, app, aID, 100)
	assertBalance(t, app, bID, 0)

	// Same account
	code, env, _ = doJSON(t, app.Router, http.MethodPost, "/api/transfers", map[string]any{
		"from_account_id": aID, "to_account_id": aID, "amount": 10, "reference_id": "T2",
	})
	if code != http.StatusBadRequest || env.Error.Code != "SAME_ACCOUNT_TRANSFER" {
		t.Fatalf("same account => %d %v", code, env.Error)
	}

	// Missing source
	code, env, _ = doJSON(t, app.Router, http.MethodPost, "/api/transfers", map[string]any{
		"from_account_id": "missing", "to_account_id": bID, "amount": 10, "reference_id": "T3",
	})
	if code != http.StatusNotFound {
		t.Fatalf("missing source => %d", code)
	}

	// Missing destination
	code, env, _ = doJSON(t, app.Router, http.MethodPost, "/api/transfers", map[string]any{
		"from_account_id": aID, "to_account_id": "missing", "amount": 10, "reference_id": "T4",
	})
	if code != http.StatusNotFound {
		t.Fatalf("missing dest => %d", code)
	}
}

func TestTransferDuplicateIdempotent(t *testing.T) {
	app := newTestApp(t)
	a := createAccount(t, app, "A")
	b := createAccount(t, app, "B")
	aID := a["id"].(string)
	bID := b["id"].(string)
	_, _, _ = doJSON(t, app.Router, http.MethodPost, "/api/transactions", map[string]any{
		"account_id": aID, "type": "credit", "amount": 50, "reference_id": "OPEN",
	})

	payload := map[string]any{
		"from_account_id": aID, "to_account_id": bID, "amount": 10, "reference_id": "IDEMPOTENT-1",
	}
	code, _, _ := doJSON(t, app.Router, http.MethodPost, "/api/transfers", payload)
	if code != http.StatusCreated {
		t.Fatalf("first transfer: %d", code)
	}
	code, env, _ := doJSON(t, app.Router, http.MethodPost, "/api/transfers", payload)
	if code != http.StatusConflict || env.Error.Code != "DUPLICATE_TRANSFER_REFERENCE" {
		t.Fatalf("duplicate => %d %v", code, env.Error)
	}
	assertBalance(t, app, aID, 40)
	assertBalance(t, app, bID, 10)
}

func TestConcurrentTransfersPreserveIntegrity(t *testing.T) {
	app := newTestApp(t)
	a := createAccount(t, app, "A")
	b := createAccount(t, app, "B")
	aID := a["id"].(string)
	bID := b["id"].(string)
	_, _, _ = doJSON(t, app.Router, http.MethodPost, "/api/transactions", map[string]any{
		"account_id": aID, "type": "credit", "amount": 100, "reference_id": "OPEN",
	})

	const workers = 20
	var wg sync.WaitGroup
	wg.Add(workers)
	var ok atomic.Int64

	for i := 0; i < workers; i++ {
		i := i
		go func() {
			defer wg.Done()
			code, _, _ := doJSON(t, app.Router, http.MethodPost, "/api/transfers", map[string]any{
				"from_account_id": aID, "to_account_id": bID, "amount": 10,
				"reference_id": fmt.Sprintf("CX-%d", i),
			})
			if code == http.StatusCreated {
				ok.Add(1)
			}
		}()
	}
	wg.Wait()

	if ok.Load() != 10 {
		t.Fatalf("expected 10 successful transfers from 100, got %d", ok.Load())
	}
	assertBalance(t, app, aID, 0)
	assertBalance(t, app, bID, 100)
}

func assertBalance(t *testing.T, app *testApp, accountID string, want int64) {
	t.Helper()
	_, env, _ := doJSON(t, app.Router, http.MethodGet, "/api/accounts/"+accountID+"/balance", nil)
	var bal map[string]any
	_ = json.Unmarshal(env.Data, &bal)
	got := int64(bal["balance"].(float64))
	if got != want {
		t.Fatalf("account %s balance=%d want=%d", accountID, got, want)
	}
}
