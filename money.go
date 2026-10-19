package money

// Error is a failure of a money rule. Constants, compared by value.
type Error string

func (e Error) Error() string { return string(e) }

const (
	ErrCurrency    Error = "money: currency must be three upper-case letters"
	ErrMismatch    Error = "money: amounts are in different currencies"
	ErrOverflow    Error = "money: amount out of range"
	ErrBasisPoints Error = "money: basis points must be between 0 and 10000"
	ErrKind        Error = "money: unknown fact kind"
	ErrSide        Error = "money: invalid side for this kind of fact"
	ErrAmount      Error = "money: the amount of a fact must be greater than zero"
	ErrSource      Error = "money: a fact needs the type and id of its source"
	ErrScope       Error = "money: a fact needs a scope"
	ErrTime        Error = "money: a fact needs the time it happened"
	ErrDueTime     Error = "money: only an accrual carries a due time"
	ErrReverses    Error = "money: a reversal needs the source of the fact it reverses, and only a reversal has one"
	ErrKeyConflict Error = "money: a fact with the same key but different content was already accepted"
	ErrHandler     Error = "money: Subscribe needs a sink and an error handler"
	ErrPayload     Error = "money: the payload of a money.fact event is not a fact"
)

// Currency is an ISO-4217 code: exactly three upper-case ASCII letters.
type Currency string

const CLP Currency = "CLP"

// Valid reports whether c is three upper-case ASCII letters.
func (c Currency) Valid() bool {
	if len(c) != 3 {
		return false
	}
	for i := 0; i < 3; i++ {
		if c[i] < 'A' || c[i] > 'Z' {
			return false
		}
	}
	return true
}

// Money is an amount in the minor unit of its currency (for CLP, one unit is one peso).
// The zero Money has an empty Currency and is only valid as "no amount yet".
type Money struct {
	Amount   int64
	Currency Currency
}

// New returns a Money; the currency must be valid (ErrCurrency).
func New(amount int64, c Currency) (Money, error) {
	if !c.Valid() {
		return Money{}, ErrCurrency
	}
	return Money{Amount: amount, Currency: c}, nil
}

func (m Money) IsZero() bool {
	return m.Amount == 0
}

func (m Money) IsNegative() bool {
	return m.Amount < 0
}

func (m Money) Neg() (Money, error) {
	if m.Amount == -9223372036854775808 { // MinInt64
		return Money{}, ErrOverflow
	}
	return Money{Amount: -m.Amount, Currency: m.Currency}, nil
}

// Add and Sub require the same currency (ErrMismatch) and report ErrOverflow instead of wrapping.
func Add(a, b Money) (Money, error) {
	if a.Currency != b.Currency {
		return Money{}, ErrMismatch
	}
	sum := a.Amount + b.Amount
	if (sum^a.Amount)&(sum^b.Amount) < 0 {
		return Money{}, ErrOverflow
	}
	return Money{Amount: sum, Currency: a.Currency}, nil
}

func Sub(a, b Money) (Money, error) {
	if a.Currency != b.Currency {
		return Money{}, ErrMismatch
	}
	diff := a.Amount - b.Amount
	if (a.Amount^b.Amount)&(a.Amount^diff) < 0 {
		return Money{}, ErrOverflow
	}
	return Money{Amount: diff, Currency: a.Currency}, nil
}

// Compare returns -1, 0 or 1; ErrMismatch for different currencies.
func Compare(a, b Money) (int, error) {
	if a.Currency != b.Currency {
		return 0, ErrMismatch
	}
	if a.Amount < b.Amount {
		return -1, nil
	}
	if a.Amount > b.Amount {
		return 1, nil
	}
	return 0, nil
}

// Share returns m multiplied by basisPoints/10000, rounded to the nearest unit, ties away from zero.
// 1 percent is 100 basis points, 50 percent is 5000. basisPoints outside 0..10000 → ErrBasisPoints.
// It never uses floating point. A product that would overflow int64 → ErrOverflow.
func Share(m Money, basisPoints int64) (Money, error) {
	if basisPoints < 0 || basisPoints > 10000 {
		return Money{}, ErrBasisPoints
	}

	amount := m.Amount
	if amount == 0 {
		return Money{Amount: 0, Currency: m.Currency}, nil
	}

	absAmount := amount
	if absAmount < 0 {
		if absAmount == -9223372036854775808 { // MinInt64
			if basisPoints == 0 {
				return Money{Amount: 0, Currency: m.Currency}, nil
			}
			return Money{}, ErrOverflow
		}
		absAmount = -absAmount
	}

	prod := absAmount * basisPoints
	if basisPoints != 0 && prod/basisPoints != absAmount {
		return Money{}, ErrOverflow
	}

	if prod > 9223372036854775807-5000 {
		return Money{}, ErrOverflow
	}

	q := (prod + 5000) / 10000

	if amount < 0 {
		q = -q
	}

	return Money{Amount: q, Currency: m.Currency}, nil
}
