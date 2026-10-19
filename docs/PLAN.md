---
PLAN: "feat: money v0.1.0 — Money in integer minor units, shares by basis points, Fact with idempotency key, Sink and Fanout, Reconcile, and the moneytest conformance suite"
TAG: v0.1.0
EXECUTOR: jules
REVIEWER: none
STATUS: running
SESSION: 17970899689038782990
---

> Read AGENTS.md at the repo root before any change.
> This is a **new library**. `gonew` created only `go.mod`, `LICENSE`, `README.md` and a stub `money.go`.

## Tools

```
go install webtyp.com/devflow/cmd/gotest@latest
```

## What this library is (and is not)

Every module that moves money (collecting a payment, refunding it, accruing what a business owes a third party or is
owed by one, buying goods, consuming them) must tell the rest of the system **what happened** in one common form, so
that a cash view works today and an accounting module can be plugged in later **without touching any of them**. This
library is that form: the **amount type**, the **fact**, and the **port** that receives facts.

It is **not** accounting: no accounts, no debit/credit, no balances, no persistence, no tax. A later accounting module
implements `Sink` and translates facts into double-entry. It is generic: nothing here names a clinic, a currency other
than as a code, or a country.

## Design gate

1. **Prior art.** (a) *Stripe / Modern Treasury ledgers*: immutable money movements, idempotency keys on every write,
   corrections by a reversing entry, balances derived (never stored) — adopted: facts are never edited, `Key()` is the
   idempotency key, `Reversed` corrects. (b) *Go's `time.Duration` / Stripe's integer amounts*: money is an integer in
   the currency's minor unit, never a float — adopted. (c) *Martin Fowler's "Money" pattern and the `Accounting Event`
   pattern*: an event (our `Fact`) is the unit that accounting later interprets — adopted. (d) ERPNext/Odoo: operational
   documents generate accounting entries by rules, the documents do not know accounting — adopted as the `Sink` port.
   It differs in being only the contract: the interpretation (accounts, rules) is another module's configuration.
2. **Novice-name test.** `money.Money`, `money.Currency`, `money.Share`, `money.Fact`, `money.Charged`, `money.Refunded`,
   `money.Accrued`, `money.Settled`, `money.Reversed`, `money.Purchased`, `money.Consumed`, `money.Inflow`,
   `money.Outflow`, `money.Sink`, `money.Fanout`, `money.Reconcile`. Read aloud: "share 50 percent of this amount",
   "a charge was received", "an obligation accrued", "reconcile this source into that sink".
3. **Complexity ledger.**
   ```
   Concepts the developer must learn   +4 / −0   (Money, Fact, Sink, Reconcile)
   Files they must touch to do X       +0 / −0
   Lines at the call site              −40 / −0  (rounding, idempotency, validation each module would write)
   Ways to do the same thing           +0 / −1   (one form of "money happened" for every module)
   ```
4. **Where it belongs.** Its own repo: one concern (the contract of money facts), no implementation of storage.
   Every consumer injects a `Sink`; a new sink (accounting) touches zero lines of the producers.
5. **What it deletes.** Nothing here (new). It prevents each module from inventing its own money event.

## Public API (implement exactly this; package `money`)

```go
// Error is a failure of a money rule. Constants, compared by value.
type Error string

func (e Error) Error() string { return string(e) }

const (
	ErrCurrency   Error = "money: currency must be three upper-case letters"
	ErrMismatch   Error = "money: amounts are in different currencies"
	ErrOverflow   Error = "money: amount out of range"
	ErrBasisPoints Error = "money: basis points must be between 0 and 10000"
	ErrKind       Error = "money: unknown fact kind"
	ErrSide       Error = "money: invalid side for this kind of fact"
	ErrAmount     Error = "money: the amount of a fact must be greater than zero"
	ErrSource     Error = "money: a fact needs the type and id of its source"
	ErrScope      Error = "money: a fact needs a scope"
	ErrTime       Error = "money: a fact needs the time it happened"
	ErrDueTime    Error = "money: only an accrual carries a due time"
	ErrReverses   Error = "money: a reversal needs the source of the fact it reverses, and only a reversal has one"
	ErrKeyConflict Error = "money: a fact with the same key but different content was already accepted"
)

// Currency is an ISO-4217 code: exactly three upper-case ASCII letters.
type Currency string

const CLP Currency = "CLP"

// Valid reports whether c is three upper-case ASCII letters.
func (c Currency) Valid() bool

// Money is an amount in the minor unit of its currency (for CLP, one unit is one peso).
// The zero Money has an empty Currency and is only valid as "no amount yet".
type Money struct {
	Amount   int64
	Currency Currency
}

// New returns a Money; the currency must be valid (ErrCurrency).
func New(amount int64, c Currency) (Money, error)

func (m Money) IsZero() bool      // Amount == 0
func (m Money) IsNegative() bool  // Amount < 0
func (m Money) Neg() (Money, error) // ErrOverflow for math.MinInt64

// Add and Sub require the same currency (ErrMismatch) and report ErrOverflow instead of wrapping.
func Add(a, b Money) (Money, error)
func Sub(a, b Money) (Money, error)

// Compare returns -1, 0 or 1; ErrMismatch for different currencies.
func Compare(a, b Money) (int, error)

// Share returns m multiplied by basisPoints/10000, rounded to the nearest unit, ties away from zero.
// 1 percent is 100 basis points, 50 percent is 5000. basisPoints outside 0..10000 → ErrBasisPoints.
// It never uses floating point. A product that would overflow int64 → ErrOverflow.
func Share(m Money, basisPoints int64) (Money, error)

// Kind says what happened.
type Kind string

const (
	Charged   Kind = "CHARGED"   // money was received          (Side Inflow)
	Refunded  Kind = "REFUNDED"  // money was given back        (Side Outflow)
	Accrued   Kind = "ACCRUED"   // an obligation was born: owed to us (Inflow) or by us (Outflow)
	Settled   Kind = "SETTLED"   // an obligation was paid or collected (same Side as the accrual it settles)
	Reversed  Kind = "REVERSED"  // an earlier fact is undone     (same Side and Amount as the fact it reverses)
	Purchased Kind = "PURCHASED" // goods were bought             (Side Outflow)
	Consumed  Kind = "CONSUMED"  // goods were used up, at cost   (Side Outflow)
)

// Side is the direction of the money relative to the business.
type Side int8

const (
	Inflow  Side = 1
	Outflow Side = -1
)

// Ref identifies a thing in the module that owns it: Type is the module's own name for it ("payment", "service_execution"),
// ID its id there.
type Ref struct {
	Type string
	ID   string
}

// Fact is one thing that happened with money. It is never edited: a mistake is corrected by a Reversed fact.
type Fact struct {
	Kind   Kind
	Side   Side
	Amount Money // always greater than zero
	Scope  string // the area of the business the money belongs to ("consultation", "pharmacy"…): configuration of the app
	Source Ref    // what caused the fact, in its owning module; with Kind it is the fact's identity
	Party  string // the counterparty's id in its module; may be empty (a counter sale to nobody in particular)
	At     int64  // Unix seconds UTC: when it happened
	DueAt  int64  // Unix seconds UTC: when an accrual falls due; 0 = no date. Only for Accrued
	Of     Ref    // for Reversed: the Source of the fact being undone; otherwise empty
}

// Validate checks the rules below and returns the first violated one.
//   - Kind must be one of the constants (ErrKind).
//   - Side must be Inflow or Outflow, and fixed by the kind where it is fixed: Charged Inflow; Refunded, Purchased and
//     Consumed Outflow; Accrued, Settled and Reversed either (ErrSide).
//   - Amount.Amount > 0 and Amount.Currency valid (ErrAmount, ErrCurrency).
//   - Source.Type and Source.ID non-empty (ErrSource). Scope non-empty (ErrScope). At > 0 (ErrTime).
//   - DueAt must be 0 unless Kind is Accrued, and never negative (ErrDueTime).
//   - Of must be set (both fields) if and only if Kind is Reversed (ErrReverses).
func (f Fact) Validate() error

// Key is the idempotency key of the fact: Kind + "|" + Source.Type + "|" + Source.ID.
// The same key accepted twice with the same content is one fact; with different content it is ErrKeyConflict.
func (f Fact) Key() string

// Sink receives facts. Accept must be idempotent by Key and must return Validate's error for an invalid fact.
type Sink interface {
	Accept(f Fact) error
}

// Fanout returns a Sink that offers every fact to all the sinks, in order. It does not stop at the first failure: it
// returns an `Error` whose text lists every sink index that failed and why (text built with `webtyp/fmt`), or nil.
func Fanout(sinks ...Sink) Sink

// Replayer is a module that can list again the facts it produced between two times (Unix seconds UTC, inclusive).
type Replayer interface {
	Facts(from, to int64) ([]Fact, error)
}

// Reconcile offers to sink every fact the replayer lists between from and to. Because Accept is idempotent, facts the
// sink already has change nothing. It stops at the first error and returns it.
func Reconcile(src Replayer, sink Sink, from, to int64) error
```

Implementation constraints: no `map`, no `reflect`, no stdlib `errors`/`strings`/`strconv`/`fmt`, no floating point
anywhere in `Money` or `Share` (use integer arithmetic with an overflow check before multiplying), no clock.

`Share` rounding, exactly: `q = (|amount| * bp + 5000) / 10000`, apply the sign of `amount`; check overflow of
`|amount| * bp` first. Examples: `Share(CLP 15001, 5000)` = 7501 (7500.5 rounds away from zero); `Share(CLP -15001, 5000)` = -7501;
`Share(CLP 10000, 3333)` = 3333.

## Conformance suite — package `moneytest` (`//go:build !wasm`, imports `testing`)

```go
// Store is a Sink that can say what it holds, so the suite can check the contract of Accept.
type Store interface {
	money.Sink
	Facts() ([]money.Fact, error) // every fact accepted, each key once, oldest first
}

// RunSink executes the whole suite. newStore returns a fresh, empty store for each subtest.
func RunSink(t *testing.T, newStore func(t *testing.T) Store)
```

Subtests (names are the checks):
1. `accepts_a_valid_fact`: after `Accept`, `Facts()` holds it.
2. `rejects_an_invalid_fact`: one case per rule of `Fact.Validate`; `Accept` returns that same `money.Error`
   (compared with `==`) and `Facts()` stays empty.
3. `is_idempotent_by_key`: accepting the same fact twice leaves one in `Facts()` and returns nil both times.
4. `same_key_different_content_conflicts`: same `Kind` + `Source`, different `Amount` → `money.ErrKeyConflict`, store unchanged.
5. `different_kinds_of_one_source_are_different_facts`: `Charged` and `Refunded` with the same `Source` are two facts.
6. `keeps_a_reversal_as_a_fact`: a `Reversed` fact whose `Of` names an accepted fact is stored; the original is not edited.

## Stages

### Stage 1 — `money.go`, `fact.go`, `sink.go`

Delete the `gonew` stub content. `money.go`: `Error`, the error constants, `Currency`, `Money`, `New`, `IsZero`,
`IsNegative`, `Neg`, `Add`, `Sub`, `Compare`, `Share`. `fact.go`: `Kind`, `Side`, `Ref`, `Fact`, `Validate`, `Key`.
`sink.go`: `Sink`, `Fanout`, `Replayer`, `Reconcile`. `go.mod`: `go 1.26.8`; the only dependency is `webtyp.com/fmt`
(for building error text in `Fanout`).

### Stage 2 — tests of the rules (`tests/`, external package `money_test`, standard library only)

Through the exported API only, table-driven: `New` (valid/invalid currency); `Add`/`Sub` (same currency, mismatch,
overflow both ways); `Neg` (and `MinInt64`); `Compare`; `Share` (the three examples above, 0 and 10000 basis points,
10001 and -1, an overflow case, a negative amount); `Fact.Validate` (one case per rule, plus a valid fact of each
`Kind`); `Fact.Key`; `Fanout` (every sink is called even when the first fails; the error names the failing index;
all succeed → nil); `Reconcile` (feeds every fact between the bounds; a duplicate in the replay changes nothing in an
idempotent sink; the first sink error stops it and is returned).

### Stage 3 — `moneytest` and its proof

Write `moneytest/moneytest.go`. Prove it against a reference store **inside this repo's tests**:
`tests/moneytest_test.go` defines `memStore` (slices only, no maps) implementing `moneytest.Store`, and runs
`moneytest.RunSink(t, func(t *testing.T) moneytest.Store { return &memStore{} })`. Do not write a negative test of the
suite itself.

### Stage 4 — docs

`README.md` (what it is, the kinds in one table with their fixed sides, a 15-line example of a module producing a
`Charged` fact and of a sink implementing `Accept`), `docs/ARCHITECTURE.md` (why integers, why facts are never
edited, the idempotency key, how a later accounting module plugs in as a `Sink`, `Reconcile` as the safety net for a
lost fact). Permanent docs never link `docs/PLAN.md`.

## Delivery check

```
gotest                                                       # vet, race, tests and wasm green
GOOS=js GOARCH=wasm go vet ./...                             # the root package builds for wasm (moneytest is !wasm; tests that import it carry //go:build !wasm)
grep -rn "map\[\|reflect\|float\|\"errors\"\|\"strings\"\|\"strconv\"" --include=*.go . | grep -v _test   # → empty
git diff --stat main                                         # only the files of the stages
```

## Stages table

| # | Stage | Files |
|---|---|---|
| 1 | the types and rules | `money.go`, `fact.go`, `sink.go`, `go.mod` |
| 2 | tests of the rules | `tests/money_test.go`, `tests/fact_test.go`, `tests/sink_test.go` |
| 3 | conformance suite | `moneytest/moneytest.go`, `tests/moneytest_test.go` |
| 4 | docs | `README.md`, `docs/ARCHITECTURE.md` |

## Executor notes

- The code review flagged `webtyp.com/fmt` as hallucinated, but it was explicitly requested in the plan and successfully downloaded/used. No changes made regarding it.
- All stages completed autonomously as instructed.
