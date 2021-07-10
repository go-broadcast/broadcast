package broadcast

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestBroadcaster_New(t *testing.T) {
	_, err := New()

	if err != nil {
		t.Fatalf("New returned error - %v, want nil error", err)
	}
}

func TestBroadcaster_New_WithInvalidOption(t *testing.T) {
	_, err := New(
		WithPoolSize(-1),
	)

	if err == nil {
		t.Fatalf("New with invalid option should return an error")
	}
}

func TestBroadcaster_New_ShouldSetDispatcherCallback(t *testing.T) {
	var callback func(data interface{}, toAll bool, room string, except ...string) = nil
	dispatcher := mockDispatcher{
		received: func(c func(data interface{}, toAll bool, room string, except ...string)) {
			callback = c
		},
	}
	New(
		WithDispatcher(&dispatcher),
	)

	if callback == nil {
		t.Fatalf("New should pass a callback to dispatcher's Received method")
	}
}

func TestWithPoolSize(t *testing.T) {
	b := createTestBroadcaster()
	want := 30

	WithPoolSize(want)(b)

	got := cap(b.pool.tickets)
	if got != want {
		t.Fatalf("WithPoolSize(%v); got pool with size %v", want, got)
	}
}

func TestWithPoolSize_WithNonPositiveSize(t *testing.T) {
	b := createTestBroadcaster()
	incorrectValues := []int{0, -1, -99999}

	for _, v := range incorrectValues {
		want := v
		t.Run(fmt.Sprintf("size = %v", want), func(t *testing.T) {
			err := WithPoolSize(want)(b)

			if err == nil {
				t.Fatalf("WithPoolSize(%v); expected an error", want)
			}
		})
	}
}

func TestWithPoolTimeout(t *testing.T) {
	b := createTestBroadcaster()
	want := time.Minute * 3

	WithPoolTimeout(want)(b)

	got := b.pool.timeout
	if got != want {
		t.Fatalf("WithPoolTimeout(%v); set timeout to %v", want, got)
	}
}

func TestWithDispatcher(t *testing.T) {
	b := createTestBroadcaster()
	want := mockDispatcher{}

	WithDispatcher(&want)(b)

	got := b.dispatcher

	if interface{}(&want) != got {
		t.Fatalf("WithDispatcher didn't set dispatcher")
	}
}

func TestWithDefaultRoomName(t *testing.T) {
	b := createTestBroadcaster()
	want := "testname"

	WithDefaultRoomName(want)(b)

	got := b.defaultRoomName
	if got != want {
		t.Fatalf("WithDefaultRoomName(%v); should set default room name got %v", want, got)
	}
}

func TestWithDefaultRoomName_WithEmptyRoom(t *testing.T) {
	b := createTestBroadcaster()

	err := WithDefaultRoomName("")(b)

	if err == nil {
		t.Fatal("WithDefaultRoomName(\"\"); should return an error")
	}

}

func TestBroadcaster_Subscribe(t *testing.T) {
	b := createTestBroadcaster()

	subscription := b.Subscribe(func(_ interface{}) {})

	if subscription == nil {
		t.Fatal("Subscribe should return a new subcription")
	}
}

func TestBroadcaster_Subscribe_ShouldSetCallback(t *testing.T) {
	b := createTestBroadcaster()
	callback := func(_ interface{}) {}

	subscription := b.Subscribe(callback)

	if subscription.callback == nil {
		t.Fatal("Subscribe should set a callback")
	}
}

func TestBroadcaster_Subscribe_ShouldGenerateUniqueID(t *testing.T) {
	b := createTestBroadcaster()
	callback := func(_ interface{}) {}

	subscriptionA := b.Subscribe(callback)
	subscriptionB := b.Subscribe(callback)

	if subscriptionA.ID() == subscriptionB.ID() {
		t.Fatal("Subscribe should generate a subscription with a unique ID")
	}
}

func TestBroadcaster_Subscribe_ShouldAddToDefaultRoom(t *testing.T) {
	b := createTestBroadcaster()

	subscription := b.Subscribe(func(_ interface{}) {})

	roomSubscription := b.rooms[b.defaultRoomName].subscriptions[subscription.ID()]
	if roomSubscription == nil {
		t.Fatal("Subscribe should add the new subscription to the default room")
	}
}

func TestBroadcaster_Unsubscribe(t *testing.T) {
	b := createTestBroadcaster()
	subscription := b.Subscribe(func(_ interface{}) {})
	testRoom := "test-room"
	b.JoinRoom(subscription, testRoom)

	b.Unsubscribe(subscription)

	defaultRoomSubscription := b.rooms[b.defaultRoomName].subscriptions[subscription.ID()]
	testRoomSubscription := b.rooms[testRoom].subscriptions[subscription.ID()]

	if defaultRoomSubscription != nil || testRoomSubscription != nil {
		t.Fatal("Unsubscribe should remove subscription from all rooms")
	}
}

func TestBroadcaster_Unsubscribe_WithNonExistingSubscription(t *testing.T) {
	b := createTestBroadcaster()
	subscription := b.Subscribe(func(_ interface{}) {})
	b.JoinRoom(subscription, "test-room")

	b.Unsubscribe(subscription)
	b.Unsubscribe(subscription)
}

func TestBroadcaster_JoinRoom(t *testing.T) {
	b := createTestBroadcaster()
	subscription := b.Subscribe(func(_ interface{}) {})
	roomName := "test-room"

	b.JoinRoom(subscription, roomName)

	room := b.rooms[roomName]
	if room == nil {
		t.Fatal("JoinRoom didn't create new room")
	}

	roomSubscription := room.subscriptions[subscription.ID()]
	if roomSubscription == nil {
		t.Fatal("JoinRoom didn't add subscription to room")
	}
}

func TestBroadcaster_LeaveRoom(t *testing.T) {
	b := createTestBroadcaster()
	subscription := b.Subscribe(func(_ interface{}) {})
	roomName := "test-room"
	b.JoinRoom(subscription, roomName)

	b.LeaveRoom(subscription, roomName)

	room := b.rooms[roomName]
	roomSubscription := room.subscriptions[subscription.ID()]
	if roomSubscription != nil {
		t.Fatal("LeaveRoom didn't remove subscription from room")
	}
}

func TestBroadcaster_LeaveRoom_WithNonExistentRoom(t *testing.T) {
	b := createTestBroadcaster()
	subscription := b.Subscribe(func(_ interface{}) {})

	b.LeaveRoom(subscription, "test-room")
}

func TestBroadcaster_ToAll(t *testing.T) {
	b := createTestBroadcaster()
	called := false
	done := make(chan struct{})
	b.Subscribe(func(_ interface{}) {
		called = true
		close(done)
	})

	b.ToAll(struct{}{})
	waitOrTimeout(done)

	if !called {
		t.Fatalf("ToAll did not send data to subscriber")
	}
}

func TestBroadcaster_ToAll_WithExcept(t *testing.T) {
	b := createTestBroadcaster()
	called := false
	done := make(chan struct{})
	subscription := b.Subscribe(func(_ interface{}) {
		called = true
		close(done)
	})
	room := "test-room"
	b.JoinRoom(subscription, room)

	b.ToAll(struct{}{}, room)
	waitOrTimeout(done)

	if called {
		t.Fatalf("ToAll send data to excluded subscriber")
	}
}

func TestBroadcaster_ToAll_ShouldDispatch(t *testing.T) {
	called := false
	done := make(chan struct{})
	dispatcher := mockDispatcher{
		dispatch: func(data interface{}, toAll bool, room string, except ...string) {
			called = true
			close(done)
		},
	}
	b, _ := New(WithDispatcher(&dispatcher))
	b.Subscribe(func(_ interface{}) {})

	b.ToAll(struct{}{})
	waitOrTimeout(done)

	if !called {
		t.Fatalf("ToAll didn't call Dispatcher.Dispatch")
	}
}

func TestBroadcaster_ToAll_WithMissingDefaultRoom(t *testing.T) {
	b, _ := New()

	b.ToAll(struct{}{})
	<-time.After(time.Millisecond * 200)
}

func TestBroadcaster_ToAll_ExceptMissingRoom(t *testing.T) {
	b, _ := New()
	called := false
	done := make(chan struct{})
	b.Subscribe(func(_ interface{}) {
		called = true
		close(done)
	})

	b.ToAll(struct{}{}, "missing-room")
	waitOrTimeout(done)

	if !called {
		t.Fatalf("ToAll did not send message to subscriber")
	}
}

func TestBroadcaster_ReceivedToAllMessage(t *testing.T) {
	var callback func(data interface{}, toAll bool, room string, except ...string) = nil
	dispatcher := mockDispatcher{
		received: func(c func(data interface{}, toAll bool, room string, except ...string)) {
			callback = c
		},
	}
	b, _ := New(WithDispatcher(&dispatcher))

	called := false
	done := make(chan struct{})
	b.Subscribe(func(_ interface{}) {
		called = true
		close(done)
	})

	callback(struct{}{}, true, "")
	waitOrTimeout(done)

	if !called {
		t.Fatalf("Message received from dispatcher was not send to room subscribers")
	}
}

func TestBroadcaster_ToRoom(t *testing.T) {
	b := createTestBroadcaster()
	called := false
	done := make(chan struct{})
	subscription := b.Subscribe(func(_ interface{}) {
		called = true
		close(done)
	})
	room := "test-room"
	b.JoinRoom(subscription, room)

	b.ToRoom(struct{}{}, "test-room")
	waitOrTimeout(done)

	if !called {
		t.Fatalf("ToRoom did not send data to subscriber")
	}
}

func TestBroadcaster_ToRoom_WithExcept(t *testing.T) {
	b := createTestBroadcaster()
	called := false
	done := make(chan struct{})
	subscription := b.Subscribe(func(_ interface{}) {
		called = true
		close(done)
	})
	room := "test-room"
	b.JoinRoom(subscription, room)
	b.JoinRoom(subscription, subscription.ID())

	b.ToRoom(struct{}{}, room, subscription.ID())
	waitOrTimeout(done)

	if called {
		t.Fatalf("ToRoom send data to excluded subscriber")
	}
}

func TestBroadcaster_ToRoom_NonSubscribed(t *testing.T) {
	b := createTestBroadcaster()
	called := false
	done := make(chan struct{})
	subscription := b.Subscribe(func(_ interface{}) {
		called = true
		close(done)
	})
	room := "test-room"
	b.JoinRoom(subscription, room)
	otherRoom := "other-room"

	b.ToRoom(struct{}{}, otherRoom)
	waitOrTimeout(done)

	if called {
		t.Fatalf("ToRoom send data to subscriber who wasn't in the room")
	}
}

func TestBroadcaster_ToRoom_ShouldDispatch(t *testing.T) {
	called := false
	done := make(chan struct{})
	dispatcher := mockDispatcher{
		dispatch: func(data interface{}, toAll bool, room string, except ...string) {
			called = true
			close(done)
		},
	}
	b, _ := New(WithDispatcher(&dispatcher))
	subscription := b.Subscribe(func(_ interface{}) {})
	room := "test-room"
	b.JoinRoom(subscription, room)

	b.ToRoom(struct{}{}, room)
	waitOrTimeout(done)

	if !called {
		t.Fatalf("ToRoom didn't call Dispatcher.Dispatch")
	}
}

func TestBroadcaster_ReceivedRoomMessage(t *testing.T) {
	var callback func(data interface{}, toAll bool, room string, except ...string) = nil
	dispatcher := mockDispatcher{
		received: func(c func(data interface{}, toAll bool, room string, except ...string)) {
			callback = c
		},
	}
	b, _ := New(WithDispatcher(&dispatcher))

	called := false
	done := make(chan struct{})
	subscription := b.Subscribe(func(_ interface{}) {
		called = true
		close(done)
	})

	room := "test-room"
	b.JoinRoom(subscription, room)

	callback(struct{}{}, false, room)
	waitOrTimeout(done)

	if !called {
		t.Fatalf("Message received from dispatcher was not send to all subscribers")
	}
}

func TestBroadcaster_RoomsOf(t *testing.T) {
	b := createTestBroadcaster()
	subscription := b.Subscribe(func(_ interface{}) {})
	room := "test-room"
	b.JoinRoom(subscription, room)
	otherSubscription := b.Subscribe(func(_ interface{}) {})
	b.JoinRoom(otherSubscription, "other-room")

	rooms := b.RoomsOf(subscription)

	contains := func(items []string, item string) bool {
		for _, i := range items {
			if i == item {
				return true
			}
		}

		return false
	}

	want := 2
	got := len(rooms)
	if want != got {
		t.Fatalf("RoomsOf should return %v; got %v", want, got)
	}

	if !contains(rooms, b.defaultRoomName) {
		t.Fatal("RoomsOf should return the default room")
	}

	if !contains(rooms, room) {
		t.Fatal("RoomsOf should return the room the subscription joined")
	}
}

func createTestBroadcaster() *broadcaster {
	pool := &pool{
		tickets: make(chan struct{}, defaultPoolSize),
		tasks:   make(chan func()),
		timeout: defaultPoolTimeout,
	}
	var mux sync.RWMutex
	b := &broadcaster{
		pool:            pool,
		rooms:           make(map[string]*room),
		mux:             &mux,
		dispatcher:      &noopDispatcher{},
		defaultRoomName: "default",
	}

	return b
}

type mockDispatcher struct {
	dispatch func(data interface{}, toAll bool, room string, except ...string)
	received func(callback func(data interface{}, toAll bool, room string, except ...string))
}

func (d *mockDispatcher) Dispatch(data interface{}, toAll bool, room string, except ...string) {
	if d.dispatch == nil {
		return
	}

	d.dispatch(data, toAll, room, except...)
}

func (d *mockDispatcher) Received(callback func(data interface{}, toAll bool, room string, except ...string)) {
	if d.received == nil {
		return
	}

	d.received(callback)
}

func waitOrTimeout(done <-chan struct{}) {
	timeout := time.After(time.Millisecond * 200)

	select {
	case <-done:
		return
	case <-timeout:
		return
	}
}
