//go:build !wasm

package moneytest

import (
	"testing"
	"webtyp.com/money"
)

// Store is a Sink that can say what it holds, so the suite can check the contract of Accept.
type Store interface {
	money.Sink
	Facts() ([]money.Fact, error) // every fact accepted, each key once, oldest first
}

// RunSink executes the whole suite. newStore returns a fresh, empty store for each subtest.
func RunSink(t *testing.T, newStore func(t *testing.T) Store) {
	t.Run("accepts_a_valid_fact", func(t *testing.T) {
		store := newStore(t)
		validAmount, _ := money.New(100, "CLP")
		f := money.Fact{
			Kind:   money.Charged,
			Side:   money.Inflow,
			Amount: validAmount,
			Scope:  "test",
			Source: money.Ref{Type: "payment", ID: "123"},
			At:     1000,
		}

		if err := store.Accept(f); err != nil {
			t.Fatalf("unexpected error accepting valid fact: %v", err)
		}

		facts, err := store.Facts()
		if err != nil {
			t.Fatalf("unexpected error listing facts: %v", err)
		}

		if len(facts) != 1 {
			t.Fatalf("expected 1 fact, got %d", len(facts))
		}
		if facts[0].Key() != f.Key() {
			t.Errorf("expected fact %v, got %v", f, facts[0])
		}
	})

	t.Run("rejects_an_invalid_fact", func(t *testing.T) {
		validAmount, _ := money.New(100, "CLP")
		validSource := money.Ref{Type: "payment", ID: "123"}
		tests := []struct {
			name string
			f    money.Fact
			err  money.Error
		}{
			{
				"invalid kind",
				money.Fact{Kind: "INVALID", Side: money.Inflow, Amount: validAmount, Scope: "test", Source: validSource, At: 1000},
				money.ErrKind,
			},
			{
				"invalid side",
				money.Fact{Kind: money.Charged, Side: money.Outflow, Amount: validAmount, Scope: "test", Source: validSource, At: 1000},
				money.ErrSide,
			},
			{
				"invalid amount",
				money.Fact{Kind: money.Charged, Side: money.Inflow, Amount: money.Money{Amount: 0, Currency: "CLP"}, Scope: "test", Source: validSource, At: 1000},
				money.ErrAmount,
			},
			{
				"invalid currency",
				money.Fact{Kind: money.Charged, Side: money.Inflow, Amount: money.Money{Amount: 100, Currency: ""}, Scope: "test", Source: validSource, At: 1000},
				money.ErrCurrency,
			},
			{
				"missing source",
				money.Fact{Kind: money.Charged, Side: money.Inflow, Amount: validAmount, Scope: "test", Source: money.Ref{}, At: 1000},
				money.ErrSource,
			},
			{
				"missing scope",
				money.Fact{Kind: money.Charged, Side: money.Inflow, Amount: validAmount, Scope: "", Source: validSource, At: 1000},
				money.ErrScope,
			},
			{
				"missing time",
				money.Fact{Kind: money.Charged, Side: money.Inflow, Amount: validAmount, Scope: "test", Source: validSource, At: 0},
				money.ErrTime,
			},
			{
				"due time on non-accrued",
				money.Fact{Kind: money.Charged, Side: money.Inflow, Amount: validAmount, Scope: "test", Source: validSource, At: 1000, DueAt: 2000},
				money.ErrDueTime,
			},
			{
				"missing Of on reversed",
				money.Fact{Kind: money.Reversed, Side: money.Inflow, Amount: validAmount, Scope: "test", Source: validSource, At: 1000},
				money.ErrReverses,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				store := newStore(t)
				err := store.Accept(tt.f)
				if err != tt.err {
					t.Errorf("expected error %q, got %v", tt.err, err)
				}
				facts, err := store.Facts()
				if err != nil {
					t.Fatalf("unexpected error listing facts: %v", err)
				}
				if len(facts) != 0 {
					t.Errorf("expected 0 facts, got %d", len(facts))
				}
			})
		}
	})

	t.Run("is_idempotent_by_key", func(t *testing.T) {
		store := newStore(t)
		validAmount, _ := money.New(100, "CLP")
		f := money.Fact{
			Kind:   money.Charged,
			Side:   money.Inflow,
			Amount: validAmount,
			Scope:  "test",
			Source: money.Ref{Type: "payment", ID: "123"},
			At:     1000,
		}

		if err := store.Accept(f); err != nil {
			t.Fatalf("unexpected error on first accept: %v", err)
		}
		if err := store.Accept(f); err != nil {
			t.Fatalf("unexpected error on second accept: %v", err)
		}

		facts, err := store.Facts()
		if err != nil {
			t.Fatalf("unexpected error listing facts: %v", err)
		}

		if len(facts) != 1 {
			t.Fatalf("expected 1 fact, got %d", len(facts))
		}
	})

	t.Run("same_key_different_content_conflicts", func(t *testing.T) {
		store := newStore(t)
		validAmount1, _ := money.New(100, "CLP")
		validAmount2, _ := money.New(200, "CLP")
		f1 := money.Fact{
			Kind:   money.Charged,
			Side:   money.Inflow,
			Amount: validAmount1,
			Scope:  "test",
			Source: money.Ref{Type: "payment", ID: "123"},
			At:     1000,
		}
		f2 := money.Fact{
			Kind:   money.Charged,
			Side:   money.Inflow,
			Amount: validAmount2,
			Scope:  "test",
			Source: money.Ref{Type: "payment", ID: "123"},
			At:     1000,
		}

		if err := store.Accept(f1); err != nil {
			t.Fatalf("unexpected error on first accept: %v", err)
		}
		err := store.Accept(f2)
		if err != money.ErrKeyConflict {
			t.Fatalf("expected ErrKeyConflict, got %v", err)
		}

		facts, err := store.Facts()
		if err != nil {
			t.Fatalf("unexpected error listing facts: %v", err)
		}

		if len(facts) != 1 {
			t.Fatalf("expected 1 fact, got %d", len(facts))
		}
		if facts[0].Amount != validAmount1 {
			t.Errorf("expected fact with amount %v, got %v", validAmount1, facts[0].Amount)
		}
	})

	t.Run("different_kinds_of_one_source_are_different_facts", func(t *testing.T) {
		store := newStore(t)
		validAmount, _ := money.New(100, "CLP")
		source := money.Ref{Type: "payment", ID: "123"}
		f1 := money.Fact{
			Kind:   money.Charged,
			Side:   money.Inflow,
			Amount: validAmount,
			Scope:  "test",
			Source: source,
			At:     1000,
		}
		f2 := money.Fact{
			Kind:   money.Refunded,
			Side:   money.Outflow,
			Amount: validAmount,
			Scope:  "test",
			Source: source,
			At:     1000,
		}

		if err := store.Accept(f1); err != nil {
			t.Fatalf("unexpected error on first accept: %v", err)
		}
		if err := store.Accept(f2); err != nil {
			t.Fatalf("unexpected error on second accept: %v", err)
		}

		facts, err := store.Facts()
		if err != nil {
			t.Fatalf("unexpected error listing facts: %v", err)
		}

		if len(facts) != 2 {
			t.Fatalf("expected 2 facts, got %d", len(facts))
		}
	})

	t.Run("keeps_a_reversal_as_a_fact", func(t *testing.T) {
		store := newStore(t)
		validAmount, _ := money.New(100, "CLP")
		source1 := money.Ref{Type: "payment", ID: "123"}
		f1 := money.Fact{
			Kind:   money.Charged,
			Side:   money.Inflow,
			Amount: validAmount,
			Scope:  "test",
			Source: source1,
			At:     1000,
		}
		source2 := money.Ref{Type: "payment_reversal", ID: "124"}
		f2 := money.Fact{
			Kind:   money.Reversed,
			Side:   money.Inflow,
			Amount: validAmount,
			Scope:  "test",
			Source: source2,
			At:     1000,
			Of:     source1,
		}

		if err := store.Accept(f1); err != nil {
			t.Fatalf("unexpected error on first accept: %v", err)
		}
		if err := store.Accept(f2); err != nil {
			t.Fatalf("unexpected error on second accept: %v", err)
		}

		facts, err := store.Facts()
		if err != nil {
			t.Fatalf("unexpected error listing facts: %v", err)
		}

		if len(facts) != 2 {
			t.Fatalf("expected 2 facts, got %d", len(facts))
		}
	})
}
