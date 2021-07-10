package broadcast

import "sync"

type room struct {
	mux           *sync.RWMutex
	subscriptions map[string]*Subscription
}

func (r *room) addSubscription(sub *Subscription) {
	r.mux.Lock()
	defer r.mux.Unlock()

	if existing := r.subscriptions[sub.id]; existing != nil {
		return
	}

	r.subscriptions[sub.id] = sub
}

func (r *room) removeSubscription(sub *Subscription) {
	r.mux.Lock()
	defer r.mux.Unlock()
	delete(r.subscriptions, sub.id)
}
