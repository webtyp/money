# money
<img src="docs/img/badges.svg">

Money amounts and economic facts shared by every module that moves money: integer minor units, shares by basis points, idempotent facts and the Sink port.

Every module that moves money (collecting a payment, refunding it, accruing what a business owes a third party or is owed by one, buying goods, consuming them) must tell the rest of the system **what happened** in one common form, so that a cash view works today and an accounting module can be plugged in later **without touching any of them**. This library is that form: the **amount type**, the **fact**, and the **port** that receives facts.

It is **not** accounting: no accounts, no debit/credit, no balances, no persistence, no tax. A later accounting module implements `Sink` and translates facts into double-entry. It is generic: nothing here names a clinic, a currency other than as a code, or a country.

## Fact Kinds and Sides

| Kind | What it means | Fixed Side |
|---|---|---|
| `CHARGED` | money was received | `Inflow` |
| `REFUNDED` | money was given back | `Outflow` |
| `ACCRUED` | an obligation was born | (Inflow or Outflow) |
| `SETTLED` | an obligation was paid or collected | (same Side as the accrual it settles) |
| `REVERSED` | an earlier fact is undone | (same Side as the fact it reverses) |
| `PURCHASED` | goods were bought | `Outflow` |
| `CONSUMED` | goods were used up, at cost | `Outflow` |

`Fact` also carries:
- `Means`: how the money moved (`MeansCash`, `MeansCard`, `MeansTransfer`). Required for `Charged` and `Refunded`, optional otherwise.
- `Actor`: the id of the user who registered it (empty if the system did).

## Example

```go
package example

import (
	"time"
	"webtyp.com/money"
)

func CollectPayment(amount int64, currency money.Currency, sink money.Sink) error {
	m, err := money.New(amount, currency)
	if err != nil {
		return err
	}
	f := money.Fact{
		Kind:   money.Charged,
		Side:   money.Inflow,
		Amount: m,
		Scope:  "sales",
		Source: money.Ref{Type: "payment", ID: "pay_123"},
		Means:  money.MeansCard,
		Actor:  "u1",
		At:     time.Now().Unix(),
	}
	return sink.Accept(f)
}

type MyStore struct{}

func (s *MyStore) Accept(f money.Fact) error {
	if err := f.Validate(); err != nil {
		return err
	}
	// ... save the fact idempotently ...
	return nil
}
```

## Publishing and receiving facts

```go
// A module that moves money announces the fact with the events.Publisher it already has.
if err := money.Publish(pub, fact); err != nil { /* the fact was invalid: nothing was published */ }

// The application attaches a sink (a cash view today, an accounting module later).
err := money.Subscribe(broker, mySink, func(err error) { report(err) }) // a rejected fact reaches the handler
```

Example: card payment: `Accrued` inflow, `Means: MeansCard`, `DueAt` = the day the processor deposits.
