package clientStub

import (
	"local/chat/rpc/api"
	"local/lib/rpc"
	"local/multicast"
)

func Display(g *multicast.Group, msg string) {
	rpc.MulticastClientStub[any](g, api.Display, api.DisplayArgs{Msg: msg})
}
