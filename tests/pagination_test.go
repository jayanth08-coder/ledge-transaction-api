package tests

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/ledge/ledge-transaction-api/internal/domain"
)

func seedTransactions(t *testing.T, app *testApp, accountID string, n int) {
	t.Helper()
	// Credit a large balance first so subsequent debits succeed.
	code, _, _ := doJSON(t, app.Router, http.MethodPost, "/api/transactions", map[string]any{
		"account_id": accountID, "type": domain.TxnTypeCredit, "amount": int64(n * 1000), "reference_id": "SEED-CREDIT",
	})
	if code != http.StatusCreated {
		t.Fatalf("seed credit: %d", code)
	}
	for i := 1; i < n; i++ {
		typ := domain.TxnTypeCredit
		amt := int64(10)
		if i%2 == 0 {
			typ = domain.TxnTypeDebit
			amt = 5
		}
		code, env, _ := doJSON(t, app.Router, http.MethodPost, "/api/transactions", map[string]any{
			"account_id": accountID, "type": typ, "amount": amt, "reference_id": fmt.Sprintf("REF-%03d", i),
			"description": fmt.Sprintf("txn %d", i),
		})
		if code != http.StatusCreated {
			t.Fatalf("seed txn %d: %d %v", i, code, env.Error)
		}
		time.Sleep(2 * time.Millisecond) // help ordering across created_at
	}
}

func TestFilteringAndPagination(t *testing.T) {
	app := newTestApp(t)
	acc := createAccount(t, app, "Page User")
	id := acc["id"].(string)
	seedTransactions(t, app, id, 15) // 1 seed credit + 14 more = 15

	// Defaults
	code, env, _ := doJSON(t, app.Router, http.MethodGet, "/api/accounts/"+id+"/transactions", nil)
	if code != http.StatusOK {
		t.Fatalf("status=%d", code)
	}
	var page map[string]any
	_ = json.Unmarshal(env.Data, &page)
	data := page["data"].([]any)
	pag := page["pagination"].(map[string]any)
	if len(data) != 3 {
		t.Fatalf("default page size should be 3, got %d", len(data))
	}
	if pag["current_page"].(float64) != 1 || pag["page_size"].(float64) != 3 {
		t.Fatalf("pagination defaults=%v", pag)
	}
	if pag["total_records"].(float64) != 15 {
		t.Fatalf("total_records=%v", pag["total_records"])
	}
	if pag["total_pages"].(float64) != 5 {
		t.Fatalf("total_pages=%v", pag["total_pages"])
	}

	// Credit filter
	code, env, _ = doJSON(t, app.Router, http.MethodGet, "/api/accounts/"+id+"/transactions?type=credit&limit=100", nil)
	_ = json.Unmarshal(env.Data, &page)
	creditTotal := int(page["pagination"].(map[string]any)["total_records"].(float64))
	for _, row := range page["data"].([]any) {
		if row.(map[string]any)["type"] != "credit" {
			t.Fatal("credit filter returned non-credit")
		}
	}

	// Debit filter
	code, env, _ = doJSON(t, app.Router, http.MethodGet, "/api/accounts/"+id+"/transactions?type=debit&limit=100", nil)
	_ = json.Unmarshal(env.Data, &page)
	debitTotal := int(page["pagination"].(map[string]any)["total_records"].(float64))
	for _, row := range page["data"].([]any) {
		if row.(map[string]any)["type"] != "debit" {
			t.Fatal("debit filter returned non-debit")
		}
	}
	if creditTotal+debitTotal != 15 {
		t.Fatalf("filter split %d+%d != 15", creditTotal, debitTotal)
	}

	// Invalid type
	code, env, _ = doJSON(t, app.Router, http.MethodGet, "/api/accounts/"+id+"/transactions?type=foo", nil)
	if code != http.StatusBadRequest || env.Error.Code != "INVALID_FILTER_TYPE" {
		t.Fatalf("invalid type => %d %v", code, env.Error)
	}

	// Invalid pagination
	for _, q := range []string{"page=0", "page=-1", "limit=0", "limit=-5", "limit=101", "page=abc"} {
		code, env, _ = doJSON(t, app.Router, http.MethodGet, "/api/accounts/"+id+"/transactions?"+q, nil)
		if code != http.StatusBadRequest {
			t.Fatalf("%s => %d %v", q, code, env.Error)
		}
	}

	// Beyond last page
	code, env, _ = doJSON(t, app.Router, http.MethodGet, "/api/accounts/"+id+"/transactions?page=99&limit=10", nil)
	_ = json.Unmarshal(env.Data, &page)
	if len(page["data"].([]any)) != 0 {
		t.Fatal("beyond last page should return empty data")
	}
	if page["pagination"].(map[string]any)["total_records"].(float64) != 15 {
		t.Fatal("metadata total should remain correct beyond last page")
	}

	// Latest-first sorting
	code, env, _ = doJSON(t, app.Router, http.MethodGet, "/api/accounts/"+id+"/transactions?limit=100", nil)
	_ = json.Unmarshal(env.Data, &page)
	rows := page["data"].([]any)
	for i := 1; i < len(rows); i++ {
		prev := rows[i-1].(map[string]any)["created_at"].(string)
		cur := rows[i].(map[string]any)["created_at"].(string)
		if prev < cur {
			t.Fatalf("not latest-first: %s then %s", prev, cur)
		}
	}
}

func TestEmptyTransactionHistory(t *testing.T) {
	app := newTestApp(t)
	acc := createAccount(t, app, "Empty")
	code, env, _ := doJSON(t, app.Router, http.MethodGet, "/api/accounts/"+acc["id"].(string)+"/transactions", nil)
	if code != http.StatusOK {
		t.Fatalf("status=%d", code)
	}
	var page map[string]any
	_ = json.Unmarshal(env.Data, &page)
	if len(page["data"].([]any)) != 0 {
		t.Fatal("expected empty data")
	}
	if page["pagination"].(map[string]any)["total_pages"].(float64) != 0 {
		t.Fatalf("total_pages=%v", page["pagination"])
	}
}
