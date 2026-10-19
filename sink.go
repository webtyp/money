package money

import (
	"webtyp.com/fmt"
)

// Sink receives facts. Accept must be idempotent by Key and must return Validate's error for an invalid fact.
type Sink interface {
	Accept(f Fact) error
}

// Fanout returns a Sink that offers every fact to all the sinks, in order. It does not stop at the first failure: it
// returns an `Error` whose text lists every sink index that failed and why (text built with `webtyp/fmt`), or nil.
func Fanout(sinks ...Sink) Sink {
	return fanoutSink{sinks: sinks}
}

type fanoutSink struct {
	sinks []Sink
}

func (s fanoutSink) Accept(f Fact) error {
	var errStr string
	hasError := false

	for i, sink := range s.sinks {
		if err := sink.Accept(f); err != nil {
			if hasError {
				errStr += "; "
			}
			errStr += fmt.Sprintf("sink %d failed: %s", i, err.Error())
			hasError = true
		}
	}

	if hasError {
		return Error(errStr)
	}
	return nil
}

// Replayer is a module that can list again the facts it produced between two times (Unix seconds UTC, inclusive).
type Replayer interface {
	Facts(from, to int64) ([]Fact, error)
}

// Reconcile offers to sink every fact the replayer lists between from and to. Because Accept is idempotent, facts the
// sink already has change nothing. It stops at the first error and returns it.
func Reconcile(src Replayer, sink Sink, from, to int64) error {
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
