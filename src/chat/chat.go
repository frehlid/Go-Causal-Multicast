package chat

import (
	"fmt"
	"local/chat/rpc/clientStub"
	"local/multicast"
)

func Post(g *multicast.Group, msg string) {
	clientStub.Display(g, msg)
}

func Display(msg string) {
	displayFunc(msg)
}

func SetDisplayFunc(f func(msg string)) {
	displayFunc = f
}

var displayFunc = func(msg string) { fmt.Print(msg) }
