package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"local/chat"
	"local/chat/rpc/serverStub"
	"local/lib/finalizer"
	"local/lib/nodeset/control"
	"local/lib/rpc"
	"local/multicast"
	"local/multicast/transport"
	"os"
	"strconv"
)

func main() {
	ctx, cancel := finalizer.WithCancel(context.Background())
	defer func() { cancel(); <-ctx.Done() }()
	control.Bind(os.Args[1])
	serverStub.Register()
	rpc.Start(ctx)

	g := multicast.NewGroup(control.Add(ctx, "chat"))

	if len(os.Args) == 3 {
		slowNodeId, err := strconv.Atoi(os.Args[2])
		if err != nil {
			panic(err)
		}
		control.AwaitSize(g.Nodeset, slowNodeId+1)
		transport.SetSlowNode(g.Nodeset.Nodeset()[slowNodeId].Addr)
	}

	fmt.Printf("Welcome to the Chat ...\n")
	reader := bufio.NewReader(os.Stdin)
	for {
		select {
		case <-ctx.Done():
			return
		default:
			msg, err := reader.ReadString('\n')
			select {
			case <-ctx.Done():
				return
			default:
				if err == io.EOF {
					return
				} else if err != nil {
					panic(err)
				}
				chat.Post(g, msg)
			}
		}
	}
}
