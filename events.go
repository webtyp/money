package money

import "webtyp.com/events"

// Publish announces a fact on Topic through the module's events.Publisher. An invalid fact is NOT published: the error of
// Fact.Validate is returned.
func Publish(p events.Publisher, f Fact) error {
	if err := f.Validate(); err != nil {
		return err
	}
	p.Publish(events.Event{Topic: Topic, Payload: &f})
	return nil
}

// Subscribe feeds every fact published on Topic to sink. An error is never dropped silently: whatever sink.Accept returns,
// and a payload that is not a *Fact, goes to onError. A nil sink or a nil onError returns ErrHandler.
func Subscribe(s events.Subscriber, sink Sink, onError func(error)) error {
	if sink == nil || onError == nil {
		return ErrHandler
	}
	s.Subscribe(Topic, func(e events.Event) {
		f, ok := e.Payload.(*Fact)
		if !ok || f == nil {
			onError(ErrPayload)
			return
		}
		if err := sink.Accept(*f); err != nil {
			onError(err)
		}
	})
	return nil
}
