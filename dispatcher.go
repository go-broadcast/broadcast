package broadcast

// Dispatcher allows messages to be dispatched to external services.
// One possible use case is to send the messages to a broker allowing
// other instances of the application to receive them.
type Dispatcher interface {
	// Dispatch sends a message to an external service.
	Dispatch(data interface{}, toAll bool, room string, except ...string)
	// Received is called with the callback the Dispatcher needs to use
	// when a message is received from an external service.
	Received(callback func(data interface{}, toAll bool, room string, except ...string))
}

type noopDispatcher struct{}

func (d *noopDispatcher) Dispatch(data interface{}, toAll bool, room string, except ...string) {
}

func (d *noopDispatcher) Received(callback func(data interface{}, toAll bool, room string, except ...string)) {
}
