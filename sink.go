package money

import "webtyp.com/time"

// Sink receives facts. Accept must be idempotent by Key and must return Validate's error for an invalid fact.
type Sink interface {
	Accept(f Fact) error
}

// Replayer is a module that can list again the facts it produced between two times (Unix seconds UTC, inclusive).
type Replayer interface {
	Facts(from, to time.Unix) ([]Fact, error)
}

// Reconcile offers to sink every fact the replayer lists between from and to. Because Accept is idempotent, facts the
// sink already has change nothing. It stops at the first error and returns it.
func Reconcile(src Replayer, sink Sink, from, to time.Unix) error {
	facts, err := src.Facts(from, to)
	if err != nil {
		return err
	}

	for _, f := range facts {
		if err := sink.Accept(f); err != nil {
			return err
		}
	}

	return nil
}
