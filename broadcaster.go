package broadcast

import (
	"errors"
	"sync"
	"time"

	"github.com/rs/xid"
)

// Broadcaster defines all broadcast operations.
type Broadcaster interface {
	Subscribe(func(interface{})) *Subscription
	Unsubscribe(*Subscription)
	JoinRoom(s *Subscription, rooms ...string)
	LeaveRoom(s *Subscription, rooms ...string)
	ToAll(data interface{}, except ...string)
	ToRoom(data interface{}, room string, except ...string)
	RoomsOf(s *Subscription) []string
}

// Option is used to change broadcaster settings.
type Option func(b *broadcaster) error

// WithPoolSize limits the amount of go routines that can be used when sending
// messages to a huge number of subscribers. Default is 1000.
func WithPoolSize(size int) Option {
	return func(b *broadcaster) error {
		if size <= 0 {
			return errors.New("pool size must be positive")
		}

		b.pool.tickets = make(chan struct{}, size)
		return nil
	}
}

// WithPoolTimeout sets the duration a go routine responsible for
// sending messages to subscribers will linger after it is done with sending mesasges.
// Default is 5 minutes.
func WithPoolTimeout(timeout time.Duration) Option {
	return func(b *broadcaster) error {
		b.pool.timeout = timeout
		return nil
	}
}

// WithDispatcher sets a Dispatcher implementation. Default dispatcher performs no actions.
func WithDispatcher(dispatcher Dispatcher) Option {
	return func(b *broadcaster) error {
		b.dispatcher = dispatcher
		return nil
	}
}

// WithDefaultRoomName sets the name of the default room.
// All subscribers are added to the default room upon creation.
func WithDefaultRoomName(name string) Option {
	return func(b *broadcaster) error {
		if len(name) == 0 {
			return errors.New("default room name cannot be empty")
		}

		b.defaultRoomName = name
		return nil
	}
}

// New creates a new Broadcaster.
func New(options ...Option) (Broadcaster, error) {
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

	for _, option := range options {
		err := option(b)

		if err != nil {
			return nil, err
		}
	}

	b.dispatcher.Received(func(data interface{}, toAll bool, room string, except ...string) {
		if toAll {
			b.toAllLocal(data, except...)
			return
		}

		b.toRoomLocal(data, room, except...)
	})

	return b, nil
}

type broadcaster struct {
	pool            *pool
	mux             *sync.RWMutex
	rooms           map[string]*room
	dispatcher      Dispatcher
	defaultRoomName string
}

// Subscribe creates a new subscription.
// All subscriptions are added to the default room upon creation.
func (b *broadcaster) Subscribe(callback func(interface{})) *Subscription {
	sub := &Subscription{
		id:       xid.New().String(),
		callback: callback,
	}

	b.JoinRoom(sub, b.defaultRoomName)

	return sub
}

// Unsubscribe removes a subscription from all rooms.
func (b *broadcaster) Unsubscribe(s *Subscription) {
	b.mux.RLock()
	defer b.mux.RUnlock()

	for _, room := range b.rooms {
		room.removeSubscription(s)
	}
}

// JoinRoom adds a subscription to one or multiple rooms.
// Subsequent calls with the same room and subscription have no effect.
func (b *broadcaster) JoinRoom(sub *Subscription, rooms ...string) {
	for _, r := range rooms {
		b.mux.RLock()
		existingRoom := b.rooms[r]
		b.mux.RUnlock()

		if existingRoom == nil {
			var roomMux sync.RWMutex
			existingRoom = &room{
				subscriptions: make(map[string]*Subscription),
				mux:           &roomMux,
			}

			b.mux.Lock()
			b.rooms[r] = existingRoom
			b.mux.Unlock()
		}

		existingRoom.addSubscription(sub)
	}
}

// LeaveRoom removes a subscription from a room.
// This method has no effect if the subscription is not part of the room.
// Removing a subscription from the default room will prevent
// the subscription from receiving messages when ToAll is called.
func (b *broadcaster) LeaveRoom(sub *Subscription, rooms ...string) {
	b.mux.RLock()
	defer b.mux.RUnlock()

	for _, r := range rooms {
		existingRoom := b.rooms[r]
		if existingRoom == nil {
			continue
		}

		existingRoom.removeSubscription(sub)
	}
}

// ToAll sends a message to all subscriptions except the subscriptions
// that are part of the rooms specified with "except".
// ToAll won't send messages to the subscriptions manually removed from the default room.
func (b *broadcaster) ToAll(data interface{}, except ...string) {
	go b.dispatcher.Dispatch(data, true, "", except...)
	b.toAllLocal(data, except...)
}

func (b *broadcaster) toAllLocal(data interface{}, except ...string) {
	b.mux.RLock()
	defaultRoom, ok := b.rooms[b.defaultRoomName]
	if !ok {
		return
	}
	b.mux.RUnlock()

	defaultRoom.mux.RLock()
	defer defaultRoom.mux.RUnlock()

	for _, sub := range defaultRoom.subscriptions {
		s := sub
		b.pool.Do(func() {
			if b.isInRooms(s, except...) {
				return
			}
			s.send(data)
		})
	}
}

// ToRoom sends a message to all subscriptions within a room except
// the subscriptions that are part of the rooms specified with "except".
func (b *broadcaster) ToRoom(data interface{}, room string, except ...string) {
	go b.dispatcher.Dispatch(data, false, room, except...)
	b.toRoomLocal(data, room, except...)
}

func (b *broadcaster) toRoomLocal(data interface{}, room string, except ...string) {
	b.mux.RLock()
	defer b.mux.RUnlock()

	existingRoom := b.rooms[room]
	if existingRoom == nil {
		return
	}

	defer existingRoom.mux.RUnlock()
	existingRoom.mux.RLock()

	for _, sub := range existingRoom.subscriptions {
		s := sub
		b.pool.Do(func() {
			if b.isInRooms(s, except...) {
				return
			}
			s.send(data)
		})
	}
}

func (b *broadcaster) isInRooms(sub *Subscription, rooms ...string) bool {
	b.mux.RLock()
	defer b.mux.RUnlock()

	for _, name := range rooms {
		room := b.rooms[name]
		if room == nil {
			continue
		}

		room.mux.RLock()
		existingSub := room.subscriptions[sub.id]
		room.mux.RUnlock()

		if existingSub != nil {
			return true
		}
	}

	return false
}

// RoomsOf returns the rooms a given subscription belongs to.
func (b *broadcaster) RoomsOf(s *Subscription) []string {
	b.mux.RLock()
	defer b.mux.RUnlock()

	roomNames := []string{}

	for name, room := range b.rooms {
		room.mux.RLock()
		_, ok := room.subscriptions[s.id]
		room.mux.RUnlock()

		if !ok {
			continue
		}

		roomNames = append(roomNames, name)
	}

	return roomNames
}
