# Broadcast

A small utility that allows you to distribute messages to groups of long-lived connections. The implementation is connection ignorant so it could be used with different communication mechanisms like web sockets and gRPC server streams.

## Installation

```bash
go get github.com/go-broadcast/broadcast
```

## Basic usage

```go
import (
	"log"
	"time"

	"github.com/go-broadcast/broadcast"
)

func main() {
	broadcaster, err := broadcast.New()
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
}
```

## More examples

- [Web sockets](https://github.com/go-broadcast/examples/tree/main/cmd/websockets)
- [gRPC server streams](https://github.com/go-broadcast/examples/tree/main/cmd/grpc)
- [Scale out with Redis](https://github.com/go-broadcast/examples/tree/main/cmd/redis)
