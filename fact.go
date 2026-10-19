package money

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
	Amount Money  // always greater than zero
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
func (f Fact) Validate() error {
	switch f.Kind {
	case Charged, Refunded, Accrued, Settled, Reversed, Purchased, Consumed:
		// valid kind
	default:
		return ErrKind
	}

	if f.Side != Inflow && f.Side != Outflow {
		return ErrSide
	}

	switch f.Kind {
	case Charged:
		if f.Side != Inflow {
			return ErrSide
		}
	case Refunded, Purchased, Consumed:
		if f.Side != Outflow {
			return ErrSide
		}
	case Accrued, Settled, Reversed:
		// can be either
	}

	if f.Amount.Amount <= 0 {
		return ErrAmount
	}

	if !f.Amount.Currency.Valid() {
		return ErrCurrency
	}

	if f.Source.Type == "" || f.Source.ID == "" {
		return ErrSource
	}

	if f.Scope == "" {
		return ErrScope
	}

	if f.At <= 0 {
		return ErrTime
	}

	if f.Kind != Accrued {
		if f.DueAt != 0 {
			return ErrDueTime
		}
	} else {
		if f.DueAt < 0 {
			return ErrDueTime
		}
	}

	if f.Kind == Reversed {
		if f.Of.Type == "" || f.Of.ID == "" {
			return ErrReverses
		}
	} else {
		if f.Of.Type != "" || f.Of.ID != "" {
			return ErrReverses
		}
	}

	return nil
}

// Key is the idempotency key of the fact: Kind + "|" + Source.Type + "|" + Source.ID.
// The same key accepted twice with the same content is one fact; with different content it is ErrKeyConflict.
func (f Fact) Key() string {
	return string(f.Kind) + "|" + f.Source.Type + "|" + f.Source.ID
}
