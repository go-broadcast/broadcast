package broadcast_test

import (
	"log"
	"time"

	"github.com/go-broadcast/broadcast"
)

func Example() {
	broadcaster, cancel, err := broadcast.New()
	if err != nil {
		log.Fatal(err)
	}

	subscription := broadcaster.Subscribe(func(data interface{}) {
		log.Printf("Received message: %v", data)
	})

	broadcaster.JoinRoom(subscription, "chat-room")
	broadcaster.ToRoom("Hello, chat!", "chat-room")

	<-time.After(time.Second * 10)

	broadcaster.ToRoom("Bye, chat!", "chat-room")
	broadcaster.Unsubscribe(subscription)

	cancel()
	<-broadcaster.Done()
}
