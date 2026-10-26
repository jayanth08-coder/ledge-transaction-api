package tests

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
)

func TestConcurrentDuplicateReference(t *testing.T) {
	app := newTestApp(t)
	acc := createAccount(t, app, "Concurrent Dup")
	id := acc["id"].(string)

	const workers = 20
	var successCount atomic.Int64
	var conflictCount atomic.Int64
	var wg sync.WaitGroup
	wg.Add(workers)

	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			code, _, _ := doJSON(t, app.Router, http.MethodPost, "/api/transactions", map[string]any{
				"account_id": id, "type": "credit", "amount": 10, "reference_id": "SAME-REF",
			})
			switch code {
			case http.StatusCreated:
				successCount.Add(1)
			case http.StatusConflict:
				conflictCount.Add(1)
			}
		}()
	}
	wg.Wait()

	if successCount.Load() != 1 {
		t.Fatalf("expected exactly 1 success, got %d (conflicts=%d)", successCount.Load(), conflictCount.Load())
	}
	if conflictCount.Load() != workers-1 {
		t.Fatalf("expected %d conflicts, got %d", workers-1, conflictCount.Load())
	}

	_, env, _ := doJSON(t, app.Router, http.MethodGet, "/api/accounts/"+id+"/balance", nil)
	var bal map[string]any
	_ = json.Unmarshal(env.Data, &bal)
	if bal["balance"].(float64) != 10 {
		t.Fatalf("balance=%v", bal["balance"])
	}
}

func TestConcurrentDebitsNeverNegative(t *testing.T) {
	app := newTestApp(t)
	acc := createAccount(t, app, "Concurrent Debit")
	id := acc["id"].(string)

	_, _, _ = doJSON(t, app.Router, http.MethodPost, "/api/transactions", map[string]any{
		"account_id": id, "type": "credit", "amount": 10, "reference_id": "OPEN",
	})

	const workers = 20
	var wg sync.WaitGroup
	wg.Add(workers)
	var okCount atomic.Int64

	for i := 0; i < workers; i++ {
		i := i
		go func() {
			defer wg.Done()
			code, _, _ := doJSON(t, app.Router, http.MethodPost, "/api/transactions", map[string]any{
				"account_id": id, "type": "debit", "amount": 1, "reference_id": fmt.Sprintf("DEB-%d", i),
			})
			if code == http.StatusCreated {
				okCount.Add(1)
			}
		}()
	}
	wg.Wait()

	if okCount.Load() != 10 {
		t.Fatalf("expected exactly 10 successful debits of 1 from 10, got %d", okCount.Load())
	}

	_, env, _ := doJSON(t, app.Router, http.MethodGet, "/api/accounts/"+id+"/balance", nil)
	var bal map[string]any
	_ = json.Unmarshal(env.Data, &bal)
	if bal["balance"].(float64) != 0 {
		t.Fatalf("balance should be 0, got %v", bal["balance"])
	}
}

func TestConcurrentCreditsNoLostUpdates(t *testing.T) {
	app := newTestApp(t)
	acc := createAccount(t, app, "Concurrent Credit")
	id := acc["id"].(string)

	const workers = 25
	var wg sync.WaitGroup
	wg.Add(workers)

	for i := 0; i < workers; i++ {
		i := i
		go func() {
			defer wg.Done()
			code, env, _ := doJSON(t, app.Router, http.MethodPost, "/api/transactions", map[string]any{
				"account_id": id, "type": "credit", "amount": 1, "reference_id": fmt.Sprintf("CR-%d", i),
			})
			if code != http.StatusCreated {
				t.Errorf("credit %d failed: %d %v", i, code, env.Error)
			}
		}()
	}
	wg.Wait()

	_, env, _ := doJSON(t, app.Router, http.MethodGet, "/api/accounts/"+id+"/balance", nil)
	var bal map[string]any
	_ = json.Unmarshal(env.Data, &bal)
	if bal["balance"].(float64) != 25 {
		t.Fatalf("expected 25, got %v", bal["balance"])
	}
}
