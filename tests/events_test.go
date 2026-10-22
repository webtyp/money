//go:build !wasm

package money_test

import (
	"testing"

	"webtyp.com/events"
	"webtyp.com/events/mock"
	"webtyp.com/model"
	"webtyp.com/money"
)

type recordingWriter struct {
	strings []string // "name=value"
	ints    []string
}

func (w *recordingWriter) String(name, val string)                    { w.strings = append(w.strings, name+"="+val) }
func (w *recordingWriter) Int(name string, val int64)                 { w.ints = append(w.ints, name+"="+itoa(val)) }
func (w *recordingWriter) Float(name string, val float64)             {}
func (w *recordingWriter) Bool(name string, val bool)                 {}
func (w *recordingWriter) Bytes(name string, val []byte)              {}
func (w *recordingWriter) Null(name string)                           {}
func (w *recordingWriter) Raw(name, val string)                       {}
func (w *recordingWriter) Object(name string, val model.Encodable)    {}
func (w *recordingWriter) Array(name string, n int) model.ArrayWriter { return nil }

func itoa(v int64) string {
	if v == 0 {
		return "0"
	}
	neg := v < 0
	if neg {
		v = -v
	}
	var b []byte
	for v > 0 {
		b = append([]byte{byte('0' + v%10)}, b...)
		v /= 10
	}
	if neg {
		b = append([]byte{'-'}, b...)
	}
	return string(b)
}

func validFact(t *testing.T) money.Fact {
	t.Helper()
	amount, err := money.New(1500, money.CLP)
	if err != nil {
		t.Fatal(err)
	}
	return money.Fact{
		Kind:   money.Charged,
		Side:   money.Inflow,
		Amount: amount,
		Scope:  "pharmacy",
		Source: money.Ref{Type: "payment", ID: "p1"},
		Party:  "c1",
		Means:  money.MeansCard,
		Actor:  "u1",
		At:     1700000000,
	}
}

func TestPublishReachesSubscribedSink(t *testing.T) {
	broker := &mock.Broker{}
	sink := &mockSink{}
	var got error
	if err := money.Subscribe(broker, sink, func(err error) { got = err }); err != nil {
		t.Fatal(err)
	}
	f := validFact(t)
	if err := money.Publish(broker, f); err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Fatalf("unexpected error: %v", got)
	}
	if sink.f == nil || *sink.f != f {
		t.Fatalf("sink received %+v, want %+v", sink.f, f)
	}
}

func TestPublishDoesNotPublishAnInvalidFact(t *testing.T) {
	broker := &mock.Broker{}
	sink := &mockSink{}
	if err := money.Subscribe(broker, sink, func(error) {}); err != nil {
		t.Fatal(err)
	}
	f := validFact(t)
	f.Scope = ""
	if err := money.Publish(broker, f); err != money.ErrScope {
		t.Fatalf("Publish = %v, want ErrScope", err)
	}
	if sink.f != nil {
		t.Fatal("an invalid fact was published")
	}
}

func TestTwoSinksBothReceive(t *testing.T) {
	broker := &mock.Broker{}
	a, b := &mockSink{}, &mockSink{}
	if err := money.Subscribe(broker, a, func(error) {}); err != nil {
		t.Fatal(err)
	}
	if err := money.Subscribe(broker, b, func(error) {}); err != nil {
		t.Fatal(err)
	}
	if err := money.Publish(broker, validFact(t)); err != nil {
		t.Fatal(err)
	}
	if a.f == nil || b.f == nil {
		t.Fatal("both sinks must receive the fact")
	}
}

func TestSinkErrorReachesOnError(t *testing.T) {
	broker := &mock.Broker{}
	sink := &mockSink{err: money.ErrKeyConflict}
	var got error
	if err := money.Subscribe(broker, sink, func(err error) { got = err }); err != nil {
		t.Fatal(err)
	}
	if err := money.Publish(broker, validFact(t)); err != nil {
		t.Fatal(err)
	}
	if got != money.ErrKeyConflict {
		t.Fatalf("onError got %v, want ErrKeyConflict", got)
	}
}

func TestSubscribeNeedsASinkAndAnErrorHandler(t *testing.T) {
	broker := &mock.Broker{}
	if err := money.Subscribe(broker, nil, func(error) {}); err != money.ErrHandler {
		t.Fatalf("nil sink: %v, want ErrHandler", err)
	}
	if err := money.Subscribe(broker, &mockSink{}, nil); err != money.ErrHandler {
		t.Fatalf("nil onError: %v, want ErrHandler", err)
	}
}

type foreignPayload struct{}

func (*foreignPayload) IsNil() bool                      { return false }
func (*foreignPayload) EncodeFields(w model.FieldWriter) {}

func TestForeignPayloadReachesOnError(t *testing.T) {
	broker := &mock.Broker{}
	var got error
	if err := money.Subscribe(broker, &mockSink{}, func(err error) { got = err }); err != nil {
		t.Fatal(err)
	}
	broker.Publish(events.Event{Topic: money.Topic, Payload: &foreignPayload{}})
	if got != money.ErrPayload {
		t.Fatalf("onError got %v, want ErrPayload", got)
	}
}

func TestFactEncodesEveryField(t *testing.T) {
	f := validFact(t)
	w := &recordingWriter{}
	f.EncodeFields(w)
	wantStrings := []string{"kind=CHARGED", "currency=CLP", "scope=pharmacy", "source_type=payment", "source_id=p1", "party=c1", "means=CARD", "actor=u1", "of_type=", "of_id="}
	wantInts := []string{"side=1", "amount=1500", "at=1700000000", "due_at=0"}
	if len(w.strings) != len(wantStrings) || len(w.ints) != len(wantInts) {
		t.Fatalf("wrote %v / %v", w.strings, w.ints)
	}
	for _, want := range wantStrings {
		found := false
		for _, got := range w.strings {
			if got == want {
				found = true
			}
		}
		if !found {
			t.Errorf("missing %q in %v", want, w.strings)
		}
	}
	for _, want := range wantInts {
		found := false
		for _, got := range w.ints {
			if got == want {
				found = true
			}
		}
		if !found {
			t.Errorf("missing %q in %v", want, w.ints)
		}
	}
	var nilFact *money.Fact
	if !nilFact.IsNil() {
		t.Fatal("(*Fact)(nil).IsNil() must be true")
	}
}
