# Phase 4: Portfolio & Transaction UI

**Goal:** Users can create portfolios, log buy/sell/dividend transactions, and see their current holdings derived from transaction history.

**Status:** Complete

---

## What Was Built

### Angular Feature Module: `features/portfolio/`

```
features/portfolio/
├── portfolio.routes.ts                              Lazy-loaded via dashboard.routes.ts
├── models/
│   ├── portfolio.model.ts                           Portfolio, CreatePortfolioInput, UpdatePortfolioInput
│   └── transaction.model.ts                         Transaction, CreateTransactionInput, Holding (derived)
├── services/
│   ├── portfolio.service.ts                         HTTP CRUD + signal state
│   ├── portfolio.service.spec.ts                    6 unit tests
│   ├── transaction.service.ts                       HTTP CRUD + computed holdings signal
│   └── transaction.service.spec.ts                  19 unit tests (7 for deriveHoldings, 12 for service)
├── pages/
│   ├── portfolio-list/                              /portfolios — PrimeNG DataTable, create/edit/delete dialogs
│   └── portfolio-detail/                            /portfolios/:id — p-tabs: Holdings + Transactions
└── components/
    ├── portfolio-form/                              Reactive form (create + edit mode), dialog-ready
    ├── transaction-form/                            Conditional form by transaction type + validation tests
    └── holdings-table/                              Pure display component (live prices deferred to Phase 5)
```

### Routing

`app.routes.ts → dashboard.routes.ts → portfolio.routes.ts`

All feature components are loaded lazily via `loadChildren` and `loadComponent`. The root path (`/`) now redirects to `/portfolios`.

---

## Holdings Derivation Logic

Holdings are **never stored** in the database. They are derived from the transaction ledger on the client using `deriveHoldings()` — a pure function exported from `transaction.service.ts`.

### Algorithm

```
For each transaction, group by symbol:
  buy / reinvested_dividend → netQty += quantity; totalCost += total_amount
  sell                      → netQty -= quantity
  dividend                  → no impact on shares or cost basis (income event only)

For each symbol where netQty > 0:
  avgCostBasis = totalCost / netQty
  Emit as Holding { symbol, quantity, avgCostBasis, totalCost }
```

**Why this approach?**

ADR 007 establishes transactions as the single source of truth. Storing a derived `quantity` or `costBasis` field would require keeping it in sync with every transaction mutation — a class of bugs that is easy to introduce and hard to detect. Computing from the ledger is always correct and requires no synchronisation logic.

**Cost basis on sell:** Sells reduce the quantity held but do **not** reduce `totalCost`. This preserves the weighted average cost basis of the remaining shares. Tax lot tracking (FIFO/LIFO) is a post-MVP feature; the transaction date and price are already captured to support it.

**Dividends:** Cash dividends are income events. They do not affect the number of shares held or the cost basis of existing shares. Reinvested dividends (`reinvested_dividend` type) do — they add to both quantity and cost basis.

---

## Transaction Form Conditional Logic

The transaction form adjusts required fields based on the selected type:

| Field              | buy | sell | dividend | reinvested_dividend |
|--------------------|-----|------|----------|---------------------|
| symbol             | req | req  | req      | req                 |
| transaction_date   | req | req  | req      | req                 |
| quantity           | req | req  | —        | req                 |
| price_per_share    | req | req  | —        | req                 |
| dividend_per_share | —   | —    | req      | req                 |
| total_amount       | req | req  | req      | req                 |

Implementation: `transaction_type` control's `valueChanges` subscription calls `setValidators()` / `clearValidators()` + `updateValueAndValidity()` on each conditional control. Clearing validators on hidden fields prevents the form from being permanently invalid for fields that don't apply to the selected type.

---

## Test Coverage

| File | Tests | Coverage |
|------|-------|----------|
| `portfolio.service.spec.ts` | 6 | loadAll, create, update, delete, loadById — all HTTP methods and signal mutations |
| `transaction.service.spec.ts` | 19 | deriveHoldings (7 edge cases) + HTTP methods + computed signal reactivity + clear() |
| `transaction-form.component.spec.ts` | 9 | Validator logic per type, form validity, submit/cancel outputs |

**Total new tests: 34** | All 52 frontend tests pass.

### deriveHoldings edge cases tested

- Empty input → empty result
- 100 buy → quantity 100
- 100 buy + 50 sell → quantity 50
- Full sell → holding disappears
- Dividend only → no holding (no shares granted)
- Reinvested dividend → adds to quantity and cost basis
- Multiple symbols → sorted alphabetically

---

## State Management Pattern

Follows the same signals-first pattern established in `AuthService` (Phase 2):

- Private `_portfolios = signal<Portfolio[]>([])` — mutable, internal
- Public `portfolios = this._portfolios.asReadonly()` — exposed to templates
- `holdings = computed(() => deriveHoldings(this._transactions()))` — automatically re-derives on every transaction change; no manual subscription required

RxJS is used only for HTTP streams (`Observable` from `HttpClient`). Signals hold all state.

---

## Known Limitations (addressed in future phases)

- **No live prices** — Holdings show cost basis only. Phase 5 adds market data (Finnhub) to calculate current value and unrealized gain/loss.
- **No edit transaction** — The backend has no PUT endpoint for transactions (by design — financial events should be immutable; delete and re-enter is the intended correction flow).
- **No update transaction endpoint** — The UI reflects this: only delete is available per transaction.
- **No pagination** — Transaction list loads all records for a portfolio. Pagination can be added if performance requires it at scale.
- **No app shell / navigation** — Phase 6 adds the sidebar navigation and dashboard overview.
