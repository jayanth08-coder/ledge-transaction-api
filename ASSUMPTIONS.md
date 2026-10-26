# Assumptions & Design Decisions

This document records explicit assumptions made while implementing the Transaction Ledger assessment. Where the brief was ambiguous, a sensible default was chosen and documented here.

---

## 1. Money representation

| Assumption | Rationale |
| --- | --- |
| Amounts and balances are **whole rupees** stored as `int64` | Matches assessment examples (₹100, ₹50) without floating-point risk |
| Clients send integers only (e.g. `"amount": 100`) | Avoids JSON float coercion issues |
| No multi-currency / FX | Out of scope for a single-ledger assessment |

**Earlier minor-unit (paise) design was replaced** so the API reads naturally as rupees for evaluators and UI users.

----

## 2. Identifiers & timestamps

| Assumption | Rationale |
| --- | --- |
| Account / transaction / transfer IDs are UUIDs generated on the server | Prevents client-controlled primary keys |
| All timestamps are UTC | Consistent across locales |
| Clients cannot set `id`, `balance`, or `created_at` on create | Mass-assignment protection via dedicated request DTOs |

---

## 3. Reference ID behaviour

| Assumption | Rationale |
| --- | --- |
| `reference_id` is **optional** on create | Improves UX for normal users |
| Blank / whitespace → server generates `AUTO-{uuid}` | Still unique; returned in API body and history |
| Explicit refs remain required for intentional duplicate tests | Evaluators can still force `409 DUPLICATE_REFERENCE_ID` |
| Uniqueness is **per account** `(account_id, reference_id)`, not global | Matches assessment Case C (same ref, different accounts) |
| Enforcement is a **database unique index**, mapped from constraint violations | Concurrent duplicates are safe without relying on a pre-check alone |

---

## 4. Pagination, filtering, sorting

| Assumption | Rationale |
| --- | --- |
| Default `page=1`, default `limit=3`, max `limit=100` | Aligns with product preference while remaining assessment-valid |
| Filter (`type`) applied **before** `Count` and page fetch | Prevents wrong totals / wrong pages |
| Fixed sort: `created_at DESC, id DESC` | Latest-first + deterministic tie-break; no SQL injection via sort params |
| Beyond-last-page returns empty `data` with correct metadata | Stable client behaviour |
| UI Prev/Next call `GET /api/accounts/:id/transactions` | Pagination is visibly backend-driven (same contract as curl) |

---

## 5. Debits, credits & integrity

| Assumption | Rationale |
| --- | --- |
| Credit/debit + balance update share one DB transaction | Atomicity requirement |
| Debit uses conditional SQL: `WHERE balance >= amount` | Concurrent debits cannot go negative |
| Failed debit creates **no** transaction row | Assessment insufficient-balance rule |
| No transaction update/delete endpoints | Immutability |
| No API to set balance directly | Balance is always the effect of committed ledger rows |

---

## 6. Transfers (bonus)

| Assumption | Rationale |
| --- | --- |
| Dedicated `transfers` table with **global** unique `reference_id` | Transfer idempotency must not be bypassed by juggling per-account refs |
| Side effects recorded as two ledger rows (`…:debit` / `…:credit`) | History remains a complete account ledger |
| Source must have sufficient funds; both accounts must exist; from ≠ to | Assessment transfer rules |
| Entire transfer rolls back on any failure | No partial debit without credit |

---

## 7. Persistence & SQLite

| Assumption | Rationale |
| --- | --- |
| SQLite file at `data/ledger.db` (configurable) | Local assessment simplicity |
| Foreign keys ON, busy timeout, WAL | Safer local integrity |
| `MaxOpenConns(1)` | Serializes writers; documents SQLite concurrency limits |
| Schema via GORM `AutoMigrate` + triggers for amount/type checks | Fast greenfield setup; limitations acknowledged in README |
| Production high-write systems may need PostgreSQL | Honest operational boundary |

---

## 8. Demo seeding

| Assumption | Rationale |
| --- | --- |
| Seed only when `SEED_DEMO=true` | Avoid surprising production-like runs |
| Six deterministic accounts + sample credits/debits | Evaluator can open UI/API immediately |
| Idempotent seed (skip ledger inserts if Alice already has rows) | Restarts do not inflate balances |
| `RESET_DB=true` in setup scripts deletes `data/*.db*` first | Clean slate for demos |

---

## 9. API & UI separation

| Assumption | Rationale |
| --- | --- |
| REST JSON endpoints are the contract for automation / evaluation | Matches submission expectations |
| HTMX UI consumes the same `/api/*` JSON for mutations & pagination | One backend path of truth |
| HTML fragments may still be used for some page shells | Faster first paint; not a second business layer |
| API JSON status panel logs backend JSON only | Debugging aid; not part of the assessment API |

---

## 10. Security & multi-tenancy

| Assumption | Rationale |
| --- | --- |
| **No authentication / authorization** in this build | Assessment does not require auth; ledger is intentionally open locally |
| Owner-key isolation was prototyped then **removed** per product feedback | Prefer an open assessment ledger over a partial auth model |
| Request size capped (~1 MiB) | Basic abuse mitigation |

If multi-user isolation is required later, introduce proper auth (sessions/JWT) and an `owner_id` foreign key — not a cookie-only workaround.

---

## 11. Testing stance

| Assumption | Rationale |
| --- | --- |
| `httptest` + temp SQLite files for integration tests | Realistic without external services |
| Race detector needs CGO (often unavailable on stock Windows Go) | Documented; concurrency correctness still asserted without `-race` |
| Pure-Go SQLite driver (`glebarez/sqlite`) | Runs without a C toolchain on Windows |

---

## 12. Out of scope (intentionally)

- Authentication, RBAC, audit log export  
- Soft deletes, transaction reversals / chargebacks as first-class types  
- Decimal/minor-unit currencies, multi-currency wallets  
- Horizontal scale / multi-node SQLite  
- Full versioned SQL migration framework (Flyway/goose-style)

These can be layered on without changing the core credit/debit invariants documented above.
