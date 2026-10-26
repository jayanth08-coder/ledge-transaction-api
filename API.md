# Transaction Ledger API — Documentation

**Base URL (local):** `http://127.0.0.1:8080`  
**Content-Type:** `application/json` (unless noted)  
**Money unit:** whole rupees (`int64`). Example: `100` means **₹100**.

---

## 1. Overview

This API implements a simple **transaction ledger**:

| Capability | Method & path |
| --- | --- |
| Create account | `POST /api/accounts` |
| List accounts | `GET /api/accounts` |
| Get account | `GET /api/accounts/:id` |
| Get balance | `GET /api/accounts/:id/balance` |
| Transaction history | `GET /api/accounts/:id/transactions` |
| Credit / debit | `POST /api/transactions` |
| Transfer (bonus) | `POST /api/transfers` |
| Health | `GET /health` |

Web UI (Templ + HTMX): `GET /`, `GET /accounts`, `GET /accounts/:id`.

---

## 2. Response envelope

### Success

```json
{
  "success": true,
  "data": { }
}
```

### Error

```json
{
  "success": false,
  "error": {
    "code": "MACHINE_READABLE_CODE",
    "message": "Human-readable explanation.",
    "fields": [
      { "field": "amount", "message": "must be > 0" }
    ]
  }
}
```

`fields` is optional (validation errors only).

---

## 3. HTTP status policy

| Status | Meaning |
| --- | --- |
| `200 OK` | Successful read |
| `201 Created` | Resource created |
| `400 Bad Request` | Malformed body, invalid type/amount/pagination/filter |
| `404 Not Found` | Account does not exist |
| `409 Conflict` | Duplicate `reference_id` (transaction or transfer) |
| `422 Unprocessable Entity` | Insufficient balance / amount overflow |
| `500 Internal Server Error` | Unexpected failure (no internal SQL leaked) |
| `503 Service Unavailable` | Database temporarily busy/unavailable |

---

## 4. Error codes

| Code | Typical status | When |
| --- | --- | --- |
| `ACCOUNT_NOT_FOUND` | 404 | Unknown account ID |
| `MISSING_ACCOUNT_NAME` | 400 | Empty name |
| `INVALID_TRANSACTION_TYPE` | 400 | Type not `credit` / `debit` |
| `INVALID_FILTER_TYPE` | 400 | Bad `type` query on history |
| `INVALID_AMOUNT` | 400 | Amount ≤ 0 |
| `AMOUNT_OVERFLOW` | 422 | Balance would overflow `int64` |
| `INVALID_REFERENCE_ID` | 400 | Explicit ref invalid/too long |
| `DUPLICATE_REFERENCE_ID` | 409 | Same `(account_id, reference_id)` again |
| `INSUFFICIENT_BALANCE` | 422 | Debit/transfer exceeds funds |
| `INVALID_PAGINATION` | 400 | Bad `page` / `limit` |
| `INVALID_REQUEST_BODY` | 400 | Unparseable JSON/form |
| `INVALID_TRANSFER` | 400 | Incomplete transfer payload |
| `SAME_ACCOUNT_TRANSFER` | 400 | From == to |
| `DUPLICATE_TRANSFER_REFERENCE` | 409 | Same transfer `reference_id` again |
| `DATABASE_UNAVAILABLE` | 503 | Lock/busy |
| `INTERNAL_ERROR` | 500 | Unexpected |

---

## 5. Accounts

### 5.1 Create account

`POST /api/accounts`

**Rules**

- Server generates `id` and `created_at` (UTC).
- Initial `balance` is always `0` (client cannot set it).
- `name` is required (trimmed, max 255).

**Request**

```json
{ "name": "Priya Sharma" }
```

**Response `201`**

```json
{
  "success": true,
  "data": {
    "id": "c5c5acc3-e457-4329-944d-65a304996640",
    "name": "Priya Sharma",
    "balance": 0,
    "created_at": "2026-10-27T00:00:00Z"
  }
}
```

Protected fields (`id`, `balance`, `created_at`) in the body are **ignored** (not mass-assigned).

---

### 5.2 List accounts

`GET /api/accounts?page=1&limit=3`

| Query | Default | Notes |
| --- | --- | --- |
| `page` | `1` | Must be ≥ 1 |
| `limit` | `3` | `1…100` |

**Response `200`**

```json
{
  "success": true,
  "data": {
    "data": [ { "id": "…", "name": "…", "balance": 120, "created_at": "…" } ],
    "pagination": {
      "current_page": 1,
      "page_size": 3,
      "total_records": 6,
      "total_pages": 2
    }
  }
}
```

---

### 5.3 Get account

`GET /api/accounts/:id` → `200` account object, or `404 ACCOUNT_NOT_FOUND`.

---

### 5.4 Get balance

`GET /api/accounts/:id/balance`

**Response `200`**

```json
{
  "success": true,
  "data": {
    "account_id": "demo-account-alice",
    "balance": 120
  }
}
```

Balance is the **stored ledger balance**, not a recalculation over a paginated history page.

---

## 6. Transactions

### 6.1 Create credit / debit

`POST /api/transactions`

**Request**

```json
{
  "account_id": "demo-account-alice",
  "type": "credit",
  "amount": 50,
  "description": "Salary credit",
  "reference_id": "PAY-001"
}
```

| Field | Required | Notes |
| --- | --- | --- |
| `account_id` | yes | Must exist |
| `type` | yes | `credit` or `debit` |
| `amount` | yes | Integer rupees, `> 0` |
| `description` | no | Max 500 chars |
| `reference_id` | no | If blank, server generates `AUTO-{uuid}` |

**Credit:** balance increases atomically with the insert.  
**Debit:** rejected with `422 INSUFFICIENT_BALANCE` if funds are short — **no** ledger row, **no** balance change.

**Response `201`**

```json
{
  "success": true,
  "data": {
    "id": "…",
    "account_id": "demo-account-alice",
    "type": "credit",
    "amount": 50,
    "description": "Salary credit",
    "reference_id": "PAY-001",
    "created_at": "2026-10-27T00:00:00Z"
  }
}
```

Transactions are **immutable** via API (no update/delete routes).

---

### 6.2 Duplicate reference ID

For the **same account**, `(account_id, reference_id)` is unique (database constraint).

| Case | Result |
| --- | --- |
| First use of `PAY-001` on account A | `201`, one row, balance updated |
| Repeat `PAY-001` on account A | `409 DUPLICATE_REFERENCE_ID`, no second row, balance unchanged |
| `PAY-001` on account B | Allowed |

**Response `409`**

```json
{
  "success": false,
  "error": {
    "code": "DUPLICATE_REFERENCE_ID",
    "message": "Duplicate reference ID — this reference already exists for the account. No new transaction was created and the balance was not changed."
  }
}
```

---

### 6.3 Transaction history (filter + pagination + sort)

`GET /api/accounts/:id/transactions`

| Query | Default | Notes |
| --- | --- | --- |
| `type` | _(all)_ | `credit`, `debit`, or omit |
| `page` | `1` | ≥ 1 |
| `limit` | `3` | `1…100` |

**Backend behaviour (assessment-aligned)**

1. Apply account + optional `type` filter in SQL.  
2. `Count` filtered rows → `total_records` / `total_pages`.  
3. Fetch page with `Limit` + `Offset`.  
4. Sort **latest-first**: `ORDER BY created_at DESC, id DESC`.

**Example**

```http
GET /api/accounts/demo-account-alice/transactions?type=credit&page=1&limit=3
```

**Response `200`**

```json
{
  "success": true,
  "data": {
    "data": [ /* transaction objects */ ],
    "pagination": {
      "current_page": 1,
      "page_size": 3,
      "total_records": 2,
      "total_pages": 1
    }
  }
}
```

Empty pages beyond the end return `"data": []` with correct pagination metadata.

---

## 7. Transfers (bonus)

`POST /api/transfers`

```json
{
  "from_account_id": "demo-account-alice",
  "to_account_id": "demo-account-bob",
  "amount": 20,
  "reference_id": "XFER-001",
  "description": "Rent share"
}
```

**Rules**

- Both accounts must exist; from ≠ to; amount `> 0`.
- Debit + credit + transfer row commit in **one** DB transaction.
- `reference_id` optional (auto `AUTO-…` if blank); **globally unique** on `transfers`.
- Ledger refs: `{reference_id}:debit` and `{reference_id}:credit`.
- Duplicate transfer ref → `409 DUPLICATE_TRANSFER_REFERENCE`.
- Insufficient source funds → `422`; nothing persisted.

---

## 8. curl examples

```bash
# Create account
curl -s -X POST http://127.0.0.1:8080/api/accounts \
  -H "Content-Type: application/json" \
  -d '{"name":"Priya Sharma"}'

# Credit ₹100
curl -s -X POST http://127.0.0.1:8080/api/transactions \
  -H "Content-Type: application/json" \
  -d '{"account_id":"ACCOUNT_ID","type":"credit","amount":100,"reference_id":"PAY-001","description":"Deposit"}'

# Balance
curl -s http://127.0.0.1:8080/api/accounts/ACCOUNT_ID/balance

# History (page 1, size 3, credits only)
curl -s "http://127.0.0.1:8080/api/accounts/ACCOUNT_ID/transactions?type=credit&page=1&limit=3"

# Transfer
curl -s -X POST http://127.0.0.1:8080/api/transfers \
  -H "Content-Type: application/json" \
  -d '{"from_account_id":"A","to_account_id":"B","amount":20,"reference_id":"XFER-001"}'
```

**PowerShell**

```powershell
Invoke-RestMethod -Method Post -Uri http://127.0.0.1:8080/api/accounts `
  -ContentType application/json -Body '{"name":"Priya Sharma"}'
```

---

## 9. Demo dataset (SEED_DEMO=true)

| Account ID | Name | Balance (₹) |
| --- | --- | --- |
| `demo-account-alice` | Alice Sharma | 120 |
| `demo-account-bob` | Bob Kumar | 80 |
| `demo-account-cara` | Cara Patel | 25 |
| `demo-account-dev` | Dev Ops Ledger | 0 |
| `demo-account-eva` | Eva Nair | 200 |
| `demo-account-finance` | Finance Pool | 150 |

Seed is **idempotent** (safe on restart). Fresh DB: use `RESET_DB=true` with the setup script.

---

## 10. Atomicity notes

Credit/debit/transfer run inside a GORM transaction:

1. Validate request  
2. Insert immutable ledger row(s) (uniqueness enforced by DB)  
3. Conditional balance `UPDATE` (`balance + ?` / `balance >= ?` then subtract)  
4. Commit or full rollback  

Concurrent duplicate refs and concurrent debits are covered by uniqueness + conditional updates (SQLite uses a single writer connection for safety).
