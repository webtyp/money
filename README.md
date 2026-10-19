# money

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
