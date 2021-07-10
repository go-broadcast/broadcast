package broadcast

import (
	"testing"

	"github.com/rs/xid"
)

func TestSubscription_send(t *testing.T) {
	subscription := createSubscriptionTestData()
	var got interface{}
	subscription.callback = func(data interface{}) {
		got = data
	}
	want := "data"

	subscription.send(want)

	if want != got {
		t.Fatalf("send called with %v; want %v", got, want)
	}
}

func TestSubscription_ID(t *testing.T) {
	subscription := createSubscriptionTestData()
	want := subscription.id

	got := subscription.ID()

	if want != got {
		t.Fatalf("ID() = %v; want %v", got, want)
	}
}

func createSubscriptionTestData() *Subscription {
	subscription := Subscription{
		id:       xid.New().String(),
		callback: func(_ interface{}) {},
	}

	return &subscription
}
