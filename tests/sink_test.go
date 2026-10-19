package money_test

import (
	"testing"
	"webtyp.com/money"
)

type mockSink struct {
	err error
	f   *money.Fact
}

func (s *mockSink) Accept(f money.Fact) error {
	s.f = &f
	return s.err
}

type mockReplayer struct {
	facts []money.Fact
	err   error
}

func (r *mockReplayer) Facts(from, to int64) ([]money.Fact, error) {
	if r.err != nil {
		return nil, r.err
	}
	var res []money.Fact
	for _, f := range r.facts {
		if f.At >= from && f.At <= to {
			res = append(res, f)
		}
	}
	return res, nil
}

func TestFanout(t *testing.T) {
	validAmount, _ := money.New(100, "CLP")
	fact := money.Fact{
		Kind:   money.Charged,
		Side:   money.Inflow,
		Amount: validAmount,
		Scope:  "test",
		Source: money.Ref{Type: "payment", ID: "123"},
		At:     1000,
	}

	t.Run("all succeed", func(t *testing.T) {
		s1 := &mockSink{}
		s2 := &mockSink{}
		fanout := money.Fanout(s1, s2)
		err := fanout.Accept(fact)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if s1.f == nil || s2.f == nil {
			t.Error("expected both sinks to be called")
		}
	})

	t.Run("some fail", func(t *testing.T) {
		s1 := &mockSink{err: money.Error("error 1")}
		s2 := &mockSink{}
		s3 := &mockSink{err: money.Error("error 3")}
		fanout := money.Fanout(s1, s2, s3)
		err := fanout.Accept(fact)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		expectedErr := "sink 0 failed: error 1; sink 2 failed: error 3"
		if err.Error() != expectedErr {
			t.Errorf("expected error %q, got %q", expectedErr, err.Error())
		}
		if s1.f == nil || s2.f == nil || s3.f == nil {
			t.Error("expected all sinks to be called even if some fail")
		}
	})
}

func TestReconcile(t *testing.T) {
	validAmount, _ := money.New(100, "CLP")
	f1 := money.Fact{
		Kind:   money.Charged,
		Side:   money.Inflow,
		Amount: validAmount,
		Scope:  "test",
		Source: money.Ref{Type: "payment", ID: "1"},
		At:     1000,
	}
	f2 := money.Fact{
		Kind:   money.Charged,
		Side:   money.Inflow,
		Amount: validAmount,
		Scope:  "test",
		Source: money.Ref{Type: "payment", ID: "2"},
		At:     2000,
	}
	f3 := money.Fact{
		Kind:   money.Charged,
		Side:   money.Inflow,
		Amount: validAmount,
		Scope:  "test",
		Source: money.Ref{Type: "payment", ID: "3"},
		At:     3000,
	}

	t.Run("success", func(t *testing.T) {
		replayer := &mockReplayer{facts: []money.Fact{f1, f2, f3}}
		sink := &mockSink{}
		err := money.Reconcile(replayer, sink, 1500, 2500)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if sink.f == nil || sink.f.Source.ID != "2" {
			t.Errorf("expected sink to receive f2, got %v", sink.f)
		}
	})

	t.Run("sink error stops replay", func(t *testing.T) {
		replayer := &mockReplayer{facts: []money.Fact{f1, f2, f3}}
		expectedErr := money.Error("sink error")
		sink := &mockSink{err: expectedErr}
		err := money.Reconcile(replayer, sink, 0, 4000)
		if err != expectedErr {
			t.Errorf("expected %v, got %v", expectedErr, err)
		}
		if sink.f == nil || sink.f.Source.ID != "1" {
			t.Errorf("expected sink to receive f1 and stop, got %v", sink.f)
		}
	})

	t.Run("replayer error", func(t *testing.T) {
		expectedErr := money.Error("replayer error")
		replayer := &mockReplayer{err: expectedErr}
		sink := &mockSink{}
		err := money.Reconcile(replayer, sink, 0, 4000)
		if err != expectedErr {
			t.Errorf("expected %v, got %v", expectedErr, err)
		}
	})
}
