//go:build !wasm

package money_test

import (
	"testing"
	"webtyp.com/money"
	"webtyp.com/money/moneytest"
)

type memStore struct {
	facts []money.Fact
}

func (s *memStore) Accept(f money.Fact) error {
	if err := f.Validate(); err != nil {
		return err
	}

	for _, existing := range s.facts {
		if existing.Key() == f.Key() {
			if existing.Amount.Amount != f.Amount.Amount || existing.Amount.Currency != f.Amount.Currency ||
				existing.Kind != f.Kind || existing.Side != f.Side || existing.Scope != f.Scope ||
				existing.Source != f.Source || existing.Party != f.Party || existing.At != f.At ||
				existing.DueAt != f.DueAt || existing.Of != f.Of {
				return money.ErrKeyConflict
			}
			return nil
		}
	}

	s.facts = append(s.facts, f)
	return nil
}

func (s *memStore) Facts() ([]money.Fact, error) {
	return s.facts, nil
}

func TestMoneyTestSuite(t *testing.T) {
	moneytest.RunSink(t, func(t *testing.T) moneytest.Store {
		return &memStore{}
	})
}
