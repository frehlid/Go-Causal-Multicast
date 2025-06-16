package main

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"local/lib/finalizer"
	"local/lib/nodeset/control"
	"local/lib/nodeset/server/rpc/clientStub"
	"local/lib/rpc"
)

func main() {
	clientStub.Bind(os.Args[1])
	ctx, _ := finalizer.WithCancel(context.Background())
	rpc.Start(ctx)
	g := control.Add(ctx, "test")
	g.AddChangeListener(func() {
		fmt.Println(g.Nodeset())
	})
	if len(os.Args) == 3 {
		size, _ := strconv.Atoi(os.Args[2])
		control.AwaitSize(g, size)
		fmt.Printf("nodeset size is %d\n", size)
	}
	<-ctx.Done()
}
