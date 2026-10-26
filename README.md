# Transaction Ledger API

Production-quality **Transaction Ledger System** built for a Full Stack Developer Assessment.

The application provides a Gin REST API and an HTMX + Templ web UI on top of GORM and SQLite. Money is stored as **whole rupees** (`int64`): **amount `100` means ?100**. Floating-point amounts are never used.

## Quick start (install + seed + run)

**macOS / Linux / Git Bash**

```bash
chmod +x scripts/setup-and-run.sh
./scripts/setup-and-run.sh
```

**Windows PowerShell**

```powershell
./scripts/setup-and-run.ps1
```

The script installs Go/templ if missing (where supported), clears `data/*.db*`, seeds demo data (`SEED_DEMO=true`), and starts the server.

Skip DB wipe: `RESET_DB=false ./scripts/setup-and-run.sh`

## Documentation

| Doc | Purpose |
| --- | --- |
| [docs/API.md](docs/API.md) | Full API reference (endpoints, schemas, errors, curl) |
| [docs/ASSUMPTIONS.md](docs/ASSUMPTIONS.md) | Assumptions & design decisions for evaluation |

## Assessment objective

Implement a correct, concurrency-safe ledger that:

- Creates accounts with a server-generated ID and zero balance
- Records immutable credit/debit transactions
- Enforces per-account `reference_id` uniqueness (HTTP 409 on duplicates)
- Rejects insufficient-balance debits without mutating state
- Supports filtered, paginated, latest-first transaction history
- Optionally transfers money atomically between accounts

## Technology stack

| Layer | Choice |
| --- | --- |
| Language | Go 1.22+ |
| HTTP | Gin |
| ORM | GORM |
| Database | SQLite via `github.com/glebarez/sqlite` (pure Go, no CGO) |
| Templates | Templ |
| Interactivity | HTMX |
| Styling | Bootstrap 5 + light custom CSS |
| Modules | Go modules |

## Architecture overview

Layered design with dependency injection at composition root (`cmd/server`):

1. **Handlers** — HTTP binding, status mapping, JSON vs HTMX fragments
2. **Services** — validation, business rules, idempotency, transfers
3. **Repositories** — GORM queries, conditional balance updates
4. **Domain** — models, DTOs, sentinel errors
5. **Database** — SQLite open/pragmas, AutoMigrate + triggers, optional demo seed

Balance-changing operations always run inside a GORM `Transaction`. Credits/debits use conditional `UPDATE ... SET balance = balance ± ?` and check `RowsAffected`. Duplicate references are detected from the database unique constraint, not a pre-check alone.

## Folder structure

```
ledge-transaction-api/
+-- cmd/server/main.go
+-- internal/
¦   +-- config/
¦   +-- database/
¦   +-- domain/
¦   +-- repository/
¦   +-- service/
¦   +-- handlers/
¦   +-- routes/
¦   +-- middleware/
+-- templates/          # Templ sources (+ generated *_templ.go)
+-- static/css|js/
+-- data/               # SQLite file created at runtime
+-- tests/
+-- .env.example
+-- Makefile
+-- README.md
```

## Prerequisites

- Go 1.22 or newer
- [Templ CLI](https://templ.guide/) (`go install github.com/a-h/templ/cmd/templ@latest`)
- Make (optional; commands below also work without it)

### Windows PowerShell notes

```powershell
$env:Path = [System.Environment]::GetEnvironmentVariable("Path","Machine") + ";" +
            [System.Environment]::GetEnvironmentVariable("Path","User") + ";" +
            "$env:USERPROFILE\go\bin"
```

## Installation

```bash
git clone <repo-url> ledge-transaction-api
cd ledge-transaction-api
go mod download
templ generate
```

Windows PowerShell:

```powershell
go mod download
templ generate
```

Generated `*_templ.go` files are committed so a fresh checkout can build even before installing the Templ CLI. Re-run `templ generate` after editing `.templ` files.

## Environment configuration

Copy `.env.example` if desired. The app runs with defaults when no `.env` is present.

| Variable | Default | Description |
| --- | --- | --- |
| `APP_ENV` | `development` | `production` enables Gin release mode |
| `APP_HOST` | `127.0.0.1` | Bind host |
| `APP_PORT` | `8080` | Bind port |
| `DATABASE_PATH` | `data/ledger.db` | SQLite file path |
| `SEED_DEMO` | `false` | Seed demo accounts when `true` |
| `LOG_LEVEL` | `info` | `debug` / `info` / `error` / `silent` |

## Database initialization

On startup the server:

1. Creates the `data/` directory if needed
2. Opens SQLite with `_foreign_keys=on`, `_busy_timeout=5000`, WAL mode
3. Runs GORM `AutoMigrate` for `accounts`, `transactions`, `transfers`
4. Installs CHECK-style triggers for positive amounts and valid types
5. Optionally seeds demo data

Existing data is preserved across restarts. Balances are never reset. The database file is gitignored.

### Migration strategy

`AutoMigrate` is used for greenfield/additive schema. Limitations: it does not drop/rename columns or safely transform arbitrary existing schemas. Supplemental SQLite triggers enforce amount/type invariants that AutoMigrate cannot express cleanly.

### Schema summary

**accounts**

| Column | Notes |
| --- | --- |
| `id` PK | UUID string |
| `name` | required, max 255 |
| `balance` | `int64` minor units, default 0 |
| `created_at` | UTC |

**transactions** (immutable via API — no update/delete routes)

| Column | Notes |
| --- | --- |
| `id` PK | UUID |
| `account_id` FK | indexed |
| `type` | `credit` \| `debit` |
| `amount` | positive `int64` minor units |
| `description` | max 500 |
| `reference_id` | optional (auto `AUTO-?` if blank); **unique with `account_id`** |
| `created_at` | UTC; sorted `DESC`, then `id DESC` |

**transfers**

| Column | Notes |
| --- | --- |
| `id` PK | Transfer UUID |
| `from_account_id` / `to_account_id` | both required |
| `amount` | positive minor units |
| `reference_id` | **globally unique** idempotency key |
| `debit_txn_id` / `credit_txn_id` | linked ledger rows |
| `description` | optional |
| `created_at` | UTC |

Ledger rows for a transfer use deterministic references `{reference_id}:debit` and `{reference_id}:credit`.

## Demo seeding

```bash
# macOS/Linux
SEED_DEMO=true go run ./cmd/server

# Windows PowerShell
$env:SEED_DEMO="true"; go run ./cmd/server
```

Idempotent demo state (only inserted when Alice has no transactions yet):

| Account ID | Name | Expected balance after seed |
| --- | --- | --- |
| `demo-account-alice` | Alice Demo | 12000 (?120.00) |
| `demo-account-bob` | Bob Demo | 5000 (?50.00) |

Alice ledger: credit 10000, credit 5000, debit 3000. Bob: credit 5000.

## Run the application

```bash
make run
# or
templ generate && go run ./cmd/server
```

Windows PowerShell:

```powershell
templ generate
go run ./cmd/server
```

Open http://127.0.0.1:8080/

Useful Make targets: `make build`, `make test`, `make test-race`, `make templ`, `make fmt`, `make vet`.

## Money representation

All API amounts and balances are **whole rupees** (integers):

- `100` ? ?100
- `30` ? ?30

Clients must send whole numbers (e.g. `"amount": 50`). Values = 0 or overflow are rejected.

`reference_id` may be omitted or blank ? the server generates an `AUTO-?` value, returns it in the JSON response, and shows it in transaction history. Supply an explicit ID to test duplicate `409` handling.

Default pagination page size is **3** (max 100).

## API endpoints

### Accounts

| Method | Path | Description |
| --- | --- | --- |
| `POST` | `/api/accounts` | Create account (balance always 0) |
| `GET` | `/api/accounts` | List accounts (`page`, `limit`) |
| `GET` | `/api/accounts/:id` | Account details |
| `GET` | `/api/accounts/:id/balance` | `{ account_id, balance }` |
| `GET` | `/api/accounts/:id/transactions` | Filtered/paginated history |

### Transactions

| Method | Path | Description |
| --- | --- | --- |
| `POST` | `/api/transactions` | Create credit/debit |

No transaction update or delete endpoints exist.

### Transfers

| Method | Path | Description |
| --- | --- | --- |
| `POST` | `/api/transfers` | Atomic transfer between accounts |

### Web UI

| Method | Path | Description |
| --- | --- | --- |
| `GET` | `/` | Dashboard |
| `GET` | `/accounts` | Account management |
| `GET` | `/accounts/:id` | Detail + forms + history |
| `GET` | `/accounts/:id/transactions` | HTMX table fragment |
| `GET` | `/health` | Health check |

## Example requests

### Create account

```bash
curl -s -X POST http://127.0.0.1:8080/api/accounts \
  -H "Content-Type: application/json" \
  -d '{"name":"Priya Sharma"}'
```

PowerShell:

```powershell
Invoke-RestMethod -Method Post -Uri http://127.0.0.1:8080/api/accounts `
  -ContentType application/json -Body '{"name":"Priya Sharma"}'
```

Success (`201`):

```json
{
  "success": true,
  "data": {
    "id": "…",
    "name": "Priya Sharma",
    "balance": 0,
    "created_at": "2026-10-27T00:00:00Z"
  }
}
```

### Credit / debit

```bash
curl -s -X POST http://127.0.0.1:8080/api/transactions \
  -H "Content-Type: application/json" \
  -d '{"account_id":"ACCOUNT_ID","type":"credit","amount":10000,"reference_id":"PAY-001","description":"Initial deposit"}'
```

### Duplicate reference (`409`)

Repeating the same `account_id` + `reference_id` returns:

```json
{
  "success": false,
  "error": {
    "code": "DUPLICATE_REFERENCE_ID",
    "message": "This reference ID already exists for the account."
  }
}
```

The same `reference_id` on a **different** account is allowed.

### Transaction history

```bash
curl -s "http://127.0.0.1:8080/api/accounts/ACCOUNT_ID/transactions?type=credit&page=1&limit=3"
```

Pagination is handled entirely in the backend with GORM `Count` / `Limit` / `Offset` (filter applied before count and page fetch). The UI Prev/Next controls call this same API.

- Default `page=1`, `limit=3`, max `limit=100`
- Filter applied **before** count/pagination
- Sort: `created_at DESC, id DESC` (latest-first)
- Invalid `type` / pagination ? `400`
- Response metadata: `current_page`, `page_size`, `total_records`, `total_pages`

Response shape:

```json
{
  "success": true,
  "data": {
    "data": [ /* transactions */ ],
    "pagination": {
      "current_page": 1,
      "page_size": 10,
      "total_records": 42,
      "total_pages": 5
    }
  }
}
```

### Transfer

```bash
curl -s -X POST http://127.0.0.1:8080/api/transfers \
  -H "Content-Type: application/json" \
  -d '{"from_account_id":"A","to_account_id":"B","amount":3000,"reference_id":"XFER-001","description":"Rent share"}'
```

Transfer idempotency: `transfers.reference_id` is globally unique. Retries return `DUPLICATE_TRANSFER_REFERENCE` (`409`) without changing balances.

## HTTP status and error codes

| Status | When |
| --- | --- |
| `200` | Successful reads |
| `201` | Created resources |
| `400` | Malformed body, validation, bad pagination/filter |
| `404` | Account not found |
| `409` | Duplicate transaction or transfer reference |
| `422` | Insufficient balance, amount overflow |
| `500` | Unexpected failure (no internal SQL leaked) |
| `503` | Database busy/unavailable |

Selected error codes: `ACCOUNT_NOT_FOUND`, `MISSING_ACCOUNT_NAME`, `INVALID_AMOUNT`, `INVALID_TRANSACTION_TYPE`, `DUPLICATE_REFERENCE_ID`, `INSUFFICIENT_BALANCE`, `INVALID_PAGINATION`, `INVALID_FILTER_TYPE`, `SAME_ACCOUNT_TRANSFER`, `DUPLICATE_TRANSFER_REFERENCE`, `DATABASE_UNAVAILABLE`, `INTERNAL_ERROR`.

## Atomicity and concurrency

For each credit/debit:

1. Begin DB transaction
2. Load account
3. Insert immutable transaction row (unique `(account_id, reference_id)`)
4. Conditionally update balance (`balance + amount` or `balance >= amount` then subtract)
5. Commit — or roll back on any failure

Invariants:

- Concurrent duplicate references ? exactly one ledger row
- Concurrent debits ? never negative balance
- Failed insert/update ? no partial balance change

SQLite note: the pool uses `MaxOpenConns(1)` so writers serialize. This is appropriate for a local assessment. Higher write concurrency in production typically requires PostgreSQL.

## HTMX frontend

- Account form posts to `/api/accounts` and shows success/error alerts
- Transaction and transfer forms post asynchronously and refresh the history panel
- Filters and pagination call `GET /api/accounts/:id/transactions` (backend JSON) and render from that response
- REST JSON endpoints remain independent of HTML fragments

## Automated tests

```bash
go test ./...
go test -race ./...   # requires CGO (not available by default on stock Windows Go installs)
```

Coverage includes account creation rules, credit/debit, insufficient funds, duplicate references, filtering/pagination/sorting, concurrent duplicates/debits/credits, and transfers.

On Windows without a C toolchain, `go test -race` reports that race requires CGO. Concurrent correctness is still covered by the non-race concurrency tests (unique constraint + conditional balance updates), and race detection can be run on Linux/macOS CI.

## Assumptions and design decisions

1. Minor units are the only monetary representation in storage and API.
2. Protected fields are omitted from request DTOs (not mass-assigned).
3. Transfer idempotency uses a dedicated `transfers` table rather than only per-account transaction refs.
4. SQLite write serialization + conditional SQL updates provide concurrency safety for the assessment scope.
5. Templ generated Go is committed for evaluator convenience.

## Known limitations

- SQLite is not ideal for high concurrent write throughput.
- `AutoMigrate` is not a full versioned migration system.
- No authentication/authorization layer (out of assessment scope).
- Single-currency minor units; no FX.

## Troubleshooting

| Issue | Fix |
| --- | --- |
| `templ: command not found` | Install CLI and ensure `$(go env GOPATH)/bin` is on `PATH` |
| Compile errors about missing `*_templ.go` | Run `templ generate` |
| `database is locked` | Retry; busy timeout is 5s; avoid multiple server processes on one DB file |
| Port in use | Set `APP_PORT` to a free port |
| Stale demo expectations | Delete `data/ledger.db*` and restart with `SEED_DEMO=true` |

## License

Assessment submission — use as required by the evaluating organization.
