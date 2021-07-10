package broadcast

// Subscription represents a receiver of messages.
type Subscription struct {
	id       string
	callback func(interface{})
}

func (s *Subscription) send(data interface{}) {
	s.callback(data)
}

// ID returns the unique identifier of the subscription.
func (s *Subscription) ID() string {
	return s.id
}
