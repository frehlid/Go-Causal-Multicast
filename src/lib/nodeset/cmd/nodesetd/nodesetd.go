package main

import (
	"context"
	"fmt"
	"local/lib/finalizer"
	"local/lib/nodeset/server/rpc/serverStub"
	"local/lib/rpc"
)

func main() {
	ctx, _ := finalizer.WithCancel(context.Background())
	rpc.Start(ctx)
	serverStub.Register()
	fmt.Println(rpc.LocalAddr().String())
	<-ctx.Done()
}
