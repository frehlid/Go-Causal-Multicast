package main

import (
	"context"
	"fmt"
	"local/chat"
	"local/chat/rpc/serverStub"
	"local/lib/finalizer"
	"local/lib/nodeset/control"
	"local/lib/rpc"
	"local/multicast"
	"local/multicast/transport"
	"os"
)

//func init() {
//	numNodes = 3
//	ws(0, "", "a", "b", "c")
//	ws(1, "c", "d", "e", "f")
//	slowChannels = []channel{{0, 2}}
//}

func init() {
	numNodes = 6
	ws(0, "", "a", "b")
	ws(1, "a", "c")
	ws(2, "c", "d")
	ws(3, "", "e", "f")
	ws(4, "f", "g")
	ws(5, "g", "h")
	slowChannels = []channel{{0, 1}, {1, 2}, {3, 4}, {4, 5}}
}

func main() {
	displayCh := make(chan string)
	ctx, cancel := finalizer.WithCancel(context.Background())
	defer func() { cancel(); <-ctx.Done() }()
	control.Bind(os.Args[1])
	serverStub.Register()
	rpc.Start(ctx)

	g := multicast.NewGroup(control.Add(ctx, "chat"))

	chat.SetDisplayFunc(func(msg string) {
		fmt.Println(msg)
		displayCh <- msg
	})

	control.AwaitSize(g.Nodeset, numNodes)
	for _, c := range slowChannels {
		if c.from == g.Nodeset.NodeId() {
			transport.SetSlowNode(g.Nodeset.Nodeset()[c.to].Addr)
		}
	}

	for _, s := range script[g.Nodeset.NodeId()] {
		if s.waitFor != "" {
			for {
				if <-displayCh == s.waitFor {
					break
				}
			}
		}
		for _, i := range s.issues {
			fmt.Println(i)
			chat.Post(g, i)
		}
	}
	for {
		<-displayCh
	}
}

type step struct {
	waitFor string
	issues  []string
}
type channel struct {
	from uint32
	to   uint32
}

var script = make(map[uint32][]step)
var slowChannels []channel
var numNodes int

func ws(nodeId uint32, waitFor string, issue ...string) {
	script[nodeId] = append(script[nodeId], step{waitFor, issue})
}
