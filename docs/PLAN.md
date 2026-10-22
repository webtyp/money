---
PLAN: "feat!: a fact says how it was paid (Means) and who registered it (Actor)"
TAG: v0.2.0
EXECUTOR: jules
REVIEWER: none
---

> Read AGENTS.md at the repo root before any change.

# Plan — `Fact.Means` and `Fact.Actor`

## Why

A cash register closes a shift by counting the **cash** one cashier took. Today a `Fact` cannot tell cash from a
card or a bank transfer, nor who registered it, so `cash_manager` counts every charge of the tenant as cash (wrong
expected cash on every shift with a card payment or two cashiers). Card money also arrives days later, net of a
fee: a consumer will publish it as an `Accrued` inflow with `DueAt` = deposit day, and needs `Means` to say it is a
card.

## Delivery rules

- **Commit after every stage.** Never run `git reset`, `git restore`, `git checkout -- <file>` or `git stash`.
- Never pause to ask permission or to report progress. When the Delivery check passes: push and open the PR. Stuck
  for 30 minutes: open the PR anyway, listing what fails.
- A PR that does not compile, or whose tests could not run, is a failed delivery.
- The code blocks below are the implementation: copy them.

## Tools

```bash
go install webtyp.com/devflow/cmd/gotest@latest
```

## Ecosystem search

`grep -rn "Means\|Method\|Actor\|CreatedBy" --include=*.go webtyp veltylabs`: payment methods exist only as
`payment_manager` string constants (`cash`, `debit`, `credit`, `transfer`, `card`), private to that module. The
fact is the shared contract between modules, so the means belongs here (**extend** `money`).

## Design gate

1. **Prior art.** Stripe balance transactions (`type` + `source` payment method), Square tenders (`CASH`, `CARD`,
   `OTHER`), ERPNext Mode of Payment on every payment entry; all record who created the entry (`created_by`).
2. **Names.** `Means` ("means of payment"), `MeansCash`, `MeansCard`, `MeansTransfer`, `Actor` ("who registered it").
3. **Ledger.** +2 fields; ways to tell cash from card: 0 → 1.
4. **Where.** `money.Fact` is the contract every module publishes and every sink reads.
5. **Deletes.** Nothing; `Validate` gets stricter (breaking: a `Charged`/`Refunded` without `Means` is rejected).

## Stage 1 — `fact.go`

After the `Side` block add:

```go
// Means is how money moved. Debit and credit cards are both MeansCard: what matters to the cash and to the deposit
// date is that the money goes through a card processor.
type Means string

const (
	MeansCash     Means = "CASH"
	MeansCard     Means = "CARD"
	MeansTransfer Means = "TRANSFER"
)

func (m Means) valid() bool { return m == MeansCash || m == MeansCard || m == MeansTransfer }
```

`Fact` gains two fields after `Party`:

```go
	Means  Means  // how it was paid; required for Charged and Refunded, optional (empty or valid) otherwise
	Actor  string // id of the user who registered it; empty = the system did
```

`money.go` error block gains: `ErrMeans Error = "money: a charge or refund needs a valid means of payment"`.

`Validate`: update the doc comment with the line
`//   - Means is required and valid for Charged and Refunded; elsewhere empty or valid (ErrMeans).`
and add, right after the `Side` checks:

```go
	switch f.Kind {
	case Charged, Refunded:
		if !f.Means.valid() {
			return ErrMeans
		}
	default:
		if f.Means != "" && !f.Means.valid() {
			return ErrMeans
		}
	}
```

`EncodeFields`: after `party` add `w.String("means", string(f.Means))` and `w.String("actor", f.Actor)`.

**Commit.**

## Stage 2 — Tests and conformance (`tests/fact_test.go`, `moneytest/`)

- In `TestFactValidate`, every existing `Charged`/`Refunded` case gets `Means: money.MeansCash` (so they keep their
  expected result). New rows:

| name | fact | wantErr |
|---|---|---|
| CHARGED without means | valid CHARGED, `Means: ""` | `ErrMeans` |
| CHARGED with unknown means | `Means: "CHEQUE"` | `ErrMeans` |
| REFUNDED by card | valid REFUNDED, `Means: MeansCard` | nil |
| ACCRUED card receivable | `Kind: Accrued, Side: Inflow, Means: MeansCard, DueAt: 2000` | nil |
| ACCRUED with unknown means | `Means: "X"` | `ErrMeans` |
| ACCRUED without means | `Means: ""` | nil |

- Any fact literal in `moneytest/` or other tests that is `Charged`/`Refunded` gets `Means: money.MeansCash`.
- An encode test: encode a fact with `Means: MeansCard, Actor: "u1"` through the existing encode helper of the
  tests (see `tests/events_test.go`) and assert `means` = `CARD`, `actor` = `u1`.

**Commit.**

## Stage 3 — Docs

`README.md`: in the `Fact` table/section add `Means` (with the three values and the rule) and `Actor`; one example:
"card payment: `Accrued` inflow, `Means: MeansCard`, `DueAt` = the day the processor deposits". **Commit.**

## Delivery check

```bash
gotest
```

## Stages

| # | What | Files |
|---|---|---|
| 1 | `Means`, `Actor`, validation, encoding | `fact.go`, `money.go` |
| 2 | Tests | `tests/fact_test.go`, `tests/events_test.go`, `moneytest/*` |
| 3 | Docs | `README.md` |
