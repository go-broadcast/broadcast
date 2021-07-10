package broadcast

import (
	"sync"
	"testing"

	"github.com/rs/xid"
)

func TestRoom_addSubscription(t *testing.T) {
	room, subscription := createRoomTestData()

	room.addSubscription(subscription)

	existingSubscription := room.subscriptions[subscription.id]

	if existingSubscription != subscription {
		t.Fatalf("addSubscription didn't add subscription")
	}
}

func TestRoom_addSubscription_WithExistingSubscription(t *testing.T) {
	room, subscription := createRoomTestData()
	room.addSubscription(subscription)
	otherSubscription := Subscription{
		id:       subscription.id,
		callback: func(_ interface{}) {},
	}

	room.addSubscription(&otherSubscription)

	existingSubscription := room.subscriptions[subscription.id]

	if existingSubscription == &otherSubscription {
		t.Fatalf("addSubscription should not override existing subscription with the same ID")
	}
}

func TestRoom_removeSubscription(t *testing.T) {
	room, subscription := createRoomTestData()
	room.addSubscription(subscription)

	room.removeSubscription(subscription)

	existingSubscription := room.subscriptions[subscription.id]

	if existingSubscription != nil {
		t.Fatalf("removeSubscription should remove subscription")
	}
}

func TestRoom_removeSubscription_WithNonExistingSubscription(t *testing.T) {
	room, subscription := createRoomTestData()

	room.removeSubscription(subscription)
}

func createRoomTestData() (*room, *Subscription) {
	var mux sync.RWMutex
	room := room{
		mux:           &mux,
		subscriptions: make(map[string]*Subscription),
	}
	subscription := Subscription{
		id:       xid.New().String(),
		callback: func(_ interface{}) {},
	}

	return &room, &subscription
}
