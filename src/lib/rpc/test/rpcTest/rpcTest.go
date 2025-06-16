package main

import (
	"context"
	"fmt"
	"os"

	"local/lib/finalizer"
	"local/lib/rpc"
	"local/lib/transport"
	"local/lib/transport/types"
)

func foobar(i int, s string) string {
	return fmt.Sprintf("(%d, %s)", i, s)
}

type foobarArgs struct {
	I int
	S string
}

func foobarClientStub(server types.NetAddr, i int, s string) string {
	return rpc.FunctionClientStub[string](server, "foobar", foobarArgs{I: i, S: s})
}

func foobarServerStub(argData []byte) []byte {
	return rpc.FunctionServerStub[foobarArgs](argData, func(args foobarArgs) any {
		return foobar(args.I, args.S)
	})
}

func main() {
	ctx, _ := finalizer.WithCancel(context.Background())
	rpc.Start(ctx)
	if os.Args[1] == "s" {
		fmt.Println(rpc.LocalAddr().String())
		rpc.RegisterFunc("foobar", foobarServerStub)
	} else {
		server := transport.ResolveNetAddr(os.Args[2])
		fmt.Println(foobarClientStub(server, 10, "hello world"))
	}
	<-ctx.Done()
}
